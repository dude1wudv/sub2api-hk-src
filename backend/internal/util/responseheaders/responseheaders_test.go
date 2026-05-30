package responseheaders

import (
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

func TestFilterHeadersDisabledUsesDefaultAllowlist(t *testing.T) {
	src := http.Header{}
	src.Add("Content-Type", "application/json")
	src.Add("X-Request-Id", "req-123")
	src.Add("X-Test", "ok")
	src.Add("Connection", "keep-alive")
	src.Add("Content-Length", "123")

	cfg := config.ResponseHeaderConfig{
		Enabled:     false,
		ForceRemove: []string{"x-request-id"},
	}

	filtered := FilterHeaders(src, CompileHeaderFilter(cfg))
	if filtered.Get("Content-Type") != "application/json" {
		t.Fatalf("expected Content-Type passthrough, got %q", filtered.Get("Content-Type"))
	}
	if filtered.Get("X-Request-Id") != "req-123" {
		t.Fatalf("expected X-Request-Id allowed, got %q", filtered.Get("X-Request-Id"))
	}
	if filtered.Get("X-Test") != "" {
		t.Fatalf("expected X-Test removed, got %q", filtered.Get("X-Test"))
	}
	if filtered.Get("Connection") != "" {
		t.Fatalf("expected Connection to be removed, got %q", filtered.Get("Connection"))
	}
	if filtered.Get("Content-Length") != "" {
		t.Fatalf("expected Content-Length to be removed, got %q", filtered.Get("Content-Length"))
	}
}

func TestFilterHeadersEnabledUsesAllowlist(t *testing.T) {
	src := http.Header{}
	src.Add("Content-Type", "application/json")
	src.Add("X-Extra", "ok")
	src.Add("X-Remove", "nope")
	src.Add("X-Blocked", "nope")

	cfg := config.ResponseHeaderConfig{
		Enabled:           true,
		AdditionalAllowed: []string{"x-extra"},
		ForceRemove:       []string{"x-remove"},
	}

	filtered := FilterHeaders(src, CompileHeaderFilter(cfg))
	if filtered.Get("Content-Type") != "application/json" {
		t.Fatalf("expected Content-Type allowed, got %q", filtered.Get("Content-Type"))
	}
	if filtered.Get("X-Extra") != "ok" {
		t.Fatalf("expected X-Extra allowed, got %q", filtered.Get("X-Extra"))
	}
	if filtered.Get("X-Remove") != "" {
		t.Fatalf("expected X-Remove removed, got %q", filtered.Get("X-Remove"))
	}
	if filtered.Get("X-Blocked") != "" {
		t.Fatalf("expected X-Blocked removed, got %q", filtered.Get("X-Blocked"))
	}
}

func TestFilterHeadersAlwaysRemovesUserVisibleQuotaSignals(t *testing.T) {
	src := http.Header{}
	src.Add("Content-Type", "text/event-stream")
	src.Add("Anthropic-Ratelimit-Unified-5h-Status", "allowed_warning")
	src.Add("X-Codex-Primary-Used-Percent", "80")
	src.Add("X-Codex-Secondary-Reset-After-Seconds", "120")
	src.Add("X-Codex-Turn-State", "keep")

	cfg := config.ResponseHeaderConfig{
		Enabled: true,
		AdditionalAllowed: []string{
			"anthropic-ratelimit-unified-5h-status",
			"x-codex-primary-used-percent",
			"x-codex-secondary-reset-after-seconds",
			"x-codex-turn-state",
		},
	}

	filtered := FilterHeaders(src, CompileHeaderFilter(cfg))
	if filtered.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("expected Content-Type allowed, got %q", filtered.Get("Content-Type"))
	}
	if filtered.Get("Anthropic-Ratelimit-Unified-5h-Status") != "" {
		t.Fatalf("expected Anthropic quota header removed, got %q", filtered.Get("Anthropic-Ratelimit-Unified-5h-Status"))
	}
	if filtered.Get("X-Codex-Primary-Used-Percent") != "" {
		t.Fatalf("expected X-Codex primary quota header removed, got %q", filtered.Get("X-Codex-Primary-Used-Percent"))
	}
	if filtered.Get("X-Codex-Secondary-Reset-After-Seconds") != "" {
		t.Fatalf("expected X-Codex secondary quota header removed, got %q", filtered.Get("X-Codex-Secondary-Reset-After-Seconds"))
	}
	if filtered.Get("X-Codex-Turn-State") != "keep" {
		t.Fatalf("expected non-quota x-codex header allowed, got %q", filtered.Get("X-Codex-Turn-State"))
	}
}
