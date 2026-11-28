package repository

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const (
	charityCostScale         = 1e8 // 1 USD = 1e8 nano-scale units, matching NUMERIC(20,8)
	charityDailyCacheTTL     = 48 * time.Hour
	charityConcurrencySlotTT = 10 * time.Minute
	charityRPMSlotWindow     = 60 * time.Second
)

type charityCache struct {
	rdb *redis.Client
}

// NewCharityCache 创建公益池 Redis 热路径缓存实现。
func NewCharityCache(rdb *redis.Client) service.CharityCache {
	return &charityCache{rdb: rdb}
}

func charityGroupCostKey(usageDate string) string {
	return "charity:usage:group:" + usageDate
}

func charityIPCostKey(usageDate, clientIP string) string {
	return "charity:usage:ip:" + usageDate + ":" + clientIP
}

func charityGlobalConcKey() string {
	return "charity:conc:global"
}

func charityGroupConcKey(groupID int64) string {
	return fmt.Sprintf("charity:conc:group:%d", groupID)
}

func charityIPConcKey(clientIP string) string {
	return "charity:conc:ip:" + clientIP
}

func charityIPRPMKey(clientIP string) string {
	return "charity:rpm:ip:" + clientIP
}

func charityRotationLockKey(dateKey string) string {
	return "charity:rotate:lock:" + dateKey
}

func usdToScaledInt(costUSD float64) int64 {
	if costUSD <= 0 || math.IsNaN(costUSD) || math.IsInf(costUSD, 0) {
		return 0
	}
	return int64(math.Round(costUSD * charityCostScale))
}

func scaledIntToUSD(v int64) float64 {
	if v <= 0 {
		return 0
	}
	return float64(v) / charityCostScale
}

func (c *charityCache) GetDailyGroupCosts(ctx context.Context, usageDate string) (map[int64]float64, bool, error) {
	if c == nil || c.rdb == nil {
		return nil, false, nil
	}
	key := charityGroupCostKey(usageDate)
	raw, err := c.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, false, err
	}
	if len(raw) == 0 {
		return nil, false, nil
	}
	out := make(map[int64]float64, len(raw))
	for field, valStr := range raw {
		if field == "_init" {
			continue
		}
		gid, parseErr := strconv.ParseInt(field, 10, 64)
		if parseErr != nil {
			continue
		}
		scaled, _ := strconv.ParseInt(valStr, 10, 64)
		out[gid] = scaledIntToUSD(scaled)
	}
	return out, true, nil
}

func (c *charityCache) SetDailyGroupCosts(ctx context.Context, usageDate string, costs map[int64]float64) error {
	if c == nil || c.rdb == nil {
		return nil
	}
	key := charityGroupCostKey(usageDate)
	fields := make(map[string]any, len(costs)+1)
	fields["_init"] = "1"
	for gid, cost := range costs {
		fields[strconv.FormatInt(gid, 10)] = usdToScaledInt(cost)
	}
	pipe := c.rdb.TxPipeline()
	pipe.Del(ctx, key)
	pipe.HSet(ctx, key, fields)
	pipe.Expire(ctx, key, charityDailyCacheTTL)
	_, err := pipe.Exec(ctx)
	return err
}

func (c *charityCache) IncrDailyGroupCost(ctx context.Context, usageDate string, groupID int64, costUSD float64) error {
	if c == nil || c.rdb == nil || groupID <= 0 || costUSD <= 0 {
		return nil
	}
	delta := usdToScaledInt(costUSD)
	if delta <= 0 {
		return nil
	}
	key := charityGroupCostKey(usageDate)
	// 仅在缓存已初始化时递增，避免把半截数据写入缓存导致低估当日已用金额。
	const script = `
		if redis.call('EXISTS', KEYS[1]) == 1 then
			redis.call('HINCRBY', KEYS[1], ARGV[1], tonumber(ARGV[2]))
			redis.call('EXPIRE', KEYS[1], tonumber(ARGV[3]))
		end
		return 1
	`
	return c.rdb.Eval(ctx, script, []string{key},
		strconv.FormatInt(groupID, 10),
		delta,
		int64(charityDailyCacheTTL.Seconds()),
	).Err()
}

