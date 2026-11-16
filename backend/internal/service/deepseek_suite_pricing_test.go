//go:build unit

package service

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDeepSeekSuiteIndependentModelPricing(t *testing.T) {
	group := &Group{ID: 179, Platform: PlatformDeepseek, RateMultiplier: .3,
		ModelPricing: []ChannelModelPricing{{Models: []string{"deepseek-flash"}, RateIndependent: true}}}
	channel := Channel{ID: 27, Status: StatusActive, RestrictModels: true, GroupIDs: []int64{179},
		ModelPricing: []ChannelModelPricing{{Platform: PlatformDeepseek, Models: []string{"deepseek-flash"},
			InputPrice: testPtrFloat64(2e-6), OutputPrice: testPtrFloat64(8e-6), CacheReadPrice: testPtrFloat64(4e-8),
			TimePricing: &ChannelTimePricing{Timezone: "UTC", Periods: []ChannelTimePricingPeriod{
				{StartTime: "01:00:00", EndTime: "04:00:00", Multiplier: 2},
			}}}}}
	cs := newTestChannelService(makeStandardRepo(channel, map[int64]string{179: PlatformDeepseek}))
	bs := newTestBillingService()
	resolver := NewModelPricingResolver(cs, bs)
	for _, rate := range []float64{0, .03, .3, 1, 9} {
		for _, hour := range []int{2, 12} {
			cost, err := bs.CalculateCostUnified(CostInput{
				Ctx: t.Context(), Model: "deepseek-flash", Group: group, GroupID: &group.ID,
				Tokens:         UsageTokens{InputTokens: 1000, OutputTokens: 200, CacheReadTokens: 100},
				RateMultiplier: rate, Resolver: resolver, PricingAt: time.Date(2026, 11, 16, hour, 0, 0, 0, time.UTC),
			})
			require.NoError(t, err)
			want := .003604
			if hour == 2 {
				want *= 2 // Preserve the model's own schedule, not the group's rate.
			}
			require.InDelta(t, want, cost.ActualCost, 1e-12)
			require.Equal(t, cost.TotalCost, cost.ActualCost)
		}
	}
	require.Nil(t, matchGroupModelPricing(group, "deepseek-flash"), "rate-only card inherits channel prices")
	require.Equal(t, .3, groupModelRateMultiplier(group, "unrelated-model", .3))
	require.False(t, cs.IsModelRestricted(t.Context(), group.ID, "deepseek-flash"))
	require.True(t, cs.IsModelRestricted(t.Context(), group.ID, "unknown-model"))
	raw, err := json.Marshal(group.ModelPricing)
	require.NoError(t, err)
	var restored []ChannelModelPricing
	require.NoError(t, json.Unmarshal(raw, &restored))
	require.True(t, restored[0].RateIndependent)
}

func TestDeepSeekSuiteIndependentReservationAndReceipt(t *testing.T) {
	group := &Group{ID: 179, Platform: PlatformDeepseek, RateMultiplier: .3,
		ModelPricing: []ChannelModelPricing{{Models: []string{"deepseek-flash"}, RateIndependent: true,
			InputPrice: testPtrFloat64(2e-6), OutputPrice: testPtrFloat64(8e-6), CacheReadPrice: testPtrFloat64(4e-8)}}}
	key := &APIKey{ID: 2, UserID: 7, User: &User{ID: 7}, GroupID: &group.ID, Group: group}
	estimate := newInflightEstimateGateway(t, nil)
	first, priced := estimate.EstimateInflightReservation(t.Context(), key, InflightEstimateRequest{
		Model: "deepseek-flash", BodyBytes: 4000, MaxTokens: 200,
	})
	require.True(t, priced)
	group.RateMultiplier = 9
	second, priced := estimate.EstimateInflightReservation(t.Context(), key, InflightEstimateRequest{
		Model: "deepseek-flash", BodyBytes: 4000, MaxTokens: 200,
	})
	require.True(t, priced)
	require.InDelta(t, first, second, 1e-12)
	group.RateMultiplier = 0
	freeGroupEstimate, priced := estimate.EstimateInflightReservation(t.Context(), key, InflightEstimateRequest{
		Model: "deepseek-flash", BodyBytes: 4000, MaxTokens: 200,
	})
	require.True(t, priced)
	require.InDelta(t, first, freeGroupEstimate, 1e-12)
	logs := &openAIRecordUsageLogRepoStub{inserted: true}
	bills := &openAIRecordUsageBillingRepoStub{result: &UsageBillingApplyResult{Applied: true}}
	svc := newOpenAIRecordUsageServiceWithBillingRepoForTest(logs, bills, &openAIRecordUsageUserRepoStub{}, &openAIRecordUsageSubRepoStub{}, nil)
	svc.resolver = NewModelPricingResolver(nil, svc.billingService)
	in := &OpenAIRecordUsageInput{
		Result: &OpenAIForwardResult{RequestID: "deepseek_suite:priced", Model: "deepseek-flash",
			Usage: OpenAIUsage{InputTokens: 1100, OutputTokens: 200, CacheReadInputTokens: 100}},
		APIKey: key, User: key.User, Account: &Account{ID: 1181, Platform: PlatformDeepseek},
	}
	require.NoError(t, svc.RecordUsage(t.Context(), in))
	require.InDelta(t, .003604, bills.lastCmd.BalanceCost, 1e-12)
	require.Equal(t, 1.0, logs.lastLog.RateMultiplier)
	require.Equal(t, logs.lastLog.TotalCost, logs.lastLog.ActualCost)
}

