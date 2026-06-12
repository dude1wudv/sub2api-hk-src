package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

// --- stubs ---

type grantRepoCapture struct {
	created   *DailyBalanceGrant
	createErr error
	nextID    int64
}

func (s *grantRepoCapture) CreateGrant(ctx context.Context, g *DailyBalanceGrant) (int64, error) {
	if s.createErr != nil {
		return 0, s.createErr
	}
	cp := *g
	s.created = &cp
	if s.nextID == 0 {
		s.nextID = 555
	}
	return s.nextID, nil
}
func (s *grantRepoCapture) SumActiveRemaining(ctx context.Context, userID, groupID int64, now time.Time) (float64, error) {
	return 0, nil
}
func (s *grantRepoCapture) ListActiveByExpiry(ctx context.Context, userID, groupID int64, now time.Time) ([]DailyBalanceGrant, error) {
	return nil, nil
}
func (s *grantRepoCapture) AtomicDecrement(ctx context.Context, grantID int64, amount float64) (bool, error) {
	return false, nil
}
func (s *grantRepoCapture) MarkExpired(ctx context.Context, now time.Time) (int64, error) {
	return 0, nil
}
func (s *grantRepoCapture) ListByUser(ctx context.Context, userID int64) ([]DailyBalanceGrant, error) {
	return nil, nil
}

// dailyGroupRepoStub 仅实现 GetByID，其余 GroupRepository 方法 panic（本测试不应触达）。
type dailyGroupRepoStub struct {
	group *Group
	err   error
}

func (s *dailyGroupRepoStub) Create(ctx context.Context, g *Group) error { panic("unexpected") }
func (s *dailyGroupRepoStub) GetByID(ctx context.Context, id int64) (*Group, error) {
	return s.group, s.err
}
func (s *dailyGroupRepoStub) GetByIDLite(ctx context.Context, id int64) (*Group, error) {
	return s.group, s.err
}
func (s *dailyGroupRepoStub) Update(ctx context.Context, g *Group) error          { panic("unexpected") }
func (s *dailyGroupRepoStub) Delete(ctx context.Context, id int64) error          { panic("unexpected") }
func (s *dailyGroupRepoStub) DeleteCascade(ctx context.Context, id int64) ([]int64, error) {
	panic("unexpected")
}
func (s *dailyGroupRepoStub) List(ctx context.Context, p pagination.PaginationParams) ([]Group, *pagination.PaginationResult, error) {
	panic("unexpected")
}
func (s *dailyGroupRepoStub) ListWithFilters(ctx context.Context, p pagination.PaginationParams, platform, status, search string, isExclusive *bool) ([]Group, *pagination.PaginationResult, error) {
	panic("unexpected")
}
func (s *dailyGroupRepoStub) ListActive(ctx context.Context) ([]Group, error) { panic("unexpected") }
func (s *dailyGroupRepoStub) ListActiveByPlatform(ctx context.Context, platform string) ([]Group, error) {
	panic("unexpected")
}
func (s *dailyGroupRepoStub) ExistsByName(ctx context.Context, name string) (bool, error) {
	panic("unexpected")
}
func (s *dailyGroupRepoStub) GetAccountCount(ctx context.Context, groupID int64) (int64, int64, error) {
	panic("unexpected")
}
func (s *dailyGroupRepoStub) DeleteAccountGroupsByGroupID(ctx context.Context, groupID int64) (int64, error) {
	panic("unexpected")
}
func (s *dailyGroupRepoStub) GetAccountIDsByGroupIDs(ctx context.Context, groupIDs []int64) ([]int64, error) {
	panic("unexpected")
}
func (s *dailyGroupRepoStub) BindAccountsToGroup(ctx context.Context, groupID int64, accountIDs []int64) error {
	panic("unexpected")
}
func (s *dailyGroupRepoStub) UpdateSortOrders(ctx context.Context, updates []GroupSortOrderUpdate) error {
	panic("unexpected")
}

type invalidatorSpy struct{ called int64 }

func (s *invalidatorSpy) InvalidateUserBalance(ctx context.Context, userID int64) error {
	s.called++
	return nil
}

// --- tests ---

func TestGrantDaily_Success(t *testing.T) {
	grantRepo := &grantRepoCapture{nextID: 42}
	groupRepo := &dailyGroupRepoStub{group: &Group{ID: 7, DailyBalanceEnabled: true}}
	spy := &invalidatorSpy{}
	svc := NewDailyGrantService(grantRepo, groupRepo, spy)

	before := time.Now()
	grant, err := svc.GrantDaily(context.Background(), GrantDailyInput{
		UserID: 1, GroupID: 7, Amount: 10,
	})
	require.NoError(t, err)
	require.Equal(t, int64(42), grant.ID)
	require.InDelta(t, 10.0, grant.Remaining, 1e-9)
	require.Equal(t, domain.DailyGrantStatusActive, grant.Status)
	require.Equal(t, domain.DailyGrantSourceAdmin, grant.Source)
	// 有效期约为 now+24h
	require.WithinDuration(t, before.Add(24*time.Hour), grant.ExpiresAt, 5*time.Second)
	// 缓存被失效一次
	require.Equal(t, int64(1), spy.called)
	// 仓储确实收到了正确的 Grant
	require.NotNil(t, grantRepo.created)
	require.Equal(t, int64(7), grantRepo.created.GroupID)
}

func TestGrantDaily_RejectNonExclusiveGroup(t *testing.T) {
	grantRepo := &grantRepoCapture{}
	groupRepo := &dailyGroupRepoStub{group: &Group{ID: 3, DailyBalanceEnabled: false}}
	svc := NewDailyGrantService(grantRepo, groupRepo, &invalidatorSpy{})

	_, err := svc.GrantDaily(context.Background(), GrantDailyInput{UserID: 1, GroupID: 3, Amount: 5})
	require.ErrorIs(t, err, ErrDailyGrantGroupNotExclusive)
	require.Nil(t, grantRepo.created, "非专属分组不应创建 Grant")
}

func TestGrantDaily_RejectNonPositiveAmount(t *testing.T) {
	grantRepo := &grantRepoCapture{}
	// 金额校验先于分组查询：groupRepo.GetByID 不应被触达
	svc := NewDailyGrantService(grantRepo, &dailyGroupRepoStub{group: &Group{DailyBalanceEnabled: true}}, &invalidatorSpy{})

	_, err := svc.GrantDaily(context.Background(), GrantDailyInput{UserID: 1, GroupID: 7, Amount: 0})
	require.ErrorIs(t, err, ErrDailyGrantAmountInvalid)
	require.Nil(t, grantRepo.created)
}

func TestGrantDaily_RedeemSourceRef(t *testing.T) {
	grantRepo := &grantRepoCapture{nextID: 9}
	groupRepo := &dailyGroupRepoStub{group: &Group{ID: 7, DailyBalanceEnabled: true}}
	svc := NewDailyGrantService(grantRepo, groupRepo, &invalidatorSpy{})

	code := "REDEEM-XYZ"
	grant, err := svc.GrantDaily(context.Background(), GrantDailyInput{
		UserID: 1, GroupID: 7, Amount: 3,
		Source: domain.DailyGrantSourceRedeem, SourceRef: &code,
	})
	require.NoError(t, err)
	require.Equal(t, domain.DailyGrantSourceRedeem, grant.Source)
	require.NotNil(t, grant.SourceRef)
	require.Equal(t, code, *grant.SourceRef)
}
