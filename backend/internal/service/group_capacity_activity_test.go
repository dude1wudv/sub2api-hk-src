package service

import (
	"context"
	"errors"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type capacityActivityProbe struct {
	calls int
	ids   []int64
	since time.Time
	err   error
}

func (p *capacityActivityProbe) GetGroupActiveUserCounts(_ context.Context, ids []int64, since time.Time) (map[int64]int, error) {
	p.calls++
	p.ids = append([]int64(nil), ids...)
	p.since = since
	return map[int64]int{10: 4}, p.err
}
func TestGroupCapacityActivityCacheAndUnknown(t *testing.T) {
	probe := &capacityActivityProbe{}
	s := &GroupCapacityService{activeUsers: probe}
	first := []GroupCapacitySummary{{GroupID: 10}, {GroupID: 20}}
	s.enrichActiveUsers(context.Background(), first)
	require.True(t, first[0].ActiveUsersKnown)
	require.Equal(t, 4, first[0].ActiveUsers)
	require.False(t, first[0].SessionsTracked)
	require.True(t, first[1].ActiveUsersKnown)
	require.Zero(t, first[1].ActiveUsers)
	require.WithinDuration(t, time.Now().Add(-5*time.Minute), probe.since, time.Second)
	second := []GroupCapacitySummary{{GroupID: 10}}
	s.enrichActiveUsers(context.Background(), second)
	require.Equal(t, 1, probe.calls)
	require.Equal(t, 4, second[0].ActiveUsers)
	s.activeCache[10] = groupActiveUsersEntry{expires: time.Now().Add(-time.Second)}
	probe.err = errors.New("DB unavailable")
	unknown := []GroupCapacitySummary{{GroupID: 10}}
	s.enrichActiveUsers(context.Background(), unknown)
	require.False(t, unknown[0].ActiveUsersKnown)
}