func TestDeepSeekSuiteRateOnlyCardAndNativeUnitNormalization(t *testing.T) {
	cards, err := normalizeGroupModelPricing(PlatformDeepseek, []ChannelModelPricing{
		{Models: []string{"deepseek-flash"}, RateIndependent: true},
		{Models: []string{"deepseek-asr"}, RateIndependent: true, MeteredUnit: "hour",
			BillingMode: BillingModePerRequest, PerRequestPrice: testPtrFloat64(.78)},
	})
	require.NoError(t, err)
	require.True(t, cards[0].RateIndependent)
	require.Equal(t, "hour", cards[1].MeteredUnit)
	_, err = normalizeGroupModelPricing(PlatformDeepseek, []ChannelModelPricing{
		{Models: []string{"deepseek-asr"}, MeteredUnit: "invalid"},
	})
	require.Error(t, err)
}

func TestDeepSeekSuitePlazaInheritsChannelPriceWithIndependentFlag(t *testing.T) {
	channel := plazaPricedChannel(27, "suite", []int64{179}, PlatformDeepseek, "deepseek-flash")
	group := Group{ID: 179, Platform: PlatformDeepseek, RateMultiplier: .3,
		ModelPricing: []ChannelModelPricing{
			{Models: []string{"deepseek-flash"}, RateIndependent: true},
			{Models: []string{"deepseek-asr"}, RateIndependent: true, MeteredUnit: "hour",
				BillingMode: BillingModePerRequest, PerRequestPrice: testPtrFloat64(.78)},
		}}
	svc := newPlazaService([]Channel{channel}, []Group{group}, nil)
	cs := newTestChannelService(makeStandardRepo(channel, map[int64]string{179: PlatformDeepseek}))
	svc.billingService = newTestBillingService()
	svc.resolver = NewModelPricingResolver(cs, svc.billingService)
	groups, err := svc.ListGroups(t.Context())
	require.NoError(t, err)
	require.Len(t, groups, 1)
	require.Len(t, groups[0].Models, 2)
	for _, model := range groups[0].Models {
		require.True(t, model.Pricing.RateIndependent)
		if model.Name == "deepseek-flash" {
			require.InDelta(t, 3e-6, *model.Pricing.InputPrice, 1e-12)
		} else {
			require.Equal(t, "hour", model.Pricing.MeteredUnit)
			require.Equal(t, .78, *model.Pricing.PerRequestPrice)
		}
	}
}

