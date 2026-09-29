//go:build unit

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSlowTTFTObservationThresholdAndEpochs(t *testing.T) {
	ctx := context.Background()
	sched, _ := newSchedulerCacheUnitWithRedis(t)
	cache := NewTempUnschedCache(sched.rdb).(*tempUnschedCache)
	cfg := service.DefaultSlowTTFTConfig()
	cfg.Enabled, cfg.Generation = true, "epoch-a"
	observe := func(attempt string, slow bool) service.SlowTTFTObservation {
		out, err := cache.ObserveSlowTTFT(ctx, 41, cfg, attempt, slow)
		require.NoError(t, err)
		return out
	}
	// A fast sample resets the consecutive count but does not trip the rolling threshold.
	require.False(t, observe("a", true).Tripped)
	require.False(t, observe("b", false).Tripped)
	require.False(t, observe("c", true).Tripped)
	tripped := observe("d", true)
	require.True(t, tripped.Tripped)
	require.Equal(t, "consecutive", tripped.Reason)

	// Duplicate attempt IDs are idempotent.
	cfg.Generation = "epoch-b"
	require.False(t, observe("same", true).Tripped)
	require.False(t, observe("same", true).Tripped)
	old := cfg
	old.Generation = "epoch-a"
	_, err := cache.ObserveSlowTTFT(ctx, 41, old, "late-old", true)
	require.NoError(t, err)
	require.True(t, observe("next", true).Tripped, "old-generation observation must not clear the one active slow sample")
}

func TestSlowTTFTRollingCountAllowsFastSamplesWithoutCountingThem(t *testing.T) {
	ctx := context.Background()
	sched, _ := newSchedulerCacheUnitWithRedis(t)
	cache := NewTempUnschedCache(sched.rdb).(*tempUnschedCache)
	cfg := service.DefaultSlowTTFTConfig()
	cfg.Enabled, cfg.Generation = true, "rolling-fast"
	cfg.ConsecutiveCount, cfg.WindowCount = 100, 3
	observe := func(id string, slow bool) service.SlowTTFTObservation {
		got, err := cache.ObserveSlowTTFT(ctx, 43, cfg, id, slow)
		require.NoError(t, err)
		return got
	}
	require.False(t, observe("slow-1", true).Tripped)
	require.False(t, observe("fast", false).Tripped)
	require.False(t, observe("slow-2", true).Tripped)
	require.True(t, observe("slow-3", true).Tripped)
}

func TestSlowTTFTRollingWindowCooldownAndReset(t *testing.T) {
	ctx := context.Background()
	sched, mr := newSchedulerCacheUnitWithRedis(t)
	cache := NewTempUnschedCache(sched.rdb).(*tempUnschedCache)
	cfg := service.DefaultSlowTTFTConfig()
	cfg.Enabled, cfg.Generation = true, "window"
	cfg.ConsecutiveCount, cfg.WindowCount, cfg.WindowSeconds, cfg.PauseSeconds = 100, 3, 10, 2
	now := time.Unix(100, 0)
	mr.SetTime(now)
	observe := func(attempt string, slow bool) service.SlowTTFTObservation {
		got, err := cache.ObserveSlowTTFT(ctx, 42, cfg, attempt, slow)
		require.NoError(t, err)
		return got
	}
	require.False(t, observe("1", true).Tripped)
	now = now.Add(9 * time.Second)
	mr.FastForward(9 * time.Second)
	mr.SetTime(now)
	require.False(t, observe("2", true).Tripped)
	now = now.Add(time.Second)
	mr.FastForward(time.Second)
	mr.SetTime(now)
	// At the exact window boundary, the first sample is removed.
	require.False(t, observe("3", true).Tripped)
	now = now.Add(time.Millisecond)
	mr.FastForward(time.Millisecond)
	mr.SetTime(now)
	trip := observe("4", true)
	require.True(t, trip.Tripped, "the sample at t=100 must have expired")

	now = now.Add(time.Second)
	mr.FastForward(time.Second)
	mr.SetTime(now)
	require.Equal(t, trip.Until, observe("5", true).Until, "cooldown must not slide")
	now = now.Add(2 * time.Second)
	mr.FastForward(2 * time.Second)
	mr.SetTime(now)
	require.False(t, observe("6", true).Tripped, "expired cooldown starts a fresh observation epoch")

	// Configuration changes take effect in a new generation, as they do on trigger.
	cfg.Generation, cfg.WindowSeconds = "window-new-epoch", 60
	require.False(t, observe("7", true).Tripped)
	now = now.Add(11 * time.Second)
	mr.FastForward(11 * time.Second)
	mr.SetTime(now)
	require.False(t, observe("8", true).Tripped)
	trip = observe("9", true)
	require.True(t, trip.Tripped, "the 60-second window retains samples beyond the old 10-second window")
}

