//go:build unit

package handler

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type deepSeekToolsSpineAccountRepo struct {
	service.AccountRepository
	account service.Account
}

func (r deepSeekToolsSpineAccountRepo) GetByID(_ context.Context, id int64) (*service.Account, error) {
	if id != r.account.ID {
		return nil, service.ErrNoAvailableAccounts
	}
	account := r.account
	return &account, nil
}

func (r deepSeekToolsSpineAccountRepo) ListSchedulableByGroupIDAndPlatform(_ context.Context, _ int64, platform string) ([]service.Account, error) {
	if platform != r.account.Platform {
		return nil, nil
	}
	return []service.Account{r.account}, nil
}

func (r deepSeekToolsSpineAccountRepo) ListSchedulableByPlatform(_ context.Context, platform string) ([]service.Account, error) {
	if platform != r.account.Platform {
		return nil, nil
	}
	return []service.Account{r.account}, nil
}

func (r deepSeekToolsSpineAccountRepo) ListSchedulableUngroupedByPlatform(_ context.Context, platform string) ([]service.Account, error) {
	return r.ListSchedulableByPlatform(context.Background(), platform)
}

type deepSeekToolsSpineUpstream struct {
	service.HTTPUpstream
	server *httptest.Server
}

func (u *deepSeekToolsSpineUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	target, err := url.Parse(u.server.URL)
	if err != nil {
		return nil, err
	}
	copyRequest := req.Clone(req.Context())
	copyRequest.Body = io.NopCloser(bytes.NewReader(body))
	copyRequest.URL.Scheme = target.Scheme
	copyRequest.URL.Host = target.Host
	copyRequest.Host = copyRequest.URL.Host
	return u.server.Client().Do(copyRequest)
}

type deepSeekToolsSpineUsageRepo struct {
	service.UsageLogRepository
	mu   sync.Mutex
	logs []*service.UsageLog
}

func (r *deepSeekToolsSpineUsageRepo) Create(_ context.Context, log *service.UsageLog) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	copyLog := *log
	r.logs = append(r.logs, &copyLog)
	return true, nil
}

type deepSeekToolsSpineBillingRepo struct {
	service.UsageBillingRepository
	mu       sync.Mutex
	commands []*service.UsageBillingCommand
}

func (r *deepSeekToolsSpineBillingRepo) Apply(_ context.Context, command *service.UsageBillingCommand) (*service.UsageBillingApplyResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	copyCommand := *command
	r.commands = append(r.commands, &copyCommand)
	return &service.UsageBillingApplyResult{Applied: true}, nil
}

func (r *deepSeekToolsSpineUsageRepo) snapshot() []*service.UsageLog {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]*service.UsageLog(nil), r.logs...)
}

func (r *deepSeekToolsSpineBillingRepo) snapshot() []*service.UsageBillingCommand {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]*service.UsageBillingCommand(nil), r.commands...)
}

type deepSeekToolsSpineUserRepo struct {
	service.UserRepository
	user *service.User
}

func (r deepSeekToolsSpineUserRepo) GetByID(context.Context, int64) (*service.User, error) {
	return r.user, nil
}

