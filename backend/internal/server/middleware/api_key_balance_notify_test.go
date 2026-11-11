//go:build unit

package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type authBalanceNotifier struct{ calls atomic.Int64 }

func (n *authBalanceNotifier) NotifyInsufficientBalance(*service.User) { n.calls.Add(1) }

func TestAPIKeyAuthNotifiesOnlyUserBalanceRejection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tt := range []struct {
		name, path, keyStatus, groupStatus, subscriptionType string
		balance, quota, used                                 float64
		runMode                                              string
		subscribed                                           bool
		want                                                 int
		notifications                                        int64
	}{
		{name: "balance zero", balance: 0, want: 403, notifications: 1},
		{name: "balance negative", balance: -1, want: 403, notifications: 1},
		{name: "positive balance", balance: 1, want: 200},
		{name: "key quota status", balance: 0, keyStatus: service.StatusAPIKeyQuotaExhausted, want: 429},
		{name: "key runtime quota", balance: 0, quota: 1, used: 1, want: 429},
		{name: "group disabled", balance: 0, groupStatus: service.StatusDisabled, want: 403},
		{name: "subscription missing", balance: 0, subscriptionType: service.SubscriptionTypeSubscription, want: 403},
		{name: "subscription quota", balance: 0, subscriptionType: service.SubscriptionTypeSubscription, subscribed: true, want: 429},
		{name: "usage lookup", path: "/v1/usage", balance: 0, want: 200},
		{name: "billing lookup", path: "/v1/sub2api/billing", balance: 0, want: 200},
		{name: "simple mode", balance: 0, runMode: config.RunModeSimple, want: 200},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{RunMode: config.RunModeStandard}
			if tt.runMode != "" {
				cfg.RunMode = tt.runMode
			}
			group := &service.Group{ID: 8, Platform: service.PlatformOpenAI, Status: service.StatusActive, SubscriptionType: tt.subscriptionType}
			if tt.groupStatus != "" {
				group.Status = tt.groupStatus
			}
			key := &service.APIKey{ID: 1, UserID: 2, Key: "sk-notify", Status: service.StatusActive, GroupID: &group.ID, Group: group, Quota: tt.quota, QuotaUsed: tt.used, User: &service.User{ID: 2, Status: service.StatusActive, Balance: tt.balance}}
			if tt.keyStatus != "" {
				key.Status = tt.keyStatus
			}
			repo := &stubApiKeyRepo{getByKey: func(context.Context, string) (*service.APIKey, error) { clone := *key; return &clone, nil }}
			keys := service.NewAPIKeyService(repo, nil, nil, nil, nil, nil, cfg)
			subRepo := &stubUserSubscriptionRepo{getActive: func(context.Context, int64, int64) (*service.UserSubscription, error) {
				if !tt.subscribed {
					return nil, service.ErrSubscriptionNotFound
				}
				now := time.Now()
				return &service.UserSubscription{ID: 1, UserID: 2, GroupID: 8, Status: service.StatusActive, ExpiresAt: now.Add(time.Hour), DailyWindowStart: &now, WeeklyWindowStart: &now, MonthlyWindowStart: &now, DailyUsageUSD: 2}, nil
			}}
			if tt.subscribed {
				limit := 1.0
				group.DailyLimitUSD = &limit
			}
			subscriptions := service.NewSubscriptionService(nil, subRepo, nil, nil, cfg)
			t.Cleanup(subscriptions.Stop)
			notifier := &authBalanceNotifier{}
			router := gin.New()
			router.Use(gin.HandlerFunc(NewSmartRoutingAPIKeyAuthMiddleware(keys, subscriptions, cfg, nil, nil, notifier)))
			path := tt.path
			if path == "" {
				path = "/test"
			}
			router.GET(path, func(c *gin.Context) { c.Status(http.StatusOK) })
			request := httptest.NewRequest(http.MethodGet, path, nil)
			request.Header.Set("x-api-key", key.Key)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			require.Equal(t, tt.want, response.Code, response.Body.String())
			require.Equal(t, tt.notifications, notifier.calls.Load())
		})
	}
}
