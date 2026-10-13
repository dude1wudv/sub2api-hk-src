package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func forcedOpenAITierTestGroup(tier string) *Group {
	return &Group{
		ID: 7, Platform: PlatformOpenAI, Status: StatusActive, Hydrated: true,
		ForceOpenAIFast: true, ForceOpenAIUltrafast: tier == OpenAIFastTierUltrafast,
	}
}

func forcedOpenAITierTestPolicy(action string) *OpenAIFastPolicySettings {
	return &OpenAIFastPolicySettings{Rules: []OpenAIFastPolicyRule{{
		ServiceTier: OpenAIFastTierAny, Action: action, Scope: BetaPolicyScopeAll,
	}}}
}

func TestGroupForcedOpenAIServiceTierTrustBoundary(t *testing.T) {
	for _, tc := range []struct {
		name    string
		group   *Group
		account *Account
		want    string
	}{
		{name: "ultrafast alone", group: &Group{ID: 1, Hydrated: true, Status: StatusActive, Platform: PlatformOpenAI, ForceOpenAIUltrafast: true}, account: &Account{Platform: PlatformOpenAI}, want: OpenAIFastTierUltrafast},
		{name: "both flags prefer ultrafast", group: forcedOpenAITierTestGroup(OpenAIFastTierUltrafast), account: &Account{Platform: PlatformOpenAI}, want: OpenAIFastTierUltrafast},
		{name: "unhydrated", group: &Group{ID: 1, Status: StatusActive, Platform: PlatformOpenAI, ForceOpenAIUltrafast: true}, account: &Account{Platform: PlatformOpenAI}},
		{name: "other group platform", group: &Group{ID: 1, Hydrated: true, Status: StatusActive, Platform: PlatformAnthropic, ForceOpenAIUltrafast: true}, account: &Account{Platform: PlatformOpenAI}},
		{name: "other account platform", group: forcedOpenAITierTestGroup(OpenAIFastTierUltrafast), account: &Account{Platform: PlatformAnthropic}},
		{name: "missing account", group: forcedOpenAITierTestGroup(OpenAIFastTierUltrafast)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.WithValue(context.Background(), ctxkey.Group, tc.group)
			require.Equal(t, tc.want, openAIGroupForcedServiceTier(ctx, tc.account))
		})
	}
}

