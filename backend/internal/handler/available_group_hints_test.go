//go:build unit

package handler

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUserAvailableGroupHintContract(t *testing.T) {
	now := time.Now().UTC()
	rate := 1.5
	change := &service.GroupRateChange{OldRate: 2, NewRate: rate, ChangedAt: now}
	g := service.Group{ID: 1, LaunchedAt: &now, LaunchRate: &rate, IsNew: true, RecentRateChange: change}
	groups := filterUserVisibleGroups([]service.AvailableGroupRef{{ID: 1}, {ID: 2}}, map[int64]struct{}{1: {}})
	applyUserGroupHints(groups, map[int64]service.Group{1: g})
	require.Len(t, groups, 1)
	for _, payload := range []any{groups[0], dto.GroupFromService(&g)} {
		data, err := json.Marshal(payload)
		require.NoError(t, err)
		var decoded map[string]any
		require.NoError(t, json.Unmarshal(data, &decoded))
		require.Equal(t, true, decoded["is_new"])
		require.Equal(t, rate, decoded["launch_rate"])
		require.NotNil(t, decoded["launched_at"])
		require.Equal(t, 2.0, decoded["recent_rate_change"].(map[string]any)["old_rate"])
	}
	g.RecentRateChange = nil
	applyUserGroupHints(groups, map[int64]service.Group{1: g})
	data, err := json.Marshal(groups[0])
	require.NoError(t, err)
	require.Contains(t, string(data), `"recent_rate_change":null`)
}
