//go:build unit

package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	middleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type smartRoutingIntegrationKeyRepo struct {
	service.APIKeyRepository
	key *service.APIKey
}

func (r *smartRoutingIntegrationKeyRepo) GetByKeyForAuth(context.Context, string) (*service.APIKey, error) {
	copy := *r.key
	return &copy, nil
}
func (*smartRoutingIntegrationKeyRepo) UpdateLastUsed(context.Context, int64, time.Time) error {
	return nil
}

type smartRoutingIntegrationUserRepo struct {
	service.UserRepository
	user *service.User
}

func (r *smartRoutingIntegrationUserRepo) GetByID(context.Context, int64) (*service.User, error) {
	return r.user, nil
}

type smartRoutingIntegrationRates struct {
	service.UserGroupRateRepository
	groups []int64
	rate   float64
}

func (r *smartRoutingIntegrationRates) GetByUserAndGroup(_ context.Context, _, groupID int64) (*float64, error) {
	r.groups = append(r.groups, groupID)
	return &r.rate, nil
}
func (*smartRoutingIntegrationRates) GetRPMOverrideByUserAndGroup(context.Context, int64, int64) (*int, error) {
	return nil, nil
}

type smartRoutingIntegrationSubscriptions struct {
	service.UserSubscriptionRepository
	sub *service.UserSubscription
}

func (r *smartRoutingIntegrationSubscriptions) GetActiveByUserIDAndGroupID(_ context.Context, userID, groupID int64) (*service.UserSubscription, error) {
	if r.sub == nil || r.sub.UserID != userID || r.sub.GroupID != groupID {
		return nil, service.ErrSubscriptionNotFound
	}
	copy := *r.sub
	return &copy, nil
}

type smartRoutingIntegrationUsageRepo struct {
	service.UsageLogRepository
	logs []*service.UsageLog
}

func (r *smartRoutingIntegrationUsageRepo) Create(_ context.Context, log *service.UsageLog) (bool, error) {
	copy := *log
	r.logs = append(r.logs, &copy)
	return true, nil
}

type smartRoutingIntegrationBillingRepo struct {
	service.UsageBillingRepository
	commands []*service.UsageBillingCommand
}

func (r *smartRoutingIntegrationBillingRepo) Apply(_ context.Context, cmd *service.UsageBillingCommand) (*service.UsageBillingApplyResult, error) {
	copy := *cmd
	r.commands = append(r.commands, &copy)
	return &service.UsageBillingApplyResult{Applied: true}, nil
}

type smartRoutingIntegrationConcurrency struct {
	fakeConcurrencyCache
	acquired, released int
}

func (c *smartRoutingIntegrationConcurrency) AcquireAccountSlot(context.Context, int64, int, string) (bool, error) {
	c.acquired++
	return true, nil
}
func (c *smartRoutingIntegrationConcurrency) ReleaseAccountSlot(context.Context, int64, string) error {
	c.released++
	return nil
}

// The port routes every outbound request to an in-process server. It never
// contacts the URL configured on the synthetic Anthropic account.
type smartRoutingLocalUpstream struct {
	base   *url.URL
	client *http.Client
}

func (u *smartRoutingLocalUpstream) Do(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	copy := req.Clone(req.Context())
	copy.URL.Scheme, copy.URL.Host = u.base.Scheme, u.base.Host
	copy.Host = u.base.Host
	return u.client.Do(copy)
}
func (u *smartRoutingLocalUpstream) DoWithTLS(req *http.Request, proxy string, accountID int64, concurrency int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return u.Do(req, proxy, accountID, concurrency)
}

const smartRoutingToolSSE = "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_tool\",\"type\":\"message\",\"role\":\"assistant\",\"model\":\"claude-sonnet-4-5\",\"content\":[],\"usage\":{\"input_tokens\":10,\"output_tokens\":0}}}\n\n" +
	"event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"tool_use\",\"id\":\"toolu_1\",\"name\":\"get_weather\",\"input\":{}}}\n\n" +
	"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"input_json_delta\",\"partial_json\":\"{\\\"city\\\":\\\"Paris\\\"}\"}}\n\n" +
	"event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\n" +
	"event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"tool_use\"},\"usage\":{\"output_tokens\":5}}\n\n" +
	"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"

