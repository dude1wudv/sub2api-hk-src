//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/stretchr/testify/require"
)

func TestCalculateOpenAIRecordUsageCostDeepSeekTool(t *testing.T) {
	groupID := int64(223)
	price := 0.013
	apiKey := &APIKey{ID: 1, GroupID: &groupID, Group: &Group{ID: groupID, Platform: PlatformOpenAI, WebSearchPricePerCall: &price}}
	svc := &OpenAIGatewayService{billingService: &BillingService{}}

	for _, kind := range []DeepSeekToolKind{DeepSeekToolSearch, DeepSeekToolFetch} {
		for _, tt := range []struct {
			name  string
			calls int
			want  float64
		}{
			{name: "billable success", calls: 1, want: 0.013},
			{name: "successful empty result", calls: 0, want: 0},
		} {
			t.Run(string(kind)+"/"+tt.name, func(t *testing.T) {
				result := &OpenAIForwardResult{Model: kind.Model(), DeepSeekToolKind: kind, BillableToolCalls: tt.calls}
				cost, err := svc.calculateOpenAIRecordUsageCost(context.Background(), result, apiKey, nil,
					9, 9, 9, 0.4, UsageTokens{}, "", boolPtr(false), false, time.Time{})
				require.NoError(t, err)
				require.Equal(t, string(BillingModePerRequest), cost.BillingMode)
				require.InDelta(t, tt.want, cost.TotalCost, 1e-12)
				require.InDelta(t, tt.want*0.4, cost.ActualCost, 1e-12, "tool pricing uses base user/group multiplier, not peak-aware token rate")
			})
		}
	}
}

func TestCalculateOpenAIRecordUsageCostDeepSeekToolExplicitFreeAndInvalidCallCounts(t *testing.T) {
	groupID := int64(223)
	free := 0.0
	apiKey := &APIKey{ID: 1, GroupID: &groupID, Group: &Group{ID: groupID, Platform: PlatformOpenAI, WebSearchPricePerCall: &free}}
	svc := &OpenAIGatewayService{billingService: &BillingService{}}

	for _, calls := range []int{0, 1} {
		cost, err := svc.calculateOpenAIRecordUsageCost(context.Background(),
			&OpenAIForwardResult{Model: "deepseek-fetch", DeepSeekToolKind: DeepSeekToolFetch, BillableToolCalls: calls},
			apiKey, []string{"nonexistent-model"}, 1, 1, 1, 1, UsageTokens{InputTokens: 1000}, "", boolPtr(false), false, time.Time{})
		require.NoError(t, err)
		require.Equal(t, string(BillingModePerRequest), cost.BillingMode)
		require.Zero(t, cost.TotalCost)
		require.Zero(t, cost.ActualCost)
	}

	for _, calls := range []int{-1, 2} {
		_, err := svc.calculateOpenAIRecordUsageCost(context.Background(),
			&OpenAIForwardResult{DeepSeekToolKind: DeepSeekToolSearch, BillableToolCalls: calls},
			apiKey, nil, 1, 1, 1, 1, UsageTokens{}, "", boolPtr(false), false, time.Time{})
		require.Error(t, err)
	}
}

