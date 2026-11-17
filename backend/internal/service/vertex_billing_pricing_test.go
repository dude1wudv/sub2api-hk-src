//go:build unit

package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

// Use a complete recorded request so endpoint pricing cannot accidentally scale
// the independently billed search and storage charges along with token costs.
func TestVertexBillingPricingTokenAndSurchargeComposition(t *testing.T) {
	const model = "gemini-3.7-flash"
	for _, source := range []string{"catalog", "fallback", "group", "channel", "rate-only group"} {
		for _, endpointFactor := range []float64{0, 1, 1.1} {
			for _, rate := range []float64{0, .4, 2} {
				t.Run(fmt.Sprintf("%s/factor=%g/rate=%g", source, endpointFactor, rate), func(t *testing.T) {
					pricing := NewPricingService(&config.Config{}, nil)
					t.Cleanup(pricing.Stop)
					if source != "fallback" {
						var err error
						pricing.pricingData, err = pricing.parsePricingData([]byte(`{"gemini-3.7-flash":{
							"litellm_provider":"gemini","input_cost_per_token":0.00000075,
							"output_cost_per_token":0.00000375,"cache_read_input_token_cost":0.000000075,
							"cache_storage_cost_per_token_per_hour":0.000001}}`))
						require.NoError(t, err)
					}
					billing := NewBillingService(&config.Config{}, pricing)
					group := &Group{ID: 5, Platform: PlatformGemini, Hydrated: true}
					searchPrice := 7.0
					group.SearchPricePer1k = &searchPrice
					configuredInput, configuredOutput, configuredCache := 2e-6, 8e-6, .4e-6
					card := ChannelModelPricing{
						Models: []string{model}, Platform: PlatformGemini, BillingMode: BillingModeToken,
						InputPrice: &configuredInput, OutputPrice: &configuredOutput, CacheReadPrice: &configuredCache,
					}
					var channels *ChannelService
					switch source {
					case "group":
						group.ModelPricing = []ChannelModelPricing{card}
					case "channel":
						channels = newTestChannelService(makeStandardRepo(Channel{
							ID: 27, Status: StatusActive, GroupIDs: []int64{group.ID}, ModelPricing: []ChannelModelPricing{card},
						}, map[int64]string{group.ID: PlatformGemini}))
					case "rate-only group":
						// Rate independence without a price card retains the default
						// token source, including its endpoint factor.
						group.ModelPricing = []ChannelModelPricing{{
							Models: []string{model}, BillingMode: BillingModeToken, RateIndependent: true,
						}}
					}
					gateway := &GatewayService{
						billingService: billing, resolver: NewModelPricingResolver(channels, billing),
					}
					key := &APIKey{GroupID: &group.ID, Group: group}
					result := &ForwardResult{
						Model: model, UpstreamPriceMultiplier: endpointFactor,
						Usage:       ClaudeUsage{InputTokens: 10000, OutputTokens: 1000, CacheReadInputTokens: 5000},
						SearchCount: 3, CacheStorageTokenHours: 20000,
					}
					factor := endpointFactor
					if factor == 0 {
						factor = 1
					}
					inputPrice, outputPrice, cachePrice := .75e-6, 3.75e-6, .075e-6
					if source == "group" || source == "channel" {
						inputPrice, outputPrice, cachePrice = configuredInput, configuredOutput, configuredCache
						factor = 1
					}
					tokenRate := rate
					if source == "rate-only group" {
						tokenRate = 1
					}
					wantInput := 10000 * inputPrice * factor
					wantOutput := 1000 * outputPrice * factor
					wantCache := 5000 * cachePrice * factor
					wantTokens := wantInput + wantOutput + wantCache
					wantSearch := 3 * searchPrice / 1000
					storagePrice := 1e-6
					if source == "fallback" {
						storagePrice = builtinGeminiCacheStoragePrice(model)
					}
					require.Positive(t, storagePrice)
					wantStorage := 20000 * storagePrice
					cost := gateway.calculateRecordUsageCost(context.Background(), result, key, model, rate, rate, time.Now())
					require.NotNil(t, cost)
					require.InDelta(t, wantInput, cost.InputCost, 1e-12)
					require.InDelta(t, wantOutput, cost.OutputCost, 1e-12)
					require.InDelta(t, wantCache, cost.CacheReadCost, 1e-12)
					require.InDelta(t, wantTokens+wantSearch+wantStorage, cost.TotalCost, 1e-12)
					require.InDelta(t, wantTokens*tokenRate+(wantSearch+wantStorage)*rate, cost.ActualCost, 1e-12)
				})
			}
		}
	}
}

func TestVertexBillingPricingSurchargeOnlyRequests(t *testing.T) {
	const model = "gemini-3.7-flash"
	pricing := NewPricingService(&config.Config{}, nil)
	t.Cleanup(pricing.Stop)
	var err error
	pricing.pricingData, err = pricing.parsePricingData([]byte(`{"gemini-3.7-flash":{
		"litellm_provider":"gemini","input_cost_per_token":0.00000075,
		"output_cost_per_token":0.00000375,"cache_storage_cost_per_token_per_hour":0.000001}}`))
	require.NoError(t, err)
	billing := NewBillingService(&config.Config{}, pricing)
	gateway := &GatewayService{billingService: billing, resolver: NewModelPricingResolver(nil, billing)}
	for _, tc := range []struct {
		name         string
		search       int
		storageHours float64
		want         float64
	}{
		{"search only", 3, 0, .021},
		{"storage only", 0, 20000, .02},
		{"search and storage", 3, 20000, .041},
	} {
		t.Run(tc.name, func(t *testing.T) {
			searchPrice := 7.0
			group := &Group{ID: 5, Platform: PlatformGemini, Hydrated: true, SearchPricePer1k: &searchPrice}
			key := &APIKey{GroupID: &group.ID, Group: group}
			result := &ForwardResult{
				Model: model, UpstreamPriceMultiplier: 1.1, SearchCount: tc.search, CacheStorageTokenHours: tc.storageHours,
			}
			cost := gateway.calculateRecordUsageCost(context.Background(), result, key, model, .4, .4, time.Now())
			require.NotNil(t, cost)
			require.Zero(t, cost.InputCost+cost.OutputCost+cost.CacheReadCost+cost.CacheCreationCost)
			require.InDelta(t, tc.want, cost.TotalCost, 1e-12)
			require.InDelta(t, tc.want*.4, cost.ActualCost, 1e-12)
		})
	}
}
