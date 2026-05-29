//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type proxyLatencyCacheStub struct {
	latencies map[int64]*ProxyLatencyInfo
}

func (s *proxyLatencyCacheStub) GetProxyLatencies(context.Context, []int64) (map[int64]*ProxyLatencyInfo, error) {
	out := make(map[int64]*ProxyLatencyInfo, len(s.latencies))
	for id, latency := range s.latencies {
		out[id] = latency
	}
	return out, nil
}

func (s *proxyLatencyCacheStub) SetProxyLatency(context.Context, int64, *ProxyLatencyInfo) error {
	return nil
}

type createAccountRepoStub struct {
	account            *Account
	boundAccountID     int64
	boundGroupIDs      []int64
	getByIDAccount     *Account
	accountsByPlatform []Account
	bulkProxyUpdates   map[int64]int64
	extraUpdates       map[int64]map[string]any
}

func (s *createAccountRepoStub) Create(ctx context.Context, account *Account) error {
	s.account = account
	account.ID = 900
	return nil
}
func (s *createAccountRepoStub) GetByID(context.Context, int64) (*Account, error) {
	if s.getByIDAccount != nil {
		cp := *s.getByIDAccount
		return &cp, nil
	}
	if s.account != nil {
		cp := *s.account
		return &cp, nil
	}
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
func (s *createAccountRepoStub) Update(_ context.Context, account *Account) error {
	s.account = account
	if s.getByIDAccount != nil {
		cp := *account
		s.getByIDAccount = &cp
	}
	return nil
}
func (s *createAccountRepoStub) Delete(context.Context, int64) error { panic("unexpected") }
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
	return append([]Account(nil), s.accountsByPlatform...), nil
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
func (s *createAccountRepoStub) BindGroups(_ context.Context, accountID int64, groupIDs []int64) error {
	s.boundAccountID = accountID
	s.boundGroupIDs = append([]int64(nil), groupIDs...)
	return nil
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
func (s *createAccountRepoStub) SetModelRateLimit(context.Context, int64, string, time.Time, ...string) error {
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
func (s *createAccountRepoStub) UpdateExtra(_ context.Context, id int64, updates map[string]any) error {
	if s.extraUpdates == nil {
		s.extraUpdates = make(map[int64]map[string]any)
	}
	s.extraUpdates[id] = updates
	return nil
}
func (s *createAccountRepoStub) BulkUpdate(_ context.Context, ids []int64, updates AccountBulkUpdate) (int64, error) {
	if len(ids) == 1 && updates.ProxyID != nil {
		if s.bulkProxyUpdates == nil {
			s.bulkProxyUpdates = make(map[int64]int64)
		}
		s.bulkProxyUpdates[ids[0]] = *updates.ProxyID
		if s.getByIDAccount != nil {
			cp := *s.getByIDAccount
			cp.ProxyID = updates.ProxyID
			s.getByIDAccount = &cp
		}
		return 1, nil
	}
	panic("unexpected")
}
func (s *createAccountRepoStub) IncrementQuotaUsed(context.Context, int64, float64) error {
	panic("unexpected")
}
func (s *createAccountRepoStub) ResetQuotaUsed(context.Context, int64) error { panic("unexpected") }

type createAccountGroupRepoStub struct {
	groups []Group
}

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
func (s *createAccountGroupRepoStub) ListActiveByPlatform(_ context.Context, _ string) ([]Group, error) {
	return append([]Group(nil), s.groups...), nil
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

func TestAdminServiceCreateAccountAutoAssignsLowestLatencyPrimaryProxyWithinLimit(t *testing.T) {
	accountRepo := &createAccountRepoStub{}
	fastLatency := int64(259)
	slowLatency := int64(410)
	proxyRepo := &assignableProxyRepoStub{
		proxies: []ProxyWithAccountCount{
			{Proxy: Proxy{ID: 20, Status: StatusActive, FailureCount: 0}, AccountCount: 0},
			{Proxy: Proxy{ID: 10, Status: StatusActive, FailureCount: 0}, AccountCount: 19},
		},
	}
	svc := &adminServiceImpl{
		accountRepo: accountRepo,
		groupRepo:   &createAccountGroupRepoStub{},
		proxyRepo:   proxyRepo,
		proxyLatencyCache: &proxyLatencyCacheStub{latencies: map[int64]*ProxyLatencyInfo{
			10: {Success: true, LatencyMs: &fastLatency},
			20: {Success: true, LatencyMs: &slowLatency},
		}},
	}

	account, err := svc.CreateAccount(context.Background(), &CreateAccountInput{
		Name:                 "openai-oauth",
		Platform:             PlatformOpenAI,
		Type:                 AccountTypeOAuth,
		SkipDefaultGroupBind: true,
	})

	require.NoError(t, err)
	require.NotNil(t, account.ProxyID)
	require.Equal(t, int64(10), *account.ProxyID)
	require.Equal(t, int64(10), *accountRepo.account.ProxyID)
}

func TestAdminServiceCreateAccountMovesToNextLatencyProxyAfterPrimaryLimit(t *testing.T) {
	accountRepo := &createAccountRepoStub{}
	fastLatency := int64(259)
	nextLatency := int64(410)
	proxyRepo := &assignableProxyRepoStub{
		proxies: []ProxyWithAccountCount{
			{Proxy: Proxy{ID: 10, Status: StatusActive, FailureCount: 0}, AccountCount: 20},
			{Proxy: Proxy{ID: 20, Status: StatusActive, FailureCount: 0}, AccountCount: 0},
		},
	}
	svc := &adminServiceImpl{
		accountRepo: accountRepo,
		groupRepo:   &createAccountGroupRepoStub{},
		proxyRepo:   proxyRepo,
		proxyLatencyCache: &proxyLatencyCacheStub{latencies: map[int64]*ProxyLatencyInfo{
			10: {Success: true, LatencyMs: &fastLatency},
			20: {Success: true, LatencyMs: &nextLatency},
		}},
	}

	account, err := svc.CreateAccount(context.Background(), &CreateAccountInput{
		Name:                 "openai-oauth",
		Platform:             PlatformOpenAI,
		Type:                 AccountTypeOAuth,
		SkipDefaultGroupBind: true,
	})

	require.NoError(t, err)
	require.NotNil(t, account.ProxyID)
	require.Equal(t, int64(20), *account.ProxyID)
	require.Equal(t, int64(20), *accountRepo.account.ProxyID)
}

func TestAdminServiceCreateAccountMovesByLatencyAfterSecondaryLimit(t *testing.T) {
	accountRepo := &createAccountRepoStub{}
	fastLatency := int64(259)
	nextLatency := int64(410)
	thirdLatency := int64(520)
	proxyRepo := &assignableProxyRepoStub{
		proxies: []ProxyWithAccountCount{
			{Proxy: Proxy{ID: 10, Status: StatusActive, FailureCount: 0}, AccountCount: 20},
			{Proxy: Proxy{ID: 20, Status: StatusActive, FailureCount: 0}, AccountCount: 10},
			{Proxy: Proxy{ID: 30, Status: StatusActive, FailureCount: 0}, AccountCount: 0},
		},
	}
	svc := &adminServiceImpl{
		accountRepo: accountRepo,
		groupRepo:   &createAccountGroupRepoStub{},
		proxyRepo:   proxyRepo,
		proxyLatencyCache: &proxyLatencyCacheStub{latencies: map[int64]*ProxyLatencyInfo{
			10: {Success: true, LatencyMs: &fastLatency},
			20: {Success: true, LatencyMs: &nextLatency},
			30: {Success: true, LatencyMs: &thirdLatency},
		}},
	}

	account, err := svc.CreateAccount(context.Background(), &CreateAccountInput{
		Name:                 "openai-oauth",
		Platform:             PlatformOpenAI,
		Type:                 AccountTypeOAuth,
		SkipDefaultGroupBind: true,
	})

	require.NoError(t, err)
	require.NotNil(t, account.ProxyID)
	require.Equal(t, int64(30), *account.ProxyID)
	require.Equal(t, int64(30), *accountRepo.account.ProxyID)
}

func TestOpenAIUpdateCodexUsageSnapshotMovesExhaustedAccountToSlowestProxy(t *testing.T) {
	accountRepo := &createAccountRepoStub{
		getByIDAccount: &Account{
			ID:          123,
			Name:        "regular",
			Platform:    PlatformOpenAI,
			Type:        AccountTypeOAuth,
			Status:      StatusActive,
			Schedulable: true,
		},
	}
	fast := int64(259)
	slow := int64(520)
	svc := &OpenAIGatewayService{
		accountRepo: accountRepo,
		proxyRepo: &assignableProxyRepoStub{proxies: []ProxyWithAccountCount{
			{Proxy: Proxy{ID: 10, Status: StatusActive}},
			{Proxy: Proxy{ID: 20, Status: StatusActive}},
		}},
		proxyLatencyCache: &proxyLatencyCacheStub{latencies: map[int64]*ProxyLatencyInfo{
			10: {Success: true, LatencyMs: &fast},
			20: {Success: true, LatencyMs: &slow},
		}},
	}

	err := svc.moveOpenAICodexExhaustedAccountToSlowestProxy(context.Background(), 123)

	require.NoError(t, err)
	require.NotNil(t, accountRepo.getByIDAccount.ProxyID)
	require.Equal(t, int64(20), *accountRepo.getByIDAccount.ProxyID)
}

func TestOpenAIRecoverAndRebalanceCodexAccounts(t *testing.T) {
	slowProxyID := int64(20)
	fastProxyID := int64(10)
	accountRepo := &createAccountRepoStub{
		accountsByPlatform: []Account{
			{
				ID:          1,
				Name:        "exhausted",
				Platform:    PlatformOpenAI,
				Type:        AccountTypeOAuth,
				Status:      StatusActive,
				Schedulable: true,
				ProxyID:     &fastProxyID,
				Extra: map[string]any{
					openAICodex7dUsedPercentExtraKey: 99.0,
				},
			},
			{
				ID:          2,
				Name:        "recovered",
				Platform:    PlatformOpenAI,
				Type:        AccountTypeOAuth,
				Status:      StatusActive,
				Schedulable: true,
				ProxyID:     &slowProxyID,
				Extra: map[string]any{
					openAICodexPrimaryUsedPercentExtraKey: 10.0,
					openAICodex5hUsedPercentExtraKey:      0.0,
					openAICodex7dUsedPercentExtraKey:      10.0,
				},
			},
			{
				ID:          3,
				Name:        "oto",
				Platform:    PlatformOpenAI,
				Type:        AccountTypeOAuth,
				Status:      StatusActive,
				Schedulable: true,
			},
		},
	}
	fast := int64(100)
	slow := int64(500)
	svc := &OpenAIGatewayService{
		accountRepo: accountRepo,
		proxyRepo: &assignableProxyRepoStub{proxies: []ProxyWithAccountCount{
			{Proxy: Proxy{ID: fastProxyID, Status: StatusActive}},
			{Proxy: Proxy{ID: slowProxyID, Status: StatusActive}},
		}},
		proxyLatencyCache: &proxyLatencyCacheStub{latencies: map[int64]*ProxyLatencyInfo{
			fastProxyID: {Success: true, LatencyMs: &fast},
			slowProxyID: {Success: true, LatencyMs: &slow},
		}},
	}

	err := svc.RecoverAndRebalanceOpenAICodexAccounts(context.Background())

	require.NoError(t, err)
	require.Equal(t, slowProxyID, accountRepo.bulkProxyUpdates[1])
	require.Equal(t, fastProxyID, accountRepo.bulkProxyUpdates[2])
	require.Contains(t, accountRepo.extraUpdates[2], openAICodexRecoveredFromSlowPoolExtraKey)
	require.NotContains(t, accountRepo.bulkProxyUpdates, int64(3))
}

func TestNextOpenAICodexQuotaResetRecoveryAt(t *testing.T) {
	now := time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC)
	earlierReset := now.Add(2 * time.Minute).Format(time.RFC3339)
	laterReset := now.Add(10 * time.Minute).Format(time.RFC3339)
	pastReset := now.Add(-time.Minute).Format(time.RFC3339)
	accountRepo := &createAccountRepoStub{
		accountsByPlatform: []Account{
			{
				ID:       1,
				Name:     "future-5h",
				Platform: PlatformOpenAI,
				Type:     AccountTypeOAuth,
				Extra: map[string]any{
					openAICodex5hUsedPercentExtraKey: 100.0,
					openAICodex5hResetAtExtraKey:     earlierReset,
				},
			},
			{
				ID:       2,
				Name:     "future-7d",
				Platform: PlatformOpenAI,
				Type:     AccountTypeOAuth,
				Extra: map[string]any{
					openAICodex7dUsedPercentExtraKey: 100.0,
					openAICodex7dResetAtExtraKey:     laterReset,
				},
			},
			{
				ID:       3,
				Name:     "already-reset",
				Platform: PlatformOpenAI,
				Type:     AccountTypeOAuth,
				Extra: map[string]any{
					openAICodex5hUsedPercentExtraKey: 100.0,
					openAICodex5hResetAtExtraKey:     pastReset,
				},
			},
			{
				ID:       4,
				Name:     "oto",
				Platform: PlatformOpenAI,
				Type:     AccountTypeOAuth,
				Extra: map[string]any{
					openAICodex5hUsedPercentExtraKey: 100.0,
					openAICodex5hResetAtExtraKey:     now.Add(time.Minute).Format(time.RFC3339),
				},
			},
		},
	}

	nextAt, err := nextOpenAICodexQuotaResetRecoveryAt(context.Background(), accountRepo, now)

	require.NoError(t, err)
	require.Equal(t, now.Add(2*time.Minute).Add(openAICodexQuotaResetWakeDelay), nextAt)
}

func TestNextOpenAICodexQuotaResetRecoveryAtNoFutureReset(t *testing.T) {
	now := time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC)
	accountRepo := &createAccountRepoStub{
		accountsByPlatform: []Account{
			{
				ID:       1,
				Name:     "low-usage",
				Platform: PlatformOpenAI,
				Type:     AccountTypeOAuth,
				Extra: map[string]any{
					openAICodex5hUsedPercentExtraKey: 94.9,
					openAICodex5hResetAtExtraKey:     now.Add(time.Minute).Format(time.RFC3339),
				},
			},
		},
	}

	nextAt, err := nextOpenAICodexQuotaResetRecoveryAt(context.Background(), accountRepo, now)

	require.NoError(t, err)
	require.True(t, nextAt.IsZero())
}

func TestAdminServiceCreateAccountSkipsFailedProxy(t *testing.T) {
	accountRepo := &createAccountRepoStub{}
	fastLatency := int64(259)
	slowLatency := int64(410)
	proxyRepo := &assignableProxyRepoStub{
		proxies: []ProxyWithAccountCount{
			{Proxy: Proxy{ID: 10, Status: StatusActive, FailureCount: 0}, AccountCount: 0, CountryCode: "JP"},
			{Proxy: Proxy{ID: 20, Status: StatusActive, FailureCount: 0}, AccountCount: 0, CountryCode: "JP"},
		},
	}
	svc := &adminServiceImpl{
		accountRepo: accountRepo,
		groupRepo:   &createAccountGroupRepoStub{},
		proxyRepo:   proxyRepo,
		proxyLatencyCache: &proxyLatencyCacheStub{latencies: map[int64]*ProxyLatencyInfo{
			10: {Success: false, LatencyMs: &fastLatency, QualityStatus: "failed"},
			20: {Success: true, LatencyMs: &slowLatency},
		}},
	}

	account, err := svc.CreateAccount(context.Background(), &CreateAccountInput{
		Name:                 "openai-oauth",
		Platform:             PlatformOpenAI,
		Type:                 AccountTypeOAuth,
		SkipDefaultGroupBind: true,
	})

	require.NoError(t, err)
	require.NotNil(t, account.ProxyID)
	require.Equal(t, int64(20), *account.ProxyID)
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

func TestAdminServiceCreateAccountNeverAssignsProxyToOto(t *testing.T) {
	accountRepo := &createAccountRepoStub{}
	proxyRepo := &assignableProxyRepoStub{
		proxies: []ProxyWithAccountCount{{Proxy: Proxy{ID: 10, Status: StatusActive}, CountryCode: "JP"}},
	}
	explicitProxyID := int64(10)
	svc := &adminServiceImpl{
		accountRepo: accountRepo,
		groupRepo:   &createAccountGroupRepoStub{},
		proxyRepo:   proxyRepo,
	}

	account, err := svc.CreateAccount(context.Background(), &CreateAccountInput{
		Name:                 "oto",
		Platform:             PlatformOpenAI,
		Type:                 AccountTypeOAuth,
		ProxyID:              &explicitProxyID,
		SkipDefaultGroupBind: true,
	})

	require.NoError(t, err)
	require.Nil(t, account.ProxyID)
	require.Nil(t, accountRepo.account.ProxyID)
}

func TestAdminServiceUpdateAccountClearsProxyForOto2(t *testing.T) {
	proxyID := int64(10)
	accountRepo := &createAccountRepoStub{
		getByIDAccount: &Account{
			ID:          42,
			Name:        "regular",
			Platform:    PlatformOpenAI,
			Type:        AccountTypeOAuth,
			ProxyID:     &proxyID,
			Credentials: map[string]any{},
			Extra:       map[string]any{},
			Status:      StatusActive,
			Schedulable: true,
		},
	}
	svc := &adminServiceImpl{
		accountRepo: accountRepo,
		groupRepo:   &createAccountGroupRepoStub{},
	}

	updated, err := svc.UpdateAccount(context.Background(), 42, &UpdateAccountInput{Name: "oto2"})

	require.NoError(t, err)
	require.NotNil(t, updated)
	require.Equal(t, "oto2", updated.Name)
	require.Nil(t, updated.ProxyID)
	require.Nil(t, accountRepo.account.ProxyID)
}

func TestAdminServiceCreateOpenAIOAuthDefaultsCompatibilityAndAllOpenAIGroups(t *testing.T) {
	accountRepo := &createAccountRepoStub{}
	svc := &adminServiceImpl{
		accountRepo: accountRepo,
		groupRepo: &createAccountGroupRepoStub{groups: []Group{
			{ID: 7, Name: "special", Platform: PlatformOpenAI, Status: StatusActive},
			{ID: 8, Name: "day-card", Platform: PlatformOpenAI, Status: StatusActive},
			{ID: 9, Name: "week-card", Platform: PlatformOpenAI, Status: StatusActive},
		}},
	}

	account, err := svc.CreateAccount(context.Background(), &CreateAccountInput{
		Name:     "openai-oauth",
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
	})

	require.NoError(t, err)
	require.NotNil(t, account)
	require.Equal(t, int64(900), accountRepo.boundAccountID)
	require.Equal(t, []int64{7, 8, 9}, accountRepo.boundGroupIDs)
	require.Equal(t, false, account.Extra["openai_passthrough"])
	require.Equal(t, OpenAIWSIngressModeOff, account.Extra["openai_oauth_responses_websockets_v2_mode"])
	require.Equal(t, false, account.Extra["openai_oauth_responses_websockets_v2_enabled"])
}

func TestAdminServiceCreateOpenAIAPIKeyKeepsExplicitDefaultGroup(t *testing.T) {
	accountRepo := &createAccountRepoStub{}
	svc := &adminServiceImpl{
		accountRepo: accountRepo,
		groupRepo: &createAccountGroupRepoStub{groups: []Group{
			{ID: 7, Name: "special", Platform: PlatformOpenAI, Status: StatusActive},
			{ID: 8, Name: "day-card", Platform: PlatformOpenAI, Status: StatusActive},
			{ID: 9, Name: "week-card", Platform: PlatformOpenAI, Status: StatusActive},
		}},
	}

	account, err := svc.CreateAccount(context.Background(), &CreateAccountInput{
		Name:     "openai-apikey",
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
	})

	require.NoError(t, err)
	require.NotNil(t, account)
	require.Equal(t, []int64{7, 8, 9}, accountRepo.boundGroupIDs)
	require.Nil(t, account.Extra)
}

func TestAdminServiceCreateOpenAIOAuthForcesNonPassthroughCompatibilityExtra(t *testing.T) {
	accountRepo := &createAccountRepoStub{}
	svc := &adminServiceImpl{
		accountRepo: accountRepo,
		groupRepo:   &createAccountGroupRepoStub{},
	}

	account, err := svc.CreateAccount(context.Background(), &CreateAccountInput{
		Name:                 "openai-oauth",
		Platform:             PlatformOpenAI,
		Type:                 AccountTypeOAuth,
		Extra:                map[string]any{"openai_passthrough": true, "openai_oauth_responses_websockets_v2_enabled": true, "openai_oauth_responses_websockets_v2_mode": OpenAIWSIngressModePassthrough},
		SkipDefaultGroupBind: true,
	})

	require.NoError(t, err)
	require.NotNil(t, account)
	require.Equal(t, false, account.Extra["openai_passthrough"])
	require.Equal(t, false, account.Extra["openai_oauth_responses_websockets_v2_enabled"])
	require.Equal(t, OpenAIWSIngressModeOff, account.Extra["openai_oauth_responses_websockets_v2_mode"])
}
