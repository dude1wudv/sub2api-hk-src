//go:build integration

package repository

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type geminiChargeFaultBillingRepository struct {
	service.UsageBillingRepository
	applyErr error
	commands []service.UsageBillingCommand
}

func (r *geminiChargeFaultBillingRepository) Apply(ctx context.Context, command *service.UsageBillingCommand) (*service.UsageBillingApplyResult, error) {
	if command != nil {
		r.commands = append(r.commands, *command)
	}
	if r.applyErr != nil {
		return nil, r.applyErr
	}
	return r.UsageBillingRepository.Apply(ctx, command)
}

type geminiChargeFaultUsageLogRepository struct {
	service.UsageLogRepository
	createErr error
}

func (r *geminiChargeFaultUsageLogRepository) Create(ctx context.Context, usage *service.UsageLog) (bool, error) {
	if r.createErr != nil {
		err := r.createErr
		r.createErr = nil
		return false, err
	}
	return r.UsageLogRepository.Create(ctx, usage)
}

func newGeminiCachedChargeGateway(usageRepo service.UsageLogRepository, billingRepo service.UsageBillingRepository) *service.GatewayService {
	cfg := &config.Config{}
	return service.NewGatewayService(
		nil, nil, usageRepo, billingRepo,
		nil, nil, nil, nil, cfg,
		nil, nil, nil, nil, nil, nil, nil, nil, nil,
		nil, nil, nil, nil, nil, nil, nil, nil, nil, nil,
	)
}

func createGeminiCachedChargeFixture(t *testing.T) (context.Context, service.GeminiCachedContentRepository, *service.GeminiCachedContent, *service.GeminiCachedContentChargeIntent, *service.User, *service.APIKey, *service.Account) {
	t.Helper()
	ctx := context.Background()
	client := testEntClient(t)
	suffix := uuid.NewString()
	windowStart := time.Now().UTC().Add(-time.Hour).Truncate(time.Second)
	user := mustCreateUser(t, client, &service.User{
		Email:   "gemini-charge-" + suffix + "@example.com",
		Balance: 20,
	})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{
		UserID:        user.ID,
		Key:           "sk-gemini-charge-" + suffix,
		Quota:         50,
		QuotaUsed:     1,
		RateLimit5h:   10,
		RateLimit1d:   20,
		RateLimit7d:   30,
		Usage5h:       0.5,
		Usage1d:       1.0,
		Usage7d:       2.0,
		Window5hStart: &windowStart,
		Window1dStart: &windowStart,
		Window7dStart: &windowStart,
	})
	account := mustCreateAccount(t, client, &service.Account{
		Name: "gemini-charge-" + suffix,
		Type: service.AccountTypeAPIKey,
		Extra: map[string]any{
			"quota_limit":        100.0,
			"quota_used":         0.5,
			"quota_daily_limit":  20.0,
			"quota_daily_used":   1.25,
			"quota_daily_start":  windowStart.Format(time.RFC3339Nano),
			"quota_weekly_limit": 50.0,
			"quota_weekly_used":  2.0,
			"quota_weekly_start": windowStart.Format(time.RFC3339Nano),
		},
	})
	publicID := "charged" + strings.ReplaceAll(suffix, "-", "")
	requestID := service.GeminiCachedContentUsageRequestIDPrefix + "create:" + publicID
	intent := &service.GeminiCachedContentChargeIntent{
		Command: service.UsageBillingCommand{
			RequestID: requestID, UserID: user.ID, APIKeyID: apiKey.ID, AccountID: account.ID,
			AccountType: service.AccountTypeAPIKey, Model: "gemini-2.5-pro",
			InputTokens: 120, OutputTokens: 30, CacheCreationTokens: 120,
			BalanceCost: 1.25, APIKeyQuotaCost: 1.25, APIKeyRateLimitCost: 0.75, AccountQuotaCost: 0.25,
			AccountQuotaHasDailyLimit: true, AccountQuotaHasWeeklyLimit: true,
		},
		Usage: service.UsageLog{
			RequestID: requestID, UserID: user.ID, APIKeyID: apiKey.ID, AccountID: account.ID,
			Model: "gemini-2.5-pro", InputTokens: 120, OutputTokens: 30,
			CacheCreationTokens: 120, InputCost: 1.25, TotalCost: 1.25, ActualCost: 1.25,
			RateMultiplier: 1, BillingType: service.BillingTypeBalance,
		},
	}
	intent.Command.Normalize()
	target := time.Now().UTC().Add(12 * time.Hour).Truncate(time.Microsecond)
	record := &service.GeminiCachedContent{
		PublicID: publicID, UserID: user.ID, APIKeyID: apiKey.ID, GroupID: 42, AccountID: account.ID,
		UpstreamName: "cachedContents/synthetic-" + strings.ReplaceAll(suffix, "-", ""),
		Model:        "gemini-2.5-pro", UpstreamModel: "models/gemini-2.5-pro", TotalTokenCount: 120,
		ExpireTime: target,
	}

	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = integrationDB.ExecContext(cleanupCtx, "DELETE FROM gemini_cached_contents WHERE api_key_id = $1", apiKey.ID)
		_, _ = integrationDB.ExecContext(cleanupCtx, "DELETE FROM usage_logs WHERE user_id = $1 AND api_key_id = $2", user.ID, apiKey.ID)
		_, _ = integrationDB.ExecContext(cleanupCtx, "DELETE FROM usage_billing_dedup WHERE api_key_id = $1", apiKey.ID)
		_, _ = integrationDB.ExecContext(cleanupCtx, "DELETE FROM accounts WHERE id = $1", account.ID)
		_, _ = integrationDB.ExecContext(cleanupCtx, "DELETE FROM api_keys WHERE id = $1", apiKey.ID)
		_, _ = integrationDB.ExecContext(cleanupCtx, "DELETE FROM users WHERE id = $1", user.ID)
	})
	return ctx, NewGeminiCachedContentRepository(integrationDB), record, intent, user, apiKey, account
}

