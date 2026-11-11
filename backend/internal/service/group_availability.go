package service

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// GroupRateChange describes the last base-rate change, never a personal rate.
type GroupRateChange struct {
	OldRate   float64   `json:"old_rate"`
	NewRate   float64   `json:"new_rate"`
	ChangedAt time.Time `json:"changed_at"`
}

// Optional capability keeps existing GroupRepository implementations compatible.
type GroupRateHistoryReader interface {
	GetRecentRateChanges(context.Context, []int64, time.Time) (map[int64]GroupRateChange, error)
}

func (g *Group) IsNewAt(now time.Time) bool {
	return g.LaunchedAt != nil && !g.LaunchedAt.After(now) && g.LaunchedAt.After(now.Add(-24*time.Hour))
}

// setLaunch snapshots only explicit publication, not ordinary rate/name edits.
func (g *Group) setLaunch(at *time.Time) {
	if at == nil {
		g.LaunchedAt, g.LaunchRate = nil, nil
		return
	}
	if g.LaunchedAt == nil || !g.LaunchedAt.Equal(*at) {
		rate := g.RateMultiplier
		g.LaunchRate = &rate
	}
	g.LaunchedAt = at
}

func (s *APIKeyService) enrichAvailableGroups(ctx context.Context, userID int64, groups []Group, now time.Time) {
	ids := make([]int64, len(groups))
	for i := range groups {
		ids[i] = groups[i].ID
		groups[i].IsNew = groups[i].IsNewAt(now)
		groups[i].RecentRateChange = nil
		groups[i].RateHintsAuthorized = false
	}
	if len(ids) == 0 {
		return
	}
	rates, err := s.GetUserGroupRates(ctx, userID)
	for i := range groups {
		if _, fixed := rates[groups[i].ID]; fixed || err != nil || s.userGroupRateRepo == nil {
			groups[i].LaunchRate = nil
		} else {
			// A history outage may suppress the recent change while this known
			// base-rate launch snapshot remains valid for the checked user.
			groups[i].RateHintsAuthorized = true
		}
	}
	if s.userGroupRateRepo == nil {
		return
	}
	if err != nil {
		// Unknown personal overrides must not expose base-rate launch hints either.
		logger.LegacyPrintf("service.group", "read personal group rates failed; omitting rate hints: %v", err)
		return
	}
	reader, ok := s.groupRepo.(GroupRateHistoryReader)
	if !ok {
		return
	}
	changes, err := reader.GetRecentRateChanges(ctx, ids, now.Add(-48*time.Hour))
	if err != nil {
		logger.LegacyPrintf("service.group", "read group rate history failed; omitting rate hints: %v", err)
		return
	}
	for i := range groups {
		if !groups[i].RateHintsAuthorized {
			continue
		}
		if change, found := changes[groups[i].ID]; found {
			groups[i].RecentRateChange = &change
		}
	}
}
