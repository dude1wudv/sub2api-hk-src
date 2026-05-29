package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
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

type OpenAIAccountRiskOverview struct {
	Drain95             int                                  `json:"drain_95"`
	Exhausted99         int                                  `json:"exhausted_99"`
	Challenge           int                                  `json:"challenge"`
	Banned              int                                  `json:"banned"`
	HighFailure         int                                  `json:"high_failure"`
	PartitionCandidates int                                  `json:"partition_candidates"`
	MoveCandidates      int                                  `json:"move_candidates"`
	SlowestProxy        *OpenAIAccountMaintenanceProxyTarget `json:"slowest_proxy,omitempty"`
	Candidates          []OpenAIAccountRiskCandidate         `json:"candidates"`
}

type OpenAIAccountRiskCandidate struct {
	AccountID        int64    `json:"account_id"`
	Name             string   `json:"name"`
	Risk             string   `json:"risk"`
	Reasons          []string `json:"reasons"`
	ErrorMessage     string   `json:"error_message,omitempty"`
	CurrentProxyID   *int64   `json:"current_proxy_id,omitempty"`
	CurrentProxyName string   `json:"current_proxy_name,omitempty"`
	TargetProxyID    int64    `json:"target_proxy_id"`
	TargetProxyName  string   `json:"target_proxy_name"`
	RequiresMove     bool     `json:"requires_move"`
	MaxUsagePercent  float64  `json:"max_usage_percent"`
}

