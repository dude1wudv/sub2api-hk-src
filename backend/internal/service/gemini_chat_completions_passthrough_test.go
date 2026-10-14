//go:build unit

package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func geminiChatPassthroughAccount() *Account {
	return &Account{
		ID:       701,
		Platform: PlatformGemini,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":  "dummy-gemini-key",
			"base_url": "https://relay.example/v1",
		},
		Extra: map[string]any{"gemini_chat_completions_passthrough": true},
	}
}

func TestAccountUsesGeminiChatCompletionsRequiresExplicitEligibleAccount(t *testing.T) {
	base := geminiChatPassthroughAccount()
	tests := []struct {
		name   string
		mutate func(*Account)
		want   bool
	}{
		{name: "enabled", want: true},
		{name: "disabled by default", mutate: func(a *Account) { a.Extra = nil }},
		{name: "explicit false", mutate: func(a *Account) { a.Extra["gemini_chat_completions_passthrough"] = false }},
		{name: "string false is not a boolean", mutate: func(a *Account) { a.Extra["gemini_chat_completions_passthrough"] = "false" }},
		{name: "string true is not a boolean", mutate: func(a *Account) { a.Extra["gemini_chat_completions_passthrough"] = "true" }},
		{name: "wrong platform", mutate: func(a *Account) { a.Platform = PlatformOpenAI }},
		{name: "wrong type", mutate: func(a *Account) { a.Type = AccountTypeOAuth }},
		{name: "missing base URL", mutate: func(a *Account) { delete(a.Credentials, "base_url") }},
		{name: "blank base URL", mutate: func(a *Account) { a.Credentials["base_url"] = "  " }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account := *base
			account.Credentials = map[string]any{"api_key": "dummy-gemini-key", "base_url": "https://relay.example/v1"}
			account.Extra = map[string]any{"gemini_chat_completions_passthrough": true}
			if tt.mutate != nil {
				tt.mutate(&account)
			}
			require.Equal(t, tt.want, account.UsesGeminiChatCompletions())
		})
	}
	require.False(t, (*Account)(nil).UsesGeminiChatCompletions())
}

func TestGeminiChatCompletionsPassthroughCredentialsOnlyApplyWhenEnabled(t *testing.T) {
	account := geminiChatPassthroughAccount()
	require.Equal(t, "https://relay.example/v1", account.GetOpenAIBaseURL())
	require.Equal(t, "dummy-gemini-key", account.GetOpenAIProtocolAPIKey())

	account.Extra = nil
	require.Empty(t, account.GetOpenAIBaseURL(), "disabled Gemini must not fall back to the OpenAI API base")
	require.Empty(t, account.GetOpenAIProtocolAPIKey())
}

func TestForwardGeminiChatCompletionsPreservesToolCycleAndMapsUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"gemini-public","stream":false,"messages":[{"role":"user","content":"read file"},{"role":"assistant","tool_calls":[{"id":"call_1","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"sample.txt\"}"}}]},{"role":"tool","tool_call_id":"call_1","content":"sample contents"},{"role":"user","content":"summarize"}],"tools":[{"type":"function","function":{"name":"read_file","description":"Read a file","parameters":{"type":"object","properties":{"path":{"type":"string"}},"required":["path"]}}}],"tool_choice":"auto"}`)
	upstreamResponse := `{"id":"chatcmpl_dummy","model":"gemini-upstream","choices":[{"index":0,"message":{"role":"assistant","tool_calls":[{"id":"call_2","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"next.txt\"}"}}]},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":100,"completion_tokens":12,"total_tokens":112,"prompt_tokens_details":{"cached_tokens":20,"cache_creation_tokens":10}}}`
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(upstreamResponse)),
	}}
	account := geminiChatPassthroughAccount()
	account.Credentials["model_mapping"] = map[string]any{"gemini-public": "gemini-upstream"}
	svc := &OpenAIGatewayService{cfg: rawChatCompletionsTestConfig(), httpUpstream: upstream}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))

	result, err := svc.ForwardGeminiChatCompletions(context.Background(), c, account, body)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, "/v1/chat/completions", upstream.lastReq.URL.Path)
	require.Equal(t, "Bearer dummy-gemini-key", upstream.lastReq.Header.Get("Authorization"))
	require.Equal(t, "gemini-upstream", gjson.GetBytes(upstream.lastBody, "model").String())
	require.Equal(t, "read_file", gjson.GetBytes(upstream.lastBody, "tools.0.function.name").String())
	require.Equal(t, "auto", gjson.GetBytes(upstream.lastBody, "tool_choice").String())
	require.Equal(t, `{"path":"sample.txt"}`, gjson.GetBytes(upstream.lastBody, "messages.1.tool_calls.0.function.arguments").String())
	require.Equal(t, "call_1", gjson.GetBytes(upstream.lastBody, "messages.2.tool_call_id").String())
	require.Equal(t, "tool_calls", gjson.GetBytes(recorder.Body.Bytes(), "choices.0.finish_reason").String())
	require.Equal(t, "{\"path\":\"next.txt\"}", gjson.GetBytes(recorder.Body.Bytes(), "choices.0.message.tool_calls.0.function.arguments").String())
	require.Equal(t, 70, result.Usage.InputTokens, "read and creation cache tokens are removed from billable input")
	require.Equal(t, 20, result.Usage.CacheReadInputTokens)
	require.Equal(t, 10, result.Usage.CacheCreationInputTokens)
	require.Equal(t, 12, result.Usage.OutputTokens)
}

