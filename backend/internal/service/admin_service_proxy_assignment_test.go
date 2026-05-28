//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type createAccountRepoStub struct {
	account *Account
}

func (s *createAccountRepoStub) Create(ctx context.Context, account *Account) error {
	s.account = account
	account.ID = 900
	return nil
}
func (s *createAccountRepoStub) GetByID(context.Context, int64) (*Account, error) {
	panic("unexpected")
}
func (s *createAccountRepoStub) GetByIDs(context.Context, []int64) ([]*Account, error) {
	panic("unexpected")
}
func (s *createAccountRepoStub) ExistsByID(context.Context, int64) (bool, error) { panic("unexpected") }
func (s *createAccountRepoStub) GetByCRSAccountID(context.Context, string) (*Account, error) {
	panic("unexpected")
}
func (s *createAccountRepoStub) FindByExtraField(context.Context, string, any) ([]Account, error) {
	panic("unexpected")
}
func (s *createAccountRepoStub) ListCRSAccountIDs(context.Context) (map[string]int64, error) {
	panic("unexpected")
}
func (s *createAccountRepoStub) Update(context.Context, *Account) error { panic("unexpected") }
func (s *createAccountRepoStub) Delete(context.Context, int64) error    { panic("unexpected") }
func (s *createAccountRepoStub) List(context.Context, pagination.PaginationParams) ([]Account, *pagination.PaginationResult, error) {
	panic("unexpected")
}
func (s *createAccountRepoStub) ListWithFilters(context.Context, pagination.PaginationParams, string, string, string, string, int64, string) ([]Account, *pagination.PaginationResult, error) {
	panic("unexpected")
}
func (s *createAccountRepoStub) ListByGroup(context.Context, int64) ([]Account, error) {
	panic("unexpected")
}
func (s *createAccountRepoStub) ListActive(context.Context) ([]Account, error) { panic("unexpected") }
func (s *createAccountRepoStub) ListByPlatform(context.Context, string) ([]Account, error) {
	panic("unexpected")
}
func (s *createAccountRepoStub) UpdateLastUsed(context.Context, int64) error { panic("unexpected") }
func (s *createAccountRepoStub) BatchUpdateLastUsed(context.Context, map[int64]time.Time) error {
	panic("unexpected")
}
func (s *createAccountRepoStub) SetError(context.Context, int64, string) error { panic("unexpected") }
func (s *createAccountRepoStub) ClearError(context.Context, int64) error       { panic("unexpected") }
func (s *createAccountRepoStub) SetSchedulable(context.Context, int64, bool) error {
	panic("unexpected")
}
func (s *createAccountRepoStub) AutoPauseExpiredAccounts(context.Context, time.Time) (int64, error) {
	panic("unexpected")
}
func (s *createAccountRepoStub) BindGroups(context.Context, int64, []int64) error {
	panic("unexpected")
}
func (s *createAccountRepoStub) ListSchedulable(context.Context) ([]Account, error) {
	panic("unexpected")
}
func (s *createAccountRepoStub) ListSchedulableByGroupID(context.Context, int64) ([]Account, error) {
	panic("unexpected")
}
func (s *createAccountRepoStub) ListSchedulableByPlatform(context.Context, string) ([]Account, error) {
	panic("unexpected")
}
func (s *createAccountRepoStub) ListSchedulableByGroupIDAndPlatform(context.Context, int64, string) ([]Account, error) {
	panic("unexpected")
}
func (s *createAccountRepoStub) ListSchedulableByPlatforms(context.Context, []string) ([]Account, error) {
	panic("unexpected")
}
func (s *createAccountRepoStub) ListSchedulableByGroupIDAndPlatforms(context.Context, int64, []string) ([]Account, error) {
	panic("unexpected")
}
func (s *createAccountRepoStub) ListSchedulableUngroupedByPlatform(context.Context, string) ([]Account, error) {
	panic("unexpected")
}
func (s *createAccountRepoStub) ListSchedulableUngroupedByPlatforms(context.Context, []string) ([]Account, error) {
	panic("unexpected")
}
func (s *createAccountRepoStub) SetRateLimited(context.Context, int64, time.Time) error {
	panic("unexpected")
}
func (s *createAccountRepoStub) SetModelRateLimit(context.Context, int64, string, time.Time) error {
	panic("unexpected")
}
func (s *createAccountRepoStub) SetOverloaded(context.Context, int64, time.Time) error {
	panic("unexpected")
}
func (s *createAccountRepoStub) SetTempUnschedulable(context.Context, int64, time.Time, string) error {
	panic("unexpected")
}
func (s *createAccountRepoStub) ClearTempUnschedulable(context.Context, int64) error {
	panic("unexpected")
}
func (s *createAccountRepoStub) ClearRateLimit(context.Context, int64) error { panic("unexpected") }
func (s *createAccountRepoStub) ClearAntigravityQuotaScopes(context.Context, int64) error {
	panic("unexpected")
}
func (s *createAccountRepoStub) ClearModelRateLimits(context.Context, int64) error {
	panic("unexpected")
}
func (s *createAccountRepoStub) UpdateSessionWindow(context.Context, int64, *time.Time, *time.Time, string) error {
	panic("unexpected")
}
func (s *createAccountRepoStub) UpdateExtra(context.Context, int64, map[string]any) error {
	panic("unexpected")
}
func (s *createAccountRepoStub) BulkUpdate(context.Context, []int64, AccountBulkUpdate) (int64, error) {
	panic("unexpected")
}
func (s *createAccountRepoStub) IncrementQuotaUsed(context.Context, int64, float64) error {
	panic("unexpected")
}
func (s *createAccountRepoStub) ResetQuotaUsed(context.Context, int64) error { panic("unexpected") }

