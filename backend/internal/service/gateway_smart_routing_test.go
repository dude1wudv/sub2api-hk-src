//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type smartRoutingGatewayAccountsRepo struct {
	AccountRepository
	accounts  []Account
	listCalls int
}

func (r *smartRoutingGatewayAccountsRepo) ListSchedulableByGroupIDAndPlatforms(_ context.Context, _ int64, _ []string) ([]Account, error) {
	r.listCalls++
	return r.accounts, nil
}

type smartRoutingGatewayConcurrencyCache struct {
	ConcurrencyCache
	acquired int
	released int
	busy     bool
}

type smartRoutingGatewayGroupRepo struct {
	GroupRepository
	group *Group
}

func (r *smartRoutingGatewayGroupRepo) GetByID(context.Context, int64) (*Group, error) {
	return r.group, nil
}

func (r *smartRoutingGatewayGroupRepo) GetByIDLite(context.Context, int64) (*Group, error) {
	return r.group, nil
}

func (c *smartRoutingGatewayConcurrencyCache) AcquireAccountSlot(context.Context, int64, int, string) (bool, error) {
	c.acquired++
	return !c.busy, nil
}

func (c *smartRoutingGatewayConcurrencyCache) ReleaseAccountSlot(context.Context, int64, string) error {
	c.released++
	return nil
}

func TestGatewaySmartRoutingCatalogUsesExplicitAndDefaultClaudeModels(t *testing.T) {
	defaultAccount := &Account{Platform: PlatformAnthropic}
	explicitAccount := &Account{Platform: PlatformAnthropic, Credentials: map[string]any{
		"model_mapping": map[string]any{
			"claude-custom": "claude-upstream-custom",
			"claude-alias":  "claude-upstream-alias",
			"claude-*":      "claude-wildcard",
			"*":             "claude-catch-all",
		},
	}}
	mixedAccount := &Account{Platform: PlatformAntigravity, Credentials: map[string]any{
		"mixed_scheduling": true,
		"model_mapping": map[string]any{
			"claude-via-mixed": "claude-3-7-sonnet",
			"gpt-via-mixed":    "gpt-5",
		},
	}}

	defaults := smartRoutingAccountModels(defaultAccount)
	require.NotEmpty(t, defaults, "unmapped Anthropic accounts must expose the project's default Claude catalog")
	require.True(t, gatewaySmartRoutingAccountClaims(defaultAccount, defaults[0]))
	require.ElementsMatch(t, []string{"claude-custom", "claude-alias"}, gatewaySmartRoutingAccountModels(explicitAccount))
	require.False(t, gatewaySmartRoutingAccountClaims(explicitAccount, "claude-anything"), "wildcard mappings must not declare arbitrary Claude models")
	require.Equal(t, []string{"claude-via-mixed"}, gatewaySmartRoutingAccountModels(mixedAccount), "mixed accounts may claim only mappings that forward to Claude")
	require.False(t, gatewaySmartRoutingAccountClaims(mixedAccount, "gpt-via-mixed"))
}

func TestGatewaySmartRoutingModelsAggregatesMixedCatalogAndAppliesGroupAllowlist(t *testing.T) {
	groupID := int64(91)
	group := &Group{ID: groupID, Platform: PlatformAnthropic, ModelAllowlist: GroupModelAllowlist{Enabled: true, Models: []string{"claude-custom", "claude-via-mixed"}}}
	repo := &smartRoutingGatewayAccountsRepo{accounts: []Account{
		{Platform: PlatformAnthropic, Credentials: map[string]any{"model_mapping": map[string]any{"claude-custom": "claude-upstream", "claude-hidden": "claude-hidden"}}},
		{Platform: PlatformAntigravity, Extra: map[string]any{"mixed_scheduling": true}, Credentials: map[string]any{"model_mapping": map[string]any{"claude-via-mixed": "claude-3-7-sonnet", "gpt-via-mixed": "gpt-5"}}},
	}}
	svc := &GatewayService{accountRepo: repo}
	got, err := svc.SmartRoutingModels(context.Background(), &APIKey{GroupID: &groupID, Group: group})
	require.NoError(t, err)
	require.Equal(t, []string{"claude-custom", "claude-via-mixed"}, got)
}

