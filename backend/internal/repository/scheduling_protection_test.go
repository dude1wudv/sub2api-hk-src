//go:build unit

package repository

import (
	"context"
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