type createAccountGroupRepoStub struct{}

func (s *createAccountGroupRepoStub) Create(context.Context, *Group) error { panic("unexpected") }
func (s *createAccountGroupRepoStub) GetByID(context.Context, int64) (*Group, error) {
	panic("unexpected")
}
func (s *createAccountGroupRepoStub) GetByIDLite(context.Context, int64) (*Group, error) {
	panic("unexpected")
}
func (s *createAccountGroupRepoStub) Update(context.Context, *Group) error { panic("unexpected") }
func (s *createAccountGroupRepoStub) Delete(context.Context, int64) error  { panic("unexpected") }
func (s *createAccountGroupRepoStub) DeleteCascade(context.Context, int64) ([]int64, error) {
	panic("unexpected")
}
func (s *createAccountGroupRepoStub) List(context.Context, pagination.PaginationParams) ([]Group, *pagination.PaginationResult, error) {
	panic("unexpected")
}
func (s *createAccountGroupRepoStub) ListWithFilters(context.Context, pagination.PaginationParams, string, string, string, *bool) ([]Group, *pagination.PaginationResult, error) {
	panic("unexpected")
}
func (s *createAccountGroupRepoStub) ListActive(context.Context) ([]Group, error) {
	panic("unexpected")
}
func (s *createAccountGroupRepoStub) ListActiveByPlatform(context.Context, string) ([]Group, error) {
	return nil, nil
}
func (s *createAccountGroupRepoStub) ExistsByName(context.Context, string) (bool, error) {
	panic("unexpected")
}
func (s *createAccountGroupRepoStub) GetAccountCount(context.Context, int64) (int64, int64, error) {
	panic("unexpected")
}
func (s *createAccountGroupRepoStub) DeleteAccountGroupsByGroupID(context.Context, int64) (int64, error) {
	panic("unexpected")
}
func (s *createAccountGroupRepoStub) GetAccountIDsByGroupIDs(context.Context, []int64) ([]int64, error) {
	panic("unexpected")
}
func (s *createAccountGroupRepoStub) BindAccountsToGroup(context.Context, int64, []int64) error {
	panic("unexpected")
}
func (s *createAccountGroupRepoStub) UpdateSortOrders(context.Context, []GroupSortOrderUpdate) error {
	panic("unexpected")
}

type assignableProxyRepoStub struct {
	proxyRepoStub
	proxies []ProxyWithAccountCount
}

func (s *assignableProxyRepoStub) ListAssignableWithAccountCount(context.Context) ([]ProxyWithAccountCount, error) {
	return append([]ProxyWithAccountCount(nil), s.proxies...), nil
}

func TestAdminServiceCreateAccountAutoAssignsUSProxyFirst(t *testing.T) {
	accountRepo := &createAccountRepoStub{}
	proxyRepo := &assignableProxyRepoStub{
		proxies: []ProxyWithAccountCount{
			{Proxy: Proxy{ID: 20, Status: StatusActive, FailureCount: 0}, AccountCount: 0, CountryCode: "JP"},
			{Proxy: Proxy{ID: 10, Status: StatusActive, FailureCount: 2}, AccountCount: 1, CountryCode: "US"},
			{Proxy: Proxy{ID: 11, Status: StatusActive, FailureCount: 0}, AccountCount: 0, Country: "United States"},
		},
	}
	svc := &adminServiceImpl{
		accountRepo: accountRepo,
		groupRepo:   &createAccountGroupRepoStub{},
		proxyRepo:   proxyRepo,
	}

	account, err := svc.CreateAccount(context.Background(), &CreateAccountInput{
		Name:                 "openai-oauth",
		Platform:             PlatformOpenAI,
		Type:                 AccountTypeOAuth,
		SkipDefaultGroupBind: true,
	})

	require.NoError(t, err)
	require.NotNil(t, account.ProxyID)
	require.Equal(t, int64(11), *account.ProxyID)
	require.Equal(t, int64(11), *accountRepo.account.ProxyID)
}

func TestAdminServiceCreateAccountKeepsExplicitProxy(t *testing.T) {
	accountRepo := &createAccountRepoStub{}
	proxyRepo := &assignableProxyRepoStub{
		proxies: []ProxyWithAccountCount{{Proxy: Proxy{ID: 10, Status: StatusActive}, CountryCode: "US"}},
	}
	explicitProxyID := int64(99)
	svc := &adminServiceImpl{
		accountRepo: accountRepo,
		groupRepo:   &createAccountGroupRepoStub{},
		proxyRepo:   proxyRepo,
	}

	account, err := svc.CreateAccount(context.Background(), &CreateAccountInput{
		Name:                 "openai-oauth",
		Platform:             PlatformOpenAI,
		Type:                 AccountTypeOAuth,
		ProxyID:              &explicitProxyID,
		SkipDefaultGroupBind: true,
	})

	require.NoError(t, err)
	require.NotNil(t, account.ProxyID)
	require.Equal(t, explicitProxyID, *account.ProxyID)
}
