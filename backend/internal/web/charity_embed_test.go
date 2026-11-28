//go:build embed

package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestServeEmbeddedCharityFrontend_MissingAssetReturns404(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(ServeEmbeddedCharityFrontend())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/assets/does-not-exist.js", nil)
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)
}
