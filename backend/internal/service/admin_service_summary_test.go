package service

import "testing"

func TestBuildAccountSummaryCodex5hSkipsFreeAccounts(t *testing.T) {
	accounts := []Account{
		{
			Platform:    PlatformOpenAI,
			Type:        AccountTypeOAuth,
			Credentials: map[string]any{"plan_type": "free"},
			Extra: map[string]any{
				"codex_5h_used_percent": 0.0,
				"codex_7d_used_percent": 20.0,
			},
		},
		{
			Platform:    PlatformOpenAI,
			Type:        AccountTypeOAuth,
			Credentials: map[string]any{"plan_type": "free"},
			Extra: map[string]any{
				"codex_5h_used_percent": 35.0,
				"codex_7d_used_percent": 80.0,
			},
		},
	}

	summary := buildAccountSummary(accounts)

	if summary.Codex5h.Sampled != 0 {
		t.Fatalf("Codex5h.Sampled = %d, want 0", summary.Codex5h.Sampled)
	}
	if summary.Codex5h.UsedPercent != nil {
		t.Fatalf("Codex5h.UsedPercent = %v, want nil", *summary.Codex5h.UsedPercent)
	}
	if summary.Codex5h.RemainingPercent != nil {
		t.Fatalf("Codex5h.RemainingPercent = %v, want nil", *summary.Codex5h.RemainingPercent)
	}
	if summary.Codex7d.Sampled != 2 {
		t.Fatalf("Codex7d.Sampled = %d, want 2", summary.Codex7d.Sampled)
	}
}

func TestBuildAccountSummaryCodex5hCountsPaidOpenAIAccountsOnly(t *testing.T) {
	proxyID := int64(12)
	accounts := []Account{
		{
			Platform:    PlatformOpenAI,
			Type:        AccountTypeOAuth,
			Credentials: map[string]any{"plan_type": "free"},
			Extra:       map[string]any{"codex_5h_used_percent": 10.0},
			ProxyID:     &proxyID,
		},
		{
			Platform:    PlatformOpenAI,
			Type:        AccountTypeOAuth,
			Credentials: map[string]any{"plan_type": "plus"},
			Extra:       map[string]any{"codex_5h_used_percent": 40.0},
			ProxyID:     &proxyID,
		},
		{
			Platform:    PlatformOpenAI,
			Type:        AccountTypeOAuth,
			Credentials: map[string]any{},
			Extra:       map[string]any{"codex_5h_used_percent": 90.0},
			ProxyID:     &proxyID,
		},
	}

	summary := buildAccountSummary(accounts)

	if summary.Codex5h.Sampled != 1 {
		t.Fatalf("Codex5h.Sampled = %d, want 1", summary.Codex5h.Sampled)
	}
	if summary.Codex5h.UsedPercent == nil || *summary.Codex5h.UsedPercent != 40.0 {
		t.Fatalf("Codex5h.UsedPercent = %v, want 40", summary.Codex5h.UsedPercent)
	}
	if summary.Codex5h.RemainingPercent == nil || *summary.Codex5h.RemainingPercent != 60.0 {
		t.Fatalf("Codex5h.RemainingPercent = %v, want 60", summary.Codex5h.RemainingPercent)
	}
	if len(summary.ProxyDistribution) != 1 {
		t.Fatalf("ProxyDistribution len = %d, want 1", len(summary.ProxyDistribution))
	}
	if got := summary.ProxyDistribution[0].Remaining5hPercent; got == nil || *got != 60.0 {
		t.Fatalf("proxy Remaining5hPercent = %v, want 60", got)
	}
}
