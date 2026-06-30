package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
)

const (
	defaultDashboardStatsFreshTTL       = 15 * time.Second
	defaultDashboardStatsCacheTTL       = 30 * time.Second
	defaultDashboardStatsRefreshTimeout = 30 * time.Second
)

// ErrDashboardStatsCacheMiss 标记仪表盘缓存未命中。
var ErrDashboardStatsCacheMiss = errors.New("仪表盘缓存未命中")

// DashboardStatsCache 定义仪表盘统计缓存接口。
type DashboardStatsCache interface {
	GetDashboardStats(ctx context.Context) (string, error)
	SetDashboardStats(ctx context.Context, data string, ttl time.Duration) error
	DeleteDashboardStats(ctx context.Context) error
}

type dashboardStatsRangeFetcher interface {
	GetDashboardStatsWithRange(ctx context.Context, start, end time.Time) (*usagestats.DashboardStats, error)
}

type dashboardStatsCacheEntry struct {
	Stats     *usagestats.DashboardStats `json:"stats"`
	UpdatedAt int64                      `json:"updated_at"`
}

// DashboardService 提供管理员仪表盘统计服务。
type DashboardService struct {
	usageRepo      UsageLogRepository
	aggRepo        DashboardAggregationRepository
	accountRepo    AccountRepository
	cache          DashboardStatsCache
	httpUpstream   HTTPUpstream
	cacheFreshTTL  time.Duration
	cacheTTL       time.Duration
	refreshTimeout time.Duration
	refreshing     int32
	aggEnabled     bool
	aggInterval    time.Duration
	aggLookback    time.Duration
	aggUsageDays   int
}

func NewDashboardService(usageRepo UsageLogRepository, aggRepo DashboardAggregationRepository, cache DashboardStatsCache, cfg *config.Config) *DashboardService {
	freshTTL := defaultDashboardStatsFreshTTL
	cacheTTL := defaultDashboardStatsCacheTTL
	refreshTimeout := defaultDashboardStatsRefreshTimeout
	aggEnabled := true
	aggInterval := time.Minute
	aggLookback := 2 * time.Minute
	aggUsageDays := 90
	if cfg != nil {
		if !cfg.Dashboard.Enabled {
			cache = nil
		}
		if cfg.Dashboard.StatsFreshTTLSeconds > 0 {
			freshTTL = time.Duration(cfg.Dashboard.StatsFreshTTLSeconds) * time.Second
		}
		if cfg.Dashboard.StatsTTLSeconds > 0 {
			cacheTTL = time.Duration(cfg.Dashboard.StatsTTLSeconds) * time.Second
		}
		if cfg.Dashboard.StatsRefreshTimeoutSeconds > 0 {
			refreshTimeout = time.Duration(cfg.Dashboard.StatsRefreshTimeoutSeconds) * time.Second
		}
		aggEnabled = cfg.DashboardAgg.Enabled
		if cfg.DashboardAgg.IntervalSeconds > 0 {
			aggInterval = time.Duration(cfg.DashboardAgg.IntervalSeconds) * time.Second
		}
		if cfg.DashboardAgg.LookbackSeconds > 0 {
			aggLookback = time.Duration(cfg.DashboardAgg.LookbackSeconds) * time.Second
		}
		if cfg.DashboardAgg.Retention.UsageLogsDays > 0 {
			aggUsageDays = cfg.DashboardAgg.Retention.UsageLogsDays
		}
	}
	if aggRepo == nil {
		aggEnabled = false
	}
	return &DashboardService{
		usageRepo:      usageRepo,
		aggRepo:        aggRepo,
		cache:          cache,
		cacheFreshTTL:  freshTTL,
		cacheTTL:       cacheTTL,
		refreshTimeout: refreshTimeout,
		aggEnabled:     aggEnabled,
		aggInterval:    aggInterval,
		aggLookback:    aggLookback,
		aggUsageDays:   aggUsageDays,
	}
}

func ProvideDashboardService(usageRepo UsageLogRepository, aggRepo DashboardAggregationRepository, cache DashboardStatsCache, cfg *config.Config, accountRepo AccountRepository, httpUpstream HTTPUpstream) *DashboardService {
	svc := NewDashboardService(usageRepo, aggRepo, cache, cfg)
	svc.SetUpstreamBalanceDeps(accountRepo, httpUpstream)
	return svc
}

