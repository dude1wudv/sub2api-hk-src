package service

import (
	"context"
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestOpenAIFastDefaultsForUnknownModels(t *testing.T) {
	service := NewBillingService(&config.Config{}, &PricingService{pricingData: map[string]*LiteLLMModelPricing{
		"gpt-future":          {InputCostPerToken: 1, OutputCostPerToken: 2},
		"custom-openai-alias": {InputCostPerToken: 1, OutputCostPerToken: 2, LiteLLMProvider: "openai"},
	}})
	for _, model := range []string{"gpt-future", "custom-openai-alias"} {
		for tier, multiplier := range map[string]float64{"priority": 2.5, "fast": 2.5, "ultrafast": 8} {
			cost, err := service.CalculateCostWithServiceTier(model, UsageTokens{InputTokens: 1, OutputTokens: 1}, 0.5, tier)
			require.NoError(t, err)
			require.InDelta(t, 3*multiplier, cost.TotalCost, 1e-12)
			require.InDelta(t, 1.5*multiplier, cost.ActualCost, 1e-12)
		}
	}
}

func TestGroupUltrafastPricingRoundTripAndCustomMultiplier(t *testing.T) {
	group := Group{Platform: PlatformOpenAI}
	require.NoError(t, json.Unmarshal([]byte(`[{"models":["custom-model"],"input_price":1,"output_price":2,"ultrafast_multiplier":6}]`), &group.ModelPricing))
	// Group uses the same pricing cards as channels; unknown aliases inherit its platform.
	service := NewBillingService(&config.Config{}, nil)
	resolver := NewModelPricingResolver(nil, service)
	for tier, multiplier := range map[string]float64{"priority": 2.5, "ultrafast": 6} {
		cost, err := service.CalculateCostUnified(CostInput{Ctx: context.Background(), Model: "custom-model", Group: &group, Tokens: UsageTokens{InputTokens: 1, OutputTokens: 1}, RateMultiplier: 0.5, ServiceTier: tier, Resolver: resolver})
		require.NoError(t, err)
		require.InDelta(t, 3*multiplier, cost.TotalCost, 1e-12)
		require.InDelta(t, 1.5*multiplier, cost.ActualCost, 1e-12)
	}
	cloned := group.ModelPricing[0].Clone()
	*cloned.UltrafastMultiplier = 4
	require.Equal(t, 6.0, *group.ModelPricing[0].UltrafastMultiplier)
}

func TestUltrafastCustomIntervalAndLongContextPricing(t *testing.T) {
	service := NewBillingService(&config.Config{}, nil)
	resolver := NewModelPricingResolver(nil, service)
	for _, multiplier := range []float64{8, 4} {
		pricing := &ChannelModelPricing{Platform: PlatformOpenAI, InputPrice: pricingMultiplier(1e-6), OutputPrice: pricingMultiplier(2e-6), CacheReadPrice: pricingMultiplier(0.1e-6), CacheWritePrice: pricingMultiplier(1.25e-6)}
		if multiplier != 8 {
			pricing.UltrafastMultiplier = pricingMultiplier(multiplier)
		}
		for _, interval := range []bool{false, true} {
			if interval {
				pricing.Intervals = []PricingInterval{{MinTokens: 272000, InputMultiplier: pricingMultiplier(3), OutputMultiplier: pricingMultiplier(2), CacheReadMultiplier: pricingMultiplier(3), CacheWriteMultiplier: pricingMultiplier(3)}}
			}
			resolved := resolver.resolveConfiguredPricing(pricing, "gpt-6-astra", PricingSourceChannel)
			resolved.longContextPricingEnabled = true
			input := CostInput{Ctx: context.Background(), Model: "gpt-6-astra", Tokens: UsageTokens{InputTokens: 300000, OutputTokens: 100, CacheReadTokens: 100, CacheCreationTokens: 100}, RateMultiplier: 0.5, Resolver: resolver, Resolved: resolved}
			standard, err := service.CalculateCostUnified(input)
			require.NoError(t, err)
			input.ServiceTier = "ultrafast"
			ultra, err := service.CalculateCostUnified(input)
			require.NoError(t, err)
			require.InDelta(t, standard.TotalCost*multiplier, ultra.TotalCost, 1e-12)
			require.InDelta(t, standard.ActualCost*multiplier, ultra.ActualCost, 1e-12)
		}
	}
}

func TestUltrafastMultiplierValidation(t *testing.T) {
	for _, value := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		require.Error(t, checkPricesNotNegative(ChannelModelPricing{UltrafastMultiplier: &value}))
	}
	require.NoError(t, checkPricesNotNegative(ChannelModelPricing{}))
	require.NoError(t, checkPricesNotNegative(ChannelModelPricing{UltrafastMultiplier: pricingMultiplier(8)}))
}

