package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"strconv"
	"strings"
	"time"
)

const (
	// maxAPIKeyUsageRangeDays bounds the aggregate window so one request cannot
	// scan an unbounded slice of usage_logs.
	maxAPIKeyUsageRangeDays = 90
)

// parseAPIKeyUsageRange resolves the range for GET /user/api-keys/:id/usage.
//
// Precedence: explicit start_date/end_date > days (default 30, same as the /usage/daily endpoint) > last 30 days.
// Both dates are inclusive and interpreted in the caller's timezone; the
// returned end bound is exclusive (start of the following day) to match the
// [start, end) SQL convention. Day labels inside the response follow the
// caller timezone rather than the database session timezone.
func parseAPIKeyUsageRange(c *gin.Context, userTZ string) (start, end, todayStart time.Time, ok bool) {
	now := timezone.NowInUserLocation(userTZ)
	todayStart = timezone.StartOfDayInUserLocation(now, userTZ)

	startDateStr := strings.TrimSpace(c.Query("start_date"))
	endDateStr := strings.TrimSpace(c.Query("end_date"))
	daysStr := strings.TrimSpace(c.Query("days"))

	var startPtr, endPtr *time.Time
	if startDateStr != "" {
		t, err := timezone.ParseInUserLocation("2006-01-02", startDateStr, userTZ)
		if err != nil {
			response.BadRequest(c, "Invalid start_date format, use YYYY-MM-DD")
			return time.Time{}, time.Time{}, time.Time{}, false
		}
		startPtr = &t
	}
	if endDateStr != "" {
		t, err := timezone.ParseInUserLocation("2006-01-02", endDateStr, userTZ)
		if err != nil {
			response.BadRequest(c, "Invalid end_date format, use YYYY-MM-DD")
			return time.Time{}, time.Time{}, time.Time{}, false
		}
		inclusiveEnd := t.AddDate(0, 0, 1)
		endPtr = &inclusiveEnd
	}

	if startPtr != nil || endPtr != nil {
		if startPtr == nil {
			start := endPtr.AddDate(0, 0, -1)
			startPtr = &start
		}
		if endPtr == nil {
			// Explicit start without end means "start .. today" so the caller
			// always gets the current day included.
			end := todayStart.AddDate(0, 0, 1)
			if end.Before(*startPtr) {
				end = startPtr.AddDate(0, 0, 1)
			}
			endPtr = &end
		}
	} else {
		days, valid := parseAPIKeyDailyUsageDays(daysStr)
		if !valid {
			response.BadRequest(c, "Invalid days, allowed range is 1-90")
			return time.Time{}, time.Time{}, time.Time{}, false
		}
		startTime, endTime := apiKeyDailyUsageRange(days, userTZ)
		startPtr, endPtr = &startTime, &endTime
	}

	startTime, endTime := *startPtr, *endPtr
	if !endTime.After(startTime) {
		response.BadRequest(c, "end_date must be on or after start_date")
		return time.Time{}, time.Time{}, time.Time{}, false
	}
	if endTime.After(startTime.AddDate(0, 0, maxAPIKeyUsageRangeDays)) {
		response.BadRequest(c, "Date range exceeds the 90 day limit")
		return time.Time{}, time.Time{}, time.Time{}, false
	}
	return startTime, endTime, todayStart, true
}

// GetMyAPIKeyUsage returns a sanitized usage summary plus a per-day breakdown
// for one API key owned by the caller.
//
// GET /api/v1/user/api-keys/:id/usage?start_date=2026-10-04&end_date=2026-11-02&timezone=Asia/Shanghai
//
// Response data:
//
//	{
//	  "api_key_id": 12,
//	  "start_date": "2026-10-04",   // inclusive
//	  "end_date":   "2026-11-02",   // inclusive
//	  "timezone":   "Asia/Shanghai",
//	  "today": { ...APIKeyUsageSummary, date: "2026-11-02" },
//	  "range": { ...APIKeyUsageSummary },
//	  "items": [{ date, requests, input_tokens, output_tokens,
//	              cache_read_tokens, cache_write_tokens, total_tokens,
//	              cost, actual_cost }]
//	}
//
// Contract: only counters for this key are returned. Account cost, endpoint or
// upstream breakdowns, model names, request ids and IP addresses are never
// exposed; ownership is verified before any aggregate query runs.
func (h *UsageHandler) GetMyAPIKeyUsage(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	apiKeyID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || apiKeyID <= 0 {
		response.BadRequest(c, "Invalid API key ID")
		return
	}
	if h.apiKeyService == nil {
		response.InternalError(c, "API key service is not configured")
		return
	}
	apiKey, err := h.apiKeyService.GetByID(c.Request.Context(), apiKeyID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if apiKey.UserID != subject.UserID {
		response.Forbidden(c, "Not authorized to access this API key's usage")
		return
	}

	userTZ := strings.TrimSpace(c.Query("timezone"))
	if userTZ == "" {
		userTZ = timezone.Now().Location().String()
	}
	if _, err := time.LoadLocation(userTZ); err != nil {
		response.BadRequest(c, "Invalid timezone")
		return
	}
	startTime, endTime, todayStart, ok := parseAPIKeyUsageRange(c, userTZ)
	if !ok {
		return
	}

	items, err := h.usageService.GetAPIKeyDailyUsageInTimezone(c.Request.Context(), subject.UserID, apiKeyID, startTime, endTime, userTZ)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	rangeSummary := service.SumAPIKeyDailyUsage(items)
	todayItems := items
	if todayStart.Before(startTime) || !todayStart.Before(endTime) {
		todayItems, err = h.usageService.GetAPIKeyDailyUsageInTimezone(c.Request.Context(), subject.UserID, apiKeyID, todayStart, todayStart.AddDate(0, 0, 1), userTZ)
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
	}
	todaySummary := service.SumAPIKeyDailyUsage(nil)
	for _, item := range todayItems {
		if item.Date == todayStart.Format("2006-01-02") {
			todaySummary = item
			break
		}
	}
	todaySummary.Date = todayStart.Format("2006-01-02")

	response.Success(c, gin.H{
		"api_key_id": apiKeyID,
		"start_date": startTime.Format("2006-01-02"),
		"end_date":   endTime.AddDate(0, 0, -1).Format("2006-01-02"),
		"timezone":   userTZ,
		"today":      todaySummary,
		"range":      rangeSummary,
		"items":      items,
	})
}
