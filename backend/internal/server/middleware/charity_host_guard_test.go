package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestCharityHostGuard_IsolatesMainSiteAndServesCharitySPA(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	charityServed := false
	r.Use(CharityHostGuard(nil, func(c *gin.Context) {
		charityServed = true
		c.String(http.StatusOK, "charity-spa")
	}))

	r.GET("/api/v1/charity/overview", func(c *gin.Context) {
		require.True(t, IsCharityHostRequest(c))
		c.String(http.StatusOK, "charity-overview")
	})
	r.POST("/v1/messages", func(c *gin.Context) {
		require.True(t, IsCharityHostRequest(c))
		c.String(http.StatusOK, "gateway-ok")
	})
	r.POST("/chat/completions", func(c *gin.Context) {
		require.True(t, IsCharityHostRequest(c))
		c.String(http.StatusOK, "gateway-ok")
	})
	r.GET("/v1/images/batches", func(c *gin.Context) {
		c.String(http.StatusOK, "batch history")
	})
	r.GET("/api/v1/admin/charity/overview", func(c *gin.Context) {
		c.String(http.StatusOK, "admin-ok")
	})
	r.GET("/login", func(c *gin.Context) {
		c.String(http.StatusOK, "main-login")
	})

	// 1. 公益域名根路径返回独立 charity SPA
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "free.xjtu.shop"
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "charity-spa", w.Body.String())
	require.True(t, charityServed)

	// 2. 公益域名允许 /api/v1/charity/overview 与 /v1/messages
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/charity/overview", nil)
	req.Host = "free.xjtu.shop"
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "charity-overview", w.Body.String())

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	req.Host = "free.xjtu.shop"
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "gateway-ok", w.Body.String())

	// 3. 公益域名严禁访问主站管理与登录接口
	for _, blocked := range []string{"/api/v1/admin/charity/overview", "/api/v1/auth/login", "/login", "/admin/charity", "/v1/images/batches", "/v1/responses"} {
		w = httptest.NewRecorder()
		req = httptest.NewRequest(http.MethodGet, blocked, nil)
		req.Host = "free.xjtu.shop"
		r.ServeHTTP(w, req)
		require.Equal(t, http.StatusNotFound, w.Code, "path %s should be blocked on charity host", blocked)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/chat/completions", nil)
	req.Host = "FREE.XJTU.SHOP:443"
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "gateway-ok", w.Body.String())

	// 4. 非公益主站域名不受影响
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/v1/admin/charity/overview", nil)
	req.Host = "sub.microedulab.com"
	req.Header.Set("X-Sub2API-Surface", "charity")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "admin-ok", w.Body.String())
}
