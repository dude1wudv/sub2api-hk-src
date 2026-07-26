package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

// UpstreamBalanceAccount is one monitored upstream account balance.
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
	RecordUpstreamBalance(context.Context, UpstreamBalanceAccount, time.Time) error
	GetUpstreamBalanceConsumptionSummary(context.Context, time.Time) (UpstreamBalanceConsumptionSummary, error)
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

func (s *DashboardService) GetUpstreamBalances(ctx context.Context) (*UpstreamBalanceSummary, error) {
	out := &UpstreamBalanceSummary{Unit: "USD", Consumption: UpstreamBalanceConsumptionSummary{Unit: "USD"}, Items: []UpstreamBalanceAccount{}}
	if s.accountRepo == nil || s.httpUpstream == nil {
		return out, nil
	}
	accounts, err := s.accountRepo.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	repo, _ := s.aggRepo.(upstreamBalanceConsumptionRepository)
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
			item.Balance = normalizeUpstreamBalance(balance, account, groupName)
			if unit != "" {
				item.Unit = unit
			}
			out.Total += item.Balance
			if repo != nil {
				if err := repo.RecordUpstreamBalance(ctx, item, observedAt); err != nil {
					slog.Warn("record upstream balance failed", "account_id", item.ID, "error", err)
				}
			}
		}
		out.Items = append(out.Items, item)
	}
	if repo != nil {
		if consumption, err := repo.GetUpstreamBalanceConsumptionSummary(ctx, observedAt); err == nil {
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
	if strings.Contains(name, "king1余额") || strings.Contains(name, "king2余额") {
		return balance * 0.08
	}
	return balance
}

func (s *DashboardService) fetchUpstreamBalance(ctx context.Context, account *Account) (float64, string, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(account.GetCredential("base_url")), "/")
	apiKey := account.GetCredential("api_key")
	if baseURL == "" || apiKey == "" {
		return 0, "", errors.New("missing base_url or api_key")
	}
	callCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	paths := []string{"/v1/usage", "/dashboard/billing/credit_grants", "/billing/credit_grants"}
	requestURLs := make([]string, 0, len(paths)+1)
	for _, path := range paths {
		requestURLs = append(requestURLs, baseURL+path)
	}
	if openAIBaseURLHasVersionSuffix(baseURL) {
		requestURLs = append(requestURLs, buildOpenAIEndpointURL(baseURL, "/v1/usage"))
	}
	for _, requestURL := range requestURLs {
		req, err := http.NewRequestWithContext(callCtx, http.MethodGet, requestURL, nil)
		if err != nil {
			return 0, "", err
		}
		req.Header.Set("Authorization", "Bearer "+apiKey)
		req.Header.Set("Accept", "application/json")
		resp, err := s.httpUpstream.Do(req, dashboardAccountProxyURL(account), account.ID, maxInt(account.Concurrency, 1))
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
		if balance, unit, ok := parseBalancePayload(body); ok {
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
	if json.Unmarshal(body, &payload) != nil {
		return 0, "", false
	}
	unit := findBalanceString(payload, "unit")
	for _, key := range []string{"total_available", "current_balance", "balance", "remaining", "available"} {
		if value, ok := findBalanceNumber(payload, key); ok {
			return value, unit, true
		}
	}
	return 0, unit, false
}

func findBalanceNumber(value any, key string) (float64, bool) {
	switch item := value.(type) {
	case map[string]any:
		if raw, ok := item[key]; ok {
			switch number := raw.(type) {
			case float64:
				return number, true
			case string:
				parsed, err := strconv.ParseFloat(strings.TrimSpace(number), 64)
				return parsed, err == nil
			}
		}
		for _, raw := range item {
			if number, ok := findBalanceNumber(raw, key); ok {
				return number, true
			}
		}
	case []any:
		for _, raw := range item {
			if number, ok := findBalanceNumber(raw, key); ok {
				return number, true
			}
		}
	}
	return 0, false
}

func findBalanceString(value any, key string) string {
	switch item := value.(type) {
	case map[string]any:
		if raw, ok := item[key].(string); ok {
			return raw
		}
		for _, raw := range item {
			if text := findBalanceString(raw, key); text != "" {
				return text
			}
		}
	case []any:
		for _, raw := range item {
			if text := findBalanceString(raw, key); text != "" {
				return text
			}
		}
	}
	return ""
}