func (s *DashboardService) SetUpstreamBalanceDeps(accountRepo AccountRepository, httpUpstream HTTPUpstream) {
	s.accountRepo = accountRepo
	s.httpUpstream = httpUpstream
}

type UpstreamBalanceAccount struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	GroupID   int64   `json:"group_id"`
	GroupName string  `json:"group_name"`
	Balance   float64 `json:"balance"`
	Unit      string  `json:"unit"`
	Error     string  `json:"error,omitempty"`
}

type UpstreamBalanceSummary struct {
	Total float64                  `json:"total"`
	Unit  string                   `json:"unit"`
	Items []UpstreamBalanceAccount `json:"items"`
}

func (s *DashboardService) GetUpstreamBalances(ctx context.Context) (*UpstreamBalanceSummary, error) {
	if s.accountRepo == nil || s.httpUpstream == nil {
		return &UpstreamBalanceSummary{Unit: "USD", Items: []UpstreamBalanceAccount{}}, nil
	}
	accounts, err := s.accountRepo.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	out := &UpstreamBalanceSummary{Unit: "USD", Items: []UpstreamBalanceAccount{}}
	for i := range accounts {
		account := &accounts[i]
		groupID, groupName, ok := balanceMonitorGroup(account)
		if !ok {
			continue
		}
		item := UpstreamBalanceAccount{ID: account.ID, Name: account.Name, GroupID: groupID, GroupName: groupName, Unit: "USD"}
		balance, unit, fetchErr := s.fetchUpstreamBalance(ctx, account)
		if fetchErr != nil {
			item.Error = fetchErr.Error()
		} else {
			item.Balance = balance
			if unit != "" {
				item.Unit = unit
			}
			out.Total += balance
		}
		out.Items = append(out.Items, item)
	}
	return out, nil
}

func balanceMonitorGroup(account *Account) (int64, string, bool) {
	if account == nil {
		return 0, "", false
	}
	for _, group := range account.Groups {
		if group != nil && strings.Contains(strings.ToLower(group.Name), "余额") {
			return group.ID, group.Name, true
		}
	}
	if strings.Contains(strings.ToLower(account.Name), "余额") {
		return 0, "", true
	}
	return 0, "", false
}

func (s *DashboardService) fetchUpstreamBalance(ctx context.Context, account *Account) (float64, string, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(account.GetCredential("base_url")), "/")
	apiKey := account.GetCredential("api_key")
	if baseURL == "" || apiKey == "" {
		return 0, "", errors.New("missing base_url or api_key")
	}
	callCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	for _, path := range []string{"/v1/usage", "/dashboard/billing/credit_grants", "/billing/credit_grants"} {
		req, err := http.NewRequestWithContext(callCtx, http.MethodGet, baseURL+path, nil)
		if err != nil {
			return 0, "", err
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "sub2api-admin-balance-monitor/1.0")
		resp, err := s.httpUpstream.Do(req.WithContext(WithHTTPUpstreamProfile(req.Context(), HTTPUpstreamProfileOpenAI)), dashboardAccountProxyURL(account), account.ID, maxInt(account.Concurrency, 1))
		if err != nil {
			return 0, "", err
		}
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		_ = resp.Body.Close()
		if readErr != nil {
			return 0, "", readErr
		}
		if resp.StatusCode >= 400 {
			continue
		}
		balance, unit, ok := parseBalancePayload(body)
		if ok {
			return balance, unit, nil
		}
	}
	return 0, "", errors.New("balance unavailable")
}

func dashboardAccountProxyURL(account *Account) string {
	if account != nil && account.ProxyID != nil && account.Proxy != nil {
		return account.Proxy.URL()
	}
	return ""
}

func parseBalancePayload(body []byte) (float64, string, bool) {
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return 0, "", false
	}
	unit := findString(payload, "unit")
	for _, key := range []string{"total_available", "current_balance", "balance", "remaining", "available"} {
		if v, ok := findNumber(payload, key); ok {
			return v, unit, true
		}
	}
	return 0, unit, false
}

func findNumber(v any, key string) (float64, bool) {
	switch x := v.(type) {
	case map[string]any:
		if raw, ok := x[key]; ok {
			if n, ok := numberValue(raw); ok {
				return n, true
			}
		}
		for _, raw := range x {
			if n, ok := findNumber(raw, key); ok {
				return n, true
			}
		}
	case []any:
		for _, raw := range x {
			if n, ok := findNumber(raw, key); ok {
				return n, true
			}
		}
	}
	return 0, false
}

