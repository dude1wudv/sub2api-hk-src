package repository

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestCharityCache_DailyCostsConcurrencyAndRPM(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	cache := NewCharityCache(rdb)
	ctx := context.Background()

	// 1. 每日分组折前费用：未初始化时 miss，初始化后原子累加
	_, hit, err := cache.GetDailyGroupCosts(ctx, "2026-11-28")
	require.NoError(t, err)
	require.False(t, hit)

	require.NoError(t, cache.SetDailyGroupCosts(ctx, "2026-11-28", map[int64]float64{10: 12.5}))
	require.NoError(t, cache.IncrDailyGroupCost(ctx, "2026-11-28", 10, 2.25))
	require.NoError(t, cache.IncrDailyGroupCost(ctx, "2026-11-28", 20, 0.75))

	costs, hit, err := cache.GetDailyGroupCosts(ctx, "2026-11-28")
	require.NoError(t, err)
	require.True(t, hit)
	require.InDelta(t, 14.75, costs[10], 1e-6)
	require.InDelta(t, 0.75, costs[20], 1e-6)

	// 2. 单 IP 日消耗缓存
	require.NoError(t, cache.SetDailyIPCost(ctx, "2026-11-28", "10.0.0.1", 1.0))
	require.NoError(t, cache.IncrDailyIPCost(ctx, "2026-11-28", "10.0.0.1", 0.5))
	ipCost, hit, err := cache.GetDailyIPCost(ctx, "2026-11-28", "10.0.0.1")
	require.NoError(t, err)
	require.True(t, hit)
	require.InDelta(t, 1.5, ipCost, 1e-6)

	// 3. 并发租约（全局 2、分组 1、单 IP 2）
	ok, reason, err := cache.AcquireConcurrencySlots(ctx, "req-1", 2, 10, 1, "10.0.0.1", 2)
	require.NoError(t, err)
	require.True(t, ok)
	require.Empty(t, reason)

	// 同分组上限 1 → 拒绝且不泄漏全局/IP 租约
	ok, reason, err = cache.AcquireConcurrencySlots(ctx, "req-2", 2, 10, 1, "10.0.0.1", 2)
	require.NoError(t, err)
	require.False(t, ok)
	require.Equal(t, "group", reason)

	// 另一分组 20 可获取第 2 个全局并发槽
	ok, reason, err = cache.AcquireConcurrencySlots(ctx, "req-3", 2, 20, 2, "10.0.0.1", 2)
	require.NoError(t, err)
	require.True(t, ok)

	live, err := cache.GetLiveConcurrency(ctx, []int64{10, 20})
	require.NoError(t, err)
	require.Equal(t, 2, live.Global)
	require.Equal(t, 1, live.ByGroup[10])
	require.Equal(t, 1, live.ByGroup[20])

	// 释放 req-1 后，分组 10 恢复可进入
	require.NoError(t, cache.ReleaseConcurrencySlots(ctx, "req-1", 10, "10.0.0.1"))
	ok, _, err = cache.AcquireConcurrencySlots(ctx, "req-4", 2, 10, 1, "10.0.0.2", 2)
	require.NoError(t, err)
	require.True(t, ok)

	// 4. 单 IP 滑动窗口 RPM
	ok, err = cache.CheckAndIncrIPRPM(ctx, "10.0.0.9", 2)
	require.NoError(t, err)
	require.True(t, ok)
	ok, err = cache.CheckAndIncrIPRPM(ctx, "10.0.0.9", 2)
	require.NoError(t, err)
	require.True(t, ok)
	ok, err = cache.CheckAndIncrIPRPM(ctx, "10.0.0.9", 2)
	require.NoError(t, err)
	require.False(t, ok)

	// 5. 每日轮换分布式锁
	locked, err := cache.TryAcquireRotationLock(ctx, "2026-11-28", time.Minute)
	require.NoError(t, err)
	require.True(t, locked)
	locked, err = cache.TryAcquireRotationLock(ctx, "2026-11-28", time.Minute)
	require.NoError(t, err)
	require.False(t, locked)
}

func TestCharityCache_ConfiguredLimitsRequireRedis(t *testing.T) {
	cache := NewCharityCache(nil)
	ctx := context.Background()
	allowed, err := cache.CheckAndIncrIPRPM(ctx, "10.0.0.1", 1)
	require.Error(t, err)
	require.False(t, allowed)
	acquired, _, err := cache.AcquireConcurrencySlots(ctx, "req", 1, 10, 0, "10.0.0.1", 0)
	require.Error(t, err)
	require.False(t, acquired)
	acquired, _, err = cache.AcquireConcurrencySlots(ctx, "req", 0, 10, 0, "10.0.0.1", 0)
	require.NoError(t, err)
	require.True(t, acquired)
}
