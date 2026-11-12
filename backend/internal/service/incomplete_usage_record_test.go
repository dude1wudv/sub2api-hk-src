//go:build unit

package service

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestEstimatedUsageUsesExistingAtomicBillingAndProvenance(t *testing.T) {
	for _, protocol := range []string{"anthropic", "openai"} {
		for _, applied := range []bool{true, false} {
			t.Run(protocol+map[bool]string{true: "/first", false: "/duplicate"}[applied], func(t *testing.T) {
				logs := &openAIRecordUsageLogRepoStub{inserted: applied}
				bills := &openAIRecordUsageBillingRepoStub{result: &UsageBillingApplyResult{Applied: applied}}
				users := &openAIRecordUsageUserRepoStub{}
				subs := &openAIRecordUsageSubRepoStub{}
				keys := &openAIRecordUsageAPIKeyQuotaStub{}
				key := &APIKey{ID: 11, Quota: 100}
				ctx, cancel := context.WithCancel(context.Background())
				cancel() // Cancelled transport must still settle through detached billing context.
				if protocol == "anthropic" {
					svc := newGatewayRecordUsageServiceWithBillingRepoForTest(logs, bills, users, subs)
					svc.billingService.fallbackPrices["claude-sonnet-4"] = &ModelPricing{InputPricePerToken: .002, OutputPricePerToken: .003}
					err := svc.RecordUsage(ctx, &RecordUsageInput{Result: &ForwardResult{RequestID: "estimated-same-request", Model: "claude-sonnet-4", Usage: ClaudeUsage{InputTokens: 5, CacheReadInputTokens: 95, OutputTokens: 7, InputTokensEstimated: true, CacheReadTokensEstimated: true, OutputTokensEstimated: true, UsageEstimationReason: "client_cancelled"}}, APIKey: key, User: &User{ID: 22}, Account: &Account{ID: 33}, APIKeyService: keys})
					require.NoError(t, err)
				} else {
					svc := newOpenAIRecordUsageServiceWithBillingRepoForTest(logs, bills, users, subs, nil)
					svc.billingService.fallbackPrices["claude-sonnet-4"] = &ModelPricing{InputPricePerToken: .002, OutputPricePerToken: .003}
					err := svc.RecordUsage(ctx, &OpenAIRecordUsageInput{Result: &OpenAIForwardResult{RequestID: "estimated-same-request", Model: "claude-sonnet-4", Usage: OpenAIUsage{InputTokens: 100, CacheReadInputTokens: 95, OutputTokens: 7, InputTokensEstimated: true, CacheReadTokensEstimated: true, OutputTokensEstimated: true, UsageEstimationReason: "client_cancelled"}}, APIKey: key, User: &User{ID: 22}, Account: &Account{ID: 33}, APIKeyService: keys})
					require.NoError(t, err)
				}
				require.Equal(t, 1, bills.calls)
				require.NoError(t, bills.lastCtxErr)
				require.NotNil(t, bills.lastCmd)
				require.Equal(t, 5, bills.lastCmd.InputTokens)
				require.Equal(t, 95, bills.lastCmd.CacheReadTokens)
				require.InDelta(t, .221*1.1, bills.lastCmd.BalanceCost, 1e-9)
				require.Equal(t, bills.lastCmd.BalanceCost, bills.lastCmd.APIKeyQuotaCost)
				require.Zero(t, bills.lastCmd.SubscriptionCost)
				require.Zero(t, users.deductCalls, "atomic repo owns balance changes")
				require.Zero(t, subs.incrementCalls)
				require.Zero(t, keys.quotaCalls, "no second quota deduction outside atomic repo")
				require.NotNil(t, logs.lastLog)
				require.True(t, logs.lastLog.InputTokensEstimated)
				require.True(t, logs.lastLog.CacheReadTokensEstimated)
				require.True(t, logs.lastLog.OutputTokensEstimated)
				require.Equal(t, "client_cancelled", logs.lastLog.UsageEstimationReason)
				require.InDelta(t, .19, logs.lastLog.CacheReadCost, 1e-9)
			})
		}
	}
}

func TestEstimatedUsageSubscriptionSharesAtomicQuota(t *testing.T) {
	for _, protocol := range []string{"anthropic", "openai"} {
		t.Run(protocol, func(t *testing.T) {
			logs := &openAIRecordUsageLogRepoStub{inserted: true}
			bills := &openAIRecordUsageBillingRepoStub{result: &UsageBillingApplyResult{Applied: true}}
			users := &openAIRecordUsageUserRepoStub{}
			subs := &openAIRecordUsageSubRepoStub{}
			keys := &openAIRecordUsageAPIKeyQuotaStub{}
			group := &Group{ID: 44, SubscriptionType: SubscriptionTypeSubscription, RateMultiplier: .5}
			key := &APIKey{ID: 11, Quota: 100, GroupID: &group.ID, Group: group}
			sub := &UserSubscription{ID: 55}
			if protocol == "anthropic" {
				svc := newGatewayRecordUsageServiceWithBillingRepoForTest(logs, bills, users, subs)
				svc.billingService.fallbackPrices["claude-sonnet-4"] = &ModelPricing{InputPricePerToken: .002, CacheReadPricePerToken: .0002}
				err := svc.RecordUsage(context.Background(), &RecordUsageInput{Result: &ForwardResult{RequestID: "estimated-sub", Model: "claude-sonnet-4", Usage: ClaudeUsage{InputTokens: 5, CacheReadInputTokens: 95, InputTokensEstimated: true, CacheReadTokensEstimated: true}}, APIKey: key, User: &User{ID: 22}, Account: &Account{ID: 33}, Subscription: sub, APIKeyService: keys})
				require.NoError(t, err)
			} else {
				svc := newOpenAIRecordUsageServiceWithBillingRepoForTest(logs, bills, users, subs, nil)
				svc.billingService.fallbackPrices["claude-sonnet-4"] = &ModelPricing{InputPricePerToken: .002, CacheReadPricePerToken: .0002}
				err := svc.RecordUsage(context.Background(), &OpenAIRecordUsageInput{Result: &OpenAIForwardResult{RequestID: "estimated-sub", Model: "claude-sonnet-4", Usage: OpenAIUsage{InputTokens: 100, CacheReadInputTokens: 95, InputTokensEstimated: true, CacheReadTokensEstimated: true}}, APIKey: key, User: &User{ID: 22}, Account: &Account{ID: 33}, Subscription: sub, APIKeyService: keys})
				require.NoError(t, err)
			}
			require.Equal(t, 1, bills.calls)
			require.Zero(t, bills.lastCmd.BalanceCost)
			require.InDelta(t, .029*.5, bills.lastCmd.SubscriptionCost, 1e-9)
			require.Equal(t, bills.lastCmd.SubscriptionCost, bills.lastCmd.APIKeyQuotaCost)
			require.Equal(t, sub.ID, *bills.lastCmd.SubscriptionID)
			require.Zero(t, users.deductCalls)
			require.Zero(t, subs.incrementCalls)
			require.Zero(t, keys.quotaCalls)
		})
	}
}
