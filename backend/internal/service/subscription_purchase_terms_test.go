package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestResolveSubscriptionExpiryPreservesLongerActiveTerm(t *testing.T) {
	now := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	fixed := now.AddDate(0, 0, 30)
	active := now.AddDate(0, 0, 60)

	require.Equal(t, fixed, ResolveSubscriptionExpiry(now, 7, &fixed, nil))
	require.Equal(t, active, ResolveSubscriptionExpiry(now, 7, &fixed, &active))
	require.Error(t, validateFixedSubscriptionExpiry(now, &now, nil))
}
