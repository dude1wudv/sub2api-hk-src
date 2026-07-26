package service

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/xai"
)

// AccountUsageWindowSummary describes aggregate quota usage for one window.
type AccountUsageWindowSummary struct {
	Sampled          int      `json:"sampled"`
	UsedPercent      *float64 `json:"used_percent,omitempty"`
	RemainingPercent *float64 `json:"remaining_percent,omitempty"`
	Exhausted        int      `json:"exhausted"`
}

type AccountQuotaPoolSummary struct {
	Total              int      `json:"total"`
	Available          int      `json:"available"`
	Sampled            int      `json:"sampled"`
	RemainingPercent   *float64 `json:"remaining_percent,omitempty"`
	Remaining5hPercent *float64 `json:"remaining_5h_percent,omitempty"`
	Remaining7dPercent *float64 `json:"remaining_7d_percent,omitempty"`
	Exhausted          int      `json:"exhausted"`
}

type AccountProxySummary struct {
	ProxyID            *int64   `json:"proxy_id"`
	Name               string   `json:"name"`
	Total              int      `json:"total"`
	Available          int      `json:"available"`
	Used5hPercent      *float64 `json:"used_5h_percent,omitempty"`
	Used7dPercent      *float64 `json:"used_7d_percent,omitempty"`
	Remaining5hPercent *float64 `json:"remaining_5h_percent,omitempty"`
	Remaining7dPercent *float64 `json:"remaining_7d_percent,omitempty"`
}

type AccountSummary struct {
	Total             int64                     `json:"total"`
	Available         int                       `json:"available"`
	Active            int                       `json:"active"`
	Inactive          int                       `json:"inactive"`
	Error             int                       `json:"error"`
	Paused            int                       `json:"paused"`
	Unschedulable     int                       `json:"unschedulable"`
	RateLimited       int                       `json:"rate_limited"`
	TempUnschedulable int                       `json:"temp_unschedulable"`
	Overloaded        int                       `json:"overloaded"`
	Expired           int                       `json:"expired"`
	QuotaExceeded     int                       `json:"quota_exceeded"`
	OpenAI            int                       `json:"openai"`
	Codex5h           AccountUsageWindowSummary `json:"codex_5h"`
	Codex7d           AccountUsageWindowSummary `json:"codex_7d"`
	OAuthPool         AccountQuotaPoolSummary   `json:"oauth_pool"`
	FreePool          AccountQuotaPoolSummary   `json:"free_pool"`
	GrokPool          AccountQuotaPoolSummary   `json:"grok_pool"`
	RecentlyUsed      int                       `json:"recently_used"`
	NeverUsed         int                       `json:"never_used"`
	ProxyDistribution []AccountProxySummary     `json:"proxy_distribution"`
}

type OAuthAccountUsageSummary struct {
	TotalStandardCost                  float64    `json:"total_standard_cost"`
	AverageStandardCostPerOAuthAccount float64    `json:"average_standard_cost_per_oauth_account"`
	OAuthAccountCount                  int64      `json:"oauth_account_count"`
	OAuthAccountsWithUsage             int64      `json:"oauth_accounts_with_usage"`
	UsageLogCount                      int64      `json:"usage_log_count"`
	DeletedOAuthAccountCount           int64      `json:"deleted_oauth_account_count"`
	ExpiredOAuthAccountCount           int64      `json:"expired_oauth_account_count"`
	FirstUsageAt                       *time.Time `json:"first_usage_at,omitempty"`
	LastUsageAt                        *time.Time `json:"last_usage_at,omitempty"`
}

type accountUsageAccumulator struct {
	count, exhausted int
	totalUsed        float64
}

func (a *accountUsageAccumulator) add(value float64) {
	value = math.Max(0, math.Min(100, value))
	a.count++
	a.totalUsed += value
	if value >= 100 {
		a.exhausted++
	}
}

func (a accountUsageAccumulator) summary() AccountUsageWindowSummary {
	out := AccountUsageWindowSummary{Sampled: a.count, Exhausted: a.exhausted}
	if a.count == 0 {
		return out
	}
	used := math.Round(a.totalUsed/float64(a.count)*10) / 10
	remaining := math.Round((100-used)*10) / 10
	out.UsedPercent, out.RemainingPercent = &used, &remaining
	return out
}

