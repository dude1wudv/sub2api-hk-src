package service

import (
	"context"
	"log/slog"
	"time"
)

// ChannelMonitorUsageCache contains only aggregate counters from local usage;
// active probe measurements and externally imported quality data are excluded.
type ChannelMonitorUsageCache struct {
	GroupID      int64
	GroupName    string
	CacheRate    *float64
	RequestCount int64
}

type channelMonitorUsageCacheRepository interface {
	ChannelMonitorUsageCache(context.Context, time.Time, time.Time) ([]ChannelMonitorUsageCache, error)
}

// Cache at most two small aggregate rows for 60 seconds. Holding the mutex
// through the bounded query prevents concurrent page loads from fanning out.
func (s *ChannelMonitorService) userUsageCache(ctx context.Context) map[string]ChannelMonitorUsageCache {
	repo, ok := s.repo.(channelMonitorUsageCacheRepository)
	if !ok {
		return nil
	}
	s.usageCacheMu.Lock()
	defer s.usageCacheMu.Unlock()
	now := time.Now().UTC()
	if now.Before(s.usageCacheUntil) {
		return s.usageCache
	}
	queryCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	rows, err := repo.ChannelMonitorUsageCache(queryCtx, now.Add(-90*time.Minute), now)
	if err != nil {
		slog.Warn("channel_monitor: local usage cache aggregate unavailable", "error", err)
		s.usageCache = nil
		s.usageCacheUntil = now.Add(time.Minute)
		return nil // Do not substitute stale counters or active probe results.
	}
	s.usageCache = make(map[string]ChannelMonitorUsageCache, len(rows))
	for _, row := range rows {
		s.usageCache[row.GroupName] = row
	}
	s.usageCacheUntil = now.Add(time.Minute)
	return s.usageCache
}
