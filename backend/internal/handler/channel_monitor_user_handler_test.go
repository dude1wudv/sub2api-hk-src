package handler

import (
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/stretchr/testify/require"
)

func TestUserMonitorDetailToResponse_IncludesModelPricing(t *testing.T) {
	inputPrice := 0.000003
	outputPrice := 0.000015
	otherPrice := 0.000099
	detail := &service.UserMonitorDetail{
		ID:        10,
		Name:      "Anthropic Monitor",
		Provider:  "anthropic",
		GroupName: "Claude",
		Models: []service.ModelDetail{
			{Model: "claude-sonnet-4.5", LatestStatus: "operational"},
			{Model: "claude-opus-4.7", LatestStatus: "failed"},
		},
	}
	channels := []service.AvailableChannel{
		{
			Status: service.StatusActive,
			Groups: []service.AvailableGroupRef{
				{Name: "Claude", Platform: "anthropic"},
			},
			SupportedModels: []service.SupportedModel{
				{
					Name:     "claude-sonnet-4.5",
					Platform: "anthropic",
					Pricing: &service.ChannelModelPricing{
						BillingMode: service.BillingModeToken,
						InputPrice:  &inputPrice,
						OutputPrice: &outputPrice,
					},
				},
				{
					Name:     "gpt-4o",
					Platform: "openai",
					Pricing: &service.ChannelModelPricing{
						BillingMode: service.BillingModeToken,
						InputPrice:  &otherPrice,
					},
				},
			},
		},
		{
			Status: service.StatusActive,
			Groups: []service.AvailableGroupRef{
				{Name: "Other", Platform: "anthropic"},
			},
			SupportedModels: []service.SupportedModel{
				{
					Name:     "claude-opus-4.7",
					Platform: "anthropic",
					Pricing: &service.ChannelModelPricing{
						BillingMode: service.BillingModeToken,
						InputPrice:  &otherPrice,
					},
				},
			},
		},
	}

	pricing := monitorPricingByModelFromAvailableChannels(detail, channels)
	resp := userMonitorDetailToResponse(detail, pricing)

	require.Len(t, resp.Models, 2)
	require.NotNil(t, resp.Models[0].Pricing)
	require.Equal(t, string(service.BillingModeToken), resp.Models[0].Pricing.BillingMode)
	require.Equal(t, inputPrice, *resp.Models[0].Pricing.InputPrice)
	require.Equal(t, outputPrice, *resp.Models[0].Pricing.OutputPrice)
	require.Nil(t, resp.Models[1].Pricing)

	raw, err := json.Marshal(resp.Models[0])
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	require.Contains(t, decoded, "pricing")
}

func TestMonitorPricingByModelFromAvailableChannels_FallsBackWhenMonitorGroupEmpty(t *testing.T) {
	inputPrice := 0.0000025
	fallbackPrice := 0.000001
	detail := &service.UserMonitorDetail{
		Provider:  "anthropic",
		GroupName: "",
		Models: []service.ModelDetail{
			{Model: "claude-haiku-4-5-20251001"},
		},
	}
	channels := []service.AvailableChannel{
		{
			Status: service.StatusActive,
			Groups: []service.AvailableGroupRef{
				{Name: "claude", Platform: "anthropic"},
			},
			SupportedModels: []service.SupportedModel{
				{
					Name:     "claude-haiku-4-5-20251001",
					Platform: "anthropic",
					Pricing: &service.ChannelModelPricing{
						BillingMode: service.BillingModeToken,
						InputPrice:  &inputPrice,
					},
				},
			},
		},
		{
			Status: service.StatusActive,
			Groups: []service.AvailableGroupRef{
				{Name: "cheap-claude", Platform: "anthropic"},
			},
			SupportedModels: []service.SupportedModel{
				{
					Name:     "claude-haiku-4-5-20251001",
					Platform: "anthropic",
					Pricing: &service.ChannelModelPricing{
						BillingMode: service.BillingModeToken,
						InputPrice:  &fallbackPrice,
					},
				},
			},
		},
	}

	pricing := monitorPricingByModelFromAvailableChannels(detail, channels)

	require.NotNil(t, pricing["claude-haiku-4-5-20251001"])
	require.Equal(t, inputPrice, *pricing["claude-haiku-4-5-20251001"].InputPrice)
}

func TestMonitorPricingByModelFromAvailableChannels_MatchesGeminiPricing(t *testing.T) {
	inputPrice := 0.000002
	detail := &service.UserMonitorDetail{
		Provider:  "gemini",
		GroupName: "",
		Models: []service.ModelDetail{
			{Model: "gemini-3.1-flash-lite-preview"},
		},
	}
	channels := []service.AvailableChannel{
		{
			Status: service.StatusActive,
			Groups: []service.AvailableGroupRef{
				{Name: "Gemini", Platform: "gemini"},
			},
			SupportedModels: []service.SupportedModel{
				{
					Name:     "gemini-3.1-flash-lite-preview",
					Platform: "gemini",
					Pricing: &service.ChannelModelPricing{
						BillingMode: service.BillingModeToken,
						InputPrice:  &inputPrice,
					},
				},
			},
		},
	}

	pricing := monitorPricingByModelFromAvailableChannels(detail, channels)

	require.NotNil(t, pricing["gemini-3.1-flash-lite-preview"])
	require.Equal(t, inputPrice, *pricing["gemini-3.1-flash-lite-preview"].InputPrice)
}
