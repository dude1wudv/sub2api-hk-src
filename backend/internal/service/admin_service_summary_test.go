package service

import "testing"

func TestBuildAccountSummaryCodexWindowsCountAllOpenAIOAuthAccounts(t *testing.T) {
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

	if summary.Codex5h.Sampled != 2 {
		t.Fatalf("Codex5h.Sampled = %d, want 2", summary.Codex5h.Sampled)
	}
	if summary.Codex5h.UsedPercent == nil || *summary.Codex5h.UsedPercent != 17.5 {
		t.Fatalf("Codex5h.UsedPercent = %v, want 17.5", summary.Codex5h.UsedPercent)
	}
	if summary.Codex5h.RemainingPercent == nil || *summary.Codex5h.RemainingPercent != 82.5 {
		t.Fatalf("Codex5h.RemainingPercent = %v, want 82.5", summary.Codex5h.RemainingPercent)
	}
	if summary.Codex7d.Sampled != 2 {
		t.Fatalf("Codex7d.Sampled = %d, want 2", summary.Codex7d.Sampled)
	}
}

func TestBuildAccountSummaryCodexWindowsCountOpenAIOAuthAccountsOnly(t *testing.T) {
	proxyID := int64(12)
	accounts := []Account{
		{
			Platform:    PlatformOpenAI,
			Type:        AccountTypeOAuth,
			Credentials: map[string]any{"plan_type": "free"},
			Extra: map[string]any{
				"codex_5h_used_percent": 10.0,
				"codex_7d_used_percent": 80.0,
			},
			ProxyID: &proxyID,
		},
		{
			Platform:    PlatformOpenAI,
			Type:        AccountTypeOAuth,
			Credentials: map[string]any{"plan_type": "plus"},
			Extra: map[string]any{
				"codex_5h_used_percent": 40.0,
				"codex_7d_used_percent": 20.0,
			},
			ProxyID: &proxyID,
		},
		{
			Platform:    PlatformOpenAI,
			Type:        AccountTypeOAuth,
			Credentials: map[string]any{},
			Extra: map[string]any{
				"codex_5h_used_percent": 90.0,
				"codex_7d_used_percent": 90.0,
			},
			ProxyID: &proxyID,
		},
		{
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Credentials: map[string]any{"plan_type": "pro"},
			Extra: map[string]any{
				"codex_5h_used_percent": 5.0,
				"codex_7d_used_percent": 5.0,
			},
			ProxyID: &proxyID,
		},
	}

	summary := buildAccountSummary(accounts)

	if summary.Codex5h.Sampled != 3 {
		t.Fatalf("Codex5h.Sampled = %d, want 3", summary.Codex5h.Sampled)
	}
	if summary.Codex5h.UsedPercent == nil || *summary.Codex5h.UsedPercent != 46.7 {
		t.Fatalf("Codex5h.UsedPercent = %v, want 46.7", summary.Codex5h.UsedPercent)
	}
	if summary.Codex5h.RemainingPercent == nil || *summary.Codex5h.RemainingPercent != 53.3 {
		t.Fatalf("Codex5h.RemainingPercent = %v, want 53.3", summary.Codex5h.RemainingPercent)
	}
	if summary.Codex7d.Sampled != 3 {
		t.Fatalf("Codex7d.Sampled = %d, want 3", summary.Codex7d.Sampled)
	}
	if summary.Codex7d.UsedPercent == nil || *summary.Codex7d.UsedPercent != 63.3 {
		t.Fatalf("Codex7d.UsedPercent = %v, want 63.3", summary.Codex7d.UsedPercent)
	}
	if len(summary.ProxyDistribution) != 1 {
		t.Fatalf("ProxyDistribution len = %d, want 1", len(summary.ProxyDistribution))
	}
	if got := summary.ProxyDistribution[0].Remaining5hPercent; got == nil || *got != 53.3 {
		t.Fatalf("proxy Remaining5hPercent = %v, want 53.3", got)
	}
	if got := summary.ProxyDistribution[0].Remaining7dPercent; got == nil || *got != 36.7 {
		t.Fatalf("proxy Remaining7dPercent = %v, want 36.7", got)
	}
}

func TestBuildAccountSummaryQuotaPoolsUseAllOpenAIOAuthAndIgnoreAPIKeys(t *testing.T) {
	accounts := []Account{
		{
			Platform:    PlatformOpenAI,
			Type:        AccountTypeOAuth,
			Credentials: map[string]any{"plan_type": "plus"},
			Status:      StatusActive,
			Schedulable: true,
			Extra: map[string]any{
				"codex_5h_used_percent": 25.0,
				"codex_7d_used_percent": 90.0,
			},
		},
		{
			Platform:    PlatformOpenAI,
			Type:        AccountTypeOAuth,
			Credentials: map[string]any{"plan_type": "pro"},
			Status:      StatusActive,
			Schedulable: true,
			Extra: map[string]any{
				"codex_5h_used_percent": 35.0,
				"codex_7d_used_percent": 55.0,
			},
		},
		{
			Platform:    PlatformOpenAI,
			Type:        AccountTypeOAuth,
			Credentials: map[string]any{"plan_type": ""},
			Status:      StatusActive,
			Schedulable: true,
			Groups:      []*Group{{Name: "Plus"}},
			Extra: map[string]any{
				"codex_5h_used_percent": 45.0,
				"codex_7d_used_percent": 30.0,
			},
		},
		{
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Credentials: map[string]any{"plan_type": "pro"},
			Status:      StatusActive,
			Schedulable: true,
			Groups:      []*Group{{Name: "Plus"}},
			Extra: map[string]any{
				"codex_5h_used_percent": 5.0,
				"codex_7d_used_percent": 5.0,
			},
		},
		{
			Platform:    PlatformOpenAI,
			Type:        AccountTypeOAuth,
			Credentials: map[string]any{"plan_type": "free"},
			Status:      StatusActive,
			Schedulable: true,
			Extra: map[string]any{
				"codex_5h_used_percent": 10.0,
				"codex_7d_used_percent": 60.0,
			},
		},
	}

	summary := buildAccountSummary(accounts)

	if summary.OAuthPool.Total != 4 || summary.OAuthPool.Sampled != 4 {
		t.Fatalf("OAuthPool totals = (%d,%d), want (4,4)", summary.OAuthPool.Total, summary.OAuthPool.Sampled)
	}
	if summary.OAuthPool.RemainingPercent == nil || *summary.OAuthPool.RemainingPercent != 71.2 {
		t.Fatalf("OAuthPool.RemainingPercent = %v, want 71.2", summary.OAuthPool.RemainingPercent)
	}
	if summary.OAuthPool.Remaining5hPercent == nil || *summary.OAuthPool.Remaining5hPercent != 71.2 {
		t.Fatalf("OAuthPool.Remaining5hPercent = %v, want 71.2", summary.OAuthPool.Remaining5hPercent)
	}
	if summary.OAuthPool.Remaining7dPercent == nil || *summary.OAuthPool.Remaining7dPercent != 41.2 {
		t.Fatalf("OAuthPool.Remaining7dPercent = %v, want 41.2", summary.OAuthPool.Remaining7dPercent)
	}
	if summary.FreePool.Total != 0 || summary.FreePool.Sampled != 0 {
		t.Fatalf("FreePool totals = (%d,%d), want (0,0)", summary.FreePool.Total, summary.FreePool.Sampled)
	}
	if summary.FreePool.RemainingPercent != nil {
		t.Fatalf("FreePool.RemainingPercent = %v, want nil", summary.FreePool.RemainingPercent)
	}
}