func TestFinishSmartRoutingReleasesFailedSessionOnceAndKeepsServedSession(t *testing.T) {
	cache := newSessionLimitReleaseCacheStub()
	svc := &GatewayService{sessionLimitCache: cache}
	account := newSessionLimitTestAccount()
	ctx := context.WithValue(context.Background(), smartRoutingSessionsContextKey{}, &smartRoutingSessions{
		gateway: svc, sessionHash: "failed-session", accounts: map[int64]*Account{account.ID: account},
	})
	FinishSmartRouting(ctx)
	FinishSmartRouting(ctx)
	require.Equal(t, []string{"failed-session"}, cache.unregistered[account.ID], "early return cleanup must release a failed session only once")

	servedCtx := context.WithValue(context.Background(), smartRoutingSessionsContextKey{}, &smartRoutingSessions{
		gateway: svc, sessionHash: "served-session", accounts: map[int64]*Account{account.ID: account}, served: true,
	})
	FinishSmartRouting(servedCtx)
	require.Equal(t, []string{"failed-session"}, cache.unregistered[account.ID], "a session that reached the handler must remain registered")
}

func TestPrepareGatewaySmartRoutingReusesSelectionAndCountTokensSkipsGenerationSlot(t *testing.T) {
	groupID := int64(93)
	group := &Group{ID: groupID, Status: StatusActive, Platform: PlatformAnthropic, Hydrated: true}
	account := Account{
		ID: 94, Status: StatusActive, Schedulable: true, Platform: PlatformAnthropic,
		Type: AccountTypeAPIKey, Concurrency: 1,
		Credentials:   map[string]any{"model_mapping": map[string]any{"claude-smart-test": "claude-sonnet-4-5"}},
		AccountGroups: []AccountGroup{{AccountID: 94, GroupID: groupID}},
	}
	repo := &smartRoutingGatewayAccountsRepo{accounts: []Account{account}}
	cache := &smartRoutingGatewayConcurrencyCache{}
	cfg := &config.Config{}
	cfg.Gateway.Scheduling.LoadBatchEnabled = false
	svc := &GatewayService{
		accountRepo: repo, groupRepo: &smartRoutingGatewayGroupRepo{group: group},
		concurrencyService: NewConcurrencyService(cache), cfg: cfg,
	}
	key := &APIKey{UserID: 7, GroupID: &groupID, Group: group}
	parsed := &ParsedRequest{Model: "claude-smart-test"}

	ctx, selected, err := svc.PrepareSmartRouting(context.Background(), key, parsed, false)
	require.NoError(t, err)
	require.True(t, selected)
	require.Equal(t, 1, cache.acquired, "generation requests must reserve their normal account slot")
	selectedAccount, err := svc.SelectAccountWithLoadAwareness(ctx, &groupID, "", parsed.Model, nil, "", key.UserID)
	require.NoError(t, err)
	require.Equal(t, account.ID, selectedAccount.Account.ID)
	require.Equal(t, 1, repo.listCalls, "the handler must consume the reservation instead of scheduling a second account")
	FinishSmartRouting(ctx)
	require.Equal(t, 1, cache.released, "the generation reservation must release exactly once")

	repo.listCalls = 0
	cache.acquired, cache.released = 0, 0
	ctx, selected, err = svc.PrepareSmartRouting(context.Background(), key, parsed, true)
	require.NoError(t, err)
	require.True(t, selected)
	require.Zero(t, cache.acquired, "count_tokens must select without reserving a generation slot")
	selectedAccount, err = svc.SelectAccountWithLoadAwareness(ctx, &groupID, "", parsed.Model, nil, "", key.UserID)
	require.NoError(t, err)
	require.Equal(t, account.ID, selectedAccount.Account.ID)
	require.Equal(t, 1, repo.listCalls, "count_tokens handler must reuse its selected account")
	FinishSmartRouting(ctx)
	require.Zero(t, cache.released)

	repo.listCalls = 0
	cache.acquired, cache.released, cache.busy = 0, 0, true
	ctx, selected, err = svc.PrepareSmartRouting(context.Background(), key, parsed, false)
	require.NoError(t, err)
	require.True(t, selected)
	reservation := ctx.Value(smartRoutingSelectionContextKey{}).(*smartRoutingSelection).selection
	require.NotNil(t, reservation.WaitPlan)
	require.Equal(t, account.ID, reservation.WaitPlan.AccountID)
	selectedAccount, err = svc.SelectAccountWithLoadAwareness(ctx, &groupID, "", parsed.Model, nil, "", key.UserID)
	require.NoError(t, err)
	require.Same(t, reservation.WaitPlan, selectedAccount.WaitPlan, "the handler must preserve the scheduler's wait plan")
	require.Equal(t, 1, repo.listCalls)
	require.Equal(t, 1, cache.acquired, "waiting must not schedule a second slot acquisition")
	FinishSmartRouting(ctx)
	require.Zero(t, cache.released, "a wait plan has no acquired slot to release")
}
