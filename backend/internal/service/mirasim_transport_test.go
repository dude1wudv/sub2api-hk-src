package service

import (
	"net/http"
	"strings"
	"testing"
)

func TestMirasimSignerRejectsNonOfficialRelayBeforeReadingCredentials(t *testing.T) {
	svc := &OpenAIGatewayService{}
	account := &Account{ID: 1, Platform: PlatformMirasim, Credentials: map[string]any{
		"refresh_token": "secret-refresh-token",
		"private_key":   "secret-private-key",
	}}
	for _, target := range []string{
		"https://relay.mirasim.ai.evil.example/v1/messages",
		"http://relay.mirasim.ai/v1/messages",
		"https://relay.mirasim.ai/v1/models",
		"https://relay.mirasim.ai/v1/messages?redirect=https://evil.example",
	} {
		req, err := http.NewRequest(http.MethodPost, target, strings.NewReader(`{"model":"claude-sonnet-5"}`))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := svc.signMirasimRequest(req, "", account); err == nil || strings.Contains(err.Error(), "secret-") {
			t.Fatalf("target %s was not safely rejected: %v", target, err)
		}
	}
}
