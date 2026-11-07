package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func newLoginEmailRateLimitTestRouter(t *testing.T, client *redis.Client, handler gin.HandlerFunc) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/login", NewProgressiveLoginLimiter(client).Middleware(), handler)
	return router
}

func sendLoginEmailRateLimitTestRequest(router http.Handler, email, ip string) *httptest.ResponseRecorder {
	body, _ := json.Marshal(map[string]string{"email": email, "password": "not-read-by-middleware"})
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = ip + ":12345"
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func newLoginEmailRateLimitTestRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return server, client
}

func fastForwardLoginRedis(server *miniredis.Miniredis, current *time.Time, duration time.Duration) {
	server.FastForward(duration)
	*current = current.Add(duration)
	server.SetTime(*current)
}

func TestProgressiveLoginLimiterSharesFailuresAcrossIPsAndNormalizesEmail(t *testing.T) {
	server, client := newLoginEmailRateLimitTestRedis(t)
	handlerCalls := 0
	router := newLoginEmailRateLimitTestRouter(t, client, func(c *gin.Context) {
		handlerCalls++
		c.Status(http.StatusUnauthorized)
	})

	for i := 0; i < loginDelayThreshold-1; i++ {
		rec := sendLoginEmailRateLimitTestRequest(router, " User@example.com ", "198.51.100."+string(rune('1'+i)))
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	}
	key := loginEmailKeyPrefix + hashLoginEmail("user@example.com")
	require.Equal(t, "4", server.HGet(key, "failures"))
	require.Empty(t, server.HGet(key, "cooldown_until_ms"))

	rec := sendLoginEmailRateLimitTestRequest(router, "USER@example.com", "203.0.113.90")
	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.Equal(t, "5", server.HGet(key, "failures"))
	require.NotEmpty(t, server.HGet(key, "cooldown_until_ms"))
	remaining := handlerCalls

	rec = sendLoginEmailRateLimitTestRequest(router, "user@example.com", "192.0.2.44")
	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	require.Equal(t, "30", rec.Header().Get("Retry-After"))
	require.Equal(t, remaining, handlerCalls, "cooldown must stop requests before password verification")
}

func TestProgressiveLoginLimiterIsolatesDifferentEmails(t *testing.T) {
	_, client := newLoginEmailRateLimitTestRedis(t)
	router := newLoginEmailRateLimitTestRouter(t, client, func(c *gin.Context) {
		c.Status(http.StatusUnauthorized)
	})
	for i := 0; i < loginDelayThreshold; i++ {
		require.Equal(t, http.StatusUnauthorized, sendLoginEmailRateLimitTestRequest(router, "first@example.com", "198.51.100.10").Code)
	}
	require.Equal(t, http.StatusTooManyRequests, sendLoginEmailRateLimitTestRequest(router, "first@example.com", "203.0.113.10").Code)
	require.Equal(t, http.StatusUnauthorized, sendLoginEmailRateLimitTestRequest(router, "second@example.com", "203.0.113.10").Code)
}

func TestProgressiveLoginLimiterDelayProgressionAndCap(t *testing.T) {
	server, client := newLoginEmailRateLimitTestRedis(t)
	redisTime := time.Now()
	server.SetTime(redisTime)
	key := loginEmailKeyPrefix + hashLoginEmail("delay@example.com")
	router := newLoginEmailRateLimitTestRouter(t, client, func(c *gin.Context) {
		c.Status(http.StatusUnauthorized)
	})

	delayForFailure := func(failure int) time.Duration {
		if failure < loginDelayThreshold {
			return 0
		}
		exponent := (failure - loginDelayThreshold) / 2
		delay := loginDelayInitial * time.Duration(1<<exponent)
		if delay > loginDelayMax {
			return loginDelayMax
		}
		return delay
	}
	for failure := 1; failure <= 17; failure++ {
		require.Equal(t, http.StatusUnauthorized, sendLoginEmailRateLimitTestRequest(router, "delay@example.com", "198.51.100.20").Code)
		if delay := delayForFailure(failure); delay > 0 {
			untilMillis := server.HGet(key, "cooldown_until_ms")
			require.NotEmpty(t, untilMillis)
			if failure < 17 {
				fastForwardLoginRedis(server, &redisTime, delay)
			}
		}
	}
	require.Equal(t, "17", server.HGet(key, "failures"))
	require.Equal(t, http.StatusTooManyRequests, sendLoginEmailRateLimitTestRequest(router, "delay@example.com", "203.0.113.20").Code)
	require.Equal(t, "900", sendLoginEmailRateLimitTestRequest(router, "delay@example.com", "203.0.113.21").Header().Get("Retry-After"))
}

