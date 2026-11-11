package repository

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestUsageInflightCacheCompletionRaceAndCrashExpiry(t *testing.T) {
	server := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: server.Addr()})
	cache := NewUsageInflightCache(rdb)
	ctx := context.Background()
	snap := service.UsageInflightSnapshot{UserID: 3, RequestID: "racing", StartedAt: time.Now(), UpdatedAt: time.Now()}
	var workers sync.WaitGroup
	for i := 0; i < 10; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for j := 0; j < 10; j++ {
				_ = cache.Save(ctx, snap)
			}
		}()
	}
	if err := cache.Delete(ctx, 3, snap.RequestID); err != nil {
		t.Fatal(err)
	}
	workers.Wait()
	rows, err := cache.ListUser(ctx, 3)
	if err != nil || len(rows) != 0 {
		t.Fatalf("late save resurrected completed request: %+v %v", rows, err)
	}
	snap.RequestID = "crashed"
	snap.UpdatedAt = time.Now().Add(-61 * time.Second)
	if err := cache.Save(ctx, snap); err != nil {
		t.Fatal(err)
	}
	rows, err = cache.ListAll(ctx)
	if err != nil || len(rows) != 0 {
		t.Fatalf("crashed heartbeat remains visible: %+v %v", rows, err)
	}
	snap.RequestID = "long-running"
	snap.StartedAt = time.Now().Add(-59 * time.Minute)
	snap.UpdatedAt = time.Now()
	if err := cache.Save(ctx, snap); err != nil {
		t.Fatal(err)
	}
	rows, err = cache.ListUser(ctx, 3)
	if err != nil || len(rows) != 1 {
		t.Fatalf("fresh heartbeat of long request lost: %+v %v", rows, err)
	}
	rows, err = cache.ListAll(ctx)
	if err != nil || len(rows) != 1 || rows[0].RequestID != "long-running" {
		t.Fatalf("administrator index lost long request with fresh heartbeat: %+v %v", rows, err)
	}
}

func TestUsageInflightCacheRoundTrip(t *testing.T) {
	server := miniredis.RunT(t)
	cache := NewUsageInflightCache(redis.NewClient(&redis.Options{Addr: server.Addr()}))
	ctx := context.Background()
	first := 40
	snap := service.UsageInflightSnapshot{
		RequestID: "req-1", UserID: 9, APIKeyID: 3, Model: "deepseek-v4",
		StartedAt: time.Now().Add(-time.Second), FirstTokenMs: &first,
		InputTokens: 12, OutputTokens: 4, ReservedAmount: 1.25, Email: "a@b.c",
	}
	if err := cache.Save(ctx, snap); err != nil {
		t.Fatal(err)
	}
	rows, err := cache.ListUser(ctx, 9)
	if err != nil || len(rows) != 1 || rows[0].ReservedAmount != 1.25 || rows[0].FirstTokenMs == nil || *rows[0].FirstTokenMs != 40 {
		t.Fatalf("rows=%+v err=%v", rows, err)
	}
	all, err := cache.ListAll(ctx)
	if err != nil || len(all) != 1 {
		t.Fatalf("all=%+v err=%v", all, err)
	}
	if err := cache.Delete(ctx, 9, "req-1"); err != nil {
		t.Fatal(err)
	}
	rows, err = cache.ListUser(ctx, 9)
	if err != nil || len(rows) != 0 {
		t.Fatalf("after delete rows=%+v err=%v", rows, err)
	}
	if err := cache.Save(ctx, snap); err != nil {
		t.Fatal(err)
	}
	rows, _ = cache.ListUser(ctx, 9)
	if len(rows) != 0 {
		t.Fatalf("tombstone should block rewrite, got %+v", rows)
	}
}

func TestUsageInflightCacheAdminOverflowIsUnknown(t *testing.T) {
	server := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	cache := NewUsageInflightCache(rdb)
	for user := int64(1); user <= 501; user++ {
		if err := cache.Save(context.Background(), service.UsageInflightSnapshot{UserID: user, RequestID: "one", StartedAt: time.Now()}); err != nil {
			t.Fatal(err)
		}
	}
	rows, err := cache.ListAll(context.Background())
	if err == nil || len(rows) != 0 {
		t.Fatal("capped administrator read must not claim a complete snapshot")
	}
	// Simulate all 501 processes disappearing, without waiting on wall time.
	// Redis expires their hashes independently of the application clock, while
	// the index scores represent the now-stale last successful heartbeats.
	stale := float64(time.Now().Add(-61 * time.Second).UnixMilli())
	pipe := rdb.Pipeline()
	for user := int64(1); user <= 501; user++ {
		pipe.ZAdd(context.Background(), usageInflightUsersKey, redis.Z{Score: stale, Member: user})
	}
	if _, err := pipe.Exec(context.Background()); err != nil {
		t.Fatal(err)
	}
	server.FastForward(usageInflightTTL + time.Second)
	fresh := service.UsageInflightSnapshot{UserID: 777, RequestID: "recovered", StartedAt: time.Now(), UpdatedAt: time.Now()}
	if err := cache.Save(context.Background(), fresh); err != nil {
		t.Fatal(err)
	}
	rows, err = cache.ListAll(context.Background())
	if err != nil || len(rows) != 1 || rows[0].UserID != fresh.UserID {
		t.Fatalf("expired overload must recover without manual Redis cleanup: rows=%+v err=%v", rows, err)
	}
	if count, err := rdb.ZCard(context.Background(), usageInflightUsersKey).Result(); err != nil || count != 1 {
		t.Fatalf("stale user index members survived pruning: count=%d err=%v", count, err)
	}
}

func TestUsageInflightCachePrunesStaleUsersBeforeCap(t *testing.T) {
	server := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	cache := NewUsageInflightCache(rdb)
	ctx := context.Background()
	stale := time.Now().Add(-61 * time.Second)
	for user := int64(1); user <= 501; user++ {
		if err := cache.Save(ctx, service.UsageInflightSnapshot{UserID: user, RequestID: "stale", StartedAt: stale, UpdatedAt: stale}); err != nil {
			t.Fatal(err)
		}
	}
	if err := cache.Save(ctx, service.UsageInflightSnapshot{UserID: 777, RequestID: "live", StartedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	rows, err := cache.ListAll(ctx)
	if err != nil || len(rows) != 1 || rows[0].UserID != 777 {
		t.Fatalf("stale heartbeats must not count against the live-user cap: %+v %v", rows, err)
	}
}

func TestUsageInflightCacheOlderSnapshotCannotAgeOutLiveUser(t *testing.T) {
	server := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	cache := NewUsageInflightCache(rdb)
	ctx := context.Background()
	now := time.Now()
	fresh := service.UsageInflightSnapshot{UserID: 9, RequestID: "fresh", StartedAt: now, UpdatedAt: now}
	if err := cache.Save(ctx, fresh); err != nil {
		t.Fatal(err)
	}
	stale := now.Add(-61 * time.Second)
	if err := cache.Save(ctx, service.UsageInflightSnapshot{UserID: 9, RequestID: "late-old-request", StartedAt: stale, UpdatedAt: stale}); err != nil {
		t.Fatal(err)
	}
	rows, err := cache.ListAll(ctx)
	if err != nil || len(rows) != 1 || rows[0].RequestID != "fresh" {
		t.Fatalf("older snapshot must not overwrite a user's newer index heartbeat: %+v %v", rows, err)
	}
}