func (s *adminServiceImpl) GetAccountSummary(ctx context.Context, platform, accountType, status, search string, groupID int64, privacyMode string) (*AccountSummary, error) {
	accounts, err := s.accountRepo.ListAllWithFilters(ctx, platform, accountType, status, search, groupID, privacyMode)
	if err != nil {
		return nil, err
	}
	return buildAccountSummary(accounts), nil
}

type accountQuotaPoolAccumulator struct {
	total, available   int
	fiveHour, sevenDay accountUsageAccumulator
}

func (a *accountQuotaPoolAccumulator) addAccount(available bool) {
	a.total++
	if available {
		a.available++
	}
}
func (a accountQuotaPoolAccumulator) summary(prefer string) AccountQuotaPoolSummary {
	fiveHour, sevenDay := a.fiveHour.summary(), a.sevenDay.summary()
	preferred := fiveHour
	if prefer == "7d" {
		preferred = sevenDay
	}
	return AccountQuotaPoolSummary{Total: a.total, Available: a.available, Sampled: preferred.Sampled,
		RemainingPercent: preferred.RemainingPercent, Remaining5hPercent: fiveHour.RemainingPercent,
		Remaining7dPercent: sevenDay.RemainingPercent, Exhausted: preferred.Exhausted}
}

type accountProxyAccumulator struct {
	id                 *int64
	name               string
	total, available   int
	fiveHour, sevenDay accountUsageAccumulator
}

func buildAccountSummary(accounts []Account) *AccountSummary {
	now := time.Now()
	out := &AccountSummary{ProxyDistribution: []AccountProxySummary{}}
	proxyStats := map[string]*accountProxyAccumulator{}
	var fiveHour, sevenDay accountUsageAccumulator
	var oauthPool, grokPool accountQuotaPoolAccumulator
	for i := range accounts {
		account := &accounts[i]
		out.Total++
		if account.Status == StatusActive {
			out.Active++
		}
		switch account.Status {
		case StatusDisabled, "inactive":
			out.Inactive++
		case StatusError:
			out.Error++
		}
		if account.LastUsedAt == nil {
			out.NeverUsed++
		} else if account.LastUsedAt.After(now.Add(-time.Hour)) {
			out.RecentlyUsed++
		}
		available := account.IsSchedulable()
		if available {
			out.Available++
		} else if account.Status != StatusError {
			out.Paused++
		}
		if account.Status == StatusActive && !account.Schedulable {
			out.Unschedulable++
		}
		if account.IsRateLimited() {
			out.RateLimited++
		}
		if account.IsOverloaded() {
			out.Overloaded++
		}
		if account.TempUnschedulableUntil != nil && now.Before(*account.TempUnschedulableUntil) {
			out.TempUnschedulable++
		}
		if account.AutoPauseOnExpired && account.ExpiresAt != nil && !now.Before(*account.ExpiresAt) {
			out.Expired++
		}
		if account.IsAPIKeyOrBedrock() && account.IsQuotaExceeded() {
			out.QuotaExceeded++
		}
		if account.IsGrokOAuth() {
			grokPool.addAccount(available)
			if value, ok := grokShortWindowUsedPercentFromExtra(account.Extra, now); ok {
				grokPool.fiveHour.add(value)
			}
			if value, ok := grokWeeklyUsedPercentFromExtra(account.Extra); ok {
				grokPool.sevenDay.add(value)
			}
		}
		if account.Platform != PlatformOpenAI {
			continue
		}
		out.OpenAI++
		key, proxyID, proxyName := accountSummaryProxyKey(account)
		proxy := proxyStats[key]
		if proxy == nil {
			proxy = &accountProxyAccumulator{id: proxyID, name: proxyName}
			proxyStats[key] = proxy
		}
		proxy.total++
		if available {
			proxy.available++
		}
		if !account.IsOpenAIOAuth() {
			continue
		}
		oauthPool.addAccount(available)
		if value, ok := codexUsagePercentFromExtra(account.Extra, "codex_5h_used_percent", "codex_5h_reset_at", now); ok {
			fiveHour.add(value)
			oauthPool.fiveHour.add(value)
			proxy.fiveHour.add(value)
		}
		if value, ok := codexUsagePercentFromExtra(account.Extra, "codex_7d_used_percent", "codex_7d_reset_at", now); ok {
			sevenDay.add(value)
			oauthPool.sevenDay.add(value)
			proxy.sevenDay.add(value)
		}
	}
	out.Codex5h, out.Codex7d = fiveHour.summary(), sevenDay.summary()
	out.OAuthPool = oauthPool.summary("5h")
	out.GrokPool = grokPool.summary("7d")
	out.ProxyDistribution = buildAccountProxySummary(proxyStats)
	return out
}

