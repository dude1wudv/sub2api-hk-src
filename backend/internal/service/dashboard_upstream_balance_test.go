package service

import "testing"

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