// Capture the actual request after conversion, policy evaluation and upstream
// request construction. Both flags are enabled for Ultrafast to cover precedence.
func TestGroupForcedOpenAIServiceTierHTTPUpstream(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, route := range []string{"responses", "passthrough", "chat", "raw_chat"} {
		for _, tier := range []string{OpenAIFastTierPriority, OpenAIFastTierUltrafast} {
			for _, action := range []string{BetaPolicyActionPass, BetaPolicyActionFilter, BetaPolicyActionBlock, OpenAIFastPolicyActionForcePriority} {
				t.Run(route+"/"+tier+"/"+action, func(t *testing.T) {
					upstream := &httpUpstreamRecorder{resp: &http.Response{
						StatusCode: http.StatusBadRequest, Header: http.Header{"Content-Type": {"application/json"}},
						Body: io.NopCloser(strings.NewReader(`{"error":{"type":"invalid_request_error","message":"stop after capture"}}`)),
					}}
					svc := newOpenAIGatewayServiceWithSettings(t, forcedOpenAITierTestPolicy(action))
					svc.cfg = &config.Config{}
					svc.httpUpstream = upstream
					account := &Account{
						ID: 123, Name: "forced-tier", Platform: PlatformOpenAI, Type: AccountTypeOAuth, Concurrency: 1,
						Credentials: map[string]any{"access_token": "test-token", "chatgpt_account_id": "test-account"},
						Extra:       map[string]any{"openai_passthrough": route == "passthrough", "openai_oauth_responses_websockets_v2_mode": OpenAIWSIngressModeOff},
						Status:      StatusActive, Schedulable: true, RateMultiplier: f64p(1),
					}
					body := []byte(`{"model":"gpt-5.6-sol","input":"hi","stream":false,"service_tier":"default"}`)
					path := "/v1/responses"
					if route == "chat" || route == "raw_chat" {
						body = []byte(`{"model":"gpt-5.6-sol","messages":[{"role":"user","content":"hi"}],"stream":false}`)
						path = "/v1/chat/completions"
					}
					if route == "raw_chat" {
						account.Type = AccountTypeAPIKey
						account.Credentials = map[string]any{"api_key": "sk-test"}
						account.Extra = map[string]any{"openai_responses_supported": false}
					}
					ctx := context.WithValue(context.Background(), ctxkey.Group, forcedOpenAITierTestGroup(tier))
					c, _ := gin.CreateTestContext(httptest.NewRecorder())
					c.Request = httptest.NewRequest(http.MethodPost, path, nil).WithContext(ctx)
					c.Request.Header.Set("User-Agent", "codex_cli_rs/0.144.1")
					var err error
					if route == "chat" || route == "raw_chat" {
						_, err = svc.ForwardAsChatCompletions(ctx, c, account, body, "", "gpt-5.6-sol")
					} else {
						_, err = svc.Forward(ctx, c, account, body)
					}
					require.Error(t, err, "synthetic upstream stops after recording the outgoing request")
					require.NotNil(t, upstream.lastBody, "the global policy must not block a forced group tier")
					require.Equal(t, tier, gjson.GetBytes(upstream.lastBody, "service_tier").String())
					if account.Type == AccountTypeOAuth {
						require.Equal(t, "model=gpt-5.6-sol;tier="+tier, upstream.lastReq.Header.Get(openAICodexRoutingHintHeader))
					}
				})
			}
		}
	}
}

func TestGroupForcedOpenAIServiceTierHTTPToWSUpstream(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tier := range []string{OpenAIFastTierPriority, OpenAIFastTierUltrafast} {
		t.Run(tier, func(t *testing.T) {
			cfg := newOpenAIWSV2TestConfig()
			cfg.Gateway.OpenAIWS.MinIdlePerAccount = 0
			conn := &openAIWSCaptureConn{events: [][]byte{[]byte(`{"type":"response.completed","response":{"id":"resp_forced","model":"gpt-5.6-sol","status":"completed","service_tier":"default","usage":{"input_tokens":1,"output_tokens":1}}}`)}}
			dialer := &openAIWSCaptureDialer{conn: conn}
			pool := newOpenAIWSConnPool(cfg)
			defer pool.Close()
			pool.setClientDialerForTest(dialer)
			svc := newOpenAIGatewayServiceWithSettings(t, forcedOpenAITierTestPolicy(BetaPolicyActionFilter))
			svc.cfg, svc.openaiWSPool = cfg, pool
			svc.httpUpstream, svc.cache = &httpUpstreamRecorder{}, &stubGatewayCache{}
			svc.openaiWSResolver, svc.toolCorrector = NewOpenAIWSProtocolResolver(cfg), NewCodexToolCorrector()
			account := &Account{ID: 123, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Concurrency: 1,
				Credentials: map[string]any{"access_token": "test-token", "chatgpt_account_id": "test-account"},
				Extra:       map[string]any{"openai_passthrough": false, "responses_websockets_v2_enabled": true}}
			ctx := context.WithValue(context.Background(), ctxkey.Group, forcedOpenAITierTestGroup(tier))
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil).WithContext(ctx)
			c.Request.Header.Set("User-Agent", "codex_cli_rs/0.144.1")
			result, err := svc.Forward(ctx, c, account, []byte(`{"model":"gpt-5.6-sol","stream":false,"input":"hi"}`))
			require.NoError(t, err)
			require.True(t, result.OpenAIWSMode)
			require.Equal(t, tier, conn.lastWrite["service_tier"])
			require.Equal(t, "model=gpt-5.6-sol;tier="+tier, dialer.lastHeaders.Get(openAICodexRoutingHintHeader))
			require.NotNil(t, result.ServiceTier)
			require.Equal(t, tier, *result.ServiceTier)
			require.Equal(t, "default", result.UpstreamResponseServiceTier, "upstream downgrade remains distinct from the forced request")
		})
	}
}