func numberValue(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case json.Number:
		n, err := x.Float64()
		return n, err == nil
	case string:
		n, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
		return n, err == nil
	default:
		return 0, false
	}
}

func findString(v any, key string) string {
	switch x := v.(type) {
	case map[string]any:
		if raw, ok := x[key].(string); ok {
			return raw
		}
		for _, raw := range x {
			if s := findString(raw, key); s != "" {
				return s
			}
		}
	case []any:
		for _, raw := range x {
			if s := findString(raw, key); s != "" {
				return s
			}
		}
	}
	return ""
}

func (s *DashboardService) GetDashboardStats(ctx context.Context) (*usagestats.DashboardStats, error) {
	if s.cache != nil {
		cached, fresh, err := s.getCachedDashboardStats(ctx)
		if err == nil && cached != nil {
			s.refreshAggregationStaleness(cached)
			if !fresh {
				s.refreshDashboardStatsAsync()
			}
			return cached, nil
		}
		if err != nil && !errors.Is(err, ErrDashboardStatsCacheMiss) {
			logger.LegacyPrintf("service.dashboard", "[Dashboard] 仪表盘缓存读取失败: %v", err)
		}
	}

	stats, err := s.refreshDashboardStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("get dashboard stats: %w", err)
	}
	return stats, nil
}

func (s *DashboardService) GetUsageTrendWithFilters(ctx context.Context, startTime, endTime time.Time, granularity string, userID, apiKeyID, accountID, groupID int64, model string, requestType *int16, stream *bool, billingType *int8) ([]usagestats.TrendDataPoint, error) {
	trend, err := s.usageRepo.GetUsageTrendWithFilters(ctx, startTime, endTime, granularity, userID, apiKeyID, accountID, groupID, model, requestType, stream, billingType)
	if err != nil {
		return nil, fmt.Errorf("get usage trend with filters: %w", err)
	}
	return trend, nil
}

func (s *DashboardService) GetModelStatsWithFilters(ctx context.Context, startTime, endTime time.Time, userID, apiKeyID, accountID, groupID int64, requestType *int16, stream *bool, billingType *int8) ([]usagestats.ModelStat, error) {
	stats, err := s.usageRepo.GetModelStatsWithFilters(ctx, startTime, endTime, userID, apiKeyID, accountID, groupID, requestType, stream, billingType)
	if err != nil {
		return nil, fmt.Errorf("get model stats with filters: %w", err)
	}
	return stats, nil
}

func (s *DashboardService) GetModelStatsWithFiltersBySource(ctx context.Context, startTime, endTime time.Time, userID, apiKeyID, accountID, groupID int64, requestType *int16, stream *bool, billingType *int8, modelSource string) ([]usagestats.ModelStat, error) {
	normalizedSource := usagestats.NormalizeModelSource(modelSource)
	if normalizedSource == usagestats.ModelSourceRequested {
		return s.GetModelStatsWithFilters(ctx, startTime, endTime, userID, apiKeyID, accountID, groupID, requestType, stream, billingType)
	}

	type modelStatsBySourceRepo interface {
		GetModelStatsWithFiltersBySource(ctx context.Context, startTime, endTime time.Time, userID, apiKeyID, accountID, groupID int64, requestType *int16, stream *bool, billingType *int8, source string) ([]usagestats.ModelStat, error)
	}

	if sourceRepo, ok := s.usageRepo.(modelStatsBySourceRepo); ok {
		stats, err := sourceRepo.GetModelStatsWithFiltersBySource(ctx, startTime, endTime, userID, apiKeyID, accountID, groupID, requestType, stream, billingType, normalizedSource)
		if err != nil {
			return nil, fmt.Errorf("get model stats with filters by source: %w", err)
		}
		return stats, nil
	}

	return s.GetModelStatsWithFilters(ctx, startTime, endTime, userID, apiKeyID, accountID, groupID, requestType, stream, billingType)
}

func (s *DashboardService) GetGroupStatsWithFilters(ctx context.Context, startTime, endTime time.Time, userID, apiKeyID, accountID, groupID int64, requestType *int16, stream *bool, billingType *int8) ([]usagestats.GroupStat, error) {
	stats, err := s.usageRepo.GetGroupStatsWithFilters(ctx, startTime, endTime, userID, apiKeyID, accountID, groupID, requestType, stream, billingType)
	if err != nil {
		return nil, fmt.Errorf("get group stats with filters: %w", err)
	}
	return stats, nil
}

