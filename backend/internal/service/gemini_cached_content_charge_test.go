//go:build unit

package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type cachedChargeCacheSpy struct {
	billingCacheWorkerStub
	balances      []int64
	subscriptions [][2]int64
	windows       []int64
	quotaReads    int
	quotaWrites   int
}

func (c *cachedChargeCacheSpy) InvalidateUserBalance(_ context.Context, userID int64) error {
	c.balances = append(c.balances, userID)
	return nil
}
func (c *cachedChargeCacheSpy) InvalidateSubscriptionCache(_ context.Context, userID, groupID int64) error {
	c.subscriptions = append(c.subscriptions, [2]int64{userID, groupID})
	return nil
}
func (c *cachedChargeCacheSpy) InvalidateAPIKeyRateLimit(_ context.Context, keyID int64) error {
	c.windows = append(c.windows, keyID)
	return nil
}
func (c *cachedChargeCacheSpy) GetUserPlatformQuotaCache(context.Context, int64, string) (*UserPlatformQuotaCacheEntry, bool, error) {
	c.quotaReads++
	return &UserPlatformQuotaCacheEntry{}, true, nil
}
func (c *cachedChargeCacheSpy) IncrUserPlatformQuotaUsageCache(context.Context, int64, string, float64, time.Duration, bool) error {
	c.quotaWrites++
	return nil
}

type cachedChargePlatformQuotaSpy struct {
	UserPlatformQuotaRepository
	writes int
}

func (r *cachedChargePlatformQuotaSpy) IncrementUsageWithReset(context.Context, int64, string, float64, time.Time) error {
	r.writes++
	return nil
}

func cachedChargeTestInput() *RecordUsageInput {
	groupID, accountRate := int64(33), 1.5
	user := &User{ID: 621, Balance: 100, PasswordHash: "synthetic-user-password-secret"}
	group := &Group{ID: groupID, Platform: PlatformGemini, RateMultiplier: 2, Hydrated: true}
	key := &APIKey{ID: 821, UserID: user.ID, Key: "synthetic-key-secret", User: user, GroupID: &groupID, Group: group, Quota: 100, RateLimit5h: 10}
	return &RecordUsageInput{
		Result: &ForwardResult{RequestID: GeminiCachedContentCreateUsageRequestID("charge-unit"),
			Model: "gemini-3.8-flash", UpstreamModel: "gemini-3.8-flash", Usage: ClaudeUsage{InputTokens: 1000},
			CacheStorageTokenHours: GeminiCachedContentStorageTokenHours(1000, 24*time.Hour)},
		APIKey: key, User: user,
		Account: &Account{ID: 713, Platform: PlatformGemini, Type: AccountTypeAPIKey, RateMultiplier: &accountRate,
			Credentials: map[string]any{"api_key": "synthetic-upstream-secret"}, Extra: map[string]any{"quota_limit": float64(100)}},
		APIKeyService: &apiKeyQuotaUpdaterStub{}, RequestPayloadHash: HashUsageRequestPayload([]byte("synthetic-request-payload-secret")),
		PricingAt: time.Date(2026, 11, 11, 12, 0, 0, 0, time.UTC),
	}
}

func TestCachedContentChargePrepareFreezesCostsWithoutSideEffects(t *testing.T) {
	svc, usage := newGeminiCacheBillingGatewayForTest(t)
	ledger := &openAIRecordUsageBillingRepoStub{}
	svc.usageBillingRepo = ledger
	input := cachedChargeTestInput()
	intent, err := svc.PrepareCachedContentCharge(context.Background(), input)
	require.NoError(t, err)
	require.Zero(t, ledger.calls, "preparation must not apply the ledger")
	require.Zero(t, usage.calls, "preparation must not persist a usage row")
	_, scheduled := svc.deferredService.lastUsedUpdates.Load(input.Account.ID)
	require.False(t, scheduled)
	require.Positive(t, intent.Command.BalanceCost)
	require.Positive(t, intent.Command.APIKeyQuotaCost)
	require.Positive(t, intent.Command.APIKeyRateLimitCost)
	require.Positive(t, intent.Command.AccountQuotaCost)
	require.NotEmpty(t, intent.Command.RequestFingerprint)
	require.Equal(t, input.RequestPayloadHash, intent.Command.RequestPayloadHash)
	require.Nil(t, intent.Usage.User)
	require.Nil(t, intent.Usage.APIKey)
	require.Nil(t, intent.Usage.Account)
	require.Nil(t, intent.Usage.Group)
	require.Nil(t, intent.Usage.Subscription)
	payload, err := json.Marshal(intent)
	require.NoError(t, err)
	for _, secret := range []string{"synthetic-user-password-secret", "synthetic-key-secret", "synthetic-upstream-secret", "synthetic-request-payload-secret"} {
		require.NotContains(t, string(payload), secret)
	}
	var recovered GeminiCachedContentChargeIntent
	require.NoError(t, json.Unmarshal(payload, &recovered))
	frozenCommand, frozenUsage := recovered.Command, recovered.Usage
	input.APIKey.Group.RateMultiplier = 100
	*input.Account.RateMultiplier = 100
	svc.cfg.Default.RateMultiplier = 100
	require.NoError(t, svc.ApplyCachedContentCharge(context.Background(), &recovered))
	require.Equal(t, frozenCommand, *ledger.lastCmd, "serialized recovery must not reprice frozen costs")
	require.Equal(t, frozenUsage.ActualCost, usage.lastLog.ActualCost)
	require.Equal(t, frozenUsage.TotalCost, usage.lastLog.TotalCost)
}

