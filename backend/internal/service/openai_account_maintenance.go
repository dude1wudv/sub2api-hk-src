package service

import (
	"context"
	"sort"
	"time"
)

const openAICodexQuotaResetWakeDelay = 30 * time.Second

type OpenAIAccountMaintenanceProxyTarget struct {
	ProxyID   int64  `json:"proxy_id"`
	Name      string `json:"name"`
	LatencyMs *int64 `json:"latency_ms,omitempty"`
}

type OpenAIAccountMaintenanceResult struct {
	Scanned             int                                  `json:"scanned"`
	MovedToSlowPool     int                                  `json:"moved_to_slow_pool"`
	MovedToNormalPool   int                                  `json:"moved_to_normal_pool"`
	AlreadyInTargetPool int                                  `json:"already_in_target_pool"`
	Skipped             int                                  `json:"skipped"`
	SlowestProxy        *OpenAIAccountMaintenanceProxyTarget `json:"slowest_proxy,omitempty"`
	NormalProxyCount    int                                  `json:"normal_proxy_count"`
}

func runOpenAICodexRecoveryMaintenance(ctx context.Context, accountRepo AccountRepository, proxyRepo ProxyRepository, proxyLatencyCache ProxyLatencyCache) (*OpenAIAccountMaintenanceResult, error) {
	result := &OpenAIAccountMaintenanceResult{}
	if accountRepo == nil || proxyRepo == nil || proxyLatencyCache == nil {
		return result, nil
	}

	proxies, err := sortedOpenAIHealthyProxies(ctx, proxyRepo, proxyLatencyCache)
	if err != nil || len(proxies) == 0 {
		return result, err
	}
	normalAssignments := buildOpenAINormalProxyAssignments(proxies)
	slowestProxy := proxies[len(proxies)-1]
	slowestProxyID := slowestProxy.ID
	result.SlowestProxy = openAIProxyTargetFromProxy(slowestProxy)
	result.NormalProxyCount = len(proxies)

	accounts, err := accountRepo.ListByPlatform(ctx, PlatformOpenAI)
	if err != nil {
		return result, err
	}

	normalIdx := 0
	for i := range accounts {
		account := &accounts[i]
		if !account.IsOpenAI() || !account.IsOAuth() || isNoProxyAccountName(account.Name) {
			result.Skipped++
			continue
		}
		result.Scanned++
		if account.IsOpenAICodexUsageExhausted() {
			if account.ProxyID == nil || *account.ProxyID != slowestProxyID {
				if err := updateOpenAIAccountProxy(ctx, accountRepo, account.ID, slowestProxyID); err != nil {
					return result, err
				}
				result.MovedToSlowPool++
			} else {
				result.AlreadyInTargetPool++
			}
			continue
		}
		if account.IsOpenAICodexUsageRecovered() || account.IsSchedulable() {
			if len(normalAssignments) == 0 {
				continue
			}
			targetProxyID := normalAssignments[normalIdx%len(normalAssignments)]
			normalIdx++
			if account.ProxyID == nil || *account.ProxyID != targetProxyID {
				if err := updateOpenAIAccountProxy(ctx, accountRepo, account.ID, targetProxyID); err != nil {
					return result, err
				}
				result.MovedToNormalPool++
				_ = accountRepo.UpdateExtra(ctx, account.ID, map[string]any{
					openAICodexRecoveredFromSlowPoolExtraKey: time.Now().Format(time.RFC3339),
				})
			} else {
				result.AlreadyInTargetPool++
			}
		}
	}
	return result, nil
}

func nextOpenAICodexQuotaResetRecoveryAt(ctx context.Context, accountRepo AccountRepository, now time.Time) (time.Time, error) {
	if accountRepo == nil {
		return time.Time{}, nil
	}
	accounts, err := accountRepo.ListByPlatform(ctx, PlatformOpenAI)
	if err != nil {
		return time.Time{}, err
	}
	var next time.Time
	for i := range accounts {
		account := &accounts[i]
		if !account.IsOpenAI() || !account.IsOAuth() || isNoProxyAccountName(account.Name) {
			continue
		}
		for _, window := range []struct {
			usedKey    string
			resetAtKey string
		}{
			{openAICodexPrimaryUsedPercentExtraKey, openAICodexPrimaryResetAtExtraKey},
			{openAICodex5hUsedPercentExtraKey, openAICodex5hResetAtExtraKey},
			{openAICodex7dUsedPercentExtraKey, openAICodex7dResetAtExtraKey},
		} {
			if account.getExtraFloat64(window.usedKey) < openAICodexQuotaRecoverThresholdPercent {
				continue
			}
			resetAt := account.getExtraTime(window.resetAtKey)
			if resetAt.IsZero() || !now.Before(resetAt) {
				continue
			}
			if next.IsZero() || resetAt.Before(next) {
				next = resetAt
			}
		}
	}
	if next.IsZero() {
		return time.Time{}, nil
	}
	return next.Add(openAICodexQuotaResetWakeDelay), nil
}

func sortedOpenAIHealthyProxies(ctx context.Context, proxyRepo ProxyRepository, proxyLatencyCache ProxyLatencyCache) ([]ProxyWithAccountCount, error) {
	proxies, err := proxyRepo.ListAssignableWithAccountCount(ctx)
	if err != nil || len(proxies) == 0 {
		return proxies, err
	}
	ids := make([]int64, 0, len(proxies))
	for i := range proxies {
		ids = append(ids, proxies[i].ID)
	}
	latencies, err := proxyLatencyCache.GetProxyLatencies(ctx, ids)
	if err != nil {
		return nil, err
	}

	healthy := make([]ProxyWithAccountCount, 0, len(proxies))
	for i := range proxies {
		info := latencies[proxies[i].ID]
		if info == nil || !info.Success || info.LatencyMs == nil || *info.LatencyMs < 0 {
			continue
		}
		if info.QualityStatus == "failed" || info.QualityStatus == "challenge" {
			continue
		}
		proxies[i].LatencyMs = info.LatencyMs
		proxies[i].LatencyStatus = "success"
		proxies[i].QualityStatus = info.QualityStatus
		healthy = append(healthy, proxies[i])
	}
	sort.SliceStable(healthy, func(i, j int) bool {
		iLatency := proxyAssignmentLatency(healthy[i])
		jLatency := proxyAssignmentLatency(healthy[j])
		if iLatency != jLatency {
			return iLatency < jLatency
		}
		return healthy[i].ID < healthy[j].ID
	})
	return healthy, nil
}

func buildOpenAINormalProxyAssignments(proxies []ProxyWithAccountCount) []int64 {
	assignments := make([]int64, 0)
	for i := range proxies {
		limit := proxyAssignmentAccountLimit(i)
		for n := int64(0); n < limit; n++ {
			assignments = append(assignments, proxies[i].ID)
		}
	}
	return assignments
}

func updateOpenAIAccountProxy(ctx context.Context, accountRepo AccountRepository, accountID int64, proxyID int64) error {
	_, err := accountRepo.BulkUpdate(ctx, []int64{accountID}, AccountBulkUpdate{ProxyID: &proxyID})
	return err
}

func openAIProxyTargetFromProxy(proxy ProxyWithAccountCount) *OpenAIAccountMaintenanceProxyTarget {
	return &OpenAIAccountMaintenanceProxyTarget{
		ProxyID:   proxy.ID,
		Name:      proxy.Name,
		LatencyMs: proxy.LatencyMs,
	}
}
