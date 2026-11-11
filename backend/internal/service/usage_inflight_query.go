package service

import (
	"context"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
)

type UsageInflightPage struct {
	Items        []UsageInflightSnapshot `json:"items"`
	Total        int                     `json:"total"`
	Page         int                     `json:"page"`
	PageSize     int                     `json:"page_size"`
	RuntimeKnown bool                    `json:"runtime_known"`
}

func (s *UsageService) liveInflight(ctx context.Context, userID int64, filters usagestats.UsageLogFilters) ([]UsageInflightSnapshot, bool) {
	rows, known := QueryUsageInflight(ctx, userID)
	if !known {
		return nil, false
	}
	own := make([]UsageInflightSnapshot, 0, len(rows))
	for _, row := range rows {
		if userID > 0 && row.UserID != userID {
			continue
		}
		own = append(own, row)
	}
	var finished map[string]struct{}
	if s != nil && s.usageRepo != nil && len(own) > 0 {
		if repo, ok := s.usageRepo.(interface {
			FinishedInflightRequests(context.Context, []UsageInflightSnapshot) (map[string]struct{}, error)
		}); ok {
			var err error
			finished, err = repo.FinishedInflightRequests(ctx, own)
			if err != nil {
				return nil, false
			}
		}
	}
	return filterUsageInflight(own, filters, finished), true
}

func (s *UsageService) ListInflight(ctx context.Context, userID int64, filters usagestats.UsageLogFilters, page, size int, admin bool) UsageInflightPage {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 10
	}
	max := 10
	if admin {
		max = 50
	}
	if size > max {
		size = max
	}
	out := UsageInflightPage{Items: []UsageInflightSnapshot{}, Page: page, PageSize: size}
	rows, known := s.liveInflight(ctx, userID, filters)
	out.RuntimeKnown = known
	if !known {
		return out
	}
	out.Total = len(rows)
	if page > 1+out.Total/size {
		return out
	}
	start := (page - 1) * size
	end := start + size
	if end > out.Total {
		end = out.Total
	}
	if start > out.Total {
		return out
	}
	for _, row := range rows[start:end] {
		row.TokensEstimated = true
		if !admin {
			row = PublicUsageInflight(row)
		}
		out.Items = append(out.Items, row)
	}
	return out
}

func (s *UsageService) UserInflightGroupLoad(ctx context.Context, userID int64) ([]UserGroupInflightLoad, bool) {
	rows, known := s.liveInflight(ctx, userID, usagestats.UsageLogFilters{})
	if !known {
		return []UserGroupInflightLoad{}, false
	}
	return SummarizeUserGroupLoad(rows), true
}