func TestProgressiveLoginLimiterBlockedRequestsDoNotExtendCooldownOrFailureTTL(t *testing.T) {
	server, client := newLoginEmailRateLimitTestRedis(t)
	key := loginEmailKeyPrefix + hashLoginEmail("ttl@example.com")
	router := newLoginEmailRateLimitTestRouter(t, client, func(c *gin.Context) {
		c.Status(http.StatusUnauthorized)
	})
	for i := 0; i < loginDelayThreshold; i++ {
		require.Equal(t, http.StatusUnauthorized, sendLoginEmailRateLimitTestRequest(router, "ttl@example.com", "198.51.100.30").Code)
	}
	initialCooldown := server.HGet(key, "cooldown_until_ms")
	initialTTL := server.TTL(key)
	server.FastForward(5 * time.Second)
	rec := sendLoginEmailRateLimitTestRequest(router, "ttl@example.com", "203.0.113.30")
	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	require.Equal(t, initialCooldown, server.HGet(key, "cooldown_until_ms"))
	require.Equal(t, initialTTL-5*time.Second, server.TTL(key))
}

func TestProgressiveLoginLimiterSuccessClearsFailureStateAndLease(t *testing.T) {
	server, client := newLoginEmailRateLimitTestRedis(t)
	redisTime := time.Now()
	server.SetTime(redisTime)
	key := loginEmailKeyPrefix + hashLoginEmail("success@example.com")
	attempts := 0
	router := newLoginEmailRateLimitTestRouter(t, client, func(c *gin.Context) {
		attempts++
		if attempts == loginDelayThreshold+1 {
			c.Status(http.StatusOK)
		} else {
			c.Status(http.StatusUnauthorized)
		}
	})
	for i := 0; i < loginDelayThreshold; i++ {
		require.Equal(t, http.StatusUnauthorized, sendLoginEmailRateLimitTestRequest(router, "success@example.com", "198.51.100.40").Code)
	}
	fastForwardLoginRedis(server, &redisTime, loginDelayInitial)
	require.Equal(t, http.StatusOK, sendLoginEmailRateLimitTestRequest(router, "success@example.com", "198.51.100.41").Code)
	require.False(t, server.Exists(key))
	require.False(t, server.Exists(key+":lease"))
	require.Equal(t, http.StatusUnauthorized, sendLoginEmailRateLimitTestRequest(router, "success@example.com", "198.51.100.42").Code)
	require.Equal(t, "1", server.HGet(key, "failures"))
}

func TestProgressiveLoginLimiterIgnoresNonCredentialFailuresAndPreservesBody(t *testing.T) {
	server, client := newLoginEmailRateLimitTestRedis(t)
	key := loginEmailKeyPrefix + hashLoginEmail("body@example.com")
	wantBody := `{"email":"body@example.com","password":"password-value"}`
	router := newLoginEmailRateLimitTestRouter(t, client, func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		require.NoError(t, err)
		require.JSONEq(t, wantBody, string(body))
		switch c.GetHeader("X-Test-Status") {
		case "401":
			c.Status(http.StatusUnauthorized)
		case "200":
			c.Status(http.StatusOK)
		case "500":
			c.Status(http.StatusInternalServerError)
		default:
			status, _ := strconv.Atoi(c.GetHeader("X-Test-Status"))
			c.Status(status)
		}
	})

	for _, status := range []int{400, 403, 429, 500, 401} {
		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(wantBody))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-Status", strconv.Itoa(status))
		req.RemoteAddr = "198.51.100.50:12345"
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		require.Equal(t, status, rec.Code)
	}
	require.Equal(t, "1", server.HGet(key, "failures"))

	// A successful response, including a password accepted before a TOTP step,
	// clears the password-failure state.
	req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(wantBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-Status", "200")
	req.RemoteAddr = "203.0.113.50:12345"
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.False(t, server.Exists(key))
}

