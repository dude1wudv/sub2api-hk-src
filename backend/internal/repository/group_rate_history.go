package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

func (r *groupRepository) GetRecentRateChanges(ctx context.Context, ids []int64, since time.Time) (map[int64]service.GroupRateChange, error) {
	out := make(map[int64]service.GroupRateChange)
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := sqlExecutorFromContext(ctx, r.sql).QueryContext(ctx, `SELECT DISTINCT ON (group_id)
        group_id, old_rate_multiplier, new_rate_multiplier, changed_at
        FROM group_rate_history WHERE group_id = ANY($1) AND changed_at >= $2
        ORDER BY group_id, changed_at DESC, id DESC`, pq.Array(ids), since)
	if err != nil {
		return nil, fmt.Errorf("query group rate history: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id int64
		var change service.GroupRateChange
		if err := rows.Scan(&id, &change.OldRate, &change.NewRate, &change.ChangedAt); err != nil {
			return nil, err
		}
		out[id] = change
	}
	return out, rows.Err()
}
