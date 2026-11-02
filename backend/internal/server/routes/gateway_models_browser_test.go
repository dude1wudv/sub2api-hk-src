package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestGatewayRoutesModelsBrowserNavigationBeforeAuthentication(t *testing.T) {
	// This registers the full gateway route chain with the real API key auth
	// middleware. A handler-only test would miss the pre-auth ordering bug.
	router, _, _ := newKeyBillingRouteTestRouter(config.RunModeSimple)
	for _, test := range []struct {
		name      string
		path      string
		headers   map[string]string
		status    int
		errorCode string
	}{
		{name: "browser document", path: "/models", status: http.StatusFound},
		{name: "versioned API remains API", path: "/v1/models", status: http.StatusUnauthorized, errorCode: "API_KEY_REQUIRED"},
		{name: "JSON fetch", path: "/models", headers: map[string]string{"Accept": "application/json"}, status: http.StatusUnauthorized, errorCode: "API_KEY_REQUIRED"},
		{name: "SDK HTML accept", path: "/models", headers: map[string]string{"Sec-Fetch-Mode": "", "Sec-Fetch-Dest": ""}, status: http.StatusUnauthorized, errorCode: "API_KEY_REQUIRED"},
		{name: "fetch request", path: "/models", headers: map[string]string{"Sec-Fetch-Mode": "cors", "Sec-Fetch-Dest": "empty"}, status: http.StatusUnauthorized, errorCode: "API_KEY_REQUIRED"},
		{name: "authorization", path: "/models", headers: map[string]string{"Authorization": "Bearer unknown-key"}, status: http.StatusUnauthorized, errorCode: "INVALID_API_KEY"},
		{name: "malformed authorization", path: "/models", headers: map[string]string{"Authorization": "Basic unknown"}, status: http.StatusUnauthorized, errorCode: "API_KEY_REQUIRED"},
		{name: "anthropic credential", path: "/models", headers: map[string]string{"x-api-key": "unknown-key"}, status: http.StatusUnauthorized, errorCode: "INVALID_API_KEY"},
		{name: "google credential", path: "/models", headers: map[string]string{"x-goog-api-key": "unknown-key"}, status: http.StatusUnauthorized, errorCode: "INVALID_API_KEY"},
		{name: "query key", path: "/models?key=unknown-key", status: http.StatusBadRequest, errorCode: "api_key_in_query_deprecated"},
		{name: "query api key", path: "/models?api_key=unknown-key", status: http.StatusBadRequest, errorCode: "api_key_in_query_deprecated"},
	} {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, test.path, nil)
			req.Header.Set("Accept", "text/html,application/xhtml+xml")
			req.Header.Set("Sec-Fetch-Mode", "navigate")
			req.Header.Set("Sec-Fetch-Dest", "document")
			for key, value := range test.headers {
				req.Header.Set(key, value)
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			require.Equal(t, test.status, w.Code, "%s", w.Body.String())
			if test.status == http.StatusFound {
				require.Equal(t, "/model-plaza", w.Header().Get("Location"))
				require.Equal(t, "no-store", w.Header().Get("Cache-Control"))
			} else {
				require.Empty(t, w.Header().Get("Location"))
				require.Contains(t, w.Header().Get("Content-Type"), "application/json")
				require.Contains(t, w.Body.String(), test.errorCode)
			}
		})
	}
}
