package service

import (
	"net/http"
	"strings"
	"testing"
)

func TestSafeUpstreamClientErrorDoesNotExposeUpstream(t *testing.T) {
	for _, status := range []int{http.StatusPaymentRequired, http.StatusForbidden, http.StatusTooManyRequests, http.StatusBadGateway, 529} {
		safe := SafeUpstreamClientError(status)
		msg := safe.MessageWithCode()
		if safe.Code == "" || safe.Message == "" || msg == "" {
			t.Fatalf("status %d returned empty safe error: %#v", status, safe)
		}
		for _, banned := range []string{"Upstream", "upstream", "balance", "group"} {
			if strings.Contains(msg, banned) {
				t.Fatalf("status %d leaked %q in %q", status, banned, msg)
			}
		}
	}
}

func TestRedactUpstreamClientCodeUsesSiteCode(t *testing.T) {
	status, code, message := RedactUpstreamClientCode(http.StatusBadGateway, "server_error", "All available accounts exhausted")
	if status != http.StatusBadGateway || code != "SUNM_SERVICE_UNAVAILABLE" {
		t.Fatalf("unexpected redaction: status=%d code=%q message=%q", status, code, message)
	}
	if strings.Contains(message, "Upstream") || strings.Contains(message, "accounts exhausted") {
		t.Fatalf("redacted message leaked internals: %q", message)
	}
}