func TestEstimateInflightDeepSeekToolUsesOneCallAndBaseRate(t *testing.T) {
	groupID := int64(223)
	price := 0.013
	userRate := 0.25
	cfg := &config.Config{}
	cfg.Default.RateMultiplier = 1
	svc := NewOpenAIGatewayService(nil, nil, nil, nil, nil, nil, nil, cfg, nil, nil,
		NewBillingService(cfg, nil), nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	apiKey := &APIKey{
		ID: 9, GroupID: &groupID, User: &User{ID: 5},
		Group: &Group{ID: groupID, Platform: PlatformOpenAI, RateMultiplier: 0.4, WebSearchPricePerCall: &price},
	}
	svc.userGroupRateResolver = newUserGroupRateResolver(
		&deepSeekToolsBillingRateRepo{rate: &userRate}, nil, time.Minute, nil, "test",
	)

	got, priced := svc.EstimateInflightReservation(context.Background(), apiKey, InflightEstimateRequest{Kind: InflightEstimateDeepSeekTool})
	require.True(t, priced)
	require.InDelta(t, 0.013*0.25, got, 1e-12)

	apiKey.Group.WebSearchPricePerCall = &[]float64{0}[0]
	got, priced = svc.EstimateInflightReservation(context.Background(), apiKey, InflightEstimateRequest{Kind: InflightEstimateDeepSeekTool})
	require.True(t, priced)
	require.Zero(t, got)
}

type deepSeekToolsBillingRateRepo struct {
	UserGroupRateRepository
	rate *float64
}

func (r *deepSeekToolsBillingRateRepo) GetByUserAndGroup(context.Context, int64, int64) (*float64, error) {
	return r.rate, nil
}

func TestResolveUsageBillingRequestIDDeepSeekToolsOverridesClientRequestID(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxkey.ClientRequestID, "reused-client-request")
	first := "deepseek_tools:request-1"
	second := "deepseek_tools:request-2"
	require.Equal(t, first, resolveUsageBillingRequestID(ctx, first))
	require.Equal(t, second, resolveUsageBillingRequestID(ctx, second))
	require.NotEqual(t, resolveUsageBillingRequestID(ctx, first), resolveUsageBillingRequestID(ctx, second))
}

func TestRecordUsageDeepSeekToolSkipsChatHealthResetAndModelStatsPricing(t *testing.T) {
	groupID := int64(223)
	toolPrice := 0.013
	customStatsPrice := 9.0

	for _, tc := range []struct {
		name  string
		calls int
		want  float64
	}{
		{name: "billable success", calls: 1, want: toolPrice},
		{name: "successful empty result", calls: 0, want: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			usageRepo := &openAIRecordUsageLogRepoStub{inserted: true}
			billingRepo := &openAIRecordUsageBillingRepoStub{result: &UsageBillingApplyResult{Applied: true}}
			svc := newOpenAIRecordUsageServiceWithBillingRepoForTest(usageRepo, billingRepo,
				&openAIRecordUsageUserRepoStub{}, &openAIRecordUsageSubRepoStub{}, nil)

			// Install a chat health counter and a custom model rule that would alter
			// account statistics for ordinary model usage. Native tools must bypass
			// both chat health recovery and model based account statistics pricing.
			counter := &openAI403CounterResetStub{}
			rateLimitSvc := NewRateLimitService(nil, nil, nil, nil, nil)
			rateLimitSvc.SetOpenAI403CounterCache(counter)
			svc.rateLimitService = rateLimitSvc
			cache := newEmptyChannelCache()
			cache.channelByGroupID[groupID] = &Channel{ID: groupID, Status: StatusActive}
			cache.pricingByGroupModel[channelModelKey{groupID: groupID, model: "custom-model"}] = &ChannelModelPricing{
				BillingMode:     BillingModePerRequest,
				PerRequestPrice: &customStatsPrice,
			}
			channelService := &ChannelService{}
			channelService.cache.Store(cache)
			svc.channelService = channelService

			apiKey := &APIKey{ID: 101, UserID: 202, GroupID: &groupID,
				Group: &Group{ID: groupID, Platform: PlatformOpenAI, RateMultiplier: 1, WebSearchPricePerCall: &toolPrice}}
			err := svc.RecordUsage(context.Background(), &OpenAIRecordUsageInput{
				Result: &OpenAIForwardResult{
					RequestID:         "deepseek-tools-stats-" + tc.name,
					Model:             "deepseek-search",
					UpstreamModel:     "custom-model",
					DeepSeekToolKind:  DeepSeekToolSearch,
					BillableToolCalls: tc.calls,
					Duration:          time.Second,
				},
				APIKey:  apiKey,
				User:    &User{ID: 202},
				Account: &Account{ID: 303, Platform: PlatformOpenAI},
			})
			require.NoError(t, err)
			require.Empty(t, counter.resetCalls, "native tools must not reset the chat-specific OpenAI 403 counter")
			require.NotNil(t, usageRepo.lastLog)
			require.InDelta(t, tc.want, usageRepo.lastLog.TotalCost, 1e-12)
			require.NotNil(t, usageRepo.lastLog.AccountStatsCost)
			require.InDelta(t, tc.want, *usageRepo.lastLog.AccountStatsCost, 1e-12,
				"account stats should use the native tool total, not custom model pricing")
		})
	}
}
