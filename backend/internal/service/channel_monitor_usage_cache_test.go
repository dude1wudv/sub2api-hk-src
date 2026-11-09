package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type monitorUsageRepoStub struct {
	ChannelMonitorRepository
	calls  int
	err    error
	window time.Duration
}

func (r *monitorUsageRepoStub) ChannelMonitorUsageCache(_ context.Context, start, end time.Time) ([]ChannelMonitorUsageCache, error) {
	r.calls++
	r.window = end.Sub(start)
	if r.err != nil {
		return nil, r.err
	}
	rate := 0.5
	return []ChannelMonitorUsageCache{{GroupID: 72, GroupName: "Exact Pro Group Name", CacheRate: &rate, RequestCount: 3}}, nil
}

func TestChannelMonitorUsageCacheBoundedWindowAndRefresh(t *testing.T) {
	repo := &monitorUsageRepoStub{}
	svc := NewChannelMonitorService(repo, nil)
	for i := 0; i < 2; i++ {
		stats := svc.userUsageCache(context.Background())
		require.Len(t, stats, 1)
		require.Equal(t, int64(72), stats["Exact Pro Group Name"].GroupID)
	}
	require.Equal(t, 1, repo.calls, "repeated user reads should reuse only aggregate counters for 60s")
	require.Equal(t, 90*time.Minute, repo.window)
	svc.usageCacheUntil = time.Now().Add(-time.Second)
	repo.err = errors.New("database unavailable")
	require.Nil(t, svc.userUsageCache(context.Background()), "error must not expose stale cache data as current")
	require.Nil(t, svc.userUsageCache(context.Background()))
	require.Equal(t, 2, repo.calls, "database failures must also be throttled")
}
