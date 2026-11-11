package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type googleRouteBalanceNotifier struct{ calls atomic.Int64 }

func (n *googleRouteBalanceNotifier) NotifyInsufficientBalance(*service.User) { n.calls.Add(1) }

func TestGatewayRoutesGoogleBalanceNotificationInjection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, path := range []string{
		"/v1beta/models/gemini-2.5-flash:generateContent",
		"/antigravity/v1beta/models/gemini-2.5-flash:generateContent",
	} {
		t.Run(path, func(t *testing.T) {
			group := &service.Group{ID: 8, Platform: service.PlatformGemini, Status: service.StatusActive, Hydrated: true, SubscriptionType: service.SubscriptionTypeStandard}
			key := &service.APIKey{ID: 1, Key: "sk-google-route-notify", UserID: 2, Status: service.StatusActive, GroupID: &group.ID, Group: group, User: &service.User{ID: 2, Status: service.StatusActive, Balance: 0}}
			cfg := &config.Config{RunMode: config.RunModeStandard, Gateway: config.GatewayConfig{MaxBodySize: 1 << 20, TextMaxBodySize: 1 << 20}}
			keys := service.NewAPIKeyService(&keyBillingRouteAPIKeyRepo{apiKey: key}, nil, nil, nil, nil, nil, cfg)
			notifier := &googleRouteBalanceNotifier{}
			router := gin.New()
			RegisterGatewayRoutes(router, &handler.Handlers{Gateway: &handler.GatewayHandler{}, OpenAIGateway: &handler.OpenAIGatewayHandler{}, AsyncImage: handler.NewAsyncImageHandler(nil, nil)}, servermiddleware.APIKeyAuthMiddleware(func(c *gin.Context) { c.Next() }), keys, nil, nil, nil, nil, cfg, notifier)
			request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"contents":[{"parts":[{"text":"hello"}]}]}`))
			request.Header.Set("x-goog-api-key", key.Key)
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			require.Equal(t, http.StatusForbidden, response.Code, response.Body.String())
			require.Equal(t, int64(1), notifier.calls.Load(), "both native Google chains must receive the injected notifier")
			wantJSON, err := json.Marshal(map[string]any{"error": map[string]any{"code": http.StatusForbidden, "message": "Insufficient account balance", "status": "PERMISSION_DENIED"}})
			require.NoError(t, err)
			require.JSONEq(t, string(wantJSON), response.Body.String())
		})
	}
}