func codexUsagePercentFromExtra(extra map[string]any, usedKey, resetAtKey string, now time.Time) (float64, bool) {
	if len(extra) == 0 {
		return 0, false
	}
	raw, ok := extra[usedKey]
	if !ok {
		return 0, false
	}
	if resetRaw, ok := extra[resetAtKey]; ok {
		if resetAt, err := parseTime(fmt.Sprint(resetRaw)); err == nil && !now.Before(resetAt) {
			return 0, true
		}
	}
	return parseExtraFloat64(raw), true
}

func accountSummaryProxyKey(account *Account) (string, *int64, string) {
	if account == nil || account.ProxyID == nil {
		return "none", nil, "No proxy"
	}
	id := *account.ProxyID
	name := fmt.Sprintf("Proxy #%d", id)
	if account.Proxy != nil && strings.TrimSpace(account.Proxy.Name) != "" {
		name = account.Proxy.Name
	}
	return fmt.Sprintf("proxy:%d", id), &id, name
}

func buildAccountProxySummary(stats map[string]*accountProxyAccumulator) []AccountProxySummary {
	out := make([]AccountProxySummary, 0, len(stats))
	for _, stat := range stats {
		fiveHour, sevenDay := stat.fiveHour.summary(), stat.sevenDay.summary()
		out = append(out, AccountProxySummary{ProxyID: stat.id, Name: stat.name, Total: stat.total, Available: stat.available,
			Used5hPercent: fiveHour.UsedPercent, Used7dPercent: sevenDay.UsedPercent,
			Remaining5hPercent: fiveHour.RemainingPercent, Remaining7dPercent: sevenDay.RemainingPercent})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Total != out[j].Total {
			return out[i].Total > out[j].Total
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func grokShortWindowUsedPercentFromExtra(extra map[string]any, now time.Time) (float64, bool) {
	snapshot, err := grokQuotaSnapshotFromExtra(extra)
	if err != nil || snapshot == nil {
		return 0, false
	}
	if value, ok := grokQuotaWindowUsedPercent(snapshot.Requests, now); ok {
		return value, true
	}
	return grokQuotaWindowUsedPercent(snapshot.Tokens, now)
}

func grokQuotaWindowUsedPercent(window *xai.QuotaWindow, now time.Time) (float64, bool) {
	if window == nil || window.Limit == nil || *window.Limit <= 0 {
		return 0, false
	}
	if window.ResetAt != "" {
		if resetAt, err := parseTime(window.ResetAt); err == nil && !now.Before(resetAt) {
			return 0, true
		}
	}
	remaining := int64(0)
	if window.Remaining != nil {
		remaining = *window.Remaining
		if remaining < 0 {
			remaining = 0
		}
	}
	used := float64(*window.Limit - remaining)
	if used < 0 {
		used = 0
	}
	return used / float64(*window.Limit) * 100, true
}

func grokWeeklyUsedPercentFromExtra(extra map[string]any) (float64, bool) {
	billing, err := grokBillingSnapshotFromExtra(extra)
	if err == nil && billing != nil {
		if billing.UsagePercent != nil {
			return *billing.UsagePercent, true
		}
		if billing.UsedPercent != nil {
			return *billing.UsedPercent, true
		}
	}
	if raw, ok := extra["grok_billing_snapshot"].(map[string]any); ok {
		if utilization, exists := raw["utilization"]; exists {
			return parseExtraFloat64(utilization), true
		}
	}
	return 0, false
}

const (
	settingKeyOpenAIOAuthUsageBaselineAccountIDs = "openai_oauth_usage_summary_baseline_account_ids"
	settingKeyOpenAIOAuthUsageCountAfter         = "openai_oauth_usage_summary_count_after"
)

func openAIOAuthUsageSummaryQuery() string {
	return `
WITH raw_oauth_usage_baseline AS (
	SELECT
		COALESCE((SELECT value FROM settings WHERE key = '` + settingKeyOpenAIOAuthUsageBaselineAccountIDs + `'), '') AS baseline_account_ids,
		NULLIF(trim(COALESCE((SELECT value FROM settings WHERE key = '` + settingKeyOpenAIOAuthUsageCountAfter + `'), '')), '')::timestamptz AS count_after
),
baseline_account_ids AS (
	SELECT DISTINCT trim(value)::bigint AS id
	FROM raw_oauth_usage_baseline,
		regexp_split_to_table(baseline_account_ids, ',') AS value
	WHERE trim(value) ~ '^[0-9]+$'
),
oauth_usage_baseline AS (
	SELECT count_after FROM raw_oauth_usage_baseline
),
target_account_rows AS (
	SELECT a.id, a.deleted_at, a.auto_pause_on_expired, a.expires_at
	FROM accounts a
	WHERE a.platform = 'openai' AND a.type = 'oauth'
	  AND lower(trim(COALESCE(a.name, ''))) NOT IN ('1day', 'opentoken')
	  AND ((NOT EXISTS (SELECT 1 FROM baseline_account_ids) AND (SELECT count_after FROM oauth_usage_baseline) IS NULL)
		OR a.id IN (SELECT id FROM baseline_account_ids)
		OR ((SELECT count_after FROM oauth_usage_baseline) IS NOT NULL
			AND a.created_at >= (SELECT count_after FROM oauth_usage_baseline)))
),
target_usage AS (
	SELECT tar.id AS account_id, ul.total_cost, ul.created_at
	FROM usage_logs ul JOIN target_account_rows tar ON tar.id = ul.account_id
),
target_counts AS (
	SELECT COUNT(*)::bigint AS account_count FROM target_account_rows
)
SELECT
	COALESCE(SUM(tu.total_cost), 0)::double precision,
	CASE WHEN tc.account_count > 0 THEN (COALESCE(SUM(tu.total_cost), 0) / tc.account_count)::double precision ELSE 0::double precision END,
	tc.account_count,
	COUNT(DISTINCT tu.account_id)::bigint AS accounts_with_usage,
	COUNT(tu.account_id)::bigint,
	(SELECT COUNT(*)::bigint FROM target_account_rows tar WHERE tar.deleted_at IS NOT NULL),
	(SELECT COUNT(*)::bigint FROM target_account_rows tar WHERE tar.auto_pause_on_expired AND tar.expires_at IS NOT NULL AND tar.expires_at <= NOW()),
	MIN(tu.created_at), MAX(tu.created_at)
FROM target_counts tc LEFT JOIN target_usage tu ON TRUE
GROUP BY tc.account_count`
}

func (s *adminServiceImpl) GetOAuthAccountUsageSummary(ctx context.Context) (*OAuthAccountUsageSummary, error) {
	out := &OAuthAccountUsageSummary{}
	if s == nil || s.entClient == nil {
		return out, nil
	}
	rows, err := s.entClient.QueryContext(ctx, openAIOAuthUsageSummaryQuery())
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return out, rows.Err()
	}
	var first, last sql.NullTime
	if err := rows.Scan(&out.TotalStandardCost, &out.AverageStandardCostPerOAuthAccount, &out.OAuthAccountCount,
		&out.OAuthAccountsWithUsage, &out.UsageLogCount, &out.DeletedOAuthAccountCount, &out.ExpiredOAuthAccountCount,
		&first, &last); err != nil {
		return nil, err
	}
	if first.Valid {
		out.FirstUsageAt = &first.Time
	}
	if last.Valid {
		out.LastUsageAt = &last.Time
	}
	return out, rows.Err()
}
