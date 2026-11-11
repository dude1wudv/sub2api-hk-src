package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

// 在途请求展示，和余额预留不是同一组 key。
//
//	usage:inflight:{uid}           HASH  field=requestID value=JSON
//	usage:inflight:users           ZSET  userID -> latest heartbeat Unix milliseconds
//	usage:inflight:done:{uid}:{id} STRING 结束墓碑，挡住迟到的刷新把行写回来
//
// Bound the administrator read to 500 active users; never SCAN Redis.
const (
	usageInflightUsersKey = "usage:inflight:users"
	usageInflightTTL      = time.Hour
	usageInflightDoneTTL  = 2 * time.Minute
	usageInflightUserCap  = 500
)

type usageInflightCache struct {
	rdb *redis.Client
}

func NewUsageInflightCache(rdb *redis.Client) service.UsageInflightStore {
	if rdb == nil {
		return nil
	}
	return &usageInflightCache{rdb: rdb}
}

const usageInflightHeartbeatStale = 60 * time.Second

// usageInflightLive 超过 1 分钟没有心跳的行当结束。旧数据没有 updated_at，就用开始时间。
func usageInflightLive(snap service.UsageInflightSnapshot, now time.Time) bool {
	if snap.ExpiresAtMs > 0 && snap.ExpiresAtMs <= now.UnixMilli() {
		return false
	}
	stamp := snap.UpdatedAt
	if stamp.IsZero() {
		stamp = snap.StartedAt
	}
	return !stamp.IsZero() && now.Sub(stamp) <= usageInflightHeartbeatStale
}

func usageInflightHashKey(userID int64) string {
	return fmt.Sprintf("usage:inflight:{%d}", userID)
}

func usageInflightDoneKey(userID int64, requestID string) string {
	return fmt.Sprintf("usage:inflight:done:{%d}:%s", userID, requestID)
}

func (c *usageInflightCache) Save(ctx context.Context, snap service.UsageInflightSnapshot) error {
	if c == nil || c.rdb == nil || snap.UserID <= 0 || snap.RequestID == "" {
		return nil
	}
	snap.ExpiresAtMs = time.Now().Add(usageInflightTTL).UnixMilli()
	heartbeat := snap.UpdatedAt
	if heartbeat.IsZero() {
		heartbeat = snap.StartedAt
	}
	payload, err := json.Marshal(snap)
	if err != nil {
		return err
	}
	return c.rdb.Eval(ctx, `
 if redis.call('EXISTS', KEYS[1]) == 1 then return 0 end
 redis.call('HSET', KEYS[2], ARGV[1], ARGV[2])
 redis.call('PEXPIRE', KEYS[2], ARGV[3])
 redis.call('ZADD', KEYS[3], 'GT', ARGV[5], ARGV[4])
 return 1`, []string{usageInflightDoneKey(snap.UserID, snap.RequestID), usageInflightHashKey(snap.UserID), usageInflightUsersKey}, snap.RequestID, payload, usageInflightTTL.Milliseconds(), snap.UserID, heartbeat.UnixMilli()).Err()

}

func (c *usageInflightCache) Delete(ctx context.Context, userID int64, requestID string) error {
	if c == nil || c.rdb == nil || userID <= 0 || requestID == "" {
		return nil
	}
	return c.rdb.Eval(ctx, `
 redis.call('SET', KEYS[1], '1', 'PX', ARGV[3])
 redis.call('HDEL', KEYS[2], ARGV[1])
 if redis.call('HLEN', KEYS[2]) == 0 then redis.call('ZREM', KEYS[3], ARGV[2]) end
 return 1`, []string{usageInflightDoneKey(userID, requestID), usageInflightHashKey(userID), usageInflightUsersKey}, requestID, userID, usageInflightDoneTTL.Milliseconds()).Err()

}

func (c *usageInflightCache) ListUser(ctx context.Context, userID int64) ([]service.UsageInflightSnapshot, error) {
	if c == nil || c.rdb == nil || userID <= 0 {
		return nil, nil
	}
	hashKey := usageInflightHashKey(userID)
	raw, err := c.rdb.HGetAll(ctx, hashKey).Result()
	if err != nil || len(raw) == 0 {
		if err == nil {
			_ = c.removeEmptyUser(ctx, userID)
		}
		return nil, err
	}
	now := time.Now()
	out := make([]service.UsageInflightSnapshot, 0, len(raw))
	var expired []string
	for field, payload := range raw {
		var snap service.UsageInflightSnapshot
		if json.Unmarshal([]byte(payload), &snap) != nil || snap.UserID != userID || !usageInflightLive(snap, now) {
			expired = append(expired, field)
			continue
		}
		out = append(out, snap)
	}
	if len(expired) > 0 {
		for _, field := range expired {
			c.dropStale(ctx, hashKey, field, raw[field])
		}
	}
	if len(out) == 0 {
		_ = c.removeEmptyUser(ctx, userID)
	}
	return out, nil
}