func TestForwardGeminiChatCompletionsResetsResponseObservationForEachAttempt(t *testing.T) {
	gin.SetMode(gin.TestMode)
	response := func(model string) *http.Response {
		payload := `{"id":"chatcmpl_dummy","model":"` + model + `","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(payload)),
		}
	}
	upstream := &httpUpstreamRecorder{responses: []*http.Response{response("first-upstream-model"), response("second-upstream-model")}}
	svc := &OpenAIGatewayService{cfg: rawChatCompletionsTestConfig(), httpUpstream: upstream}
	account := geminiChatPassthroughAccount()
	body := []byte(`{"model":"gemini-public","messages":[{"role":"user","content":"hello"}]}`)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))

	first, err := svc.ForwardGeminiChatCompletions(context.Background(), c, account, body)
	require.NoError(t, err)
	require.Equal(t, "first-upstream-model", first.UpstreamResponseModel)
	require.False(t, first.UpstreamResponseModelConflict)

	second, err := svc.ForwardGeminiChatCompletions(context.Background(), c, account, body)
	require.NoError(t, err)
	require.Equal(t, "second-upstream-model", second.UpstreamResponseModel)
	require.False(t, second.UpstreamResponseModelConflict, "a retry must not inherit response observations from the prior attempt")
}

func TestForwardGeminiChatCompletionsPreservesStreamToolFragmentsAndUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	body := []byte(`{"model":"gemini-public","stream":true,"messages":[{"role":"user","content":"read file"}],"tools":[{"type":"function","function":{"name":"read_file","parameters":{"type":"object"}}}],"tool_choice":"auto"}`)
	stream := strings.Join([]string{
		`data: {"id":"chatcmpl_dummy","object":"chat.completion.chunk","model":"gemini-upstream","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_","type":"function","function":{"name":"read_file","arguments":"{\"path\":"}}]},"finish_reason":null}]}`,
		`data: {"id":"chatcmpl_dummy","object":"chat.completion.chunk","model":"gemini-upstream","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"sample.txt\"}"}}]},"finish_reason":null}]}`,
		`data: {"id":"chatcmpl_dummy","object":"chat.completion.chunk","model":"gemini-upstream","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`,
		`data: {"id":"chatcmpl_dummy","object":"chat.completion.chunk","model":"gemini-upstream","choices":[],"usage":{"prompt_tokens":20,"completion_tokens":5,"total_tokens":25,"prompt_tokens_details":{"cached_tokens":7}}}`,
		"data: [DONE]",
		"",
	}, "\n")
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader(stream)),
	}}
	svc := &OpenAIGatewayService{cfg: rawChatCompletionsTestConfig(), httpUpstream: upstream}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))

	result, err := svc.ForwardGeminiChatCompletions(context.Background(), c, geminiChatPassthroughAccount(), body)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "/v1/chat/completions", upstream.lastReq.URL.Path)
	require.Equal(t, "read_file", gjson.GetBytes(upstream.lastBody, "tools.0.function.name").String())
	require.True(t, gjson.GetBytes(upstream.lastBody, "stream_options.include_usage").Bool())
	require.Contains(t, recorder.Body.String(), `"arguments":"{\"path\":"`)
	require.Contains(t, recorder.Body.String(), `"arguments":"\"sample.txt\"}"`)
	require.Contains(t, recorder.Body.String(), `"finish_reason":"tool_calls"`)
	require.Contains(t, recorder.Body.String(), `data: [DONE]`)
	require.Equal(t, 13, result.Usage.InputTokens)
	require.Equal(t, 7, result.Usage.CacheReadInputTokens)
	require.Equal(t, 5, result.Usage.OutputTokens)
}

func TestForwardGeminiChatCompletionsUsesRawErrorHandling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusBadRequest,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"dummy upstream rejection"}}`)),
	}}
	svc := &OpenAIGatewayService{cfg: rawChatCompletionsTestConfig(), httpUpstream: upstream}
	body := []byte(`{"model":"gemini-public","messages":[{"role":"user","content":"hello"}]}`)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))

	result, err := svc.ForwardGeminiChatCompletions(context.Background(), c, geminiChatPassthroughAccount(), body)
	require.Error(t, err)
	require.Nil(t, result)
	require.GreaterOrEqual(t, recorder.Code, http.StatusBadRequest)
	require.Contains(t, recorder.Body.String(), "dummy upstream rejection")
}
