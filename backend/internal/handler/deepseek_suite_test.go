//go:build unit

package handler

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/repository"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

type suiteTestAccounts struct {
	service.AccountRepository
	accounts []service.Account
}

func (r suiteTestAccounts) GetByID(_ context.Context, id int64) (*service.Account, error) {
	for _, account := range r.accounts {
		if account.ID == id {
			return &account, nil
		}
	}
	return nil, service.ErrNoAvailableAccounts
}

func (r suiteTestAccounts) ListSchedulableByGroupIDAndPlatform(context.Context, int64, string) ([]service.Account, error) {
	return r.accounts, nil
}

type suiteTestUsage struct {
	deepSeekToolsSpineUsageRepo
	service.DeepSeekSuiteRepository
	operations map[string]service.DeepSeekSuiteOperation
}

func (r *suiteTestUsage) CreateDeepSeekOperation(_ context.Context, op *service.DeepSeekSuiteOperation) error {
	r.operations[op.ID] = *op
	return nil
}

func (r *suiteTestUsage) SaveDeepSeekOperation(ctx context.Context, op *service.DeepSeekSuiteOperation) error {
	return r.CreateDeepSeekOperation(ctx, op)
}

func (r *suiteTestUsage) ListDeepSeekPending(context.Context) ([]*service.DeepSeekSuiteOperation, error) {
	return nil, nil
}

func TestDeepSeekSuiteHandlerUsesOneAccountPerOperation(t *testing.T) {
	for _, tc := range []struct {
		name       string
		status     int
		body       string
		wantStatus string
		wantBills  int
	}{
		{"success", http.StatusOK, `{"results":[{"title":"test","url":"https://example.com","snippet":"test"}]}`, "settled", 1},
		{"unsupported", http.StatusNotFound, `{"error":"upstream unavailable"}`, "failed", 0},
		{"not implemented", http.StatusNotImplemented, `{"error":"upstream unavailable"}`, "failed", 0},
		{"ambiguous failure", http.StatusBadGateway, `{"error":"upstream unavailable"}`, "awaiting_meter", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			var upstreamCalls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				upstreamCalls.Add(1)
				if r.Header.Get("Authorization") != "Bearer synthetic-kedaya" {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.body)
			}))
			defer server.Close()
			groupID := int64(223)
			user := &service.User{ID: 100, Status: service.StatusActive, Balance: 10}
			group := &service.Group{ID: groupID, Platform: service.PlatformOpenAI, Status: service.StatusActive,
				SubscriptionType: service.SubscriptionTypeStandard, RateMultiplier: .4}
			key := &service.APIKey{ID: 501, Status: service.StatusActive, User: user, UserID: user.ID, GroupID: &groupID, Group: group}
			account := service.Account{ID: 1112, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey,
				Status: service.StatusActive, Schedulable: true, Priority: 1, Concurrency: 2,
				GroupIDs: []int64{groupID}, Groups: []*service.Group{group},
				Credentials: map[string]any{"base_url": "https://kedaya.ai", "api_key": "synthetic-kedaya"},
				Extra: map[string]any{"deepseek_tools_group_ids": []any{float64(groupID)},
					"deepseek_suite_fallback_account_id": float64(1182)}}
			target := account
			target.ID, target.Priority = 1182, 2
			target.Credentials = map[string]any{"base_url": "https://vip.mdkj.lol", "api_key": "synthetic-mdkj"}
			target.Extra = map[string]any{"deepseek_tools_group_ids": []any{float64(groupID)}}
			usage := &suiteTestUsage{operations: map[string]service.DeepSeekSuiteOperation{}}
			bills := &deepSeekToolsSpineBillingRepo{}
			cfg := &config.Config{RunMode: config.RunModeStandard}
			users := deepSeekToolsSpineUserRepo{user: user}
			admission := service.NewBillingCacheService(nil, users, nil, nil, nil, nil, &config.Config{RunMode: config.RunModeSimple}, nil)
			t.Cleanup(admission.Stop)
			rdb := redis.NewClient(&redis.Options{Addr: miniredis.RunT(t).Addr()})
			t.Cleanup(func() { _ = rdb.Close() })
			concurrency := service.NewConcurrencyService(repository.NewConcurrencyCache(rdb, 5, 30))
			gateway := service.NewOpenAIGatewayService(suiteTestAccounts{accounts: []service.Account{account, target}},
				usage, bills, users, nil, nil, nil, cfg, nil, nil, service.NewBillingService(cfg, nil), nil,
				admission, &deepSeekToolsSpineUpstream{server: server}, &service.DeferredService{}, nil, nil, nil, nil, nil, nil, nil)
			t.Cleanup(gateway.CloseOpenAIWSPool)
			h := NewOpenAIGatewayHandler(gateway, concurrency, admission,
				service.NewAPIKeyService(nil, nil, nil, nil, nil, nil, cfg), nil, nil, nil, nil, cfg)
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/tools/search", strings.NewReader(`{"query":"test"}`))
			c.Request.Header.Set("Content-Type", "application/json")
			c.Set(string(middleware2.ContextKeyAPIKey), key)
			c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID, Concurrency: 2})
			h.handleDeepSeekSuite(c, "search", false)
			require.Equal(t, tc.status, recorder.Code, recorder.Body.String())
			require.JSONEq(t, tc.body, recorder.Body.String())
			require.Equal(t, int32(1), upstreamCalls.Load())
			require.Len(t, bills.snapshot(), tc.wantBills)
			require.Len(t, usage.operations, 1)
			operationID := recorder.Header().Get("X-DeepSeek-Operation-ID")
			require.Contains(t, usage.operations, operationID)
			op := usage.operations[operationID]
			require.Equal(t, account.ID, op.AccountID)
			require.Equal(t, tc.wantStatus, op.Status)
		})
	}
}
