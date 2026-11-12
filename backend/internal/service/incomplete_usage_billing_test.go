//go:build unit

package service

import (
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestEstimatedCacheReadPricingMissingVersusFree(t *testing.T) {
	b := NewBillingService(&config.Config{}, nil)
	for _, tc := range []struct {
		name                string
		explicit, estimated bool
		price, want         float64
	}{
		{"missing estimated falls back", false, true, 0, 190},
		{"explicit free estimated stays free", true, true, 0, 0},
		{"configured discount", true, true, .2, 19},
		{"measured missing price unchanged", false, false, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := &ModelPricing{InputPricePerToken: 2, OutputPricePerToken: 3, CacheReadPricePerToken: tc.price, CacheReadPriceExplicit: tc.explicit}
			got := b.computeTokenBreakdown(p, UsageTokens{InputTokens: 5, CacheReadTokens: 95, CacheReadEstimated: tc.estimated, OutputTokens: 7}, .5, "", false)
			require.InDelta(t, tc.want, got.CacheReadCost, 1e-9)
			require.Equal(t, 10.0, got.InputCost)
			require.Equal(t, 21.0, got.OutputCost)
			require.InDelta(t, (31+tc.want)*.5, got.ActualCost, 1e-9)
		})
	}
}

func TestEstimatedCacheReadPreservesTierAndContextMultipliers(t *testing.T) {
	b := NewBillingService(&config.Config{}, nil)
	p := &ModelPricing{InputPricePerToken: 2, InputPricePerTokenPriority: 4, LongContextInputThreshold: 50, LongContextInputMultiplier: 2, LongContextOutputMultiplier: 1}
	tokens := UsageTokens{InputTokens: 5, CacheReadTokens: 95, CacheReadEstimated: true}
	got := b.computeTokenBreakdown(p, tokens, .5, "priority", true)
	require.Equal(t, 760.0, got.CacheReadCost)
	require.Equal(t, 400.0, got.ActualCost)
	p.CacheReadPricePriorityExplicit = true
	got = b.computeTokenBreakdown(p, tokens, .5, "priority", true)
	require.Zero(t, got.CacheReadCost, "explicit free priority cache rate must not fall back")
	p = &ModelPricing{InputPricePerToken: 2, CacheReadPricePerToken: .2, CacheReadPricePriorityExplicit: true}
	got = b.computeTokenBreakdown(p, tokens, 1, "priority", false)
	require.Zero(t, got.CacheReadCost, "a lone free priority cache rate survives generic tier fallback")
	require.Equal(t, 20.0, got.InputCost, "other fields retain the generic priority multiplier")
}

func TestEstimatedCacheReadPricePresenceFromCatalogAndOverrides(t *testing.T) {
	svc := NewPricingService(&config.Config{}, nil)
	data, err := svc.parsePricingData([]byte(`{"missing":{"input_cost_per_token":2,"output_cost_per_token":3},"free":{"input_cost_per_token":2,"output_cost_per_token":3,"cache_read_input_token_cost":0,"cache_read_input_token_cost_priority":0}}`))
	require.NoError(t, err)
	require.False(t, data["missing"].CacheReadInputTokenCostExplicit)
	require.True(t, data["free"].CacheReadInputTokenCostExplicit)
	require.True(t, data["free"].CacheReadInputTokenCostPriorityExplicit)
	svc.pricingData = data
	b := NewBillingService(&config.Config{}, svc)
	for _, model := range []string{"missing", "free"} {
		p, err := b.GetModelPricing(model)
		require.NoError(t, err)
		require.Equal(t, model == "free", p.CacheReadPriceExplicit)
	}
	zero := 0.0
	p := &ModelPricing{InputPricePerToken: 2, CacheReadPricePerToken: .2}
	applyChannelTokenPriceOverrides(p, &ChannelModelPricing{CacheReadPrice: &zero})
	require.True(t, p.CacheReadPriceExplicit)
	require.Zero(t, p.CacheReadPricePerToken)
	interval := intervalToModelPricing(&PricingInterval{CacheReadPrice: &zero}, &ModelPricing{InputPricePerToken: 2, CacheReadPricePerToken: .2}, nil)
	require.True(t, interval.CacheReadPriceExplicit)
	require.Zero(t, interval.CacheReadPricePerToken)
	// A standard-only override must not manufacture a free priority rate.
	positive := .4
	p = &ModelPricing{InputPricePerToken: 2, InputPricePerTokenPriority: 4}
	applyChannelTokenPriceOverrides(p, &ChannelModelPricing{CacheReadPrice: &positive})
	got := b.computeTokenBreakdown(p, UsageTokens{CacheReadTokens: 95, CacheReadEstimated: true}, 1, "priority", false)
	require.Equal(t, 38.0, got.CacheReadCost)
	// Leaving the channel value blank inherits an explicitly free catalog rate.
	p = &ModelPricing{InputPricePerToken: 2, CacheReadPriceExplicit: true}
	applyChannelTokenPriceOverrides(p, &ChannelModelPricing{})
	got = b.computeTokenBreakdown(p, UsageTokens{CacheReadTokens: 95, CacheReadEstimated: true}, 1, "", false)
	require.Zero(t, got.CacheReadCost)
	cost := calculateTokenStatsCost(&ChannelModelPricing{InputPrice: ptrIncompletePrice(2)}, UsageTokens{InputTokens: 5, CacheReadTokens: 95, CacheReadEstimated: true})
	require.NotNil(t, cost)
	require.Equal(t, 200.0, *cost)
	cost = calculateTokenStatsCost(&ChannelModelPricing{InputPrice: ptrIncompletePrice(2), CacheReadPrice: &zero}, UsageTokens{InputTokens: 5, CacheReadTokens: 95, CacheReadEstimated: true})
	require.Equal(t, 10.0, *cost)
}

func ptrIncompletePrice(value float64) *float64 { return &value }
