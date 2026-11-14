package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestLiveLeaseReplacesRegularSlotsAndCountsTowardLimits(t *testing.T) {
	redisServer := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	regular := NewConcurrencyCache(client, 15, 900)
	live, ok := regular.(service.LiveConcurrencyCache)
	require.True(t, ok)
	ctx := context.Background()

	accountAcquired, err := regular.AcquireAccountSlot(ctx, 10, 1, "regular-account")
	require.NoError(t, err)
	require.True(t, accountAcquired)
	userAcquired, err := regular.AcquireUserSlot(ctx, 20, 1, "regular-user")
	require.NoError(t, err)
	require.True(t, userAcquired)

	acquired, err := live.AcquireLiveLease(ctx, 10, 1, 20, 1, 30, "live-lease", true)
	require.NoError(t, err)
	require.True(t, acquired)
	require.NoError(t, regular.ReleaseAccountSlot(ctx, 10, "regular-account"))
	require.NoError(t, regular.ReleaseUserSlot(ctx, 20, "regular-user"))

	accountCount, err := regular.GetAccountConcurrency(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, 1, accountCount)
	userCount, err := regular.GetUserConcurrency(ctx, 20)
	require.NoError(t, err)
	require.Equal(t, 1, userCount)
	accountAcquired, err = regular.AcquireAccountSlot(ctx, 10, 1, "ordinary-blocked")
	require.NoError(t, err)
	require.False(t, accountAcquired)

	refreshed, err := live.RefreshLiveLease(ctx, 10, 20, 30, "live-lease")
	require.NoError(t, err)
	require.True(t, refreshed)
	require.NoError(t, live.ReleaseLiveLease(ctx, 10, 20, 30, "live-lease"))
	accountAcquired, err = regular.AcquireAccountSlot(ctx, 10, 1, "ordinary-allowed")
	require.NoError(t, err)
	require.True(t, accountAcquired)
}

func TestLiveLeaseExpiresWithoutRefresh(t *testing.T) {
	redisServer := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	regular := NewConcurrencyCache(client, 15, 900)
	live, ok := regular.(service.LiveConcurrencyCache)
	require.True(t, ok)
	ctx := context.Background()

	acquired, err := live.AcquireLiveLease(ctx, 10, 1, 20, 1, 30, "expired-live", false)
	require.NoError(t, err)
	require.True(t, acquired)

	redisServer.FastForward(61 * time.Second)
	acquired, err = regular.AcquireAccountSlot(ctx, 10, 1, "ordinary-after-expiry")
	require.NoError(t, err)
	require.True(t, acquired)
	refreshed, err := live.RefreshLiveLease(ctx, 10, 20, 30, "expired-live")
	require.NoError(t, err)
	require.False(t, refreshed)
}

func TestChatAuxiliaryRedisLeaseDoesNotExpireOrResurrect(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	cache := NewConcurrencyCache(client, 15, 900).(*concurrencyCache)
	ctx := t.Context()
	now := time.Unix(1_800_000_000, 0)
	server.SetTime(now)
	ok, err := cache.AcquireUserSlot(ctx, 1, 1, "aux")
	require.NoError(t, err)
	require.True(t, ok)
	ok, err = cache.AcquireAccountSlot(ctx, 2, 1, "aux")
	require.NoError(t, err)
	require.True(t, ok)
	ok, err = cache.AcquireChatAuxiliaryKindSlot(ctx, 2, "tts", 50, "aux")
	require.NoError(t, err)
	require.True(t, ok)
	for range 48 {
		now = now.Add(20 * time.Second)
		server.FastForward(20 * time.Second)
		server.SetTime(now)
		ok, err = cache.RefreshChatAuxiliarySlots(ctx, 1, 2, "tts", true, "aux")
		require.NoError(t, err)
		require.True(t, ok)
	}
	ok, err = cache.AcquireAccountSlot(ctx, 2, 1, "ordinary")
	require.NoError(t, err)
	require.False(t, ok)
	require.NoError(t, cache.ReleaseAccountSlot(ctx, 2, "aux"))
	ok, err = cache.RefreshChatAuxiliarySlots(ctx, 1, 2, "tts", true, "aux")
	require.NoError(t, err)
	require.False(t, ok)
	require.Zero(t, client.ZCard(ctx, accountSlotKey(2)).Val())
	require.Zero(t, client.ZCard(ctx, apiKeySlotKey(3)).Val())
}

func TestChatAuxiliaryRedisKindLimits(t *testing.T) {
	for _, kind := range []string{"asr", "tts", "ocr"} {
		server := miniredis.RunT(t)
		client := redis.NewClient(&redis.Options{Addr: server.Addr()})
		t.Cleanup(func() { _ = client.Close() })
		cache := NewConcurrencyCache(client, 15, 900).(*concurrencyCache)
		limit := 50
		if kind == "ocr" {
			limit = 15
		}
		for i := range limit + 1 {
			ok, err := cache.AcquireChatAuxiliaryKindSlot(t.Context(), 2, kind, limit, string(rune('a'+i)))
			require.NoError(t, err)
			require.Equal(t, i < limit, ok)
		}
	}
}
