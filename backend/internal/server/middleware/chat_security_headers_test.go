package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
)

func TestChatFramingOnlyAllowsChatAssetNamespace(t *testing.T) {
	for _, path := range []string{"/chat-app", "/chat-app/", "/chat-app/static/a.js", "/chat", "/chat-models", "/dashboard", "/chat-app-other", "/api/v1/chat/presets", "/v1/models"} {
		t.Run(path, func(t *testing.T) {
			router := gin.New()
			router.Use(SecurityHeaders(config.CSPConfig{Enabled: true, Policy: "default-src 'self'; frame-ancestors 'none'; connect-src 'self'; FRAME-ANCESTORS https://example.com"}, nil))
			router.NoRoute(func(c *gin.Context) { c.Status(http.StatusOK) })
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
			want := "DENY"
			isChat := path == "/chat-app" || strings.HasPrefix(path, "/chat-app/")
			if isChat {
				want = "SAMEORIGIN"
			}
			if w.Header().Get("X-Frame-Options") != want {
				t.Fatalf("unexpected framing policy: %s", w.Header().Get("X-Frame-Options"))
			}
			policy := w.Header().Get("Content-Security-Policy")
			if isChat && (!strings.Contains(policy, "frame-ancestors 'self'") || strings.Count(strings.ToLower(policy), "frame-ancestors") != 1) {
				t.Fatalf("chat CSP must have one self ancestor directive: %s", policy)
			}
			if !isChat && path != "/v1/models" && !strings.Contains(policy, "frame-ancestors 'none'") {
				t.Fatalf("non-chat ancestor policy changed: %s", policy)
			}
			if w.Header().Get("X-Content-Type-Options") != "nosniff" || w.Header().Get("Referrer-Policy") != "strict-origin-when-cross-origin" {
				t.Fatal("unrelated security headers changed")
			}
		})
	}
}

func TestChatFrameAncestorsPreservesOtherDirectives(t *testing.T) {
	policy := chatFrameAncestors("default-src 'self'; frame-ancestors 'none'; connect-src https://api.example; style-src 'self' 'unsafe-inline'")
	for _, preserved := range []string{"default-src 'self'", "connect-src https://api.example", "style-src 'self' 'unsafe-inline'"} {
		if !strings.Contains(policy, preserved) {
			t.Fatalf("unrelated CSP directive lost: %s", preserved)
		}
	}
}
