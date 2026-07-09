package xai

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

const (
	BillingPeriodWeekly  = "weekly"
	BillingPeriodMonthly = "monthly"
	BillingPeriodUnknown = "unknown"

	BillingStateObserved = "observed"
	BillingStateNoData   = "no_data"
	BillingStateError    = "error"

	DefaultBillingURL = DefaultCLIBaseURL + "/billing?format=credits"
	TokenAuthCLI      = "xai-grok-cli"
	BillingUserAgent  = "Grok Build"
)

// BillingSnapshot is the normalized weekly/monthly subscription pool from
// GET cli-chat-proxy.grok.com/v1/billing?format=credits (openusage/tokscale).
type BillingSnapshot struct {
	State             string  `json:"state"`
	Period            string  `json:"period,omitempty"`
	Utilization       float64 `json:"utilization,omitempty"`
	Used              *float64 `json:"used,omitempty"`
	Limit             *float64 `json:"limit,omitempty"`
	Remaining         *float64 `json:"remaining,omitempty"`
	PeriodStart       string  `json:"period_start,omitempty"`
	PeriodEnd         string  `json:"period_end,omitempty"`
	ResetAt           string  `json:"reset_at,omitempty"`
	StatusCode        int     `json:"status_code,omitempty"`
	ObservationSource string  `json:"observation_source,omitempty"`
	LastProbeAt       string  `json:"last_probe_at,omitempty"`
	Error             string  `json:"error,omitempty"`
	UpdatedAt         string  `json:"updated_at"`
}

func BuildBillingURL() string {
	return DefaultBillingURL
}

func ParseBillingJSON(raw []byte, statusCode int, source string) *BillingSnapshot {
	now := time.Now().UTC().Format(time.RFC3339)
	snapshot := &BillingSnapshot{
		State:             BillingStateNoData,
		Period:            BillingPeriodUnknown,
		StatusCode:        statusCode,
		ObservationSource: strings.TrimSpace(source),
		UpdatedAt:         now,
	}
	if snapshot.ObservationSource == "active_probe" || snapshot.ObservationSource == "billing_probe" {
		snapshot.LastProbeAt = now
	}
	if len(raw) == 0 {
		return snapshot
	}

	var root any
	if err := json.Unmarshal(raw, &root); err != nil {
		snapshot.State = BillingStateError
		snapshot.Error = "invalid billing json"
		return snapshot
	}

	metric := findBillingMetric(root)
	if metric == nil {
		return snapshot
	}

	snapshot.State = BillingStateObserved
	snapshot.Period = metric.period
	snapshot.Utilization = metric.utilization
	snapshot.Used = metric.used
	snapshot.Limit = metric.limit
	snapshot.Remaining = metric.remaining
	snapshot.PeriodStart = metric.periodStart
	snapshot.PeriodEnd = metric.periodEnd
	snapshot.ResetAt = metric.periodEnd
	return snapshot
}

func NewBillingErrorSnapshot(statusCode int, source, errMsg string) *BillingSnapshot {
	now := time.Now().UTC().Format(time.RFC3339)
	return &BillingSnapshot{
		State:             BillingStateError,
		Period:            BillingPeriodUnknown,
		StatusCode:        statusCode,
		ObservationSource: strings.TrimSpace(source),
		LastProbeAt:       now,
		Error:             strings.TrimSpace(errMsg),
		UpdatedAt:         now,
	}
}

type billingMetric struct {
	utilization float64
	used        *float64
	limit       *float64
	remaining   *float64
	period      string
	periodStart string
	periodEnd   string
}

func findBillingMetric(value any) *billingMetric {
	if metric := parseBillingObject(value); metric != nil {
		return metric
	}
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			if metric := findBillingMetric(item); metric != nil {
				return metric
			}
		}
	case map[string]any:
		for _, item := range typed {
			if metric := findBillingMetric(item); metric != nil {
				return metric
			}
		}
	}
	return nil
}

