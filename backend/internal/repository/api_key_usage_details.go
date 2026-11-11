package repository

import (
	"context"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"time"
)

func (r *usageLogRepository) GetAPIKeyDailyUsageInTimezone(ctx context.Context, userID, keyID int64, start, end time.Time, zone string) ([]usagestats.APIKeyDailyUsagePoint, error) {
	if _, err := time.LoadLocation(zone); err != nil {
		return nil, err
	}
	rows, err := r.sql.QueryContext(ctx, `SELECT TO_CHAR(created_at AT TIME ZONE $5,'YYYY-MM-DD'),COUNT(*),
 COALESCE(SUM(input_tokens),0),COALESCE(SUM(output_tokens),0),COALESCE(SUM(cache_read_tokens),0),COALESCE(SUM(cache_creation_tokens),0),
 COALESCE(SUM(input_tokens+output_tokens+cache_read_tokens+cache_creation_tokens),0),COALESCE(SUM(total_cost),0),COALESCE(SUM(actual_cost),0)
 FROM usage_logs WHERE user_id=$1 AND api_key_id=$2 AND created_at >= $3 AND created_at < $4 GROUP BY 1 ORDER BY 1`, userID, keyID, start, end, zone)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]usagestats.APIKeyDailyUsagePoint, 0)
	for rows.Next() {
		var p usagestats.APIKeyDailyUsagePoint
		if err := rows.Scan(&p.Date, &p.Requests, &p.InputTokens, &p.OutputTokens, &p.CacheReadTokens, &p.CacheWriteTokens, &p.TotalTokens, &p.Cost, &p.ActualCost); err != nil {
			return nil, err
		}
		items = append(items, p)
	}
	return items, rows.Err()
}
