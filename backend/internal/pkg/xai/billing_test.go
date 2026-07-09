package xai

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseBillingJSONWeekly(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"billingCycle": {
			"billingPeriodStart": "2026-07-01T00:00:00Z",
			"billingPeriodEnd": "2026-07-08T00:00:00Z"
		},
		"monthlyLimit": {"val": 10000},
		"usage": {"totalUsed": {"val": 2500}}
	}`)
	snap := ParseBillingJSON(raw, 200, "billing_probe")
	require.NotNil(t, snap)
	require.Equal(t, BillingStateObserved, snap.State)
	require.Equal(t, BillingPeriodWeekly, snap.Period)
	require.InDelta(t, 25.0, snap.Utilization, 0.01)
	require.NotNil(t, snap.Used)
	require.EqualValues(t, 2500, *snap.Used)
	require.NotNil(t, snap.Limit)
	require.EqualValues(t, 10000, *snap.Limit)
	require.NotNil(t, snap.Remaining)
	require.EqualValues(t, 7500, *snap.Remaining)
	require.Equal(t, "2026-07-08T00:00:00Z", snap.ResetAt)
}

func TestParseBillingJSONMonthly(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"billingCycle": {
			"billingPeriodStart": "2026-06-01T00:00:00Z",
			"billingPeriodEnd": "2026-07-01T00:00:00Z"
		},
		"monthlyLimit": 10000,
		"usage": {"totalUsed": 1250}
	}`)
	snap := ParseBillingJSON(raw, 200, "billing_probe")
	require.Equal(t, BillingStateObserved, snap.State)
	require.Equal(t, BillingPeriodMonthly, snap.Period)
	require.InDelta(t, 12.5, snap.Utilization, 0.01)
}

func TestParseBillingJSONNoData(t *testing.T) {
	t.Parallel()
	snap := ParseBillingJSON([]byte(`{"ok":true}`), 200, "billing_probe")
	require.Equal(t, BillingStateNoData, snap.State)
	require.Equal(t, BillingPeriodUnknown, snap.Period)
}

func TestParseBillingJSONPercentOnly(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"usagePercent": 80,
		"billingPeriodStart": "2026-07-01T00:00:00Z",
		"billingPeriodEnd": "2026-07-08T00:00:00Z"
	}`)
	snap := ParseBillingJSON(raw, 200, "billing_probe")
	require.Equal(t, BillingStateObserved, snap.State)
	require.Equal(t, BillingPeriodWeekly, snap.Period)
	require.InDelta(t, 80.0, snap.Utilization, 0.01)
}
