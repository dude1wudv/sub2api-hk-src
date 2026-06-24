package service

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestCollectSelectionFailureStats(t *testing.T) {
	svc := &GatewayService{}
	model := "gpt-5.4"
	resetAt := time.Now().Add(2 * time.Minute).Format(time.RFC3339)

	accounts := []Account{
		// excluded
		{
			ID:          1,
			Platform:    PlatformOpenAI,
			Status:      StatusActive,
			Schedulable: true,
		},
		// unschedulable
		{
			ID:          2,
			Platform:    PlatformOpenAI,
			Status:      StatusActive,
			Schedulable: false,
		},
		// platform filtered
		{
			ID:          3,
			Platform:    PlatformAntigravity,
			Status:      StatusActive,
			Schedulable: true,
		},
		// model unsupported
		{
			ID:          4,
			Platform:    PlatformOpenAI,
			Status:      StatusActive,
			Schedulable: true,
			Credentials: map[string]any{
				"model_mapping": map[string]any{
					"gpt-image": "gpt-image",
				},
			},
		},
		// model rate limited
		{
			ID:          5,
			Platform:    PlatformOpenAI,
			Status:      StatusActive,
			Schedulable: true,
			Extra: map[string]any{
				"model_rate_limits": map[string]any{
					model: map[string]any{
						"rate_limit_reset_at": resetAt,
					},
				},
			},
		},
		// eligible
		{
			ID:          6,
			Platform:    PlatformOpenAI,
			Status:      StatusActive,
			Schedulable: true,
		},
	}

	excluded := map[int64]struct{}{1: {}}
	stats := svc.collectSelectionFailureStats(context.Background(), accounts, model, PlatformOpenAI, excluded, false)

	if stats.Total != 6 {
		t.Fatalf("total=%d want=6", stats.Total)
	}
	if stats.Excluded != 1 {
		t.Fatalf("excluded=%d want=1", stats.Excluded)
	}
	if stats.Unschedulable != 1 {
		t.Fatalf("unschedulable=%d want=1", stats.Unschedulable)
	}
	if stats.PlatformFiltered != 1 {
		t.Fatalf("platform_filtered=%d want=1", stats.PlatformFiltered)
	}
	if stats.ModelUnsupported != 1 {
		t.Fatalf("model_unsupported=%d want=1", stats.ModelUnsupported)
	}
	if stats.ModelRateLimited != 1 {
		t.Fatalf("model_rate_limited=%d want=1", stats.ModelRateLimited)
	}
	if stats.Eligible != 1 {
		t.Fatalf("eligible=%d want=1", stats.Eligible)
	}
}

func TestUpstreamFailoverErrorErrorIncludesSafeReason(t *testing.T) {
	err := &UpstreamFailoverError{StatusCode: 429, Reason: "rate limit exceeded"}

	got := err.Error()

	if !strings.Contains(got, "429") {
		t.Fatalf("expected status code in error, got %q", got)
	}
	if !strings.Contains(got, "rate limit exceeded") {
		t.Fatalf("expected failover reason in error, got %q", got)
	}
}

func TestUpstreamFailoverErrorErrorOmitsEmptyReason(t *testing.T) {
	err := &UpstreamFailoverError{StatusCode: 502}

	got := err.Error()

	if got != "upstream error: 502 (failover)" {
		t.Fatalf("unexpected error string: %q", got)
	}
}

func TestUpstreamFailoverErrorErrorRedactsCredentialLikeReason(t *testing.T) {
	err := &UpstreamFailoverError{
		StatusCode: 502,
		Reason:     `proxy http://user:proxy-pass@example.test failed; Authorization: Bearer sk-secret; api_key=key-secret; password=db-secret; {"api_key":"json-key","access_token":"json-token","password":"json-pass"}`,
	}

	got := err.Error()

	for _, leaked := range []string{"proxy-pass", "sk-secret", "key-secret", "db-secret", "json-key", "json-token", "json-pass"} {
		if strings.Contains(got, leaked) {
			t.Fatalf("expected %q to be redacted from %q", leaked, got)
		}
	}
	if !strings.Contains(got, "[redacted]") {
		t.Fatalf("expected redaction marker in %q", got)
	}
}

func TestGatewayService_FailoverPolicy_ClientCredentialStatusesSwitchAccountWithoutSameAccountRetry(t *testing.T) {
	svc := &GatewayService{}
	account := &Account{Platform: PlatformAnthropic, Type: AccountTypeAPIKey}

	for _, status := range []int{402, 404} {
		if !svc.shouldFailoverUpstreamError(status) {
			t.Fatalf("%d should trigger account failover", status)
		}
		if svc.shouldRetryUpstreamError(account, status) {
			t.Fatalf("%d should not retry on the same API-key account", status)
		}
	}
}

func TestDiagnoseSelectionFailure_UnschedulableDetail(t *testing.T) {
	svc := &GatewayService{}
	acc := &Account{
		ID:          7,
		Platform:    PlatformOpenAI,
		Status:      StatusActive,
		Schedulable: false,
	}

	diagnosis := svc.diagnoseSelectionFailure(context.Background(), acc, "gpt-5.4", PlatformOpenAI, map[int64]struct{}{}, false)
	if diagnosis.Category != "unschedulable" {
		t.Fatalf("category=%s want=unschedulable", diagnosis.Category)
	}
	if diagnosis.Detail != "generic_unschedulable" {
		t.Fatalf("detail=%s want=generic_unschedulable", diagnosis.Detail)
	}
}

func TestDiagnoseSelectionFailure_ModelRateLimitedDetail(t *testing.T) {
	svc := &GatewayService{}
	model := "gpt-5.4"
	resetAt := time.Now().Add(2 * time.Minute).UTC().Format(time.RFC3339)
	acc := &Account{
		ID:          8,
		Platform:    PlatformOpenAI,
		Status:      StatusActive,
		Schedulable: true,
		Extra: map[string]any{
			"model_rate_limits": map[string]any{
				model: map[string]any{
					"rate_limit_reset_at": resetAt,
				},
			},
		},
	}

	diagnosis := svc.diagnoseSelectionFailure(context.Background(), acc, model, PlatformOpenAI, map[int64]struct{}{}, false)
	if diagnosis.Category != "model_rate_limited" {
		t.Fatalf("category=%s want=model_rate_limited", diagnosis.Category)
	}
	if !strings.Contains(diagnosis.Detail, "remaining=") {
		t.Fatalf("detail=%s want contains remaining=", diagnosis.Detail)
	}
}
