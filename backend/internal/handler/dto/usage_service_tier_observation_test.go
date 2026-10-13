package dto

import (
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUsageServiceTierObservationAdminOnly(t *testing.T) {
	requested, upstream, billed := "ultrafast", "default", "ultrafast"
	for _, downgraded := range []bool{true, false} {
		log := &service.UsageLog{ServiceTier: &billed, RequestedServiceTier: &requested, UpstreamServiceTier: &upstream, ServiceTierDowngraded: &downgraded}
		userJSON, err := json.Marshal(UsageLogFromService(log))
		require.NoError(t, err)
		var user map[string]any
		require.NoError(t, json.Unmarshal(userJSON, &user))
		adminJSON, err := json.Marshal(UsageLogFromServiceAdmin(log))
		require.NoError(t, err)
		var admin map[string]any
		require.NoError(t, json.Unmarshal(adminJSON, &admin))
		require.Equal(t, billed, admin["service_tier"])
		require.Equal(t, requested, admin["requested_service_tier"])
		require.Equal(t, upstream, admin["upstream_service_tier"])
		require.Equal(t, downgraded, admin["service_tier_downgraded"])
		for _, field := range []string{"requested_service_tier", "upstream_service_tier", "service_tier_downgraded"} {
			require.NotContains(t, user, field)
		}
	}
	legacyJSON, err := json.Marshal(UsageLogFromServiceAdmin(&service.UsageLog{}))
	require.NoError(t, err)
	for _, field := range []string{"requested_service_tier", "upstream_service_tier", "service_tier_downgraded"} {
		require.NotContains(t, string(legacyJSON), field)
	}
}
