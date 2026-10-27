//go:build unit

package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestGatewayRoutesDeepSeekToolsAreRegistered(t *testing.T) {
	router := newGatewayRoutesTestRouter()
	registered := make(map[string]bool)
	for _, route := range router.Routes() {
		registered[route.Method+" "+route.Path] = true
	}

	require.True(t, registered["POST /v1/tools/search"])
	require.True(t, registered["POST /v1/tools/fetch"])
}

func TestGatewayRoutesDeepSeekToolsRejectUnsupportedGroup(t *testing.T) {
	for _, path := range []string{"/v1/tools/search", "/v1/tools/fetch"} {
		t.Run(path, func(t *testing.T) {
			router := newGatewayRoutesTestRouter(service.PlatformGrok)
			req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"query":"DeepSeek"}`))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			require.Equal(t, http.StatusNotFound, w.Code)
			require.Equal(t, "not_found_error", gjson.Get(w.Body.String(), "error.type").String())
		})
	}
}
