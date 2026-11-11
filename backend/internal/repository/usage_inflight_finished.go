package repository

import (
	"context"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
	"time"
)

func (r *usageLogRepository) FinishedInflightRequests(ctx context.Context, snapshots []service.UsageInflightSnapshot) (map[string]struct{}, error) {
	done := make(map[string]struct{})
	if len(snapshots) == 0 {
		return done, nil
	}
	ids := make([]string, 0, len(snapshots))
	users := make([]int64, 0, len(snapshots))
	start := time.Now()
	for _, s := range snapshots {
		ids = append(ids, service.NormalizeUsageRequestIDKey(s.RequestID))
		users = append(users, s.UserID)
		if s.StartedAt.Before(start) {
			start = s.StartedAt
		}
	}
	rows, err := r.sql.QueryContext(ctx, `SELECT user_id,request_id FROM usage_logs WHERE user_id=ANY($1) AND created_at >= $2 AND (request_id=ANY($3) OR request_id=ANY($4) OR request_id=ANY($5))`, pq.Array(users), start.Add(-time.Minute), pq.Array(ids), pq.Array(prefixInflightIDs(ids, "client:")), pq.Array(prefixInflightIDs(ids, "local:")))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var user int64
		var id string
		if err := rows.Scan(&user, &id); err != nil {
			return nil, err
		}
		for _, s := range snapshots {
			if s.UserID == user && service.NormalizeUsageRequestIDKey(s.RequestID) == service.NormalizeUsageRequestIDKey(id) {
				done[service.InflightFinishedKey(user, id)] = struct{}{}
			}
		}
	}
	return done, rows.Err()
}
func prefixInflightIDs(ids []string, prefix string) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = prefix + id
	}
	return out
}