func geminiCachedContentBillingQuery(t *testing.T, user *service.User, apiKey *service.APIKey, account *service.Account, requestID string, wantUsageCount int) {
	t.Helper()
	ctx := context.Background()
	var balance, quotaUsed, usage5h, usage1d, usage7d float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT balance FROM users WHERE id = $1", user.ID).Scan(&balance))
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT quota_used, usage_5h, usage_1d, usage_7d FROM api_keys WHERE id = $1", apiKey.ID).Scan(&quotaUsed, &usage5h, &usage1d, &usage7d))
	require.InDelta(t, 18.75, balance, 0.000001)
	require.InDelta(t, 2.25, quotaUsed, 0.000001)
	require.InDelta(t, 1.25, usage5h, 0.000001)
	require.InDelta(t, 1.75, usage1d, 0.000001)
	require.InDelta(t, 2.75, usage7d, 0.000001)

	var totalUsed, dailyUsed, weeklyUsed float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT (extra->>'quota_used')::double precision,
 (extra->>'quota_daily_used')::double precision, (extra->>'quota_weekly_used')::double precision
 FROM accounts WHERE id = $1`, account.ID).Scan(&totalUsed, &dailyUsed, &weeklyUsed))
	require.InDelta(t, 0.75, totalUsed, 0.000001)
	require.InDelta(t, 1.5, dailyUsed, 0.000001)
	require.InDelta(t, 2.25, weeklyUsed, 0.000001)

	var dedupCount, usageCount int
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM usage_billing_dedup WHERE request_id = $1 AND api_key_id = $2", requestID, apiKey.ID).Scan(&dedupCount))
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM usage_logs WHERE request_id = $1 AND api_key_id = $2", requestID, apiKey.ID).Scan(&usageCount))
	require.Equal(t, 1, dedupCount)
	require.Equal(t, wantUsageCount, usageCount)
}

func attachGeminiCachedContentCharge(repo service.GeminiCachedContentRepository, gateway *service.GatewayService) *service.GeminiCachedContentService {
	cache := service.NewGeminiCachedContentService(repo, nil, &config.Config{})
	cache.SetBillingGateway(gateway)
	return cache
}

func TestGeminiCachedContentChargeRecoveryAfterLedgerFailureRestartsService(t *testing.T) {
	ctx, repo, record, intent, user, apiKey, account := createGeminiCachedChargeFixture(t)
	usageRepo := NewUsageLogRepository(testEntClient(t), integrationDB)
	ledger := NewUsageBillingRepository(testEntClient(t), integrationDB)
	failingLedger := &geminiChargeFaultBillingRepository{UsageBillingRepository: ledger, applyErr: errors.New("temporary ledger outage")}
	firstService := attachGeminiCachedContentCharge(repo, newGeminiCachedChargeGateway(usageRepo, failingLedger))
	require.NoError(t, firstService.SaveCharged(ctx, record, intent))
	require.ErrorContains(t, firstService.SettleCharge(ctx, record), "temporary ledger outage")

	groupID := record.GroupID
	_, err := firstService.Resolve(ctx, apiKey.ID, &groupID, record.PublicName())
	require.ErrorIs(t, err, service.ErrGeminiCachedContentNotFound, "unsettled content remains invisible")
	acquired, err := repo.BeginMutation(ctx, record.ID, record.ExpireTime, "premature-patch", time.Minute)
	require.NoError(t, err)
	require.False(t, acquired, "unsettled content cannot be patched")
	loaded, err := repo.GetForOwner(ctx, apiKey.ID, record.PublicID)
	require.NoError(t, err)
	require.Equal(t, intent, loaded.ChargeIntent, "the persisted intent freezes the original costs")

	restartedLedger := &geminiChargeFaultBillingRepository{UsageBillingRepository: ledger}
	restartedService := attachGeminiCachedContentCharge(repo, newGeminiCachedChargeGateway(usageRepo, restartedLedger))
	count, err := restartedService.RecoverPendingCharges(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)
	require.Len(t, restartedLedger.commands, 1)
	require.Equal(t, intent.Command, restartedLedger.commands[0], "recovery applies the persisted billing command without repricing")
	geminiCachedContentBillingQuery(t, user, apiKey, account, intent.Command.RequestID, 1)

	loaded, err = repo.GetForOwner(ctx, apiKey.ID, record.PublicID)
	require.NoError(t, err)
	require.False(t, loaded.BillingPending)
	require.Nil(t, loaded.ChargeIntent)
	require.True(t, loaded.ExpireTime.Equal(record.ExpireTime))
	resolved, err := restartedService.Resolve(ctx, apiKey.ID, &groupID, record.PublicName())
	require.NoError(t, err)
	require.Equal(t, record.PublicID, resolved.PublicID)
}

func TestGeminiCachedContentChargeRecoveryDoesNotRedeductAfterUsageLogFailure(t *testing.T) {
	ctx, repo, record, intent, user, apiKey, account := createGeminiCachedChargeFixture(t)
	usageRepo := NewUsageLogRepository(testEntClient(t), integrationDB)
	ledger := NewUsageBillingRepository(testEntClient(t), integrationDB)
	failingUsageRepo := &geminiChargeFaultUsageLogRepository{UsageLogRepository: usageRepo, createErr: errors.New("temporary usage log outage")}
	firstService := attachGeminiCachedContentCharge(repo, newGeminiCachedChargeGateway(failingUsageRepo, ledger))
	require.NoError(t, firstService.SaveCharged(ctx, record, intent))
	require.ErrorContains(t, firstService.SettleCharge(ctx, record), "temporary usage log outage")

	groupID := record.GroupID
	_, err := firstService.Resolve(ctx, apiKey.ID, &groupID, record.PublicName())
	require.ErrorIs(t, err, service.ErrGeminiCachedContentNotFound, "unsettled content remains invisible after ledger commit")
	loaded, err := repo.GetForOwner(ctx, apiKey.ID, record.PublicID)
	require.NoError(t, err)
	require.True(t, loaded.BillingPending)
	require.Equal(t, intent, loaded.ChargeIntent)
	geminiCachedContentBillingQuery(t, user, apiKey, account, intent.Command.RequestID, 0)

	restartedLedger := &geminiChargeFaultBillingRepository{UsageBillingRepository: ledger}
	restartedService := attachGeminiCachedContentCharge(repo, newGeminiCachedChargeGateway(usageRepo, restartedLedger))
	count, err := restartedService.RecoverPendingCharges(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)
	require.Len(t, restartedLedger.commands, 1)
	require.Equal(t, intent.Command, restartedLedger.commands[0], "recovery retries the frozen command")
	geminiCachedContentBillingQuery(t, user, apiKey, account, intent.Command.RequestID, 1)

	loaded, err = repo.GetForOwner(ctx, apiKey.ID, record.PublicID)
	require.NoError(t, err)
	require.False(t, loaded.BillingPending)
	require.Nil(t, loaded.ChargeIntent)
	resolved, err := restartedService.Resolve(ctx, apiKey.ID, &groupID, record.PublicName())
	require.NoError(t, err)
	require.Equal(t, record.PublicID, resolved.PublicID)
}

func TestGeminiCachedContentDurableChargeVisibilityAndMutation(t *testing.T) {
	ctx := context.Background()
	repo := &geminiCachedContentRepository{sql: testTx(t)}
	baseline := time.Now().UTC().Add(time.Hour).Truncate(time.Microsecond)
	intent := &service.GeminiCachedContentChargeIntent{Command: service.UsageBillingCommand{RequestID: "gcache:create:test", UserID: 1, APIKeyID: 910055, AccountID: 99}, Usage: service.UsageLog{RequestID: "gcache:create:test", UserID: 1, APIKeyID: 910055, AccountID: 99}}
	record := &service.GeminiCachedContent{PublicID: "charged" + time.Now().Format("150405000000"), UserID: 1, APIKeyID: 910055, GroupID: 42, AccountID: 99, UpstreamName: "cachedContents/synthetic", Model: "gemini-2.5-pro", UpstreamModel: "models/gemini-2.5-pro", ExpireTime: baseline, BillingPending: true, ChargeIntent: intent, ChargeTargetExpire: &baseline}
	require.NoError(t, repo.Create(ctx, record))
	loaded, err := repo.GetForOwner(ctx, record.APIKeyID, record.PublicID)
	require.NoError(t, err)
	require.True(t, loaded.BillingPending)
	require.Equal(t, intent.Command.RequestID, loaded.ChargeIntent.Command.RequestID)
	list, err := repo.ListForOwner(ctx, record.APIKeyID, record.GroupID, time.Now(), 0, 10)
	require.NoError(t, err)
	require.Empty(t, list, "unsettled creation is absent from public listing")
	acquired, err := repo.BeginMutation(ctx, record.ID, baseline, "cannot-change", time.Minute)
	require.NoError(t, err)
	require.False(t, acquired)
	require.ErrorIs(t, repo.CompleteCharge(ctx, record.ID, "foreign-charge"), service.ErrGeminiCachedContentMutationConflict)
	require.NoError(t, repo.CompleteCharge(ctx, record.ID, intent.Command.RequestID))
	require.NoError(t, repo.CompleteCharge(ctx, record.ID, intent.Command.RequestID), "concurrent recovery completion is idempotent")
	loaded, err = repo.GetForOwner(ctx, record.APIKeyID, record.PublicID)
	require.NoError(t, err)
	require.False(t, loaded.BillingPending)
	require.Nil(t, loaded.ChargeIntent)
	acquired, err = repo.BeginMutation(ctx, record.ID, baseline, "patch-token", time.Minute)
	require.NoError(t, err)
	require.True(t, acquired)
	target := baseline.Add(time.Hour)
	patchIntent := *intent
	patchIntent.Command.RequestID = "gcache:patch:test:unique"
	patchIntent.Usage.RequestID = patchIntent.Command.RequestID
	require.NoError(t, repo.CommitChargedMutation(ctx, record.ID, "patch-token", target, &patchIntent))
	loaded, err = repo.GetForOwner(ctx, record.APIKeyID, record.PublicID)
	require.NoError(t, err)
	require.True(t, loaded.ExpireTime.Equal(baseline), "target and intent are saved together while the old expiration remains committed")
	require.True(t, loaded.ChargeTargetExpire.Equal(target))
	require.True(t, loaded.BillingPending)
	require.Equal(t, patchIntent.Command.RequestID, loaded.ChargeIntent.Command.RequestID)
	require.ErrorIs(t, repo.CompleteCharge(ctx, record.ID, intent.Command.RequestID), service.ErrGeminiCachedContentMutationConflict, "old create completion cannot unlock a newer unpaid extension")
	require.NoError(t, repo.CompleteCharge(ctx, record.ID, patchIntent.Command.RequestID))
	loaded, err = repo.GetForOwner(ctx, record.APIKeyID, record.PublicID)
	require.NoError(t, err)
	require.True(t, loaded.ExpireTime.Equal(target))
	require.False(t, loaded.BillingPending)
	list, err = repo.ListForOwner(ctx, record.APIKeyID, record.GroupID, time.Now(), 0, 10)
	require.NoError(t, err)
	require.Len(t, list, 1)
}
