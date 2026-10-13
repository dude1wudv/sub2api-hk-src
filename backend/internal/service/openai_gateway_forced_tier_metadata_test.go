//go:build unit

package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai_compat"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestForwardAsAnthropic_RawChatGroupForcedTier(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tier := range []string{OpenAIFastTierPriority, OpenAIFastTierUltrafast} {
		for _, stream := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/stream=%t", tier, stream), func(t *testing.T) {
				body := []byte(fmt.Sprintf(`{"model":"gpt-5.4","max_tokens":32,"messages":[{"role":"user","content":"hello"}],"stream":%t}`, stream))
				response := `{"id":"chat_forced","object":"chat.completion","model":"gpt-5.4","service_tier":"default","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":7,"completion_tokens":3,"total_tokens":10}}`
				contentType := "application/json"
				if stream {
					response = "data: " + `{"id":"chat_forced","object":"chat.completion.chunk","model":"gpt-5.4","service_tier":"default","choices":[{"index":0,"delta":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":7,"completion_tokens":3,"total_tokens":10}}` + "\n\ndata: [DONE]\n\n"
					contentType = "text/event-stream"
				}
				upstream := &httpUpstreamRecorder{resp: &http.Response{
					StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {contentType}},
					Body: io.NopCloser(strings.NewReader(response)),
				}}
				svc := newOpenAIGatewayServiceWithSettings(t, forcedOpenAITierTestPolicy(BetaPolicyActionBlock))
				svc.cfg, svc.httpUpstream = rawChatCompletionsTestConfig(), upstream
				account := forceChatMessagesFallbackAccount()
				ctx := context.WithValue(context.Background(), ctxkey.Group, forcedOpenAITierTestGroup(tier))
				c, _ := gin.CreateTestContext(httptest.NewRecorder())
				c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil).WithContext(ctx)

				result, err := svc.ForwardAsAnthropic(ctx, c, account, body, "", "")
				require.NoError(t, err)
				require.NotNil(t, result)
				require.Equal(t, "http://upstream.example/v1/chat/completions", upstream.lastReq.URL.String())
				require.Equal(t, tier, gjson.GetBytes(upstream.lastBody, "service_tier").String())
				require.NotNil(t, result.ServiceTier)
				require.Equal(t, tier, *result.ServiceTier)
				require.Equal(t, 7, result.Usage.InputTokens)
				require.Equal(t, 3, result.Usage.OutputTokens)
				require.Equal(t, "default", result.UpstreamResponseServiceTier)
				resolution := ApplyOpenAIServiceTierBillingResolution(account, result)
				require.Equal(t, tier, resolution.Requested)
				require.True(t, resolution.Downgraded, "upstream API-key downgrade stays authoritative")
				require.Equal(t, "default", *result.ServiceTier)
			})
		}
	}
}

func TestForwardAsAnthropic_RawChatGroupForcedTierLeavesOtherPlatformsUntouched(t *testing.T) {
	gin.SetMode(gin.TestMode)
	account := forceChatMessagesFallbackAccount()
	account.Platform = PlatformDeepseek
	upstream := &httpUpstreamRecorder{err: errors.New("stop after capture")}
	svc := newOpenAIGatewayServiceWithSettings(t, DefaultOpenAIFastPolicySettings())
	svc.cfg, svc.httpUpstream = rawChatCompletionsTestConfig(), upstream
	ctx := context.WithValue(context.Background(), ctxkey.Group, forcedOpenAITierTestGroup(OpenAIFastTierUltrafast))
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil).WithContext(ctx)
	body := []byte(`{"model":"gpt-5.4","max_tokens":32,"messages":[{"role":"user","content":"hello"}]}`)
	_, err := svc.forwardAnthropicViaRawChatCompletions(ctx, c, account, body, "")
	require.Error(t, err)
	require.NotNil(t, upstream.lastBody)
	require.False(t, gjson.GetBytes(upstream.lastBody, "service_tier").Exists())
}

