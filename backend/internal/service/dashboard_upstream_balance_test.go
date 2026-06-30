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