func TestProgressiveLoginLimiterInvalidEmailDoesNotAllocateStateAndRestoresBody(t *testing.T) {
	server, client := newLoginEmailRateLimitTestRedis(t)
	var seenBody []byte
	router := newLoginEmailRateLimitTestRouter(t, client, func(c *gin.Context) {
		seenBody, _ = io.ReadAll(c.Request.Body)
		c.Status(http.StatusUnauthorized)
	})
	for _, body := range []string{`{}`, `{"email":"bad"}`, `{"email":""}`} {
		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "198.51.100.60:12345"
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		require.Equal(t, http.StatusUnauthorized, rec.Code)
		require.Equal(t, body, string(seenBody))
	}
	require.Empty(t, server.Keys())
}

func TestProgressiveLoginLimiterFailsClosedWhenRedisUnavailable(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1", DialTimeout: 20 * time.Millisecond, ReadTimeout: 20 * time.Millisecond, WriteTimeout: 20 * time.Millisecond})
	t.Cleanup(func() { _ = client.Close() })
	handlerCalled := false
	router := newLoginEmailRateLimitTestRouter(t, client, func(c *gin.Context) {
		handlerCalled = true
		c.Status(http.StatusUnauthorized)
	})
	rec := sendLoginEmailRateLimitTestRequest(router, "redis@example.com", "198.51.100.70")
	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	require.False(t, handlerCalled)
}

func TestProgressiveLoginLimiterSerializesConcurrentAttempts(t *testing.T) {
	_, client := newLoginEmailRateLimitTestRedis(t)
	var mu sync.Mutex
	handlerCalls := 0
	router := newLoginEmailRateLimitTestRouter(t, client, func(c *gin.Context) {
		mu.Lock()
		handlerCalls++
		mu.Unlock()
		time.Sleep(20 * time.Millisecond)
		c.Status(http.StatusUnauthorized)
	})
	const workers = 12
	var wg sync.WaitGroup
	results := make(chan int, workers)
	start := make(chan struct{})
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			rec := sendLoginEmailRateLimitTestRequest(router, "parallel@example.com", "198.51.100."+strconv.Itoa(i+1))
			results <- rec.Code
		}(i)
	}
	close(start)
	wg.Wait()
	close(results)
	allowed := 0
	for status := range results {
		if status == http.StatusUnauthorized {
			allowed++
		} else {
			require.Equal(t, http.StatusTooManyRequests, status)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, 1, allowed)
	require.Equal(t, 1, handlerCalls)
}

func TestProgressiveLoginLimiterFailureCounterExpires(t *testing.T) {
	server, client := newLoginEmailRateLimitTestRedis(t)
	key := loginEmailKeyPrefix + hashLoginEmail("expire@example.com")
	router := newLoginEmailRateLimitTestRouter(t, client, func(c *gin.Context) { c.Status(http.StatusUnauthorized) })
	require.Equal(t, http.StatusUnauthorized, sendLoginEmailRateLimitTestRequest(router, "expire@example.com", "198.51.100.80").Code)
	require.Equal(t, "1", server.HGet(key, "failures"))
	server.FastForward(loginFailureWindow)
	require.False(t, server.Exists(key))
	// Recreating a limiter is representative of application process restart;
	// the Redis state remains shared while no stale account lock is left behind.
	newRouter := newLoginEmailRateLimitTestRouter(t, client, func(c *gin.Context) { c.Status(http.StatusUnauthorized) })
	require.Equal(t, http.StatusUnauthorized, sendLoginEmailRateLimitTestRequest(newRouter, "expire@example.com", "203.0.113.80").Code)
	require.Equal(t, "1", server.HGet(key, "failures"))
	_, err := client.Ping(context.Background()).Result()
	require.NoError(t, err)
}

