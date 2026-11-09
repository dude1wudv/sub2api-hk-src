package repository

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *channelMonitorRepository) ChannelMonitorUsageCache(ctx context.Context, start, end time.Time) ([]service.ChannelMonitorUsageCache, error) {
	// The group/time index bounds work to two groups over 90 minutes. Local
	// monitor probes have a dedicated user agent and never enter these counters.
	rows, err := r.db.QueryContext(ctx, `SELECT g.id,g.name,COUNT(ul.id),
		COALESCE(SUM(ul.input_tokens),0),COALESCE(SUM(ul.cache_creation_tokens),0),COALESCE(SUM(ul.cache_read_tokens),0)
		FROM groups g LEFT JOIN usage_logs ul ON ul.group_id=g.id AND ul.created_at >= $1 AND ul.created_at < $2
		AND ul.actual_cost > 0 AND COALESCE(ul.user_agent,'') NOT LIKE 'Sub2API-ChannelMonitor/%'
		WHERE g.id IN (72,237) AND g.deleted_at IS NULL GROUP BY g.id,g.name`, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	stats := make([]service.ChannelMonitorUsageCache, 0, 2)
	for rows.Next() {
		var stat service.ChannelMonitorUsageCache
		var input, creation, read int64
		if err := rows.Scan(&stat.GroupID, &stat.GroupName, &stat.RequestCount, &input, &creation, &read); err != nil {
			return nil, err
		}
		if total := input + creation + read; total > 0 {
			rate := float64(read) / float64(total)
			stat.CacheRate = &rate
		}
		stats = append(stats, stat)
	}
	return stats, rows.Err()
}
