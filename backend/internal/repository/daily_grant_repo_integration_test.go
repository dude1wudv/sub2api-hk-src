//go:build integration

package repository

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

// uniqueGrantUserID 为每个测试分配独立的 user_id，使各测试的 Grant 行互不干扰。
var grantUserSeq int64 = 9_000_000

func nextGrantUserID() int64 {
	return atomic.AddInt64(&grantUserSeq, 1)
}

func newDailyGrantRepoForTest(t *testing.T) (*dailyGrantRepository, int64) {
	t.Helper()
	repo := &dailyGrantRepository{client: integrationEntClient, sql: integrationDB}
	userID := nextGrantUserID()
	t.Cleanup(func() {
		_, _ = integrationDB.ExecContext(context.Background(),
			"DELETE FROM daily_balance_grants WHERE user_id = $1", userID)
	})
	return repo, userID
}

func mustCreateGrant(t *testing.T, repo *dailyGrantRepository, userID, groupID int64, remaining float64, expiresAt time.Time) int64 {
	t.Helper()
	id, err := repo.CreateGrant(context.Background(), &service.DailyBalanceGrant{
		UserID:    userID,
		GroupID:   groupID,
		Amount:    remaining,
		Remaining: remaining,
		Status:    domain.DailyGrantStatusActive,
		Source:    domain.DailyGrantSourceAdmin,
		ExpiresAt: expiresAt,
	})
	require.NoError(t, err)
	return id
}

func TestDailyGrantRepo_SumActiveRemaining(t *testing.T) {
	repo, userID := newDailyGrantRepoForTest(t)
	ctx := context.Background()
	now := time.Now()
	const groupID = 42

	mustCreateGrant(t, repo, userID, groupID, 10, now.Add(24*time.Hour))
	mustCreateGrant(t, repo, userID, groupID, 5, now.Add(12*time.Hour))
	// 过期的不计入
	mustCreateGrant(t, repo, userID, groupID, 100, now.Add(-time.Hour))
	// 别的分组不计入
	mustCreateGrant(t, repo, userID, 99, 7, now.Add(24*time.Hour))

	sum, err := repo.SumActiveRemaining(ctx, userID, groupID, now)
	require.NoError(t, err)
	require.InDelta(t, 15.0, sum, 1e-9)
}

func TestDailyGrantRepo_ListActiveByExpiry_Order(t *testing.T) {
	repo, userID := newDailyGrantRepoForTest(t)
	ctx := context.Background()
	now := time.Now()
	const groupID = 42

	later := mustCreateGrant(t, repo, userID, groupID, 10, now.Add(20*time.Hour))
	sooner := mustCreateGrant(t, repo, userID, groupID, 10, now.Add(2*time.Hour))

	grants, err := repo.ListActiveByExpiry(ctx, userID, groupID, now)
	require.NoError(t, err)
	require.Len(t, grants, 2)
	require.Equal(t, sooner, grants[0].ID, "先到期的应排在前")
	require.Equal(t, later, grants[1].ID)
}

func TestDailyGrantRepo_AtomicDecrement_Exhausts(t *testing.T) {
	repo, userID := newDailyGrantRepoForTest(t)
	ctx := context.Background()
	now := time.Now()
	id := mustCreateGrant(t, repo, userID, 42, 10, now.Add(24*time.Hour))

	// 扣减一部分仍 active
	ok, err := repo.AtomicDecrement(ctx, id, 4)
	require.NoError(t, err)
	require.True(t, ok)

	// 扣减超过剩余 -> 失败，不变
	ok, err = repo.AtomicDecrement(ctx, id, 100)
	require.NoError(t, err)
	require.False(t, ok, "超过剩余应拒绝")

	// 扣减剩余全部 -> 成功且置 exhausted
	ok, err = repo.AtomicDecrement(ctx, id, 6)
	require.NoError(t, err)
	require.True(t, ok)

	grants, err := repo.ListByUser(ctx, userID)
	require.NoError(t, err)
	require.Len(t, grants, 1)
	require.Equal(t, domain.DailyGrantStatusExhausted, grants[0].Status)
	require.InDelta(t, 0.0, grants[0].Remaining, 1e-9)

	// 已 exhausted 不能再扣
	ok, err = repo.AtomicDecrement(ctx, id, 1)
	require.NoError(t, err)
	require.False(t, ok)
}

// TestDailyGrantRepo_AtomicDecrement_Concurrent 验证并发扣减不会超卖：
// 100 个 goroutine 各扣 1，初始余额 60 → 恰好 60 次成功、40 次失败，剩余 0。
func TestDailyGrantRepo_AtomicDecrement_Concurrent(t *testing.T) {
	repo, userID := newDailyGrantRepoForTest(t)
	ctx := context.Background()
	now := time.Now()
	id := mustCreateGrant(t, repo, userID, 42, 60, now.Add(24*time.Hour))

	const goroutines = 100
	var success int64
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			ok, err := repo.AtomicDecrement(ctx, id, 1)
			require.NoError(t, err)
			if ok {
				atomic.AddInt64(&success, 1)
			}
		}()
	}
	wg.Wait()

	require.Equal(t, int64(60), atomic.LoadInt64(&success), "成功次数应恰好等于初始余额，无超卖")

	grants, err := repo.ListByUser(ctx, userID)
	require.NoError(t, err)
	require.InDelta(t, 0.0, grants[0].Remaining, 1e-9)
	require.Equal(t, domain.DailyGrantStatusExhausted, grants[0].Status)
}

func TestDailyGrantRepo_MarkExpired(t *testing.T) {
	repo, userID := newDailyGrantRepoForTest(t)
	ctx := context.Background()
	now := time.Now()

	expired := mustCreateGrant(t, repo, userID, 42, 10, now.Add(-time.Hour))
	active := mustCreateGrant(t, repo, userID, 42, 10, now.Add(time.Hour))

	n, err := repo.MarkExpired(ctx, now)
	require.NoError(t, err)
	require.GreaterOrEqual(t, n, int64(1))

	grants, err := repo.ListByUser(ctx, userID)
	require.NoError(t, err)
	byID := map[int64]service.DailyBalanceGrant{}
	for _, g := range grants {
		byID[g.ID] = g
	}
	require.Equal(t, domain.DailyGrantStatusExpired, byID[expired].Status)
	require.Equal(t, domain.DailyGrantStatusActive, byID[active].Status)
}
