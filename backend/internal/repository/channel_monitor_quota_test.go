package repository

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestChannelMonitorQuotaSnapshotDecode(t *testing.T) {
	balance := 12.5
	fetchedAt := time.Date(2026, time.August, 18, 12, 0, 0, 0, time.UTC)
	want := domain.MonitorQuotaSnapshot{
		Source:    "cn_balance",
		Success:   true,
		Balance:   &balance,
		Currency:  "CNY",
		PlanLevel: "pro",
		Tiers: []domain.MonitorQuotaTier{{
			Window:      "5h",
			UsedPercent: 25,
			Used:        5,
			Limit:       20,
			ResetAt:     fetchedAt.Add(time.Hour).Format(time.RFC3339),
		}},
		FetchedAt: fetchedAt,
	}
	raw, err := json.Marshal(want)
	require.NoError(t, err)

	got := scanMonitorQuota(raw)
	require.NotNil(t, got)
	require.Equal(t, want, *got)
	require.Nil(t, scanMonitorQuota(nil), "legacy probe history has NULL quota")
	require.Nil(t, scanMonitorQuota([]byte(`{`)), "invalid historical JSON must not break list rendering")
}