func TestClaudeSmartRoutingProtocolsAndSelectedGroupBilling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	endpoints := []struct {
		path, request, resultPath, streamMarker string
		call                                    func(*GatewayHandler, *gin.Context)
	}{
		{"/v1/messages", `"max_tokens":128,"messages":[{"role":"user","content":"hello"},{"role":"assistant","content":[{"type":"tool_use","id":"toolu_previous","name":"get_weather","input":{"city":"London"}}]},{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_previous","content":"sunny"}]}],"tools":[{"name":"get_weather","input_schema":{"type":"object","properties":{"city":{"type":"string"}}}}]`, "content.0.name", "event: message_stop", (*GatewayHandler).Messages},
		{"/v1/chat/completions", `"messages":[{"role":"user","content":"hello"},{"role":"assistant","tool_calls":[{"id":"call_previous","type":"function","function":{"name":"get_weather","arguments":"{\"city\":\"London\"}"}}]},{"role":"tool","tool_call_id":"call_previous","content":"sunny"}],"tools":[{"type":"function","function":{"name":"get_weather","parameters":{"type":"object","properties":{"city":{"type":"string"}}}}}]`, "choices.0.message.tool_calls.0.function.name", "data: [DONE]", (*GatewayHandler).ChatCompletions},
		{"/v1/responses", `"input":[{"type":"message","role":"user","content":"hello"},{"type":"function_call","call_id":"call_previous","name":"get_weather","arguments":"{\"city\":\"London\"}"},{"type":"function_call_output","call_id":"call_previous","output":"sunny"}],"tools":[{"type":"function","name":"get_weather","parameters":{"type":"object","properties":{"city":{"type":"string"}}}}]`, "output.0.name", "event: response.completed", (*GatewayHandler).Responses},
		{"/v1/messages/count_tokens", `"messages":[{"role":"user","content":"hello"}]`, "input_tokens", "", (*GatewayHandler).CountTokens},
	}
	for _, endpoint := range endpoints {
		for _, streaming := range []bool{false, true} {
			if strings.HasSuffix(endpoint.path, "count_tokens") && streaming {
				continue
			}
			for _, subscription := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/stream=%t/subscription=%t", endpoint.path, streaming, subscription), func(t *testing.T) {
					var upstreamBodies [][]byte
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						body, err := io.ReadAll(r.Body)
						require.NoError(t, err)
						upstreamBodies = append(upstreamBodies, body)
						w.Header().Set("x-request-id", "smart-route-usage")
						if strings.HasSuffix(endpoint.path, "count_tokens") {
							require.Equal(t, "/v1/messages/count_tokens", r.URL.Path)
							w.Header().Set("Content-Type", "application/json")
							_, _ = io.WriteString(w, `{"input_tokens":42}`)
							return
						}
						require.Equal(t, "/v1/messages", r.URL.Path)
						require.Equal(t, "get_weather", gjson.GetBytes(body, "tools.0.name").String())
						require.Contains(t, string(body), "tool_result", "tool history must survive protocol conversion")
						if gjson.GetBytes(body, "stream").Bool() {
							w.Header().Set("Content-Type", "text/event-stream")
							_, _ = io.WriteString(w, smartRoutingToolSSE)
						} else {
							w.Header().Set("Content-Type", "application/json")
							_, _ = io.WriteString(w, `{"id":"msg_tool","type":"message","role":"assistant","model":"claude-sonnet-4-5","content":[{"type":"tool_use","id":"toolu_1","name":"get_weather","input":{"city":"Paris"}}],"stop_reason":"tool_use","usage":{"input_tokens":10,"output_tokens":5}}`)
						}
					}))
					defer server.Close()
					base, err := url.Parse(server.URL)
					require.NoError(t, err)
					first := &service.Group{ID: 91, Hydrated: true, Status: service.StatusActive, Platform: service.PlatformAnthropic, RateMultiplier: 9}
					selected := &service.Group{ID: 92, Hydrated: true, Status: service.StatusActive, Platform: service.PlatformAnthropic, RateMultiplier: 2}
					if subscription {
						selected.SubscriptionType = service.SubscriptionTypeSubscription
					}
					user := &service.User{ID: 7, Status: service.StatusActive, Balance: 100, Concurrency: 10}
					key := &service.APIKey{ID: 8, Key: "synthetic-smart-key", Status: service.StatusActive, UserID: user.ID, User: user, GroupID: &first.ID, Group: first, RoutingGroupIDs: []int64{first.ID, selected.ID}}
					groups := &groupMapRepo{fakeGroupRepo: &fakeGroupRepo{}, groups: map[int64]*service.Group{first.ID: first, selected.ID: selected}}
					account := &service.Account{ID: 93, Status: service.StatusActive, Schedulable: true, Platform: service.PlatformAnthropic, Type: service.AccountTypeAPIKey, Concurrency: 1,
						Credentials:   map[string]any{"api_key": "synthetic-upstream-key", "model_mapping": map[string]any{"claude-smart-test": "claude-sonnet-4-5"}},
						AccountGroups: []service.AccountGroup{{AccountID: 93, GroupID: selected.ID}}}
					snapshot := service.NewSchedulerSnapshotService(&groupScopedSchedulerCache{fakeSchedulerCache: &fakeSchedulerCache{accounts: []*service.Account{account}}}, nil, nil, nil, nil)
					concurrency := &smartRoutingIntegrationConcurrency{}
					concurrencyService := service.NewConcurrencyService(concurrency)
					usage := &smartRoutingIntegrationUsageRepo{}
					billing := &smartRoutingIntegrationBillingRepo{}
					rates := &smartRoutingIntegrationRates{rate: 0.25}
					now := time.Now()
					subs := &smartRoutingIntegrationSubscriptions{sub: &service.UserSubscription{ID: 94, UserID: user.ID, GroupID: selected.ID, Status: service.SubscriptionStatusActive, StartsAt: now, ExpiresAt: now.Add(7 * 24 * time.Hour), DailyWindowStart: &now, WeeklyWindowStart: &now, MonthlyWindowStart: &now}}
					cfg := &config.Config{RunMode: config.RunModeStandard}
					cfg.Gateway.MaxLineSize = 1024 * 1024
					cfg.Default.RateMultiplier = 1
					// Billing admission is isolated from external caches. The actual
					// usage service below remains in standard mode and emits billing commands.
					admission := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, &config.Config{RunMode: config.RunModeSimple}, nil)
					defer admission.Stop()
					pricing := service.NewBillingService(cfg, nil)
					gateway := service.NewGatewayService(nil, groups, usage, billing, nil, subs, rates, nil, cfg, snapshot, concurrencyService, pricing, &service.RateLimitService{}, admission, nil,
						&smartRoutingLocalUpstream{base: base, client: server.Client()}, &service.DeferredService{}, nil, nil, nil, nil, nil, nil, nil, service.NewModelPricingResolver(nil, pricing), nil, nil, nil)
					h := &GatewayHandler{gatewayService: gateway, billingCacheService: admission, concurrencyHelper: NewConcurrencyHelper(concurrencyService, SSEPingFormatClaude, 0), maxAccountSwitches: 1, cfg: cfg}
					keys := service.NewAPIKeyService(&smartRoutingIntegrationKeyRepo{key: key}, &smartRoutingIntegrationUserRepo{user: user}, groups, subs, rates, nil, cfg)
					subService := service.NewSubscriptionService(groups, subs, nil, nil, cfg)
					defer subService.Stop()
					router := gin.New()
					router.Use(gin.HandlerFunc(middleware.NewSmartRoutingAPIKeyAuthMiddleware(keys, subService, cfg, nil, gateway)))
					router.POST(endpoint.path, func(c *gin.Context) { endpoint.call(h, c) })
					request := fmt.Sprintf(`{"model":"claude-smart-test","stream":%t,%s}`, streaming, endpoint.request)
					recorder := httptest.NewRecorder()
					req := httptest.NewRequest(http.MethodPost, endpoint.path, strings.NewReader(request))
					req.Header.Set("Content-Type", "application/json")
					req.Header.Set("x-api-key", key.Key)
					router.ServeHTTP(recorder, req)
					require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
					require.Len(t, upstreamBodies, 1)
					require.Equal(t, "claude-sonnet-4-5", gjson.GetBytes(upstreamBodies[0], "model").String())
					if strings.HasSuffix(endpoint.path, "count_tokens") {
						require.Equal(t, int64(42), gjson.Get(recorder.Body.String(), endpoint.resultPath).Int())
						require.Zero(t, concurrency.acquired)
						require.Zero(t, concurrency.released)
						require.Empty(t, usage.logs)
						require.Empty(t, billing.commands)
						return
					}
					if streaming {
						require.Contains(t, recorder.Body.String(), endpoint.streamMarker)
						require.Contains(t, recorder.Body.String(), "get_weather")
					} else {
						require.True(t, json.Valid(recorder.Body.Bytes()))
						require.Equal(t, "get_weather", gjson.Get(recorder.Body.String(), endpoint.resultPath).String())
					}
					require.Contains(t, recorder.Body.String(), "Paris")
					require.Equal(t, 1, concurrency.acquired, "handler must consume the selected account rather than acquire a second slot")
					require.Equal(t, 1, concurrency.released, "handler and auth cleanup must release once")
					require.Len(t, usage.logs, 1)
					log := usage.logs[0]
					require.Equal(t, selected.ID, *log.GroupID)
					require.Equal(t, account.ID, log.AccountID)
					require.Equal(t, 0.25, log.RateMultiplier)
					require.Equal(t, []int64{selected.ID}, rates.groups)
					require.Equal(t, 10, log.InputTokens)
					require.Equal(t, 5, log.OutputTokens)
					require.Greater(t, log.TotalCost, float64(0))
					require.InDelta(t, log.TotalCost*0.25, log.ActualCost, 1e-8)
					require.Len(t, billing.commands, 1)
					cmd := billing.commands[0]
					require.Equal(t, key.ID, cmd.APIKeyID)
					if subscription {
						require.Equal(t, subs.sub.ID, *cmd.SubscriptionID)
						require.InDelta(t, log.ActualCost, cmd.SubscriptionCost, 1e-8)
						require.Zero(t, cmd.BalanceCost)
					} else {
						require.Nil(t, cmd.SubscriptionID)
						require.InDelta(t, log.ActualCost, cmd.BalanceCost, 1e-8)
						require.Zero(t, cmd.SubscriptionCost)
					}
				})
			}
		}
	}
}