func TestFreeOpenAITiersAreIndependentAndPreserveUpstreamCost(t *testing.T) {
	for _, tier := range []string{"priority", "ultrafast"} {
		for _, freeFast := range []bool{false, true} {
			for _, freeUltra := range []bool{false, true} {
				usageRepo := &openAIRecordUsageLogRepoStub{inserted: true}
				svc := newOpenAIRecordUsageServiceForTest(usageRepo, &openAIRecordUsageUserRepoStub{}, &openAIRecordUsageSubRepoStub{}, nil)
				svc.resolver = NewModelPricingResolver(nil, svc.billingService)
				groupID := int64(77)
				group := &Group{ID: groupID, Platform: PlatformOpenAI, Hydrated: true, Status: StatusActive, RateMultiplier: 0.5, FreeOpenAIFast: freeFast, FreeOpenAIUltrafast: freeUltra}
				group.ModelPricing = []ChannelModelPricing{{Models: []string{"gpt-6-astra"}, InputPrice: pricingMultiplier(0.001), OutputPrice: pricingMultiplier(0.002)}}
				err := svc.RecordUsage(context.Background(), &OpenAIRecordUsageInput{Result: &OpenAIForwardResult{RequestID: "resp_free_ultra", Model: "gpt-6-astra", ServiceTier: &tier, Usage: OpenAIUsage{InputTokens: 100, OutputTokens: 50}, Duration: time.Second}, APIKey: &APIKey{ID: 101, GroupID: &groupID, Group: group}, User: &User{ID: 201}, Account: &Account{ID: 301, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}})
				require.NoError(t, err)
				require.NotNil(t, usageRepo.lastLog)
				multiplier, free := 2.5, freeFast
				if tier == "ultrafast" {
					multiplier, free = 8, freeUltra
				}
				require.Equal(t, tier, *usageRepo.lastLog.ServiceTier)
				require.InDelta(t, 0.2*multiplier, usageRepo.lastLog.TotalCost, 1e-12)
				actual := 0.1 * multiplier
				if free {
					actual = 0.1
				}
				require.InDelta(t, actual, usageRepo.lastLog.ActualCost, 1e-12)
			}
		}
	}
}

func TestCompositeAliasFastBillingUsesActualAccountPlatform(t *testing.T) {
	groupID := int64(77)
	group := &Group{
		ID: groupID, Platform: PlatformComposite, Hydrated: true, Status: StatusActive, RateMultiplier: 0.5,
		ForceOpenAIFast: true,
		ModelPricing:    []ChannelModelPricing{{Platform: PlatformComposite, Models: []string{"all"}, InputPrice: pricingMultiplier(0.001), OutputPrice: pricingMultiplier(0.002)}},
	}
	// Reuse the same card across requests: platform-specific defaults must not leak
	// into the shared Composite configuration used by a later Anthropic request.
	for _, tc := range []struct {
		name, platform string
		explicit       *float64
		want           float64
	}{
		{name: "OpenAI default", platform: PlatformOpenAI, want: 2.5},
		{name: "Anthropic default unchanged", platform: PlatformAnthropic, want: 2},
		{name: "OpenAI explicit override", platform: PlatformOpenAI, explicit: pricingMultiplier(4), want: 4},
		{name: "Anthropic explicit override", platform: PlatformAnthropic, explicit: pricingMultiplier(3), want: 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			group.ModelPricing[0].FastMultiplier = tc.explicit
			usageRepo := &openAIRecordUsageLogRepoStub{inserted: true}
			svc := newOpenAIRecordUsageServiceForTest(usageRepo, &openAIRecordUsageUserRepoStub{}, &openAIRecordUsageSubRepoStub{}, nil)
			svc.resolver = NewModelPricingResolver(nil, svc.billingService)
			tier := "priority"
			err := svc.RecordUsage(context.Background(), &OpenAIRecordUsageInput{
				Result: &OpenAIForwardResult{RequestID: "resp_composite_alias", Model: "all", ServiceTier: &tier, Usage: OpenAIUsage{InputTokens: 100, OutputTokens: 50}, Duration: time.Second},
				APIKey: &APIKey{ID: 101, GroupID: &groupID, Group: group}, User: &User{ID: 201},
				Account: &Account{ID: 301, Platform: tc.platform, Type: AccountTypeAPIKey},
			})
			require.NoError(t, err)
			require.NotNil(t, usageRepo.lastLog)
			require.InDelta(t, 0.2*tc.want, usageRepo.lastLog.TotalCost, 1e-12)
			require.InDelta(t, 0.1*tc.want, usageRepo.lastLog.ActualCost, 1e-12)
		})
	}
}

