package repository

import (
	"context"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

var _ service.ChatAuxiliaryLeaseCache = (*concurrencyCache)(nil)

var chatAuxiliaryAcquireScript = redis.NewScript(`
redis.replicate_commands()
local now = tonumber(redis.call('TIME')[1])
redis.call('ZREMRANGEBYSCORE', KEYS[1], '-inf', now - 60)
if redis.call('ZCARD', KEYS[1]) >= tonumber(ARGV[1]) then return 0 end
redis.call('ZADD', KEYS[1], now, ARGV[2])
redis.call('EXPIRE', KEYS[1], 60)
return 1
`)

// Check every member before updating any member. A lost/expired member must
// cancel the upstream operation, never be recreated by a late refresh.
var chatAuxiliaryRefreshScript = redis.NewScript(`
redis.replicate_commands()
local now = tonumber(redis.call('TIME')[1])
local ttl = tonumber(ARGV[1])
for i, key in ipairs(KEYS) do
 local lifetime = ttl
 if i == 3 then lifetime = 60 end
 local score = redis.call('ZSCORE', key, ARGV[2])
 if score == false or tonumber(score) <= now - lifetime then return {0, now} end
end
for i, key in ipairs(KEYS) do
 local lifetime = ttl
 if i == 3 then lifetime = 60 end
 redis.call('ZADD', key, 'XX', now, ARGV[2])
 redis.call('EXPIRE', key, lifetime)
end
return {1, now}
`)

func chatAuxiliaryKindSlotKey(accountID int64, kind string) string {
	return fmt.Sprintf("concurrency:chat_auxiliary:%d:%s", accountID, kind)
}

func (c *concurrencyCache) AcquireChatAuxiliaryKindSlot(ctx context.Context, accountID int64, kind string, limit int, requestID string) (bool, error) {
	n, err := chatAuxiliaryAcquireScript.Run(ctx, c.rdb, []string{chatAuxiliaryKindSlotKey(accountID, kind)}, limit, requestID).Int()
	return n == 1, err
}

func (c *concurrencyCache) RefreshChatAuxiliarySlots(ctx context.Context, userID, accountID int64, kind string, kindHeld bool, requestID string) (bool, error) {
	keys := []string{userSlotKey(userID), accountSlotKey(accountID)}
	if kindHeld {
		keys = append(keys, chatAuxiliaryKindSlotKey(accountID, kind))
	}
	ok, now, err := runScriptInt64Pair(ctx, c.rdb, chatAuxiliaryRefreshScript, keys, c.slotTTLSeconds, requestID)
	if err == nil && ok == 1 {
		c.touchActiveIndexAt(ctx, userActiveIndexKey, userID, now+int64(c.slotTTLSeconds))
		c.touchActiveIndexAt(ctx, accountActiveIndexKey, accountID, now+int64(c.slotTTLSeconds))
	}
	return ok == 1, err
}

func (c *concurrencyCache) ReleaseChatAuxiliaryKindSlot(ctx context.Context, accountID int64, kind, requestID string) error {
	return c.rdb.ZRem(ctx, chatAuxiliaryKindSlotKey(accountID, kind), requestID).Err()
}
