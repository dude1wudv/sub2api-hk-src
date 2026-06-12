package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// expirySweepRepoStub 仅记录 MarkExpired 调用次数，其余方法 panic（本测试不应触达）。
type expirySweepRepoStub struct {
	markCalls int
}

func (s *expirySweepRepoStub) CreateGrant(ctx context.Context, g *DailyBalanceGrant) (int64, error) {
	panic("unexpected")
}
func (s *expirySweepRepoStub) SumActiveRemaining(ctx context.Context, userID, groupID int64, now time.Time) (float64, error) {
	panic("unexpected")
}
func (s *expirySweepRepoStub) ListActiveByExpiry(ctx context.Context, userID, groupID int64, now time.Time) ([]DailyBalanceGrant, error) {
	panic("unexpected")
}
func (s *expirySweepRepoStub) AtomicDecrement(ctx context.Context, grantID int64, amount float64) (bool, error) {
	panic("unexpected")
}
func (s *expirySweepRepoStub) MarkExpired(ctx context.Context, now time.Time) (int64, error) {
	s.markCalls++
	return 0, nil
}
func (s *expirySweepRepoStub) ListByUser(ctx context.Context, userID int64) ([]DailyBalanceGrant, error) {
	panic("unexpected")
}

func TestDailyGrantExpiry_SkipsSweepWhenNotLeader(t *testing.T) {
	cache := &fakeLeaderLockCache{}
	// 同伴已持有该 leader lock。
	_, _ = cache.TryAcquireLeaderLock(context.Background(), dailyGrantExpiryLeaderLockKey, "peer", time.Minute)

	repo := &expirySweepRepoStub{}
	svc := NewDailyGrantExpiryService(repo, time.Minute)
	svc.SetLeaderLock(cache, nil)

	svc.runOnce()
	require.Zero(t, repo.markCalls, "非 leader 不应执行过期扫描")
}

func TestDailyGrantExpiry_SweepsWhenLeader(t *testing.T) {
	repo := &expirySweepRepoStub{}
	svc := NewDailyGrantExpiryService(repo, time.Minute)
	svc.SetLeaderLock(&fakeLeaderLockCache{}, nil)

	svc.runOnce()
	require.Equal(t, 1, repo.markCalls, "leader 应执行一次过期扫描")
}

// 单实例正确性：每轮结束释放锁，同实例必须能在后续每轮重新获取并执行（无自锁死）。
func TestDailyGrantExpiry_RunsEveryCycleSingleInstance(t *testing.T) {
	cases := map[string]LeaderLockCache{
		"with_cache": &fakeLeaderLockCache{},
		"no_backend": nil,
	}
	for name, cache := range cases {
		t.Run(name, func(t *testing.T) {
			repo := &expirySweepRepoStub{}
			svc := NewDailyGrantExpiryService(repo, time.Minute)
			svc.SetLeaderLock(cache, nil)

			svc.runOnce()
			svc.runOnce()
			svc.runOnce()
			require.Equal(t, 3, repo.markCalls, "单实例每轮都应执行")
		})
	}
}
