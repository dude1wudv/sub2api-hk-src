//go:build unit

package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAnthropicMergeKeepsConfirmedUsageAndMeteredTail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name, tail string
		wantOutput int
	}{
		{"text", `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"你好"}}`, 9},
		{"thinking", `{"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"思考"}}`, 9},
		{"tool start and arguments", `{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","name":"工具","input":{}}}` + "\n\n" +
			`data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"参数"}}`, 11},
	} {
		for _, completeBoundary := range []bool{false, true} {
			t.Run(tc.name+map[bool]string{false: "/EOF complete JSON", true: "/event boundary"}[completeBoundary], func(t *testing.T) {
				wire := "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"model\":\"actual-model\",\"usage\":{\"input_tokens\":23}}}\n\n" +
					"event: message_delta\ndata: {\"type\":\"message_delta\",\"usage\":{\"output_tokens\":7}}\n\n" +
					"data: " + tc.tail
				if completeBoundary {
					wire += "\n\n"
				}
				rec := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(rec)
				c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
				resp := &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(wire))}
				result, err := newMinimalGatewayService().handleStreamingResponse(context.Background(), resp, c, &Account{ID: 1}, time.Now(), "model", "model", false)
				require.ErrorContains(t, err, "missing terminal event")
				require.NotNil(t, result)
				require.Equal(t, 23, result.usage.InputTokens)
				require.Equal(t, tc.wantOutput, result.usage.OutputTokens, "confirmed prefix plus only the received unmetered tail")
				require.True(t, result.usage.OutputTokensEstimated)
				require.NotContains(t, rec.Body.String(), "event: message_stop", "an incomplete stream cannot manufacture success")
				require.Equal(t, "actual-model", observedUpstreamResponseModel(c))
			})
		}
	}
}

func TestGrokMergeProtectionPrecedesFlatErrorsAndContentPolicy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, started := range []bool{false, true} {
		for _, status := range []int{http.StatusBadRequest, http.StatusForbidden} {
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", nil)
			c.Request.Header.Set("X-Client-Request-ID", "local-request")
			SetUpstreamErrorProtectionEnabled(c, true)
			rule := newNonFailoverPassthroughRule(status, "permission-denied", http.StatusTeapot, "provider-private")
			rules := &ErrorPassthroughService{}
			rules.setLocalCache([]*model.ErrorPassthroughRule{rule})
			BindErrorPassthroughService(c, rules)
			if started {
				c.Header("Content-Type", "text/event-stream")
				_, err := c.Writer.WriteString(": queue ping\n\n")
				require.NoError(t, err)
				c.Writer.Flush()
			}
			svc := &OpenAIGatewayService{accountRepo: &grokQuotaAccountRepo{}}
			_, err := svc.handleGrokMediaErrorResponse(context.Background(), &http.Response{
				StatusCode: status, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"code":"permission-denied","error":"Content violates usage guidelines. provider-private"}`)),
			}, c, &Account{ID: 1, Platform: PlatformGrok, Type: AccountTypeAPIKey}, "remote-request", "grok-imagine-image")
			require.Error(t, err)
			var failover *UpstreamFailoverError
			require.False(t, errors.As(err, &failover))
			require.True(t, IsResponseCommitted(c))
			wantStatus, wantType, wantMsg := ProtectedUpstreamError(c, status)
			require.Contains(t, rec.Body.String(), wantType)
			require.Contains(t, rec.Body.String(), wantMsg)
			require.Contains(t, rec.Body.String(), "local-request")
			require.NotContains(t, rec.Body.String(), "provider-private")
			require.NotContains(t, rec.Body.String(), "Content violates")
			require.NotContains(t, rec.Body.String(), "remote-request")
			if started {
				require.Equal(t, http.StatusOK, rec.Code)
				requireGatewayKeepaliveErrorEvent(t, rec.Body.String(), ": queue ping\n\n")
				require.Equal(t, "text/event-stream", rec.Result().Header.Get("Content-Type"))
				streamErr, ok := GetOpsStreamError(c)
				require.True(t, ok)
				require.Equal(t, wantStatus, streamErr.IntendedStatus)
			} else {
				require.Equal(t, wantStatus, rec.Code)
				require.NotContains(t, rec.Body.String(), "event: error")
			}
		}
	}
}

func TestGrokMergeWritesFlatErrorAfterCommittedSSE(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/edits", nil)
	c.Header("Content-Type", "text/event-stream")
	_, err := c.Writer.WriteString(": queue ping\n\n")
	require.NoError(t, err)
	c.Writer.Flush()
	writeGrokMediaErrorResponse(c, http.StatusBadRequest, "invalid_request_error", "invalid image", "invalid_image")
	payload := requireGatewayKeepaliveErrorEvent(t, rec.Body.String(), ": queue ping\n\n")
	require.Equal(t, "invalid image", payload.Error.Message)
	require.Contains(t, rec.Body.String(), `"code":"invalid_image"`)
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestNativeAnthropicMergeTerminalUsageForbidsFailover(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, started := range []bool{false, true} {
		for _, protected := range []bool{false, true} {
			body := []byte(`{"model":"deepseek-chat","stream":true,"messages":[{"role":"user","content":"hello"}]}`)
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(string(body)))
			SetUpstreamErrorProtectionEnabled(c, protected)
			if started {
				c.Header("Content-Type", "text/event-stream")
				_, err := c.Writer.WriteString(": queue ping\n\n")
				require.NoError(t, err)
				c.Writer.Flush()
			}
			wire := `{"type":"error","model":"deepseek-actual","error":{"type":"overloaded_error","message":"provider-private"},"usage":{"input_tokens":13,"output_tokens":2,"server_tool_use":{"tool_search":2}}}`
			upstream := &httpUpstreamRecorder{resp: &http.Response{StatusCode: http.StatusServiceUnavailable, Header: http.Header{},
				Body: io.NopCloser(strings.NewReader(wire))}}
			svc := &OpenAIGatewayService{cfg: rawChatCompletionsTestConfig(), httpUpstream: upstream}
			account := newDeepSeekToolsAccount()
			account.Platform = PlatformDeepseek
			account.Credentials["base_url"] = "https://api.deepseek.com/anthropic"
			account.Credentials["api_protocol"] = APIProtocolAnthropic
			ctx := context.WithValue(context.Background(), chatNativeContextKey{}, &chatNativeRequest{extensions: true})
			result, err := svc.forwardAnthropicViaNativeAnthropicEndpoint(ctx, c, account, body, "")
			require.Error(t, err)
			var failover *UpstreamFailoverError
			require.False(t, errors.As(err, &failover), "authoritative tool usage makes this request billable, even on HTTP 503")
			require.NotNil(t, result)
			require.Equal(t, 13, result.Usage.InputTokens)
			require.Equal(t, 2, result.Usage.OutputTokens)
			require.Equal(t, "/v1/messages", result.UpstreamEndpoint)
			require.Equal(t, "deepseek-actual", result.UpstreamResponseModel)
			require.NotNil(t, result.ChatAuxiliaryUsage)
			cost, err := result.ChatAuxiliaryUsage.cost()
			require.NoError(t, err)
			require.Positive(t, cost)
			require.True(t, IsResponseCommitted(c))
			if protected {
				require.NotContains(t, rec.Body.String(), "provider-private")
			}
			if started {
				requireGatewayKeepaliveErrorEvent(t, rec.Body.String(), ": queue ping\n\n")
				require.Equal(t, http.StatusOK, rec.Code)
			} else {
				require.Equal(t, http.StatusServiceUnavailable, rec.Code)
			}
		}
	}
}
