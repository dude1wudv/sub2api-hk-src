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