func TestIndependentSchedulingPriorityAndCapacity(t *testing.T) {
	ctx := context.Background()
	cache, _ := newSchedulerCacheUnitWithRedis(t)
	concurrency := NewConcurrencyCache(cache.rdb, 1, 1).(*concurrencyCache)
	seq := 0
	pick := func(candidates ...service.SchedulingCandidate) int64 {
		seq++
		id, err := concurrency.AcquireScheduledAccount(ctx, candidates, fmt.Sprintf("select-%d", seq))
		require.NoError(t, err)
		return id
	}
	// Priority values 10/9/9: both members of the strongest (9) tier are used before tier 10.
	prioritySet := []service.SchedulingCandidate{
		{ID: 10, Priority: 10, LoadFactor: 1, Concurrency: 1},
		{ID: 9, Priority: 9, LoadFactor: 1, Concurrency: 1},
		{ID: 8, Priority: 9, LoadFactor: 1, Concurrency: 1},
	}
	first := pick(prioritySet...)
	require.Contains(t, []int64{9, 8}, first)
	second := pick(prioritySet...)
	require.Contains(t, []int64{9, 8}, second)
	require.NotEqual(t, first, second)
	require.Equal(t, int64(10), pick(prioritySet...))
	require.Zero(t, pick(prioritySet...))

	// Tied utilization in one priority layer is resolved among both candidates.
	tied := map[int64]bool{}
	for i := range 40 {
		got := pick(service.SchedulingCandidate{ID: 1000 + int64(i), Priority: 5, LoadFactor: 1, Concurrency: 1}, service.SchedulingCandidate{ID: 2000 + int64(i), Priority: 5, LoadFactor: 1, Concurrency: 1})
		tied[got] = true
	}
	sawFirst, sawSecond := false, false
	for id := range tied {
		if id >= 1000 && id < 2000 {
			sawFirst = true
		}
		if id >= 2000 {
			sawSecond = true
		}
	}
	require.True(t, sawFirst)
	require.Equal(t, int64(18), pick(service.SchedulingCandidate{ID: 19, Priority: 6, LoadFactor: 1, Concurrency: 4}, service.SchedulingCandidate{ID: 18, Priority: 6, LoadFactor: 2, Concurrency: 4}), "higher load factor has lower projected utilization at the same tier")
	require.True(t, sawSecond)

	// A full higher layer gives way to the next priority layer.
	require.Equal(t, int64(30), pick(service.SchedulingCandidate{ID: 30, Priority: 1, LoadFactor: 1, Concurrency: 1}, service.SchedulingCandidate{ID: 31, Priority: 2, LoadFactor: 1, Concurrency: 2}))
	require.Equal(t, int64(31), pick(service.SchedulingCandidate{ID: 30, Priority: 1, LoadFactor: 1, Concurrency: 1}, service.SchedulingCandidate{ID: 31, Priority: 2, LoadFactor: 1, Concurrency: 2}))
	// Legacy and independent scheduling use the same global account-slot set; release makes it available again.
	ok, err := concurrency.AcquireAccountSlot(ctx, 50, 1, "legacy")
	require.NoError(t, err)
	require.True(t, ok)
	require.Zero(t, pick(service.SchedulingCandidate{ID: 50, Priority: 0, LoadFactor: 1, Concurrency: 1}))
	require.NoError(t, concurrency.ReleaseAccountSlot(ctx, 50, "legacy"))
	require.Equal(t, int64(50), pick(service.SchedulingCandidate{ID: 50, Priority: 0, LoadFactor: 1, Concurrency: 1}))
}

func TestIndependentSchedulingPauseGenerationAndExemption(t *testing.T) {
	ctx := context.Background()
	cache, _ := newSchedulerCacheUnitWithRedis(t)
	concurrency := NewConcurrencyCache(cache.rdb, 1, 1).(*concurrencyCache)
	candidate := service.SchedulingCandidate{ID: 77, Priority: 0, LoadFactor: 1, Concurrency: 1, PauseGeneration: "g1"}
	pauseKey := slowTTFTKeys(77, "g1")[2]
	require.NoError(t, cache.rdb.Set(ctx, pauseKey, "g1", time.Minute).Err())
	id, err := concurrency.AcquireScheduledAccount(ctx, []service.SchedulingCandidate{candidate}, "paused")
	require.NoError(t, err)
	require.Zero(t, id)
	candidate.IgnoreSlowTTFT = true
	id, err = concurrency.AcquireScheduledAccount(ctx, []service.SchedulingCandidate{candidate}, "exempt")
	require.NoError(t, err)
	require.Equal(t, int64(77), id)
	require.NoError(t, concurrency.ReleaseAccountSlot(ctx, 77, "exempt"))
	candidate.PauseGeneration, candidate.IgnoreSlowTTFT = "g2", false
	id, err = concurrency.AcquireScheduledAccount(ctx, []service.SchedulingCandidate{candidate}, "new-epoch")
	require.NoError(t, err)
	require.Equal(t, int64(77), id)
}
