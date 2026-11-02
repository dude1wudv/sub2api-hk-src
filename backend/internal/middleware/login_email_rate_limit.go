package middleware

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const (
	loginEmailBodyLimit = 64 << 10
	loginEmailKeyPrefix = "rate_limit:auth-email:v1:"
	loginAttemptLease   = 30 * time.Second
	loginFailureWindow  = 24 * time.Hour
	loginDelayThreshold = 5
	loginDelayInitial   = 30 * time.Second
	loginDelayMax       = 15 * time.Minute
)

var loginAttemptBeginScript = redis.NewScript(`
local cooldown_until = tonumber(redis.call('HGET', KEYS[1], 'cooldown_until_ms') or '0')
local now_parts = redis.call('TIME')
local now_ms = tonumber(now_parts[1]) * 1000 + math.floor(tonumber(now_parts[2]) / 1000)
if cooldown_until > now_ms then
  return {0, cooldown_until - now_ms}
end
if redis.call('SET', KEYS[2], ARGV[1], 'NX', 'PX', ARGV[2]) == false then
  return {0, 1000}
end
return {1, 0}
`)

var loginAttemptFinishScript = redis.NewScript(`
-- An expired attempt must not reset/update a newer attempt's account state.
if redis.call('GET', KEYS[2]) ~= ARGV[1] then
  return 0
end
local status = tonumber(ARGV[2])
if status >= 200 and status < 300 then
  redis.call('DEL', KEYS[1])
elseif status == 401 then
  local failures = redis.call('HINCRBY', KEYS[1], 'failures', 1)
  local ttl = redis.call('PTTL', KEYS[1])
  if ttl == -1 then
    redis.call('PEXPIRE', KEYS[1], ARGV[3])
  end
  if failures >= tonumber(ARGV[4]) then
    local exponent = math.floor((failures - tonumber(ARGV[4])) / tonumber(ARGV[5]))
    local delay = tonumber(ARGV[6]) * (2 ^ exponent)
    if delay > tonumber(ARGV[7]) then
      delay = tonumber(ARGV[7])
    end
    local now_parts = redis.call('TIME')
    local now_ms = tonumber(now_parts[1]) * 1000 + math.floor(tonumber(now_parts[2]) / 1000)
    redis.call('HSET', KEYS[1], 'cooldown_until_ms', now_ms + delay)
  else
    redis.call('HDEL', KEYS[1], 'cooldown_until_ms')
  end
end
redis.call('DEL', KEYS[2])
return 1
`)

// ProgressiveLoginLimiter tracks password failures by normalized account email
// so rotating client IPs cannot reset the account's delay. Email addresses are
// hashed before they are used as Redis keys and are never logged.
type ProgressiveLoginLimiter struct {
	redis *redis.Client
}

func NewProgressiveLoginLimiter(redisClient *redis.Client) *ProgressiveLoginLimiter {
	return &ProgressiveLoginLimiter{redis: redisClient}
}

// Middleware serializes attempts for one email with a short Redis lease,
// checks any active cooldown, and records only credential failures (HTTP 401).
// CAPTCHA, request validation, and service errors do not increase the counter.
func (l *ProgressiveLoginLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if l == nil || l.redis == nil {
			abortLoginRateLimitUnavailable(c)
			return
		}

		email, ok := readLoginEmail(c)
		if !ok {
			if !c.IsAborted() {
				c.Next()
			}
			return
		}

		stateKey := loginEmailKeyPrefix + hashLoginEmail(email)
		leaseKey := stateKey + ":lease"
		leaseToken, err := newLoginAttemptLeaseToken()
		if err != nil {
			abortLoginRateLimitUnavailable(c)
			return
		}
		result, err := loginAttemptBeginScript.Run(
			c.Request.Context(), l.redis, []string{stateKey, leaseKey},
			leaseToken, loginAttemptLease.Milliseconds(),
		).Slice()
		if err != nil {
			log.Printf("[AuthRateLimit] account login precheck failed: %v", err)
			abortLoginRateLimitUnavailable(c)
			return
		}
		if len(result) != 2 {
			log.Printf("[AuthRateLimit] account login precheck returned %d values", len(result))
			abortLoginRateLimitUnavailable(c)
			return
		}
		allowed, parseErr := parseInt64(result[0])
		retryMillis, retryErr := parseInt64(result[1])
		if parseErr != nil || retryErr != nil {
			log.Printf("[AuthRateLimit] account login precheck returned invalid values")
			abortLoginRateLimitUnavailable(c)
			return
		}
		if allowed != 1 {
			abortAccountLoginRateLimited(c, time.Duration(retryMillis)*time.Millisecond)
			return
		}

		status := 0
		defer func() {
			ctx, cancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), 2*time.Second)
			defer cancel()
			_, finishErr := loginAttemptFinishScript.Run(
				ctx, l.redis, []string{stateKey, leaseKey},
				leaseToken,
				status,
				loginFailureWindow.Milliseconds(),
				loginDelayThreshold,
				2,
				loginDelayInitial.Milliseconds(),
				loginDelayMax.Milliseconds(),
			).Result()
			if finishErr != nil {
				log.Printf("[AuthRateLimit] account login result update failed: %v", finishErr)
			}
		}()

		c.Next()
		status = c.Writer.Status()
	}
}

func readLoginEmail(c *gin.Context) (string, bool) {
	if c.Request.Body == nil {
		return "", false
	}
	original := c.Request.Body
	raw, err := io.ReadAll(io.LimitReader(original, loginEmailBodyLimit+1))
	c.Request.Body = &restoredLoginBody{
		Reader: io.MultiReader(bytes.NewReader(raw), original),
		closer: original,
	}
	if len(raw) > loginEmailBodyLimit {
		c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
			"error": "request too large", "message": "Login request exceeds the maximum size",
		})
		return "", false
	}
	if err != nil {
		abortInvalidLoginBody(c)
		return "", false
	}

	var payload struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		// Gin's decoder otherwise accepts the first JSON value and can ignore
		// trailing data. Reject it here instead of bypassing the email limiter.
		abortInvalidLoginBody(c)
		return "", false
	}
	email := strings.ToLower(strings.TrimSpace(payload.Email))
	if email == "" || len(email) > 254 {
		return "", false
	}
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email {
		return "", false
	}
	return email, true
}

func abortInvalidLoginBody(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
		"error": "invalid request", "message": "Login request must contain a complete JSON object",
	})
}

type restoredLoginBody struct {
	io.Reader
	closer io.Closer
}

func (b *restoredLoginBody) Close() error { return b.closer.Close() }

func hashLoginEmail(email string) string {
	sum := sha256.Sum256([]byte(email))
	return hex.EncodeToString(sum[:])
}

func newLoginAttemptLeaseToken() (string, error) {
	var token [16]byte
	if _, err := rand.Read(token[:]); err != nil {
		return "", fmt.Errorf("generate account login lease: %w", err)
	}
	return hex.EncodeToString(token[:]), nil
}

func abortLoginRateLimitUnavailable(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
		"error":   "rate limit unavailable",
		"message": "Login is temporarily unavailable, please try again later",
	})
}

func abortAccountLoginRateLimited(c *gin.Context, retryAfter time.Duration) {
	if retryAfter <= 0 {
		retryAfter = time.Second
	}
	seconds := int64(retryAfter / time.Second)
	if retryAfter%time.Second > 0 {
		seconds++
	}
	c.Header("Retry-After", strconv.FormatInt(seconds, 10))
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
		"error":   "login temporarily rate limited",
		"message": "Too many failed login attempts, please try again later",
	})
}