func TestDeepSeekSuiteEveryNativeMeterIgnoresGroupRate(t *testing.T) {
	for _, kind := range []string{"search", "fetch", "image_search", "ocr", "document_parse", "asr", "tts"} {
		for _, rate := range []float64{.03, .4, 9} {
			n := int64(7)
			usage, err := chatAuxiliaryUsage(kind, &n, "confirmed")
			require.NoError(t, err)
			want, err := usage.cost()
			require.NoError(t, err)
			logs := &openAIRecordUsageLogRepoStub{inserted: true}
			bills := &openAIRecordUsageBillingRepoStub{result: &UsageBillingApplyResult{Applied: true}}
			svc := newOpenAIRecordUsageServiceWithBillingRepoForTest(logs, bills, &openAIRecordUsageUserRepoStub{}, &openAIRecordUsageSubRepoStub{}, nil)
			key := suiteTestKey()
			key.Group.RateMultiplier = rate
			err = svc.RecordUsage(t.Context(), &OpenAIRecordUsageInput{
				Result: &OpenAIForwardResult{RequestID: "native:" + kind, Model: "deepseek-" + kind,
					chatAuxiliaryOnly: true, ChatAuxiliaryUsage: usage},
				APIKey: key, User: key.User, Account: suiteTestAccount(),
			})
			require.NoError(t, err)
			require.InDelta(t, want, bills.lastCmd.BalanceCost, 1e-10)
			require.Equal(t, 1.0, logs.lastLog.RateMultiplier)
		}
	}
}

func TestDeepSeekTextDiscountInheritsChannelAcrossAliases(t *testing.T) {
	aliases := []string{"deepseek-flash", "deepseek-pro", "deepseek-v4.1-flash-1017",
		"deepseek-v4-pro-0813", "deepseek/deepseek-v4.1-flash-fast", "deepseek-v9-pro-2099"}
	group := &Group{ID: 179, Platform: PlatformDeepseek, RateMultiplier: .3,
		ModelPricing: []ChannelModelPricing{{Models: []string{"deepseek-search"}, RateIndependent: true,
			BillingMode: BillingModePerRequest, PerRequestPrice: testPtrFloat64(.005)}}}
	channel := Channel{ID: 27, Status: StatusActive, RestrictModels: true, GroupIDs: []int64{179},
		ModelPricing: []ChannelModelPricing{{Platform: PlatformDeepseek, Models: aliases,
			InputPrice: testPtrFloat64(2e-6), OutputPrice: testPtrFloat64(8e-6), CacheReadPrice: testPtrFloat64(4e-8),
			TimePricing: &ChannelTimePricing{Timezone: "UTC", Periods: []ChannelTimePricingPeriod{
				{StartTime: "01:00:00", EndTime: "04:00:00", Multiplier: 2},
			}}}}}
	cs := newTestChannelService(makeStandardRepo(channel, map[int64]string{179: PlatformDeepseek}))
	bs := newTestBillingService()
	resolver := NewModelPricingResolver(cs, bs)
	for _, model := range aliases {
		for _, rate := range []float64{0, .03, .3, .4, 1} {
			for _, hour := range []int{2, 12} {
				cost, err := bs.CalculateCostUnified(CostInput{
					Ctx: t.Context(), Model: model, Group: group, GroupID: &group.ID,
					Tokens:         UsageTokens{InputTokens: 1000, OutputTokens: 200, CacheReadTokens: 100},
					RateMultiplier: rate, Resolver: resolver, PricingAt: time.Date(2026, 11, 16, hour, 0, 0, 0, time.UTC),
				})
				require.NoError(t, err, model)
				base := .003604
				if hour == 2 {
					base *= 2
				}
				require.InDelta(t, base, cost.TotalCost, 1e-12, model)
				require.InDelta(t, base*rate, cost.ActualCost, 1e-12, model)
				require.Equal(t, rate, groupModelRateMultiplier(group, model, rate))
			}
		}
	}
	require.Equal(t, 1.0, groupModelRateMultiplier(group, "deepseek-search", .3))
	estimate := newInflightEstimateGateway(t, cs)
	key := &APIKey{User: &User{ID: 7}, GroupID: &group.ID, Group: group}
	for _, model := range aliases {
		group.RateMultiplier = 1
		full, priced := estimate.EstimateInflightReservation(t.Context(), key, InflightEstimateRequest{
			Model: model, BodyBytes: 4000, MaxTokens: 200,
		})
		require.True(t, priced)
		group.RateMultiplier = .3
		discounted, priced := estimate.EstimateInflightReservation(t.Context(), key, InflightEstimateRequest{
			Model: model, BodyBytes: 4000, MaxTokens: 200,
		})
		require.True(t, priced)
		require.InDelta(t, full*.3, discounted, 1e-12, model)
	}
}