// GetGroupUsageSummary returns today's and cumulative cost for all groups.
func (s *DashboardService) GetGroupUsageSummary(ctx context.Context, todayStart time.Time) ([]usagestats.GroupUsageSummary, error) {
	results, err := s.usageRepo.GetAllGroupUsageSummary(ctx, todayStart)
	if err != nil {
		return nil, fmt.Errorf("get group usage summary: %w", err)
	}
	return results, nil
}

func (s *DashboardService) getCachedDashboardStats(ctx context.Context) (*usagestats.DashboardStats, bool, error) {
	data, err := s.cache.GetDashboardStats(ctx)
	if err != nil {
		return nil, false, err
	}

	var entry dashboardStatsCacheEntry
	if err := json.Unmarshal([]byte(data), &entry); err != nil {
		s.evictDashboardStatsCache(err)
		return nil, false, ErrDashboardStatsCacheMiss
	}
	if entry.Stats == nil {
		s.evictDashboardStatsCache(errors.New("仪表盘缓存缺少统计数据"))
		return nil, false, ErrDashboardStatsCacheMiss
	}

	age := time.Since(time.Unix(entry.UpdatedAt, 0))
	return entry.Stats, age <= s.cacheFreshTTL, nil
}

func (s *DashboardService) refreshDashboardStats(ctx context.Context) (*usagestats.DashboardStats, error) {
	stats, err := s.fetchDashboardStats(ctx)
	if err != nil {
		return nil, err
	}
	s.applyAggregationStatus(ctx, stats)
	cacheCtx, cancel := s.cacheOperationContext()
	defer cancel()
	s.saveDashboardStatsCache(cacheCtx, stats)
	return stats, nil
}

func (s *DashboardService) refreshDashboardStatsAsync() {
	if s.cache == nil {
		return
	}
	if !atomic.CompareAndSwapInt32(&s.refreshing, 0, 1) {
		return
	}

	go func() {
		defer atomic.StoreInt32(&s.refreshing, 0)

		ctx, cancel := context.WithTimeout(context.Background(), s.refreshTimeout)
		defer cancel()

		stats, err := s.fetchDashboardStats(ctx)
		if err != nil {
			logger.LegacyPrintf("service.dashboard", "[Dashboard] 仪表盘缓存异步刷新失败: %v", err)
			return
		}
		s.applyAggregationStatus(ctx, stats)
		cacheCtx, cancel := s.cacheOperationContext()
		defer cancel()
		s.saveDashboardStatsCache(cacheCtx, stats)
	}()
}

func (s *DashboardService) fetchDashboardStats(ctx context.Context) (*usagestats.DashboardStats, error) {
	if !s.aggEnabled {
		if fetcher, ok := s.usageRepo.(dashboardStatsRangeFetcher); ok {
			now := time.Now().UTC()
			start := truncateToDayUTC(now.AddDate(0, 0, -s.aggUsageDays))
			return fetcher.GetDashboardStatsWithRange(ctx, start, now)
		}
	}
	return s.usageRepo.GetDashboardStats(ctx)
}

func (s *DashboardService) saveDashboardStatsCache(ctx context.Context, stats *usagestats.DashboardStats) {
	if s.cache == nil || stats == nil {
		return
	}

	entry := dashboardStatsCacheEntry{
		Stats:     stats,
		UpdatedAt: time.Now().Unix(),
	}
	data, err := json.Marshal(entry)
	if err != nil {
		logger.LegacyPrintf("service.dashboard", "[Dashboard] 仪表盘缓存序列化失败: %v", err)
		return
	}

	if err := s.cache.SetDashboardStats(ctx, string(data), s.cacheTTL); err != nil {
		logger.LegacyPrintf("service.dashboard", "[Dashboard] 仪表盘缓存写入失败: %v", err)
	}
}

func (s *DashboardService) evictDashboardStatsCache(reason error) {
	if s.cache == nil {
		return
	}
	cacheCtx, cancel := s.cacheOperationContext()
	defer cancel()

	if err := s.cache.DeleteDashboardStats(cacheCtx); err != nil {
		logger.LegacyPrintf("service.dashboard", "[Dashboard] 仪表盘缓存清理失败: %v", err)
	}
	if reason != nil {
		logger.LegacyPrintf("service.dashboard", "[Dashboard] 仪表盘缓存异常，已清理: %v", reason)
	}
}

