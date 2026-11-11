package handler

import "github.com/Wei-Shaw/sub2api/internal/service"

// Metadata has already passed APIKeyService's permission and personal-rate rules.
func applyUserGroupHints(groups []userAvailableGroup, visible map[int64]service.Group) {
	for i := range groups {
		if g, ok := visible[groups[i].ID]; ok {
			groups[i].LaunchedAt = g.LaunchedAt
			groups[i].LaunchRate = g.LaunchRate
			groups[i].IsNew = g.IsNew
			groups[i].RecentRateChange = g.RecentRateChange
		}
	}
}
