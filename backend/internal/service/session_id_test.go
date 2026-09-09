//go:build unit

package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newSessionHeaderContext(t *testing.T, headers map[string]string) *gin.Context {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	c.Request = req
	return c
}

func TestSanitizeSessionID(t *testing.T) {
	longRunes := strings.Repeat("a", maxPersistedSessionIDLength+50)
	multibyte := strings.Repeat("好", maxPersistedSessionIDLength+10)

	tests := []struct {
		name string
		in   string
		want string
	}{
		{"empty", "", ""},
		{"whitespace only", "   \t  ", ""},
		{"trims surrounding whitespace", "  sess-123  ", "sess-123"},
		{"plain value", "conv_abc-123.XYZ", "conv_abc-123.XYZ"},
		{"uuid", "550e8400-e29b-41d4-a716-446655440000", "550e8400-e29b-41d4-a716-446655440000"},
		{"reject CR", "sess\r123", ""},
		{"reject LF", "sess\n123", ""},
		{"reject CRLF injection", "sess-1\r\nSet-Cookie: x=y", ""},
		{"reject tab inside", "sess\t123", ""},
		{"reject NUL", "sess\x00123", ""},
		{"reject DEL", "sess\x7f123", ""},
		{"reject invalid UTF-8", string([]byte{'s', 'e', 's', 's', '-', 0xff}), ""},
		{"accepts value at column bound", strings.Repeat("b", maxPersistedSessionIDLength), strings.Repeat("b", maxPersistedSessionIDLength)},
		{"rejects overlong value", longRunes, ""},
		{"rejects overlong multibyte value", multibyte, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitizeSessionID(tc.in)
			require.Equal(t, tc.want, got, "sanitizeSessionID(%q)", tc.in)
			// Sanitized output must never exceed the DB column bound (rune-counted).
			require.LessOrEqual(t, len([]rune(got)), maxPersistedSessionIDLength)
		})
	}
}

func TestExtractClientSessionID_NilContext(t *testing.T) {
	require.Equal(t, "", ExtractClientSessionID(nil))
}

func TestExtractClientSessionID_NilRequest(t *testing.T) {
	require.Equal(t, "", ExtractClientSessionID(&gin.Context{}))
}

func TestExtractClientSessionID_AbsentReturnsEmpty(t *testing.T) {
	c := newSessionHeaderContext(t, nil)
	require.Equal(t, "", ExtractClientSessionID(c))
}

func TestExtractClientSessionID_SupportedHeaders(t *testing.T) {
	tests := []struct {
		name   string
		header string
		value  string
	}{
		{"session_id", "session_id", "sess-A"},
		{"conversation_id", "conversation_id", "conv-B"},
		{"X-Session-Affinity", openCodeSessionAffinityHeader, "aff-C"},
		{"X-Session-Id", openCodeSessionIDHeader, "sid-D"},
		{"X-OpenCode-Session", openCodeNativeSessionHeader, "oc-E"},
		{"X-Conversation-ID", codeBuddyConversationHeader, "cb-F"},
		{"X-Claude-Code-Session-Id", claudeCodeSessionHeader, "cc-G"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := newSessionHeaderContext(t, map[string]string{tc.header: tc.value})
			require.Equal(t, tc.value, ExtractClientSessionID(c))
		})
	}
}

func TestExtractClientSessionID_HeaderPrecedence(t *testing.T) {
	// session_id ranks ahead of conversation_id and the X-* variants.
	c := newSessionHeaderContext(t, map[string]string{
		"session_id":                "primary",
		"conversation_id":           "secondary",
		openCodeSessionIDHeader:     "tertiary",
		codeBuddyConversationHeader: "quaternary",
	})
	require.Equal(t, "primary", ExtractClientSessionID(c))
}

func TestExtractClientSessionID_Sanitizes(t *testing.T) {
	c := newSessionHeaderContext(t, map[string]string{openCodeSessionIDHeader: "  clean-123  "})
	require.Equal(t, "clean-123", ExtractClientSessionID(c))
}

func TestExtractClientSessionID_IgnoresNonSessionHeaders(t *testing.T) {
	// prompt_cache_key, request/message ids, and a Grok conversation header on a
	// non-Grok request are NOT persisted as session_id.
	c := newSessionHeaderContext(t, map[string]string{
		"prompt_cache_key": "cache-key-should-not-persist",
		"X-Request-Id":     "req-should-not-persist",
		"x-grok-conv-id":   "grok-conv-should-not-persist",
	})
	require.Equal(t, "", ExtractClientSessionID(c))
}

func TestExtractClientSessionID_GrokConversationHeader(t *testing.T) {
	c := newSessionHeaderContext(t, map[string]string{
		grokConversationIDHeader: "grok-native-session",
	})
	c.Set("api_key", &APIKey{
		ID:    42,
		Group: &Group{Platform: PlatformGrok},
	})

	require.Equal(t, "grok-native-session", ExtractClientSessionID(c))
}

func TestExtractClientSessionID_GrokConversationHeaderForCompositeRoute(t *testing.T) {
	c := newSessionHeaderContext(t, map[string]string{
		grokConversationIDHeader: "grok-composite-session",
	})
	c.Set("api_key", &APIKey{
		ID:    43,
		Group: &Group{Platform: PlatformComposite},
	})
	c.Request = c.Request.WithContext(WithResolvedTargetPlatform(context.Background(), PlatformGrok))

	require.Equal(t, "grok-composite-session", ExtractClientSessionID(c))
}

func TestExtractClientSessionID_InjectionHeaderDropped(t *testing.T) {
	// A supported header carrying a CRLF payload is rejected, not persisted mangled.
	c := newSessionHeaderContext(t, map[string]string{"session_id": "abc"})
	c.Request.Header.Set("session_id", "abc\r\nX-Injected: 1")
	require.Equal(t, "", ExtractClientSessionID(c))
}

func TestExtractResponsesUsageSessionID(t *testing.T) {
	c := newSessionHeaderContext(t, nil)
	for _, tc := range []struct {
		name string
		body string
		want string
	}{
		{"top-level session", `{"session_id":"session-1"}`, "session-1"},
		{"top-level conversation", `{"conversation_id":"conversation-1"}`, "conversation-1"},
		{"metadata session", `{"metadata":{"session_id":"session-2"}}`, "session-2"},
		{"metadata conversation", `{"metadata":{"conversation_id":"conversation-2"}}`, "conversation-2"},
		{"prompt cache fallback", `{"prompt_cache_key":"  cache-session  "}`, "cache-session"},
		{"metadata prompt cache fallback", `{"metadata":{"prompt_cache_key":"meta-cache-session"}}`, "meta-cache-session"},
		{"missing correlation", `{}`, ""},
		{"non-string correlation", `{"session_id":42,"prompt_cache_key":false}`, ""},
		{"invalid correlation", `{"prompt_cache_key":"bad\nvalue"}`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, ExtractResponsesUsageSessionID(c, []byte(tc.body)))
		})
	}

	t.Run("explicit header wins over every body field", func(t *testing.T) {
		withHeader := newSessionHeaderContext(t, map[string]string{"session_id": "header-session"})
		body := []byte(`{"session_id":"body-session","prompt_cache_key":"cache-session"}`)
		require.Equal(t, "header-session", ExtractResponsesUsageSessionID(withHeader, body))
	})

	t.Run("invalid higher priority candidate falls through", func(t *testing.T) {
		body := []byte(`{"session_id":"bad\nvalue","prompt_cache_key":"valid-fallback"}`)
		require.Equal(t, "valid-fallback", ExtractResponsesUsageSessionID(c, body))
	})
}
