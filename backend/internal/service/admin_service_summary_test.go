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

func TestBuildAccountSummaryGrokPoolAggregatesShortAndWeeklyWindows(t *testing.T) {
	accounts := []Account{
		{
			Platform:    PlatformGrok,
			Type:        AccountTypeOAuth,
			Status:      StatusActive,
			Schedulable: true,
			Extra: map[string]any{
				"grok_usage_snapshot": map[string]any{
					"requests": map[string]any{
						"limit":     int64(100),
						"remaining": int64(40),
					},
				},
				"grok_billing_snapshot": map[string]any{
					"state":       "observed",
					"utilization": 20.0,
				},
			},
		},
		{
			Platform:    PlatformGrok,
			Type:        AccountTypeOAuth,
			Status:      StatusActive,
			Schedulable: true,
			Extra: map[string]any{
				// No requests window — fall back to tokens.
				"grok_usage_snapshot": map[string]any{
					"tokens": map[string]any{
						"limit":     int64(200),
						"remaining": int64(50),
					},
				},
				"grok_billing_snapshot": map[string]any{
					"state":       "observed",
					"utilization": 80.0,
				},
			},
		},
		{
			Platform:    PlatformGrok,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Extra: map[string]any{
				"grok_usage_snapshot": map[string]any{
					"requests": map[string]any{
						"limit":     int64(100),
						"remaining": int64(0),
					},
				},
				"grok_billing_snapshot": map[string]any{
					"state":       "observed",
					"utilization": 99.0,
				},
			},
		},
		{
			Platform:    PlatformOpenAI,
			Type:        AccountTypeOAuth,
			Status:      StatusActive,
			Schedulable: true,
			Extra: map[string]any{
				"codex_5h_used_percent": 10.0,
				"codex_7d_used_percent": 10.0,
			},
		},
	}

	summary := buildAccountSummary(accounts)

	if summary.GrokPool.Total != 2 || summary.GrokPool.Available != 2 {
		t.Fatalf("GrokPool totals = (%d,%d), want (2,2)", summary.GrokPool.Total, summary.GrokPool.Available)
	}
	// Primary meter prefers weekly: used (20 + 80) / 2 = 50 → remaining 50
	if summary.GrokPool.Sampled != 2 {
		t.Fatalf("GrokPool.Sampled = %d, want 2", summary.GrokPool.Sampled)
	}
	if summary.GrokPool.RemainingPercent == nil || *summary.GrokPool.RemainingPercent != 50 {
		t.Fatalf("GrokPool.RemainingPercent = %v, want 50", summary.GrokPool.RemainingPercent)
	}
	// Short-window used: (60 + 75) / 2 = 67.5 → remaining 32.5
	if summary.GrokPool.Remaining5hPercent == nil || *summary.GrokPool.Remaining5hPercent != 32.5 {
		t.Fatalf("GrokPool.Remaining5hPercent = %v, want 32.5", summary.GrokPool.Remaining5hPercent)
	}
	if summary.GrokPool.Remaining7dPercent == nil || *summary.GrokPool.Remaining7dPercent != 50 {
		t.Fatalf("GrokPool.Remaining7dPercent = %v, want 50", summary.GrokPool.Remaining7dPercent)
	}
	if summary.GrokPool.Exhausted != 0 {
		t.Fatalf("GrokPool.Exhausted = %d, want 0", summary.GrokPool.Exhausted)
	}
}
