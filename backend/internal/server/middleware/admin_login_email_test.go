//go:build unit

package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAdminLoginEmailRejectsExistingSessions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tt := range []struct {
		name, email, role     string
		adminRoute, websocket bool
		status                int
	}{
		{"admin blocked", "admin@example.com", service.RoleAdmin, true, false, 401},
		{"admin websocket blocked", "admin@example.com", service.RoleAdmin, true, true, 401},
		{"user route admin blocked", "admin@example.com", service.RoleAdmin, false, false, 401},
		{"admin allowed", "admin@sub.sunmmyapi.xyz", service.RoleAdmin, true, false, 200},
		{"ordinary user allowed", "user@example.com", service.RoleUser, false, false, 200},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{JWT: config.JWTConfig{Secret: "test-secret", ExpireHour: 1}}
			auth := service.NewAuthService(nil, nil, nil, nil, cfg, nil, nil, nil, nil, nil, nil, nil, nil)
			user := &service.User{ID: 1, Email: tt.email, Role: tt.role, Status: service.StatusActive}
			token, err := auth.GenerateToken(context.Background(), user)
			require.NoError(t, err)
			// Token predates activation of the administrator domain restriction.
			cfg.Security.AdminLoginEmailDomain = "sub.sunmmyapi.xyz"
			users := service.NewUserService(&stubJWTUserRepo{users: map[int64]*service.User{1: user}}, nil, nil, nil)
			router := gin.New()
			if tt.adminRoute {
				router.Use(gin.HandlerFunc(NewAdminAuthMiddleware(auth, users, nil, nil)))
			} else {
				router.Use(gin.HandlerFunc(NewJWTAuthMiddleware(auth, users, nil, nil)))
			}
			router.GET("/test", func(c *gin.Context) { c.Status(http.StatusOK) })
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.websocket {
				req.Header.Set("Upgrade", "websocket")
				req.Header.Set("Connection", "Upgrade")
				req.Header.Set("Sec-WebSocket-Protocol", "sub2api-admin, jwt."+token)
			} else {
				req.Header.Set("Authorization", "Bearer "+token)
			}
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			require.Equal(t, tt.status, w.Code)
			if tt.status == 401 {
				require.Contains(t, w.Body.String(), "INVALID_CREDENTIALS")
			}
		})
	}
}
