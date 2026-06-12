package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

// balanceCacheStub 是一个只关心 GetUserBalance 的 BillingCache 桩，
// 其余方法返回零值/nil，足以驱动 checkBalanceEligibility 的单元测试。
type balanceCacheStub struct {
	billingCacheWorkerStub
	balance float64
	getErr  error
}

func (b *balanceCacheStub) GetUserBalance(ctx context.Context, userID int64) (float64, error) {
	return b.balance, b.getErr
}

// dailyGrantRepoStub 仅实现 SumActiveRemaining，其余方法 panic（本测试不应触达）。
type dailyGrantRepoStub struct {
	remaining float64
	sumErr    error
	called    bool
}

func (s *dailyGrantRepoStub) CreateGrant(ctx context.Context, g *DailyBalanceGrant) (int64, error) {
	panic("unexpected")
}
func (s *dailyGrantRepoStub) SumActiveRemaining(ctx context.Context, userID, groupID int64, now time.Time) (float64, error) {
	s.called = true
	return s.remaining, s.sumErr
}
func (s *dailyGrantRepoStub) ListActiveByExpiry(ctx context.Context, userID, groupID int64, now time.Time) ([]DailyBalanceGrant, error) {
	panic("unexpected")
}
func (s *dailyGrantRepoStub) AtomicDecrement(ctx context.Context, grantID int64, amount float64) (bool, error) {
	panic("unexpected")
}
func (s *dailyGrantRepoStub) MarkExpired(ctx context.Context, now time.Time) (int64, error) {
	panic("unexpected")
}
func (s *dailyGrantRepoStub) ListByUser(ctx context.Context, userID int64) ([]DailyBalanceGrant, error) {
	panic("unexpected")
}

func newEligibilitySvc(t *testing.T, balance float64, grantRepo DailyGrantRepository) *BillingCacheService {
	t.Helper()
	svc := NewBillingCacheService(&balanceCacheStub{balance: balance}, nil, nil, nil, nil, nil, &config.Config{}, nil)
	t.Cleanup(svc.Stop)
	if grantRepo != nil {
		svc.SetDailyGrantRepo(grantRepo)
	}
	return svc
}

func TestCheckBalanceEligibility_PositiveBalance(t *testing.T) {
	svc := newEligibilitySvc(t, 5, nil)
	// 普通分组，余额>0 放行
	require.NoError(t, svc.checkBalanceEligibility(context.Background(), 1, &Group{ID: 1}))
}

func TestCheckBalanceEligibility_ZeroBalance_NonExclusive_Rejected(t *testing.T) {
	grant := &dailyGrantRepoStub{remaining: 100} // 即便有额度，非专属分组也不查
	svc := newEligibilitySvc(t, 0, grant)
	err := svc.checkBalanceEligibility(context.Background(), 1, &Group{ID: 1, DailyBalanceEnabled: false})
	require.ErrorIs(t, err, ErrInsufficientBalance)
	require.False(t, grant.called, "非专属分组不应查询每日额度")
}

func TestCheckBalanceEligibility_ZeroBalance_ExclusiveWithGrant_Allowed(t *testing.T) {
	grant := &dailyGrantRepoStub{remaining: 3}
	svc := newEligibilitySvc(t, 0, grant)
	err := svc.checkBalanceEligibility(context.Background(), 1, &Group{ID: 7, DailyBalanceEnabled: true})
	require.NoError(t, err, "专属分组余额为0但有有效每日额度应放行")
	require.True(t, grant.called)
}

func TestCheckBalanceEligibility_ZeroBalance_ExclusiveNoGrant_Rejected(t *testing.T) {
	grant := &dailyGrantRepoStub{remaining: 0}
	svc := newEligibilitySvc(t, 0, grant)
	err := svc.checkBalanceEligibility(context.Background(), 1, &Group{ID: 7, DailyBalanceEnabled: true})
	require.ErrorIs(t, err, ErrInsufficientBalance)
	require.True(t, grant.called)
}

func TestCheckBalanceEligibility_ExclusiveGrantQueryError_FallsBackToReject(t *testing.T) {
	grant := &dailyGrantRepoStub{remaining: 100, sumErr: errors.New("db down")}
	svc := newEligibilitySvc(t, 0, grant)
	// 查询失败时不误伤放行，回退到纯余额判定（余额<=0 → 拒绝）
	err := svc.checkBalanceEligibility(context.Background(), 1, &Group{ID: 7, DailyBalanceEnabled: true})
	require.ErrorIs(t, err, ErrInsufficientBalance)
}

func TestCheckBalanceEligibility_ExclusiveButNoRepoInjected_Rejected(t *testing.T) {
	// dailyGrantRepo 未注入时，专属分组余额为 0 仍按纯余额判定拒绝
	svc := newEligibilitySvc(t, 0, nil)
	err := svc.checkBalanceEligibility(context.Background(), 1, &Group{ID: 7, DailyBalanceEnabled: true})
	require.ErrorIs(t, err, ErrInsufficientBalance)
}