func TestProgressiveLoginLimiterRejectsBodyParsingBypassesBeforeLogin(t *testing.T) {
	server, client := newLoginEmailRateLimitTestRedis(t)
	handlerCalls := 0
	router := newLoginEmailRateLimitTestRouter(t, client, func(c *gin.Context) {
		handlerCalls++
		c.Status(http.StatusOK)
	})
	valid := `{"email":"body@example.com","password":"password-value"}`
	for _, tc := range []struct {
		name string
		body string
		want int
	}{
		{"trailing junk", valid + "ignored", http.StatusBadRequest},
		{"second JSON value", valid + `{}`, http.StatusBadRequest},
		{"incomplete JSON", strings.TrimSuffix(valid, "}"), http.StatusBadRequest},
		{"large unknown field", `{"email":"body@example.com","padding":"` + strings.Repeat("x", loginEmailBodyLimit) + `","password":"password-value"}`, http.StatusRequestEntityTooLarge},
		{"large trailing whitespace", valid + strings.Repeat(" ", loginEmailBodyLimit), http.StatusRequestEntityTooLarge},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			require.Equal(t, tc.want, rec.Code)
		})
	}
	require.Zero(t, handlerCalls)
	require.Empty(t, server.Keys())

	// Legitimate trailing whitespace and unknown fields stay compatible.
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"email":"body@example.com","password":"password-value","client":"sdk"}`+" \n"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, 1, handlerCalls)
}

type failedLoginBodyReader struct{}

func (failedLoginBodyReader) Read(p []byte) (int, error) {
	return copy(p, `{"email":"body@example.com","password":"value"}`), io.ErrUnexpectedEOF
}
func (failedLoginBodyReader) Close() error { return nil }

func TestProgressiveLoginLimiterRejectsReadFailureBeforeLogin(t *testing.T) {
	_, client := newLoginEmailRateLimitTestRedis(t)
	handlerCalled := false
	router := newLoginEmailRateLimitTestRouter(t, client, func(c *gin.Context) { handlerCalled = true })
	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	req.Body = failedLoginBodyReader{}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.False(t, handlerCalled)
}

func TestProgressiveLoginLimiterExpiredAttemptCannotChangeNewerState(t *testing.T) {
	server, client := newLoginEmailRateLimitTestRedis(t)
	key := loginEmailKeyPrefix + hashLoginEmail("stale@example.com")
	for _, status := range []int{http.StatusOK, http.StatusUnauthorized} {
		server.HSet(key, "failures", "6", "cooldown_until_ms", "12345")
		require.NoError(t, server.Set(key+":lease", "newer-attempt"))
		result, err := loginAttemptFinishScript.Run(context.Background(), client, []string{key, key + ":lease"},
			"expired-attempt", status, loginFailureWindow.Milliseconds(), loginDelayThreshold, 2,
			loginDelayInitial.Milliseconds(), loginDelayMax.Milliseconds()).Int()
		require.NoError(t, err)
		require.Zero(t, result)
		require.Equal(t, "6", server.HGet(key, "failures"))
		require.Equal(t, "12345", server.HGet(key, "cooldown_until_ms"))
		lease, err := server.Get(key + ":lease")
		require.NoError(t, err)
		require.Equal(t, "newer-attempt", lease)
	}
}

func TestProgressiveLoginLimiterPanicReleasesOwnedLeaseWithoutRecordingResult(t *testing.T) {
	server, client := newLoginEmailRateLimitTestRedis(t)
	key := loginEmailKeyPrefix + hashLoginEmail("panic@example.com")
	server.HSet(key, "failures", "4")
	server.SetTTL(key, loginFailureWindow)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.POST("/login", NewProgressiveLoginLimiter(client).Middleware(), func(c *gin.Context) {
		panic("synthetic login panic")
	})

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(`{"email":"panic@example.com","password":"value"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.False(t, server.Exists(key+":lease"))
	require.Equal(t, "4", server.HGet(key, "failures"), "an incomplete attempt must not clear or increment failure state")
	require.Empty(t, server.HGet(key, "cooldown_until_ms"))
}