func (s *DashboardService) cacheOperationContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), s.refreshTimeout)
}

func (s *DashboardService) applyAggregationStatus(ctx context.Context, stats *usagestats.DashboardStats) {
	if stats == nil {
		return
	}
	updatedAt := s.fetchAggregationUpdatedAt(ctx)
	stats.StatsUpdatedAt = updatedAt.UTC().Format(time.RFC3339)
	stats.StatsStale = s.isAggregationStale(updatedAt, time.Now().UTC())
}

func (s *DashboardService) refreshAggregationStaleness(stats *usagestats.DashboardStats) {
	if stats == nil {
		return
	}
	updatedAt := parseStatsUpdatedAt(stats.StatsUpdatedAt)
	stats.StatsStale = s.isAggregationStale(updatedAt, time.Now().UTC())
}

func (s *DashboardService) fetchAggregationUpdatedAt(ctx context.Context) time.Time {
	if s.aggRepo == nil {
		return time.Unix(0, 0).UTC()
	}
	updatedAt, err := s.aggRepo.GetAggregationWatermark(ctx)
	if err != nil {
		logger.LegacyPrintf("service.dashboard", "[Dashboard] 读取聚合水位失败: %v", err)
		return time.Unix(0, 0).UTC()
	}
	if updatedAt.IsZero() {
		return time.Unix(0, 0).UTC()
	}
	return updatedAt.UTC()
}

func (s *DashboardService) isAggregationStale(updatedAt, now time.Time) bool {
	if !s.aggEnabled {
		return true
	}
	epoch := time.Unix(0, 0).UTC()
	if !updatedAt.After(epoch) {
		return true
	}
	threshold := s.aggInterval + s.aggLookback
	return now.Sub(updatedAt) > threshold
}

func parseStatsUpdatedAt(raw string) time.Time {
	if raw == "" {
		return time.Unix(0, 0).UTC()
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Unix(0, 0).UTC()
	}
	return parsed.UTC()
}

func (s *DashboardService) GetAPIKeyUsageTrend(ctx context.Context, startTime, endTime time.Time, granularity string, limit int) ([]usagestats.APIKeyUsageTrendPoint, error) {
	trend, err := s.usageRepo.GetAPIKeyUsageTrend(ctx, startTime, endTime, granularity, limit)
	if err != nil {
		return nil, fmt.Errorf("get api key usage trend: %w", err)
	}
	return trend, nil
}

func (s *DashboardService) GetUserUsageTrend(ctx context.Context, startTime, endTime time.Time, granularity string, limit int) ([]usagestats.UserUsageTrendPoint, error) {
	trend, err := s.usageRepo.GetUserUsageTrend(ctx, startTime, endTime, granularity, limit)
	if err != nil {
		return nil, fmt.Errorf("get user usage trend: %w", err)
	}
	return trend, nil
}

func (s *DashboardService) GetUserSpendingRanking(ctx context.Context, startTime, endTime time.Time, limit int) (*usagestats.UserSpendingRankingResponse, error) {
	ranking, err := s.usageRepo.GetUserSpendingRanking(ctx, startTime, endTime, limit)
	if err != nil {
		return nil, fmt.Errorf("get user spending ranking: %w", err)
	}
	return ranking, nil
}

func (s *DashboardService) GetUserBreakdownStats(ctx context.Context, startTime, endTime time.Time, dim usagestats.UserBreakdownDimension, limit int) ([]usagestats.UserBreakdownItem, error) {
	stats, err := s.usageRepo.GetUserBreakdownStats(ctx, startTime, endTime, dim, limit)
	if err != nil {
		return nil, fmt.Errorf("get user breakdown stats: %w", err)
	}
	return stats, nil
}

func (s *DashboardService) GetBatchUserUsageStats(ctx context.Context, userIDs []int64, startTime, endTime time.Time) (map[int64]*usagestats.BatchUserUsageStats, error) {
	stats, err := s.usageRepo.GetBatchUserUsageStats(ctx, userIDs, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("get batch user usage stats: %w", err)
	}
	return stats, nil
}

func (s *DashboardService) GetBatchAPIKeyUsageStats(ctx context.Context, apiKeyIDs []int64, startTime, endTime time.Time) (map[int64]*usagestats.BatchAPIKeyUsageStats, error) {
	stats, err := s.usageRepo.GetBatchAPIKeyUsageStats(ctx, apiKeyIDs, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("get batch api key usage stats: %w", err)
	}
	return stats, nil
}
