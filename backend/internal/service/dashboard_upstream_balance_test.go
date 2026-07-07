package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
)

func TestParseBalancePayload(t *testing.T) {
	balance, unit, ok := parseBalancePayload([]byte(`{"data":{"current_balance":"12.34","unit":"USD"}}`))
	if !ok {
		t.Fatal("expected balance")
	}
	if balance != 12.34 || unit != "USD" {
		t.Fatalf("unexpected balance payload: %v %q", balance, unit)
	}
}

func TestNormalizeUpstreamBalanceKing2(t *testing.T) {
	balance := normalizeUpstreamBalance(100, &Account{Name: "king 余额"}, "king 2 余额")
	if balance != 8 {
		t.Fatalf("unexpected normalized balance: %v", balance)
	}
}

func TestNormalizeUpstreamBalanceKing1(t *testing.T) {
	balance := normalizeUpstreamBalance(100, &Account{Name: "king 余额"}, "king 1 余额")
	if balance != 8 {
		t.Fatalf("unexpected normalized balance: %v", balance)
	}
}

func TestEvaluateUpstreamBalanceAlerts(t *testing.T) {
	state := map[string]bool{}
	item := UpstreamBalanceAccount{ID: 42, Name: "otok余额", Balance: 4.9}

	alerts := evaluateUpstreamBalanceAlerts([]UpstreamBalanceAccount{item}, state)
	if len(alerts) != 1 || alerts[0].Threshold != 5 {
		t.Fatalf("expected only threshold 5 alert, got %#v", alerts)
	}
	if !state["42:5"] || state["42:2"] {
		t.Fatalf("unexpected alert state: %#v", state)
	}

	item.Balance = 1.9
	alerts = evaluateUpstreamBalanceAlerts([]UpstreamBalanceAccount{item}, state)
	if len(alerts) != 1 || alerts[0].Threshold != 2 {
		t.Fatalf("expected threshold 2 alert after 5 was sent, got %#v", alerts)
	}
	if !state["42:5"] || !state["42:2"] {
		t.Fatalf("unexpected alert state after threshold 2: %#v", state)
	}

	alerts = evaluateUpstreamBalanceAlerts([]UpstreamBalanceAccount{item}, state)
	if len(alerts) != 0 {
		t.Fatalf("expected no duplicate alerts, got %#v", alerts)
	}

	item.Balance = 5.1
	alerts = evaluateUpstreamBalanceAlerts([]UpstreamBalanceAccount{item}, state)
	if len(alerts) != 0 || len(state) != 0 {
		t.Fatalf("expected recovery to clear alert state, alerts=%#v state=%#v", alerts, state)
	}
}

type upstreamBalanceAccountRepoStub struct {
	AccountRepository
	accounts []Account
}

func (s upstreamBalanceAccountRepoStub) ListActive(ctx context.Context) ([]Account, error) {
	return s.accounts, nil
}

type upstreamBalanceHTTPStub struct{}

func (upstreamBalanceHTTPStub) Do(req *http.Request, proxyURL string, accountID int64, accountConcurrency int) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"balance":12.5,"unit":"USD"}`)),
	}, nil
}

func (u upstreamBalanceHTTPStub) DoWithTLS(req *http.Request, proxyURL string, accountID int64, accountConcurrency int, profile *tlsfingerprint.Profile) (*http.Response, error) {
	return u.Do(req, proxyURL, accountID, accountConcurrency)
}

type upstreamBalanceConsumptionRepoStub struct {
	dashboardAggregationRepoStub
	recorded []UpstreamBalanceAccount
	summary  UpstreamBalanceConsumptionSummary
}

func (s *upstreamBalanceConsumptionRepoStub) RecordUpstreamBalance(ctx context.Context, item UpstreamBalanceAccount, observedAt time.Time) error {
	s.recorded = append(s.recorded, item)
	return nil
}

func (s *upstreamBalanceConsumptionRepoStub) GetUpstreamBalanceConsumptionSummary(ctx context.Context, now time.Time) (UpstreamBalanceConsumptionSummary, error) {
	return s.summary, nil
}

func TestGetUpstreamBalancesRecordsConsumptionAndReturnsSummary(t *testing.T) {
	agg := &upstreamBalanceConsumptionRepoStub{
		summary: UpstreamBalanceConsumptionSummary{Last24h: 3, Yesterday: 2, Today: 1, Total: 6, Unit: "USD"},
	}
	svc := NewDashboardService(nil, agg, nil, nil)
	svc.SetUpstreamBalanceDeps(upstreamBalanceAccountRepoStub{accounts: []Account{{
		ID:          7,
		Name:        "provider",
		Credentials: map[string]any{"base_url": "https://example.test", "api_key": "sk-test"},
		Groups:      []*Group{{ID: 9, Name: "余额组"}},
	}}}, upstreamBalanceHTTPStub{})

	summary, err := svc.GetUpstreamBalances(context.Background())
	if err != nil {
		t.Fatalf("GetUpstreamBalances error: %v", err)
	}
	if len(agg.recorded) != 1 || agg.recorded[0].Balance != 12.5 {
		t.Fatalf("expected one recorded balance, got %#v", agg.recorded)
	}
	if summary.Consumption.Total != 6 || summary.Consumption.Today != 1 {
		t.Fatalf("unexpected consumption summary: %#v", summary.Consumption)
	}
}