func TestDeepSeekToolsSpine(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var upstreamBodies [][]byte
	var upstreamAuth []string
	var mu sync.Mutex
	var emptySearch bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		mu.Lock()
		upstreamBodies = append(upstreamBodies, append([]byte(nil), body...))
		upstreamAuth = append(upstreamAuth, r.Header.Get("Authorization"))
		mu.Unlock()
		switch r.URL.Path {
		case "/v1/tools/search":
			w.Header().Set("Content-Type", "application/json")
			mu.Lock()
			wantEmpty := emptySearch
			mu.Unlock()
			if wantEmpty {
				_, _ = io.WriteString(w, `{"results":[]}`)
			} else {
				_, _ = io.WriteString(w, `{"results":[{"title":"DeepSeek","url":"https://example.com","snippet":"snippet","summary":"summary"}]}`)
			}
		case "/v1/tools/fetch":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"title":"page","content":"# fetched"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	const secret = "test-only-deepseek-upstream-key"
	user := &service.User{ID: 100, Status: service.StatusActive, Balance: 10}
	groupID := int64(223)
	searchPrice := 0.013
	group := &service.Group{ID: groupID, Name: "Deepseek test", Platform: service.PlatformOpenAI,
		Status: service.StatusActive, SubscriptionType: service.SubscriptionTypeStandard, RateMultiplier: 0.4,
		WebSearchPricePerCall: &searchPrice}
	apiKey := &service.APIKey{ID: 501, Key: "test-only-inbound-key", Status: service.StatusActive, UserID: user.ID,
		User: user, GroupID: &groupID, Group: group}
	account := service.Account{ID: 1112, Name: "synthetic Kedaya", Platform: service.PlatformOpenAI,
		Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Concurrency: 0,
		GroupIDs: []int64{groupID}, AccountGroups: []service.AccountGroup{{AccountID: 1112, GroupID: groupID}}, Groups: []*service.Group{group},
		Credentials: map[string]any{"api_key": secret, "base_url": "https://api.deepseek.com"},
		Extra:       map[string]any{"deepseek_tools_group_ids": []any{float64(groupID)}}}
	usageRepo := &deepSeekToolsSpineUsageRepo{}
	billingRepo := &deepSeekToolsSpineBillingRepo{}
	cfg := &config.Config{}
	cfg.RunMode = config.RunModeStandard
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	userRepo := deepSeekToolsSpineUserRepo{user: user}
	billingCache := service.NewBillingCacheService(nil, userRepo, nil, nil, nil, nil, &config.Config{RunMode: config.RunModeSimple}, nil)
	t.Cleanup(billingCache.Stop)
	accountRepo := deepSeekToolsSpineAccountRepo{account: account}
	upstream := &deepSeekToolsSpineUpstream{server: server}
	gateway := service.NewOpenAIGatewayService(accountRepo, usageRepo, billingRepo, userRepo, nil, nil, nil,
		cfg, nil, nil, service.NewBillingService(cfg, nil), nil, billingCache, upstream, &service.DeferredService{},
		nil, nil, nil, nil, nil, nil, nil)
	h := NewOpenAIGatewayHandler(gateway, service.NewConcurrencyService(nil), billingCache,
		service.NewAPIKeyService(nil, nil, nil, nil, nil, nil, cfg), nil, nil, nil, nil, cfg)
	h.concurrencyHelper = NewConcurrencyHelper(service.NewConcurrencyService(nil), SSEPingFormatNone, 0)

	call := func(path, request string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(http.MethodPost, path, strings.NewReader(request))
		c.Request.Header.Set("Content-Type", "application/json")
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), ctxkey.ClientRequestID, "reused-client-id"))
		c.Set("api_key", apiKey)
		c.Set(string(middleware2.ContextKeyAPIKey), apiKey)
		c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID, Concurrency: 0})
		if path == "/v1/tools/search" {
			h.DeepSeekSearch(c)
		} else {
			h.DeepSeekFetch(c)
		}
		return recorder
	}

	searchRequest := `{"query":"DeepSeek site:github.com","count":1,"need_content":true,"extra_client_field":"preserve me"}`
	for _, tc := range []struct{ path, request string }{
		{"/v1/tools/search", searchRequest},
		{"/v1/tools/fetch", `{"url":"https://example.com/article"}`},
	} {
		recorder := call(tc.path, tc.request)
		require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	}

	logs := usageRepo.snapshot()
	commands := billingRepo.snapshot()
	require.Len(t, logs, 2)
	require.Len(t, commands, 2)
	require.Equal(t, []string{"deepseek_tools:...", "deepseek_tools:..."}, []string{logs[0].RequestID[:15] + "...", logs[1].RequestID[:15] + "..."})
	require.NotEqual(t, logs[0].RequestID, logs[1].RequestID, "reused downstream request IDs must not merge calls")
	for i, label := range []string{"deepseek-search", "deepseek-fetch"} {
		require.Equal(t, label, logs[i].Model)
		require.InDelta(t, 0.013, logs[i].TotalCost, 1e-12)
		require.InDelta(t, 0.013*0.4, logs[i].ActualCost, 1e-12)
		require.Equal(t, logs[i].RequestID, commands[i].RequestID)
	}
	require.Equal(t, []string{"Bearer " + secret, "Bearer " + secret}, upstreamAuth)
	require.Contains(t, string(upstreamBodies[0]), `"extra_client_field":"preserve me"`)
	require.NotContains(t, string(upstreamBodies[0]), `"model"`)

	// A valid but empty search still gets a usage row and immutable request ID,
	// while recording the per-request billing mode at zero cost.
	mu.Lock()
	emptySearch = true
	mu.Unlock()
	emptyRecorder := call("/v1/tools/search", `{"query":"no match"}`)
	require.Equal(t, http.StatusOK, emptyRecorder.Code)
	logs = usageRepo.snapshot()
	require.Len(t, logs, 3)
	require.Zero(t, logs[2].TotalCost)
	require.Zero(t, logs[2].ActualCost)
	require.NotNil(t, logs[2].BillingMode)
	require.Equal(t, string(service.BillingModePerRequest), *logs[2].BillingMode)
	require.Len(t, billingRepo.snapshot(), 3)

	// A paid query under group 81 must fail before it reaches the synthetic upstream.
	sharedGroupID := int64(81)
	sharedKey := *apiKey
	sharedKey.GroupID = &sharedGroupID
	sharedKey.Group = &service.Group{ID: sharedGroupID, Platform: service.PlatformOpenAI, Status: service.StatusActive, RateMultiplier: 1}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/tools/search", strings.NewReader(`{"query":"must be denied"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("api_key", &sharedKey)
	c.Set(string(middleware2.ContextKeyAPIKey), &sharedKey)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: user.ID, Concurrency: 0})
	h.DeepSeekSearch(c)
	require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
	require.Len(t, upstreamBodies, 3)
	require.Len(t, usageRepo.snapshot(), 3)
}