type OpenAIAccountRiskPartitionResult struct {
	Preview            OpenAIAccountRiskOverview `json:"preview"`
	Moved              int                       `json:"moved"`
	AlreadyPartitioned int                       `json:"already_partitioned"`
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

func buildOpenAIAccountRiskOverview(ctx context.Context, accountRepo AccountRepository, proxyRepo ProxyRepository, proxyLatencyCache ProxyLatencyCache) (*OpenAIAccountRiskOverview, error) {
	overview := &OpenAIAccountRiskOverview{}
	if accountRepo == nil || proxyRepo == nil || proxyLatencyCache == nil {
		return overview, nil
	}
	proxies, err := sortedOpenAIHealthyProxies(ctx, proxyRepo, proxyLatencyCache)
	if err != nil {
		return overview, err
	}
	proxyNames := make(map[int64]string, len(proxies))
	for i := range proxies {
		proxyNames[proxies[i].ID] = proxies[i].Name
	}
	if len(proxies) > 0 {
		slowestProxy := proxies[len(proxies)-1]
		overview.SlowestProxy = openAIProxyTargetFromProxy(slowestProxy)
	}

	accounts, err := accountRepo.ListByPlatform(ctx, PlatformOpenAI)
	if err != nil {
		return overview, err
	}
	now := time.Now()
	for i := range accounts {
		account := &accounts[i]
		if !account.IsOpenAI() || !account.IsOAuth() || isNoProxyAccountName(account.Name) {
			continue
		}
		maxUsage := maxOpenAICodexEffectiveUsagePercent(account)
		if maxUsage >= openAICodexQuotaRecoverThresholdPercent {
			overview.Drain95++
		}
		if maxUsage >= openAICodexQuotaDrainThresholdPercent {
			overview.Exhausted99++
		}

		risk, reasons := classifyOpenAIAccountOperationalRisk(account, now)
		switch risk {
		case "challenge":
			overview.Challenge++
		case "banned":
			overview.Banned++
		case "high_failure":
			overview.HighFailure++
		}
		if risk == "" || overview.SlowestProxy == nil {
			continue
		}
		candidate := OpenAIAccountRiskCandidate{
			AccountID:       account.ID,
			Name:            account.Name,
			Risk:            risk,
			Reasons:         reasons,
			ErrorMessage:    account.ErrorMessage,
			CurrentProxyID:  account.ProxyID,
			TargetProxyID:   overview.SlowestProxy.ProxyID,
			TargetProxyName: overview.SlowestProxy.Name,
			MaxUsagePercent: roundPercent(maxUsage),
		}
		if account.ProxyID != nil {
			candidate.CurrentProxyName = proxyNames[*account.ProxyID]
			candidate.RequiresMove = *account.ProxyID != overview.SlowestProxy.ProxyID
		} else {
			candidate.CurrentProxyName = "No proxy"
			candidate.RequiresMove = true
		}
		overview.PartitionCandidates++
		if candidate.RequiresMove {
			overview.MoveCandidates++
		}
		overview.Candidates = append(overview.Candidates, candidate)
	}

	sort.SliceStable(overview.Candidates, func(i, j int) bool {
		ri := openAIAccountRiskRank(overview.Candidates[i].Risk)
		rj := openAIAccountRiskRank(overview.Candidates[j].Risk)
		if ri != rj {
			return ri < rj
		}
		if overview.Candidates[i].RequiresMove != overview.Candidates[j].RequiresMove {
			return overview.Candidates[i].RequiresMove
		}
		return overview.Candidates[i].AccountID < overview.Candidates[j].AccountID
	})
	return overview, nil
}

func applyOpenAIAccountRiskPartition(ctx context.Context, accountRepo AccountRepository, proxyRepo ProxyRepository, proxyLatencyCache ProxyLatencyCache) (*OpenAIAccountRiskPartitionResult, error) {
	preview, err := buildOpenAIAccountRiskOverview(ctx, accountRepo, proxyRepo, proxyLatencyCache)
	if err != nil {
		return nil, err
	}
	result := &OpenAIAccountRiskPartitionResult{Preview: *preview}
	if preview.SlowestProxy == nil {
		return result, nil
	}
	for _, candidate := range preview.Candidates {
		if !candidate.RequiresMove {
			result.AlreadyPartitioned++
			continue
		}
		if err := updateOpenAIAccountProxy(ctx, accountRepo, candidate.AccountID, preview.SlowestProxy.ProxyID); err != nil {
			return result, err
		}
		result.Moved++
	}
	return result, nil
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

func maxOpenAICodexEffectiveUsagePercent(account *Account) float64 {
	if account == nil {
		return 0
	}
	values := []float64{
		account.openAICodexEffectiveUsedPercent(openAICodexPrimaryUsedPercentExtraKey, openAICodexPrimaryResetAtExtraKey),
		account.openAICodexEffectiveUsedPercent(openAICodex5hUsedPercentExtraKey, openAICodex5hResetAtExtraKey),
		account.openAICodexEffectiveUsedPercent(openAICodex7dUsedPercentExtraKey, openAICodex7dResetAtExtraKey),
	}
	max := 0.0
	for _, value := range values {
		if value > max {
			max = value
		}
	}
	return max
}

func classifyOpenAIAccountOperationalRisk(account *Account, now time.Time) (string, []string) {
	if account == nil {
		return "", nil
	}
	reasons := make([]string, 0, 3)
	raw := strings.ToLower(strings.Join([]string{
		account.ErrorMessage,
		account.TempUnschedulableReason,
		fmt.Sprint(account.Extra["last_error"]),
		fmt.Sprint(account.Extra["last_error_message"]),
		fmt.Sprint(account.Extra["forbidden_type"]),
	}, " "))

	if containsAny(raw, "challenge", "captcha", "cloudflare", "cf_challenge", "turnstile", "人机", "验证") {
		reasons = append(reasons, "challenge")
		return "challenge", reasons
	}
	if containsAny(raw, "banned", "ban", "disabled", "deactivated", "violation", "access_denied", "ip_blocked", "organization has been disabled", "封禁", "被封", "禁用") {
		reasons = append(reasons, "banned_or_disabled")
		return "banned", reasons
	}

	if account.Status == StatusError {
		reasons = append(reasons, "status_error")
	}
	if account.TempUnschedulableUntil != nil && now.Before(*account.TempUnschedulableUntil) {
		reasons = append(reasons, "temp_unschedulable")
	}
	if account.RateLimitedAt != nil || account.IsOverloaded() {
		reasons = append(reasons, "runtime_limited")
	}
	if containsAny(raw, "failure", "failed", "timeout", "connection", "upstream", "eof", "reset", " 5xx", " 500", " 502", " 503", " 529", "429") {
		reasons = append(reasons, "upstream_failure")
	}
	if len(reasons) == 0 {
		return "", nil
	}
	return "high_failure", dedupeOpenAIAccountRiskReasons(reasons)
}

func containsAny(value string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}

func dedupeOpenAIAccountRiskReasons(values []string) []string {
	out := values[:0]
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func openAIAccountRiskRank(risk string) int {
	switch risk {
	case "banned":
		return 0
	case "challenge":
		return 1
	case "high_failure":
		return 2
	default:
		return 9
	}
}