func (c *usageInflightCache) ListAll(ctx context.Context) ([]service.UsageInflightSnapshot, error) {
	if c == nil || c.rdb == nil {
		return nil, nil
	}
	// Prune crash leftovers before checking the cap. Limit the returned index to
	// cap+1 candidates so overload cannot cause an unbounded Redis reply. ZADD GT
	// prevents an older request snapshot from aging out another live request by
	// the same user; pruning and reading run atomically with concurrent saves.
	cutoff := time.Now().Add(-usageInflightHeartbeatStale).UnixMilli()
	ids, err := c.rdb.Eval(ctx, liveInflightUsersScript, []string{usageInflightUsersKey}, cutoff, usageInflightUserCap).StringSlice()
	if err != nil || len(ids) == 0 {
		return nil, err
	}
	if len(ids) > usageInflightUserCap {
		return nil, fmt.Errorf("in-flight administrator snapshot exceeds %d active users", usageInflightUserCap)
	}
	pipe := c.rdb.Pipeline()
	type inflightHashCmd struct {
		userID int64
		id     string
		cmd    *redis.MapStringStringCmd
	}
	cmds := make([]inflightHashCmd, 0, len(ids))
	for _, id := range ids {
		userID, convErr := strconv.ParseInt(id, 10, 64)
		if convErr != nil {
			pipe.ZRem(ctx, usageInflightUsersKey, id)
			continue
		}
		cmds = append(cmds, inflightHashCmd{userID: userID, id: id, cmd: pipe.HGetAll(ctx, usageInflightHashKey(userID))})
	}
	if _, err = pipe.Exec(ctx); err != nil && err != redis.Nil {
		return nil, err
	}
	now := time.Now()
	out := make([]service.UsageInflightSnapshot, 0, len(cmds))
	stale := make([]string, 0)
	type inflightDrop struct {
		userID  int64
		field   string
		payload string
	}
	drops := make([]inflightDrop, 0)
	for _, item := range cmds {
		raw, cmdErr := item.cmd.Result()
		if cmdErr != nil || len(raw) == 0 {
			if cmdErr == nil || cmdErr == redis.Nil {
				stale = append(stale, item.id)
			}
			continue
		}
		live := 0
		for field, payload := range raw {
			var snap service.UsageInflightSnapshot
			if json.Unmarshal([]byte(payload), &snap) != nil || snap.UserID != item.userID || !usageInflightLive(snap, now) {
				drops = append(drops, inflightDrop{userID: item.userID, field: field, payload: payload})
				continue
			}
			live++
			out = append(out, snap)
		}
		if live == 0 {
			stale = append(stale, item.id)
		}
	}
	if len(drops) > 0 {
		clean := c.rdb.Pipeline()
		for _, drop := range drops {
			clean.Eval(ctx, dropStaleInflightScript, []string{usageInflightHashKey(drop.userID)}, drop.field, drop.payload)
		}
		_, _ = clean.Exec(ctx)
	}
	if len(stale) > 0 {
		clean := c.rdb.Pipeline()
		for _, id := range stale {
			if uid, convErr := strconv.ParseInt(id, 10, 64); convErr == nil {
				clean.Eval(ctx, removeEmptyInflightUserScript, []string{usageInflightHashKey(uid), usageInflightUsersKey}, id)
			}
		}
		_, _ = clean.Exec(ctx)
	}
	return out, nil
}

const dropStaleInflightScript = `if redis.call('HGET', KEYS[1], ARGV[1]) == ARGV[2] then return redis.call('HDEL', KEYS[1], ARGV[1]) end return 0`

func (c *usageInflightCache) dropStale(ctx context.Context, key, field, payload string) {
	_ = c.rdb.Eval(ctx, dropStaleInflightScript, []string{key}, field, payload).Err()
}

const liveInflightUsersScript = `redis.call('ZREMRANGEBYSCORE', KEYS[1], '-inf', '(' .. ARGV[1])
return redis.call('ZRANGE', KEYS[1], 0, ARGV[2])`

const removeEmptyInflightUserScript = `if redis.call('HLEN', KEYS[1]) == 0 then return redis.call('ZREM', KEYS[2], ARGV[1]) end return 0`

func (c *usageInflightCache) removeEmptyUser(ctx context.Context, userID int64) error {
	return c.rdb.Eval(ctx, removeEmptyInflightUserScript, []string{usageInflightHashKey(userID), usageInflightUsersKey}, userID).Err()
}