func (c *charityCache) GetDailyIPCost(ctx context.Context, usageDate string, clientIP string) (float64, bool, error) {
	if c == nil || c.rdb == nil || strings.TrimSpace(clientIP) == "" {
		return 0, false, nil
	}
	valStr, err := c.rdb.Get(ctx, charityIPCostKey(usageDate, clientIP)).Result()
	if err == redis.Nil {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	scaled, _ := strconv.ParseInt(valStr, 10, 64)
	return scaledIntToUSD(scaled), true, nil
}

func (c *charityCache) SetDailyIPCost(ctx context.Context, usageDate string, clientIP string, costUSD float64) error {
	if c == nil || c.rdb == nil || strings.TrimSpace(clientIP) == "" {
		return nil
	}
	return c.rdb.Set(ctx, charityIPCostKey(usageDate, clientIP), usdToScaledInt(costUSD), charityDailyCacheTTL).Err()
}

func (c *charityCache) IncrDailyIPCost(ctx context.Context, usageDate string, clientIP string, costUSD float64) error {
	if c == nil || c.rdb == nil || strings.TrimSpace(clientIP) == "" || costUSD <= 0 {
		return nil
	}
	delta := usdToScaledInt(costUSD)
	if delta <= 0 {
		return nil
	}
	const script = `
		if redis.call('EXISTS', KEYS[1]) == 1 then
			redis.call('INCRBY', KEYS[1], tonumber(ARGV[1]))
			redis.call('EXPIRE', KEYS[1], tonumber(ARGV[2]))
		end
		return 1
	`
	return c.rdb.Eval(ctx, script, []string{charityIPCostKey(usageDate, clientIP)},
		delta,
		int64(charityDailyCacheTTL.Seconds()),
	).Err()
}

func (c *charityCache) CheckAndIncrIPRPM(ctx context.Context, clientIP string, limit int) (bool, error) {
	if limit <= 0 || strings.TrimSpace(clientIP) == "" {
		return true, nil
	}
	if c == nil || c.rdb == nil {
		return false, errors.New("charity RPM cache unavailable")
	}
	nowMs := time.Now().UnixMilli()
	windowStart := nowMs - charityRPMSlotWindow.Milliseconds()
	member := fmt.Sprintf("%d:%d", nowMs, time.Now().UnixNano())
	const script = `
		redis.call('ZREMRANGEBYSCORE', KEYS[1], '-inf', tonumber(ARGV[1]))
		local count = redis.call('ZCARD', KEYS[1])
		if count >= tonumber(ARGV[2]) then
			return 0
		end
		redis.call('ZADD', KEYS[1], tonumber(ARGV[3]), ARGV[4])
		redis.call('PEXPIRE', KEYS[1], tonumber(ARGV[5]))
		return 1
	`
	res, err := c.rdb.Eval(ctx, script, []string{charityIPRPMKey(clientIP)},
		windowStart,
		limit,
		nowMs,
		member,
		(charityRPMSlotWindow + 10*time.Second).Milliseconds(),
	).Int()
	if err != nil {
		return false, err
	}
	return res == 1, nil
}

// AcquireConcurrencySlots 原子获取公益全局 / 分组 / 单 IP 并发租约。
// 任一维度满载时全量回滚，不残留半释放租约。
func (c *charityCache) AcquireConcurrencySlots(
	ctx context.Context,
	requestID string,
	globalLimit int,
	groupID int64,
	groupLimit int,
	clientIP string,
	ipLimit int,
) (bool, string, error) {
	if c == nil || c.rdb == nil {
		if globalLimit > 0 || groupLimit > 0 || (strings.TrimSpace(clientIP) != "" && ipLimit > 0) {
			return false, "", errors.New("charity concurrency cache unavailable")
		}
		return true, "", nil
	}
	nowMs := time.Now().UnixMilli()
	staleBefore := nowMs - charityConcurrencySlotTT.Milliseconds()
	ttlMs := (charityConcurrencySlotTT + time.Minute).Milliseconds()

	ipKey := ""
	if strings.TrimSpace(clientIP) != "" {
		ipKey = charityIPConcKey(clientIP)
	}

	const script = `
		local globalKey = KEYS[1]
		local groupKey  = KEYS[2]
		local ipKey     = KEYS[3]
		local reqID       = ARGV[1]
		local nowMs       = tonumber(ARGV[2])
		local staleBefore = tonumber(ARGV[3])
		local ttlMs       = tonumber(ARGV[4])
		local globalLimit = tonumber(ARGV[5])
		local groupLimit  = tonumber(ARGV[6])
		local ipLimit     = tonumber(ARGV[7])

		redis.call('ZREMRANGEBYSCORE', globalKey, '-inf', staleBefore)
		redis.call('ZREMRANGEBYSCORE', groupKey, '-inf', staleBefore)
		if ipKey ~= '' then
			redis.call('ZREMRANGEBYSCORE', ipKey, '-inf', staleBefore)
		end

		if globalLimit > 0 and redis.call('ZCARD', globalKey) >= globalLimit then
			return 'global'
		end
		if groupLimit > 0 and redis.call('ZCARD', groupKey) >= groupLimit then
			return 'group'
		end
		if ipKey ~= '' and ipLimit > 0 and redis.call('ZCARD', ipKey) >= ipLimit then
			return 'ip'
		end

		redis.call('ZADD', globalKey, nowMs, reqID)
		redis.call('PEXPIRE', globalKey, ttlMs)
		redis.call('ZADD', groupKey, nowMs, reqID)
		redis.call('PEXPIRE', groupKey, ttlMs)
		if ipKey ~= '' then
			redis.call('ZADD', ipKey, nowMs, reqID)
			redis.call('PEXPIRE', ipKey, ttlMs)
		end
		return 'ok'
	`
	res, err := c.rdb.Eval(ctx, script,
		[]string{charityGlobalConcKey(), charityGroupConcKey(groupID), ipKey},
		requestID,
		nowMs,
		staleBefore,
		ttlMs,
		globalLimit,
		groupLimit,
		ipLimit,
	).Text()
	if err != nil {
		return false, "", err
	}
	if res == "ok" {
		return true, "", nil
	}
	return false, res, nil
}

func (c *charityCache) ReleaseConcurrencySlots(ctx context.Context, requestID string, groupID int64, clientIP string) error {
	if c == nil || c.rdb == nil || requestID == "" {
		return nil
	}
	pipe := c.rdb.Pipeline()
	pipe.ZRem(ctx, charityGlobalConcKey(), requestID)
	if groupID > 0 {
		pipe.ZRem(ctx, charityGroupConcKey(groupID), requestID)
	}
	if strings.TrimSpace(clientIP) != "" {
		pipe.ZRem(ctx, charityIPConcKey(clientIP), requestID)
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (c *charityCache) GetLiveConcurrency(ctx context.Context, groupIDs []int64) (*service.CharityLiveConcurrency, error) {
	out := &service.CharityLiveConcurrency{
		ByGroup: make(map[int64]int, len(groupIDs)),
	}
	if c == nil || c.rdb == nil {
		return out, nil
	}
	staleBefore := strconv.FormatInt(time.Now().UnixMilli()-charityConcurrencySlotTT.Milliseconds(), 10)
	pipe := c.rdb.Pipeline()
	pipe.ZRemRangeByScore(ctx, charityGlobalConcKey(), "-inf", staleBefore)
	globalCmd := pipe.ZCard(ctx, charityGlobalConcKey())
	groupCmds := make(map[int64]*redis.IntCmd, len(groupIDs))
	for _, gid := range groupIDs {
		k := charityGroupConcKey(gid)
		pipe.ZRemRangeByScore(ctx, k, "-inf", staleBefore)
		groupCmds[gid] = pipe.ZCard(ctx, k)
	}
	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		return out, err
	}
	out.Global = int(globalCmd.Val())
	for gid, cmd := range groupCmds {
		out.ByGroup[gid] = int(cmd.Val())
	}
	return out, nil
}

func (c *charityCache) GetLiveIPConcurrency(ctx context.Context, clientIP string) (int, error) {
	if c == nil || c.rdb == nil || strings.TrimSpace(clientIP) == "" {
		return 0, nil
	}
	k := charityIPConcKey(clientIP)
	staleBefore := strconv.FormatInt(time.Now().UnixMilli()-charityConcurrencySlotTT.Milliseconds(), 10)
	pipe := c.rdb.Pipeline()
	pipe.ZRemRangeByScore(ctx, k, "-inf", staleBefore)
	cntCmd := pipe.ZCard(ctx, k)
	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		return 0, err
	}
	return int(cntCmd.Val()), nil
}

func (c *charityCache) TryAcquireRotationLock(ctx context.Context, dateKey string, ttl time.Duration) (bool, error) {
	if c == nil || c.rdb == nil {
		return true, nil
	}
	if ttl <= 0 {
		ttl = 60 * time.Second
	}
	return c.rdb.SetNX(ctx, charityRotationLockKey(dateKey), time.Now().UTC().Format(time.RFC3339Nano), ttl).Result()
}