func TestCachedContentChargeModesEvictFinancialSnapshotsWithoutDeltas(t *testing.T) {
	for _, mode := range []string{"balance", "subscription", "simple", "simple_key_windows"} {
		t.Run(mode, func(t *testing.T) {
			svc, usage := newGeminiCacheBillingGatewayForTest(t)
			ledger := &openAIRecordUsageBillingRepoStub{}
			svc.usageBillingRepo = ledger
			cache := &cachedChargeCacheSpy{}
			writes := make(chan cacheWriteTask, 8)
			svc.billingCacheService = &BillingCacheService{cfg: svc.cfg, cache: cache, cacheWriteChan: writes}
			platformQuota := &cachedChargePlatformQuotaSpy{}
			svc.userPlatformQuotaRepo = platformQuota
			input := cachedChargeTestInput()
			if mode == "subscription" {
				input.APIKey.Group.SubscriptionType = SubscriptionTypeSubscription
				input.Subscription = &UserSubscription{ID: 901, UserID: input.User.ID, GroupID: *input.APIKey.GroupID, Notes: "synthetic-subscription-secret"}
			}
			if mode == "simple" || mode == "simple_key_windows" {
				svc.cfg.RunMode = config.RunModeSimple
				svc.cfg.SimpleModeKeyRateLimitEnabled = mode == "simple_key_windows"
			}
			intent, err := svc.PrepareCachedContentCharge(context.Background(), input)
			require.NoError(t, err)
			require.Positive(t, intent.Usage.ActualCost, "usage retains the true price even when billing is disabled")
			switch mode {
			case "balance":
				require.Positive(t, intent.Command.BalanceCost)
				require.Zero(t, intent.Command.SubscriptionCost)
			case "subscription":
				require.Zero(t, intent.Command.BalanceCost)
				require.Positive(t, intent.Command.SubscriptionCost)
				require.Equal(t, input.Subscription.ID, *intent.Command.SubscriptionID)
			case "simple":
				require.Zero(t, intent.Command.BalanceCost)
				require.Zero(t, intent.Command.SubscriptionCost)
				require.Zero(t, intent.Command.APIKeyQuotaCost)
				require.Zero(t, intent.Command.APIKeyRateLimitCost)
				require.Zero(t, intent.Command.AccountQuotaCost)
			case "simple_key_windows":
				require.True(t, intent.SimpleModeKeyRateLimitOnly)
				require.Positive(t, intent.Command.APIKeyRateLimitCost)
				require.Zero(t, intent.Command.BalanceCost)
				require.Zero(t, intent.Command.SubscriptionCost)
				require.Zero(t, intent.Command.APIKeyQuotaCost)
				require.Zero(t, intent.Command.AccountQuotaCost)
			}
			// A restart may also switch run mode. Recovery obeys the saved
			// command rather than interpreting the retained usage price again.
			svc.cfg.RunMode = config.RunModeStandard
			require.NoError(t, svc.ApplyCachedContentCharge(context.Background(), intent))
			require.Equal(t, 1, ledger.calls)
			require.Equal(t, 1, usage.calls)
			require.Equal(t, []int64{input.User.ID}, cache.balances)
			require.Equal(t, []int64{input.APIKey.ID}, cache.windows)
			if mode == "subscription" {
				require.Equal(t, [][2]int64{{input.User.ID, *input.APIKey.GroupID}}, cache.subscriptions)
			} else {
				require.Empty(t, cache.subscriptions)
			}
			require.Empty(t, writes, "never queue balance, subscription or key-window deltas after invalidation")
			require.Zero(t, cache.balanceUpdates)
			require.Zero(t, cache.subscriptionUpdates)
			require.Zero(t, cache.quotaWrites)
			require.Zero(t, platformQuota.writes)
			if mode != "balance" {
				require.Zero(t, cache.quotaReads, "unbilled and subscription requests must not consume balance platform quotas")
			}
			_, scheduled := svc.deferredService.lastUsedUpdates.Load(input.Account.ID)
			require.True(t, scheduled)
		})
	}
}

