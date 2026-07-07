package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
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

type UpstreamBalanceConsumptionSummary struct {
	Last24h   float64 `json:"last_24h"`
	Yesterday float64 `json:"yesterday"`
	Today     float64 `json:"today"`
	Total     float64 `json:"total"`
	Unit      string  `json:"unit"`
}

type UpstreamBalanceSummary struct {
	Total       float64                           `json:"total"`
	Unit        string                            `json:"unit"`
	Consumption UpstreamBalanceConsumptionSummary `json:"consumption"`
	Items       []UpstreamBalanceAccount          `json:"items"`
}

type upstreamBalanceConsumptionRepository interface {
	RecordUpstreamBalance(ctx context.Context, item UpstreamBalanceAccount, observedAt time.Time) error
	GetUpstreamBalanceConsumptionSummary(ctx context.Context, now time.Time) (UpstreamBalanceConsumptionSummary, error)
}

type upstreamBalanceAlert struct {
	Account   UpstreamBalanceAccount
	Threshold float64
	Key       string
}

func (s *DashboardService) GetUpstreamBalances(ctx context.Context) (*UpstreamBalanceSummary, error) {
	out := &UpstreamBalanceSummary{
		Unit:        "USD",
		Consumption: UpstreamBalanceConsumptionSummary{Unit: "USD"},
		Items:       []UpstreamBalanceAccount{},
	}
	if s.accountRepo == nil || s.httpUpstream == nil {
		return out, nil
	}
	accounts, err := s.accountRepo.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	consumptionRepo, _ := s.aggRepo.(upstreamBalanceConsumptionRepository)
	observedAt := time.Now()
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
			balance = normalizeUpstreamBalance(balance, account, groupName)
			item.Balance = balance
			if unit != "" {
				item.Unit = unit
			}
			out.Total += balance
			if consumptionRepo != nil {
				if err := consumptionRepo.RecordUpstreamBalance(ctx, item, observedAt); err != nil {
					slog.Warn("record upstream balance consumption failed", "account_id", item.ID, "error", err)
				}
			}
		}
		out.Items = append(out.Items, item)
	}
	if consumptionRepo != nil {
		consumption, err := consumptionRepo.GetUpstreamBalanceConsumptionSummary(ctx, observedAt)
		if err != nil {
			slog.Warn("query upstream balance consumption failed", "error", err)
		} else {
			out.Consumption = consumption
		}
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

func normalizeUpstreamBalance(balance float64, account *Account, groupName string) float64 {
	if account == nil {
		return balance
	}
	name := strings.Join(strings.Fields(strings.ToLower(groupName+" "+account.Name)), "")
	if strings.Contains(name, "king1余额") || strings.Contains(name, "king2余额") ||
		(strings.Contains(name, "king") && (strings.Contains(name, "1余额") || strings.Contains(name, "2余额"))) {
		return balance * 0.08
	}
	return balance
}

func evaluateUpstreamBalanceAlerts(items []UpstreamBalanceAccount, sent map[string]bool) []upstreamBalanceAlert {
	thresholds := []float64{5, 2}
	alerts := make([]upstreamBalanceAlert, 0)
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		if item.Error != "" {
			continue
		}
		for _, threshold := range thresholds {
			key := fmt.Sprintf("%d:%g", item.ID, threshold)
			seen[key] = struct{}{}
			if item.Balance < threshold {
				if !sent[key] {
					alerts = append(alerts, upstreamBalanceAlert{Account: item, Threshold: threshold, Key: key})
					sent[key] = true
				}
			} else {
				delete(sent, key)
			}
		}
	}
	for key := range sent {
		if _, ok := seen[key]; !ok {
			delete(sent, key)
		}
	}
	return alerts
}

type UpstreamBalanceAlertService struct {
	dashboard  *DashboardService
	notifyURL  string
	interval   time.Duration
	httpClient *http.Client

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.Mutex
	sent   map[string]bool
}

func NewUpstreamBalanceAlertService(dashboard *DashboardService, notifyURL string, interval time.Duration) *UpstreamBalanceAlertService {
	ctx, cancel := context.WithCancel(context.Background())
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	return &UpstreamBalanceAlertService{
		dashboard:  dashboard,
		notifyURL:  strings.TrimSpace(notifyURL),
		interval:   interval,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		ctx:        ctx,
		cancel:     cancel,
		sent:       map[string]bool{},
	}
}

func ProvideUpstreamBalanceAlertService(dashboard *DashboardService) *UpstreamBalanceAlertService {
	url := strings.TrimSpace(os.Getenv("UPSTREAM_BALANCE_ALERT_URL"))
	if url == "" {
		url = strings.TrimSpace(os.Getenv("FEISHU_ALERT_BRIDGE_URL"))
	}
	svc := NewUpstreamBalanceAlertService(dashboard, url, 5*time.Minute)
	svc.Start()
	return svc
}

func (s *UpstreamBalanceAlertService) Start() {
	if s == nil || s.dashboard == nil || s.notifyURL == "" {
		return
	}
	s.wg.Add(1)
	go s.loop()
}

func (s *UpstreamBalanceAlertService) Stop() {
	if s == nil {
		return
	}
	s.cancel()
	s.wg.Wait()
}

func (s *UpstreamBalanceAlertService) loop() {
	defer s.wg.Done()
	s.checkOnce()
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.checkOnce()
		}
	}
}

func (s *UpstreamBalanceAlertService) checkOnce() {
	ctx, cancel := context.WithTimeout(s.ctx, 45*time.Second)
	defer cancel()
	summary, err := s.dashboard.GetUpstreamBalances(ctx)
	if err != nil {
		slog.Warn("upstream_balance_alert: fetch balances failed", "error", err)
		return
	}
	s.mu.Lock()
	alerts := evaluateUpstreamBalanceAlerts(summary.Items, s.sent)
	s.mu.Unlock()
	for _, alert := range alerts {
		if err := s.sendAlert(ctx, alert); err != nil {
			s.mu.Lock()
			delete(s.sent, alert.Key)
			s.mu.Unlock()
			slog.Warn("upstream_balance_alert: send alert failed", "account_id", alert.Account.ID, "threshold", alert.Threshold, "error", err)
		}
	}
}

func (s *UpstreamBalanceAlertService) sendAlert(ctx context.Context, alert upstreamBalanceAlert) error {
	payload := map[string]any{
		"msg": fmt.Sprintf("🚨 上游余额低于 %.0f：%s 当前 %.4f %s", alert.Threshold, upstreamBalanceAlertName(alert.Account), alert.Account.Balance, alert.Account.Unit),
		"monitor": map[string]any{
			"name": "Sub2API 上游余额 " + upstreamBalanceAlertName(alert.Account),
			"url":  "https://sub.sunmmyapi.xyz/admin",
		},
		"heartbeat": map[string]any{
			"status":        0,
			"msg":           fmt.Sprintf("账号ID: %d\n分组: %s\n阈值: %.0f\n当前余额: %.4f %s", alert.Account.ID, alert.Account.GroupName, alert.Threshold, alert.Account.Balance, alert.Account.Unit),
			"localDateTime": time.Now().Format("2006-01-02 15:04:05"),
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.notifyURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("notify bridge status %d", resp.StatusCode)
	}
	return nil
}

func upstreamBalanceAlertName(item UpstreamBalanceAccount) string {
	if item.GroupName == "" {
		return item.Name
	}
	if item.Name == "" {
		return item.GroupName
	}
	return item.GroupName + " / " + item.Name
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