func parseBillingObject(value any) *billingMetric {
	obj, ok := value.(map[string]any)
	if !ok {
		return nil
	}

	limit := numberAt(obj, "monthlyLimit")
	if limit == nil {
		if cfg, ok := obj["config"].(map[string]any); ok {
			limit = numberAt(cfg, "monthlyLimit")
		}
	}
	used := numberAtPath(obj, "usage", "totalUsed")
	if used == nil {
		used = numberAt(obj, "totalUsed")
	}
	if used == nil {
		if cfg, ok := obj["config"].(map[string]any); ok {
			used = numberAtPath(cfg, "usage", "totalUsed")
		}
	}

	var utilization *float64
	if limit != nil && used != nil && *limit > 0 {
		pct := clampPercent((*used / *limit) * 100)
		utilization = &pct
	} else {
		utilization = numberAt(obj, "usedPercent")
		if utilization == nil {
			utilization = numberAt(obj, "usagePercent")
		}
		if utilization == nil {
			utilization = numberAt(obj, "creditUsagePercent")
		}
		if utilization != nil {
			pct := clampPercent(*utilization)
			utilization = &pct
		}
	}
	if utilization == nil {
		return nil
	}

	periodStart := stringAtPath(obj, "billingCycle", "billingPeriodStart")
	if periodStart == "" {
		periodStart = stringAt(obj, "billingPeriodStart")
	}
	if periodStart == "" {
		periodStart = epochAt(obj, "billingPeriodStart")
	}
	periodEnd := stringAtPath(obj, "billingCycle", "billingPeriodEnd")
	if periodEnd == "" {
		periodEnd = stringAt(obj, "billingPeriodEnd")
	}
	if periodEnd == "" {
		periodEnd = epochAt(obj, "billingPeriodEnd")
	}

	metric := &billingMetric{
		utilization: *utilization,
		used:        used,
		limit:       limit,
		period:      cyclePeriod(periodStart, periodEnd),
		periodStart: periodStart,
		periodEnd:   periodEnd,
	}
	if limit != nil && used != nil {
		remaining := math.Max(0, *limit-*used)
		metric.remaining = &remaining
	}
	return metric
}

func cyclePeriod(start, end string) string {
	if start == "" || end == "" {
		return BillingPeriodUnknown
	}
	startAt, err1 := time.Parse(time.RFC3339, start)
	endAt, err2 := time.Parse(time.RFC3339, end)
	if err1 != nil || err2 != nil {
		return BillingPeriodUnknown
	}
	days := int(endAt.Sub(startAt).Hours() / 24)
	switch {
	case days >= 6 && days <= 8:
		return BillingPeriodWeekly
	case days >= 27 && days <= 33:
		return BillingPeriodMonthly
	default:
		return BillingPeriodUnknown
	}
}

func clampPercent(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}

func numberAt(obj map[string]any, key string) *float64 {
	if obj == nil {
		return nil
	}
	return numericValue(obj[key])
}

func numberAtPath(obj map[string]any, path ...string) *float64 {
	current := any(obj)
	for _, segment := range path {
		m, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = m[segment]
	}
	return numericValue(current)
}

func stringAt(obj map[string]any, key string) string {
	if obj == nil {
		return ""
	}
	if s, ok := obj[key].(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

func stringAtPath(obj map[string]any, path ...string) string {
	current := any(obj)
	for _, segment := range path {
		m, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = m[segment]
	}
	if s, ok := current.(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}

func epochAt(obj map[string]any, key string) string {
	n := numberAt(obj, key)
	if n == nil {
		return ""
	}
	ts := int64(*n)
	if ts > 1_000_000_000_000 {
		ts = ts / 1000
	}
	return time.Unix(ts, 0).UTC().Format(time.RFC3339)
}

func numericValue(value any) *float64 {
	switch typed := value.(type) {
	case float64:
		if math.IsNaN(typed) || math.IsInf(typed, 0) {
			return nil
		}
		return &typed
	case float32:
		v := float64(typed)
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil
		}
		return &v
	case int:
		v := float64(typed)
		return &v
	case int64:
		v := float64(typed)
		return &v
	case json.Number:
		v, err := typed.Float64()
		if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
			return nil
		}
		return &v
	case string:
		v, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
			return nil
		}
		return &v
	case map[string]any:
		if v := numericValue(typed["val"]); v != nil {
			return v
		}
		return numericValue(typed["value"])
	default:
		return nil
	}
}

func FormatBillingProbeError(statusCode int, body string) string {
	body = truncateRunes(strings.TrimSpace(body), 160)
	if body == "" {
		return fmt.Sprintf("billing probe returned %d", statusCode)
	}
	return fmt.Sprintf("billing probe returned %d: %s", statusCode, body)
}

func truncateRunes(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max]
}