type cachedChargeCompletionSpy struct {
	GeminiCachedContentRepository
	GeminiCachedContentChargeRepository
	id      int64
	request string
}

func (r *cachedChargeCompletionSpy) CompleteCharge(_ context.Context, id int64, request string) error {
	r.id, r.request = id, request
	return nil
}

type cachedChargeAuthSpy struct {
	users  []int64
	keys   []string
	groups []int64
}

func (a *cachedChargeAuthSpy) InvalidateAuthCacheByUserID(_ context.Context, id int64) {
	a.users = append(a.users, id)
}
func (a *cachedChargeAuthSpy) InvalidateAuthCacheByKey(_ context.Context, key string) {
	a.keys = append(a.keys, key)
}
func (a *cachedChargeAuthSpy) InvalidateAuthCacheByGroupID(_ context.Context, id int64) {
	a.groups = append(a.groups, id)
}

func TestCachedContentChargeSettlementInvalidatesAuthenticationByUser(t *testing.T) {
	gateway, usage := newGeminiCacheBillingGatewayForTest(t)
	ledger := &openAIRecordUsageBillingRepoStub{result: &UsageBillingApplyResult{Applied: true, APIKeyQuotaExhausted: true}}
	gateway.usageBillingRepo = ledger
	input := cachedChargeTestInput()
	intent, err := gateway.PrepareCachedContentCharge(context.Background(), input)
	require.NoError(t, err)
	repo := &cachedChargeCompletionSpy{}
	auth := &cachedChargeAuthSpy{}
	svc := NewGeminiCachedContentService(repo, nil, nil)
	svc.SetBillingGateway(gateway)
	svc.SetAuthCacheInvalidator(auth)
	target := time.Now().Add(2 * time.Hour)
	record := &GeminiCachedContent{ID: 101, UserID: input.User.ID, APIKeyID: input.APIKey.ID, AccountID: input.Account.ID,
		ExpireTime: target.Add(-time.Hour), BillingPending: true, ChargeIntent: intent, ChargeTargetExpire: &target}
	require.NoError(t, svc.SettleCharge(context.Background(), record))
	require.Equal(t, []int64{input.User.ID}, auth.users)
	require.Empty(t, auth.keys, "settlement does not need the raw API key")
	require.Empty(t, auth.groups)
	require.Equal(t, int64(101), repo.id)
	require.Equal(t, intent.Command.RequestID, repo.request)
	require.Equal(t, 1, usage.calls)
	require.False(t, record.BillingPending)
	require.Nil(t, record.ChargeIntent)
	require.Equal(t, target, record.ExpireTime)
}

func TestCachedContentChargeUsageFailureRetriesFrozenCommandWithoutFinancialDeltas(t *testing.T) {
	svc, usage := newGeminiCacheBillingGatewayForTest(t)
	ledger := &simpleModeUsageBillingRepoStub{}
	svc.usageBillingRepo = ledger
	cache := &cachedChargeCacheSpy{}
	writes := make(chan cacheWriteTask, 8)
	svc.billingCacheService = &BillingCacheService{cfg: svc.cfg, cache: cache, cacheWriteChan: writes}
	intent, err := svc.PrepareCachedContentCharge(context.Background(), cachedChargeTestInput())
	require.NoError(t, err)
	usage.err = errors.New("usage temporarily unavailable")
	require.Error(t, svc.ApplyCachedContentCharge(context.Background(), intent))
	usage.err = nil
	require.NoError(t, svc.ApplyCachedContentCharge(context.Background(), intent))
	require.Len(t, ledger.seen, 1)
	require.Len(t, ledger.cmds, 2)
	require.Equal(t, ledger.cmds[0].RequestFingerprint, ledger.cmds[1].RequestFingerprint)
	require.Equal(t, 2, usage.calls)
	require.Len(t, cache.balances, 2, "even deduplicated recovery invalidates potentially stale snapshots")
	require.Len(t, cache.windows, 2)
	require.Empty(t, writes)
}
