package service

import (
	"context"
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"time"
)

// Dedicated optional projection avoids exposing account costs or raw log fields.
func (s *UsageService) GetAPIKeyDailyUsageInTimezone(ctx context.Context, userID, keyID int64, start, end time.Time, zone string) ([]usagestats.APIKeyDailyUsagePoint, error) {
	repo, ok := s.usageRepo.(interface {
		GetAPIKeyDailyUsageInTimezone(context.Context, int64, int64, time.Time, time.Time, string) ([]usagestats.APIKeyDailyUsagePoint, error)
	})
	if !ok {
		return nil, fmt.Errorf("timezone-aware key usage unavailable")
	}
	return repo.GetAPIKeyDailyUsageInTimezone(ctx, userID, keyID, start, end, zone)
}
func SumAPIKeyDailyUsage(items []usagestats.APIKeyDailyUsagePoint) usagestats.APIKeyDailyUsagePoint {
	var sum usagestats.APIKeyDailyUsagePoint
	for _, r := range items {
		sum.Requests += r.Requests
		sum.InputTokens += r.InputTokens
		sum.OutputTokens += r.OutputTokens
		sum.CacheReadTokens += r.CacheReadTokens
		sum.CacheWriteTokens += r.CacheWriteTokens
		sum.TotalTokens += r.TotalTokens
		sum.Cost += r.Cost
		sum.ActualCost += r.ActualCost
	}
	return sum
}