func TestOpenAIResponsesCompat_PartialStreamPreservesForcedTierAndEffort(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, ingress := range []string{"chat", "messages"} {
		for _, tier := range []string{OpenAIFastTierPriority, OpenAIFastTierUltrafast} {
			for _, failure := range []string{"read_error", "response_failed"} {
				t.Run(ingress+"/"+tier+"/"+failure, func(t *testing.T) {
					body := []byte(`{"model":"gpt-5.4","messages":[{"role":"user","content":"hello"}],"reasoning_effort":"high","stream":true}`)
					path := "/v1/chat/completions"
					if ingress == "messages" {
						body = []byte(`{"model":"gpt-5.4","max_tokens":32,"messages":[{"role":"user","content":"hello"}],"output_config":{"effort":"high"},"stream":true}`)
						path = "/v1/messages"
					}
					// Compatible upstreams can send progressive usage before failing.
					payload := "data: " + `{"type":"response.created","response":{"id":"resp_partial","model":"gpt-5.4","status":"in_progress","service_tier":"default","usage":{"input_tokens":7,"output_tokens":3}}}` + "\n\n" +
						"data: " + `{"type":"response.output_text.delta","output_index":0,"content_index":0,"delta":"partial"}` + "\n\n"
					var responseBody io.ReadCloser = &errTailReader{data: []byte(payload), err: errors.New("simulated upstream read failure")}
					if failure == "response_failed" {
						payload += "data: " + `{"type":"response.failed","response":{"id":"resp_partial","model":"gpt-5.4","status":"failed","service_tier":"default","usage":{"input_tokens":7,"output_tokens":3},"error":{"type":"server_error","code":"server_error","message":"upstream interrupted"}}}` + "\n\n"
						responseBody = io.NopCloser(strings.NewReader(payload))
					}
					upstream := &httpUpstreamRecorder{resp: &http.Response{
						StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: responseBody,
					}}
					svc := newOpenAIGatewayServiceWithSettings(t, forcedOpenAITierTestPolicy(BetaPolicyActionFilter))
					svc.cfg, svc.httpUpstream = rawChatCompletionsTestConfig(), upstream
					account := rawChatCompletionsTestAccount()
					account.Extra = map[string]any{openai_compat.ExtraKeyResponsesSupported: true}
					ctx := context.WithValue(context.Background(), ctxkey.Group, forcedOpenAITierTestGroup(tier))
					rec := httptest.NewRecorder()
					c, _ := gin.CreateTestContext(rec)
					c.Request = httptest.NewRequest(http.MethodPost, path, nil).WithContext(ctx)
					var result *OpenAIForwardResult
					var err error
					if ingress == "chat" {
						result, err = svc.ForwardAsChatCompletions(ctx, c, account, body, "", "")
					} else {
						result, err = svc.ForwardAsAnthropic(ctx, c, account, body, "", "")
					}
					require.Error(t, err)
					require.NotNil(t, result)
					require.Contains(t, rec.Body.String(), "partial")
					require.Equal(t, "http://upstream.example/v1/responses", upstream.lastReq.URL.String())
					require.Equal(t, tier, gjson.GetBytes(upstream.lastBody, "service_tier").String())
					require.Equal(t, "high", gjson.GetBytes(upstream.lastBody, "reasoning.effort").String())
					require.Equal(t, 7, result.Usage.InputTokens)
					require.Equal(t, 3, result.Usage.OutputTokens)
					require.NotNil(t, result.ServiceTier, "an interrupted billable stream keeps its actual outbound tier")
					require.Equal(t, tier, *result.ServiceTier)
					require.NotNil(t, result.ReasoningEffort)
					require.Equal(t, "high", *result.ReasoningEffort)
					resolution := ApplyOpenAIServiceTierBillingResolution(account, result)
					require.Equal(t, tier, resolution.Requested)
					if failure == "response_failed" {
						require.Equal(t, "default", resolution.Observed)
						require.True(t, resolution.Downgraded)
						require.Equal(t, "default", *result.ServiceTier)
					} else {
						// A response.created tier is provisional; a broken read has
						// no terminal declaration that can lower the outbound tier.
						require.Empty(t, resolution.Observed)
						require.False(t, resolution.Downgraded)
						require.Equal(t, tier, *result.ServiceTier)
					}
					require.Equal(t, 7, result.Usage.InputTokens)
					require.Equal(t, 3, result.Usage.OutputTokens)
				})
			}
		}
	}
}
