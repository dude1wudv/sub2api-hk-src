//go:build embed

package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gin-gonic/gin"
)

func TestChatEmbeddedResourcesNeverFallBack(t *testing.T) {
	dist := fstest.MapFS{
		"index.html":                       {Data: []byte("MAIN-SPA")},
		"chat-app/index.html":              {Data: []byte("CHAT-APP")},
		"chat-app/static/main.12345678.js": {Data: []byte("chat-script")},
		"chat-app/LICENSE":                 {Data: []byte("MIT")},
	}
	for _, tc := range []struct {
		path   string
		status int
		body   string
	}{
		{"/chat-app", http.StatusPermanentRedirect, ""},
		{"/chat-app/", http.StatusOK, "CHAT-APP"},
		{"/chat-app/index.html", http.StatusOK, "CHAT-APP"},
		{"/chat-app/static/main.12345678.js", http.StatusOK, "chat-script"},
		{"/chat-app/LICENSE", http.StatusOK, "MIT"},
		{"/chat-app/missing.js", http.StatusNotFound, "Chat resource not found"},
		{"/chat-app/static/", http.StatusNotFound, "Chat resource not found"},
		{"/chat-app/../index.html", http.StatusNotFound, "Chat resource not found"},
		{"/chat-app-unrelated", http.StatusOK, "MAIN-SPA"},
		{"/dashboard", http.StatusOK, "MAIN-SPA"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			router := gin.New()
			router.Use(func(c *gin.Context) {
				if !serveChatStatic(c, dist) {
					c.String(http.StatusOK, "MAIN-SPA")
					c.Abort()
				}
			})
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if w.Code != tc.status || (tc.body != "" && strings.TrimSpace(w.Body.String()) != tc.body) {
				t.Fatalf("got %d %q; want %d %q", w.Code, w.Body.String(), tc.status, tc.body)
			}
			if tc.path == "/chat-app" && w.Header().Get("Location") != "/chat-app/" {
				t.Fatal("missing slash redirect")
			}
			cacheControl := w.Header().Get("Cache-Control")
			if tc.path == "/chat-app/" || tc.path == "/chat-app/index.html" {
				if cacheControl != "no-cache, no-transform" {
					t.Fatalf("chat entry must remain untransformed: %q", cacheControl)
				}
			} else if strings.Contains(cacheControl, "no-transform") {
				t.Fatalf("chat HTML policy leaked to %s", tc.path)
			}
		})
	}
}

func TestChatEmbeddedMissingBundleIs404(t *testing.T) {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		serveChatStatic(c, fstest.MapFS{"index.html": {Data: []byte("MAIN-SPA")}})
	})
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/chat-app/", nil))
	if w.Code != http.StatusNotFound || strings.Contains(w.Body.String(), "MAIN-SPA") {
		t.Fatalf("missing chat must not serve main SPA: %d %s", w.Code, w.Body.String())
	}
}
