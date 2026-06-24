package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveUserBillingRateMultipliers_StacksAccountUserRate(t *testing.T) {
	accountUserRate := 1.5
	apiKey := &APIKey{Group: &Group{RateMultiplier: 0.1}}
	tokenMultiplier, imageMultiplier := resolveUserBillingRateMultipliers(apiKey, 0.1, &Account{UserRateMultiplier: &accountUserRate})

	require.InDelta(t, 0.15, tokenMultiplier, 1e-12)
	require.InDelta(t, 0.15, imageMultiplier, 1e-12)
}

func TestResolveUserBillingRateMultipliers_StacksAccountUserRateOnIndependentImageRate(t *testing.T) {
	accountUserRate := 1.5
	apiKey := &APIKey{Group: &Group{
		RateMultiplier:       0.1,
		ImageRateIndependent: true,
		ImageRateMultiplier:  1,
	}}
	tokenMultiplier, imageMultiplier := resolveUserBillingRateMultipliers(apiKey, 0.1, &Account{UserRateMultiplier: &accountUserRate})

	require.InDelta(t, 0.15, tokenMultiplier, 1e-12)
	require.InDelta(t, 1.5, imageMultiplier, 1e-12)
}