// Adapt the staged upstream to support both pooled WriteJSON and passthrough
// WriteFrame, while keeping reads blocked until the test returns each turn.
type forcedTierStagedWSConn struct{ *stagedPassthroughConn }

func (c *forcedTierStagedWSConn) WriteJSON(ctx context.Context, value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.WriteFrame(ctx, coderws.MessageText, payload)
}

func TestGroupForcedOpenAIServiceTierWSEveryTurn(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, mode := range []string{OpenAIWSIngressModePassthrough, OpenAIWSIngressModeCtxPool} {
		for _, tier := range []string{OpenAIFastTierPriority, OpenAIFastTierUltrafast} {
			t.Run(mode+"/"+tier, func(t *testing.T) {
				cfg := passthroughLifecycleConfig()
				cfg.Gateway.OpenAIFirstOutputTimeoutSeconds = 5
				cfg.Gateway.OpenAIWS.ReadTimeoutSeconds = 5
				cfg.Gateway.OpenAIWS.IngressInterTurnIdleTimeoutSeconds = 5
				cfg.Gateway.OpenAIWS.MinIdlePerAccount = 0
				upstream := newStagedPassthroughConn()
				dialer := &stagedPassthroughDialer{conn: &forcedTierStagedWSConn{upstream}}
				svc := newPassthroughLifecycleService(cfg, upstream)
				svc.settingService = newOpenAIGatewayServiceWithSettings(t, forcedOpenAITierTestPolicy(BetaPolicyActionBlock)).settingService
				svc.openaiWSPassthroughDialer = dialer
				pool := newOpenAIWSConnPool(cfg)
				defer pool.Close()
				pool.setClientDialerForTest(dialer)
				svc.openaiWSPool = pool
				account := passthroughLifecycleAccount()
				account.Extra["openai_apikey_responses_websockets_v2_mode"] = mode
				ctx := context.WithValue(context.Background(), ctxkey.Group, forcedOpenAITierTestGroup(tier))
				ctx, cancel := context.WithCancel(ctx)
				defer cancel()
				server, serverErr := startPassthroughLifecycleServer(t, ctx, svc, account)
				defer server.Close()
				client := dialPassthroughLifecycleClient(t, server)
				defer func() { _ = client.CloseNow() }()

				for turn := 1; turn <= 2; turn++ {
					if turn == 2 {
						writeCtx, cancelWrite := context.WithTimeout(ctx, 3*time.Second)
						err := client.Write(writeCtx, coderws.MessageText, []byte(`{"type":"response.create","model":"gpt-5.1","service_tier":"default"}`))
						cancelWrite()
						require.NoError(t, err)
					}
					frame := requirePassthroughUpstreamWrite(t, upstream, 3*time.Second)
					require.Equal(t, tier, gjson.GetBytes(frame, "service_tier").String(), "turn %d must carry the group tier", turn)
					upstream.Send(fmt.Sprintf(`{"type":"response.completed","response":{"id":"resp_forced_%d","model":"gpt-5.1","usage":{"input_tokens":1,"output_tokens":1}}}`, turn))
					event, err := readPassthroughLifecycleFrame(t, client, 3*time.Second)
					require.NoError(t, err)
					require.Equal(t, "response.completed", gjson.GetBytes(event, "type").String())
				}
				_ = client.Close(coderws.StatusNormalClosure, "done")
				select {
				case err := <-serverErr:
					if err != nil {
						require.Contains(t, err.Error(), "StatusNormalClosure")
					}
				case <-time.After(5 * time.Second):
					t.Fatal("websocket session did not finish")
				}
			})
		}
	}
}
