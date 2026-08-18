package handler

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUserMonitorViewToItemQuotaPrivacyGate(t *testing.T) {
	snapshot := &domain.MonitorQuotaSnapshot{
		Source:  "usage",
		Success: true,
		Tiers: []domain.MonitorQuotaTier{{
			Window:      "5h",
			UsedPercent: 25,
		}},
	}
	view := &service.UserMonitorView{
		ID:           1,
		Name:         "quota-monitor",
		Provider:     "openai",
		PrimaryModel: "quota",
		LatestQuota:  snapshot,
	}

	hidden := userMonitorViewToItem(view, false)
	require.Nil(t, hidden.LatestQuota, "quota must be removed from the user response while the public setting is off")

	shown := userMonitorViewToItem(view, true)
	require.Same(t, snapshot, shown.LatestQuota)
}
