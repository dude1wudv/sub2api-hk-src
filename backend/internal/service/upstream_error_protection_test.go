package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestProtectedUpstreamErrorMapping(t *testing.T) {
	for _, tc := range []struct {
		upstream, status int
		typ              string
	}{
		{400, 400, "invalid_request_error"}, {413, 413, "invalid_request_error"},
		{422, 400, "invalid_request_error"}, {401, 502, "api_error"},
		{429, 429, "rate_limit_error"}, {529, 503, "overloaded_error"},
		{500, 502, "api_error"}, {504, 504, "api_error"}, {418, 502, "api_error"},
	} {
		status, typ, msg := ProtectedUpstreamError(nil, tc.upstream)
		require.Equal(t, tc.status, status)
		require.Equal(t, tc.typ, typ)
		require.NotContains(t, strings.ToLower(msg), "upstream")
	}
}

func TestWriteProtectedUpstreamErrorUsesLocalRequestIDAndHidesBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	req = req.WithContext(contextWithClientRequestID(req.Context(), "local-request-1"))
	c.Request = req
	SetUpstreamErrorProtectionEnabled(c, true)
	WriteProtectedUpstreamError(c, 500)
	require.Equal(t, http.StatusBadGateway, rec.Code)
	require.Equal(t, "local-request-1", rec.Header().Get("X-Client-Request-ID"))
	var body map[string]map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "local-request-1", body["error"]["request_id"])
	require.NotContains(t, rec.Body.String(), "api.openai.com")
}

// Small test-only wrapper keeps the production helper free of test fixtures.
func contextWithClientRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxkey.ClientRequestID, id)
}