func TestUnknownAliasFastBillingUsesAccountPlatformWithoutResolver(t *testing.T) {
	for _, platform := range []string{PlatformOpenAI, PlatformAnthropic} {
		usageRepo := &openAIRecordUsageLogRepoStub{inserted: true}
		svc := newOpenAIRecordUsageServiceForTest(usageRepo, &openAIRecordUsageUserRepoStub{}, &openAIRecordUsageSubRepoStub{}, nil)
		svc.billingService = NewBillingService(&config.Config{}, &PricingService{pricingData: map[string]*LiteLLMModelPricing{
			"all": {InputCostPerToken: 0.001, OutputCostPerToken: 0.002},
		}})
		tier := "priority"
		err := svc.RecordUsage(context.Background(), &OpenAIRecordUsageInput{
			Result: &OpenAIForwardResult{RequestID: "resp_unknown_alias", Model: "all", ServiceTier: &tier, Usage: OpenAIUsage{InputTokens: 100, OutputTokens: 50}, Duration: time.Second},
			APIKey: &APIKey{ID: 101}, User: &User{ID: 201}, Account: &Account{ID: 301, Platform: platform, Type: AccountTypeAPIKey},
		})
		require.NoError(t, err)
		require.NotNil(t, usageRepo.lastLog)
		want := 0.4
		if platform == PlatformOpenAI {
			want = 0.5
		}
		require.InDelta(t, want, usageRepo.lastLog.TotalCost, 1e-12)
	}
}

func TestActualOpenAIPlatformPreservesExplicitAnthropicPricing(t *testing.T) {
	service := NewBillingService(&config.Config{}, &PricingService{pricingData: map[string]*LiteLLMModelPricing{
		"custom-anthropic": {InputCostPerToken: 1, OutputCostPerToken: 2, LiteLLMProvider: PlatformAnthropic},
	}})
	cost, err := service.CalculateCostUnified(CostInput{Model: "custom-anthropic", Platform: PlatformOpenAI, Tokens: UsageTokens{InputTokens: 1, OutputTokens: 1}, RateMultiplier: 1, ServiceTier: "priority"})
	require.NoError(t, err)
	require.InDelta(t, 6, cost.TotalCost, 1e-12)
}

func TestOpenAICompatibleCardsPreserveClaudeFastPricing(t *testing.T) {
	for _, catalog := range []*PricingService{nil, {pricingData: map[string]*LiteLLMModelPricing{
		"claude-opus-4-8": {LiteLLMProvider: PlatformAnthropic, InputCostPerToken: 5e-6, OutputCostPerToken: 25e-6, InputCostPerTokenPriority: 10e-6, OutputCostPerTokenPriority: 50e-6},
	}}} {
		service := NewBillingService(&config.Config{}, catalog)
		for _, withCard := range []bool{false, true} {
			for _, explicit := range []*float64{nil, pricingMultiplier(4)} {
				input := CostInput{Ctx: context.Background(), Model: "claude-opus-4-8", Platform: PlatformOpenAI, Tokens: UsageTokens{InputTokens: 1000, OutputTokens: 100}, RateMultiplier: 1}
				if withCard {
					input.Resolver = NewModelPricingResolver(nil, service)
					input.Group = &Group{Platform: PlatformOpenAI, ModelPricing: []ChannelModelPricing{{Platform: PlatformOpenAI, Models: []string{input.Model}, FastMultiplier: explicit}}}
				}
				standard, err := service.CalculateCostUnified(input)
				require.NoError(t, err)
				input.ServiceTier = "priority"
				fast, err := service.CalculateCostUnified(input)
				require.NoError(t, err)
				multiplier := 2.0
				if withCard && explicit != nil {
					multiplier = *explicit
				}
				require.InDelta(t, standard.TotalCost*multiplier, fast.TotalCost, 1e-12)
			}
		}
	}
}
