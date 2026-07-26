package service

import "testing"

func TestParseBalancePayload(t *testing.T) {
	balance, unit, ok := parseBalancePayload([]byte(`{"data":{"current_balance":"12.34","unit":"USD"}}`))
	if !ok || balance != 12.34 || unit != "USD" {
		t.Fatalf("unexpected balance payload: %v %q %v", balance, unit, ok)
	}
}

func TestNormalizeUpstreamBalanceKingGroups(t *testing.T) {
	for _, group := range []string{"king 1 余额", "king 2 余额"} {
		if got := normalizeUpstreamBalance(100, &Account{Name: "king 余额"}, group); got != 8 {
			t.Fatalf("unexpected normalized balance for %q: %v", group, got)
		}
	}
}
