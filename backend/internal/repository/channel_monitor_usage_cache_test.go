package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

func TestChannelMonitorUsageCacheUsesLocalUsageAndExcludesActiveProbes(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	now := time.Now().UTC()
	start := now.Add(-90 * time.Minute)
	mock.ExpectQuery(`(?s)SELECT g.id,g.name,COUNT\(ul.id\).*FROM groups g LEFT JOIN usage_logs ul.*ul.created_at >= \$1 AND ul.created_at < \$2.*ul.actual_cost > 0.*NOT LIKE 'Sub2API-ChannelMonitor/%'.*WHERE g.id IN \(72,237\).*GROUP BY g.id,g.name`).WithArgs(start, now).WillReturnRows(sqlmock.NewRows([]string{"id", "name", "requests", "input", "creation", "read"}).AddRow(72, "Pro stable", 4, 50, 25, 25).AddRow(237, "Plus", 0, 0, 0, 0))
	stats, err := (&channelMonitorRepository{db: db}).ChannelMonitorUsageCache(context.Background(), start, now)
	require.NoError(t, err)
	require.Len(t, stats, 2)
	require.Equal(t, int64(72), stats[0].GroupID)
	require.Equal(t, int64(4), stats[0].RequestCount)
	require.InDelta(t, 0.25, *stats[0].CacheRate, 0.000001)
	require.Nil(t, stats[1].CacheRate, "no local usage must remain unknown, never zero or synthetic cache rate")
	require.NoError(t, mock.ExpectationsWereMet())
}
