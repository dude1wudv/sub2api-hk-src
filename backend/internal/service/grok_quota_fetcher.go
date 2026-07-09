package service

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/xai"
)

const grokQuotaSnapshotExtraKey = "grok_usage_snapshot"

type GrokQuotaFetcher struct{}

func NewGrokQuotaFetcher() *GrokQuotaFetcher {
	return &GrokQuotaFetcher{}
}

func (f *GrokQuotaFetcher) BuildUsageInfo(account *Account) *UsageInfo {
	now := time.Now()
	usage := &UsageInfo{
		Source:    "passive",
		UpdatedAt: &now,
	}
	if account == nil {
		usage.ErrorCode = "quota_unknown"
		usage.Error = "Grok quota is unknown until the first upstream response includes xAI rate-limit headers"
		return usage
	}

	snapshot, err := grokQuotaSnapshotFromExtra(account.Extra)
	if err != nil || snapshot == nil {
		usage.ErrorCode = "quota_unknown"
		usage.Error = "Grok quota is unknown until the first upstream response includes xAI rate-limit headers"
	} else {
		if parsedAt, err := time.Parse(time.RFC3339, snapshot.UpdatedAt); err == nil {
			usage.UpdatedAt = &parsedAt
		}
		usage.GrokRequestQuota = snapshot.Requests
		usage.GrokTokenQuota = snapshot.Tokens
		usage.GrokRetryAfterSeconds = snapshot.RetryAfterSeconds
		usage.SubscriptionTier = snapshot.SubscriptionTier
		usage.SubscriptionTierRaw = snapshot.SubscriptionTier
		usage.GrokEntitlementStatus = snapshot.EntitlementStatus
		usage.GrokLastQuotaProbeAt = snapshot.LastProbeAt
		usage.GrokLastHeadersSeenAt = snapshot.LastHeadersSeenAt
		usage.GrokLastStatusCode = snapshot.StatusCode
		if snapshot.HasObservedHeaders() {
			usage.GrokQuotaSnapshotState = "observed"
		} else {
			usage.GrokQuotaSnapshotState = "no_headers"
			usage.ErrorCode = "quota_unknown"
			usage.Error = "No xAI quota headers observed on the latest Grok probe"
		}

		switch snapshot.StatusCode {
		case 401:
			usage.NeedsReauth = true
			usage.ErrorCode = "unauthenticated"
		case 403:
			usage.IsForbidden = true
			usage.ForbiddenType = "forbidden"
			usage.ErrorCode = "forbidden"
			if usage.GrokEntitlementStatus == "" {
				usage.GrokEntitlementStatus = "forbidden"
			}
		case 429:
			usage.ErrorCode = "rate_limited"
		}
	}

	if billing, _ := grokBillingSnapshotFromExtra(account.Extra); billing != nil {
		usage.GrokWeeklyQuota = &GrokWeeklyQuotaInfo{
			Utilization: billing.Utilization,
			Used:        billing.Used,
			Limit:       billing.Limit,
			Remaining:   billing.Remaining,
			ResetAt:     billing.ResetAt,
			Period:      billing.Period,
			State:       billing.State,
			Error:       billing.Error,
		}
		if billing.LastProbeAt != "" && usage.GrokLastQuotaProbeAt == "" {
			usage.GrokLastQuotaProbeAt = billing.LastProbeAt
		}
		if billing.StatusCode == 401 {
			usage.NeedsReauth = true
			if usage.ErrorCode == "" {
				usage.ErrorCode = "unauthenticated"
			}
		}
		if billing.StatusCode == 403 {
			usage.IsForbidden = true
			if usage.ForbiddenType == "" {
				usage.ForbiddenType = "forbidden"
			}
			if usage.ErrorCode == "" {
				usage.ErrorCode = "forbidden"
			}
		}
	}

	return usage
}

func grokQuotaSnapshotFromExtra(extra map[string]any) (*xai.QuotaSnapshot, error) {
	if extra == nil {
		return nil, nil
	}
	raw, ok := extra[grokQuotaSnapshotExtraKey]
	if !ok || raw == nil {
		return nil, nil
	}
	switch snapshot := raw.(type) {
	case *xai.QuotaSnapshot:
		return snapshot, nil
	case xai.QuotaSnapshot:
		return &snapshot, nil
	case map[string]any:
		data, err := json.Marshal(snapshot)
		if err != nil {
			return nil, err
		}
		var out xai.QuotaSnapshot
		if err := json.Unmarshal(data, &out); err != nil {
			return nil, err
		}
		return &out, nil
	default:
		data, err := json.Marshal(raw)
		if err != nil {
			return nil, fmt.Errorf("marshal grok quota snapshot: %w", err)
		}
		var out xai.QuotaSnapshot
		if err := json.Unmarshal(data, &out); err != nil {
			return nil, err
		}
		return &out, nil
	}
}

func grokBillingSnapshotFromExtra(extra map[string]any) (*xai.BillingSnapshot, error) {
	if extra == nil {
		return nil, nil
	}
	raw, ok := extra[grokBillingSnapshotKey]
	if !ok || raw == nil {
		return nil, nil
	}
	switch snapshot := raw.(type) {
	case *xai.BillingSnapshot:
		return snapshot, nil
	case xai.BillingSnapshot:
		return &snapshot, nil
	case map[string]any:
		data, err := json.Marshal(snapshot)
		if err != nil {
			return nil, err
		}
		var out xai.BillingSnapshot
		if err := json.Unmarshal(data, &out); err != nil {
			return nil, err
		}
		return &out, nil
	default:
		data, err := json.Marshal(raw)
		if err != nil {
			return nil, fmt.Errorf("marshal grok billing snapshot: %w", err)
		}
		var out xai.BillingSnapshot
		if err := json.Unmarshal(data, &out); err != nil {
			return nil, err
		}
		return &out, nil
	}
}
