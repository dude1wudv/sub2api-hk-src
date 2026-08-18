//go:build unit

package service

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestAccountTestService_TestAccountConnection_DeepSeekProtocols(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name              string
		protocol          string
		baseURL           string
		model             string
		prompt            string
		stream            string
		wantURL           string
		wantAuthorization string
		wantXAPIKey       string
		assertPayload     func(*testing.T, []byte)
	}{
		{
			name:              "chat completions",
			protocol:          APIProtocolChatCompletions,
			baseURL:           "https://deepseek.test",
			model:             "deepseek-chat",
			prompt:            "chat probe",
			stream:            "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n",
			wantURL:           "https://deepseek.test/v1/chat/completions",
			wantAuthorization: "Bearer ds-test-key",
			assertPayload: func(t *testing.T, body []byte) {
				t.Helper()
				require.Equal(t, "deepseek-chat", gjson.GetBytes(body, "model").String())
				require.Equal(t, "user", gjson.GetBytes(body, "messages.0.role").String())
				require.Equal(t, "chat probe", gjson.GetBytes(body, "messages.0.content").String())
				require.True(t, gjson.GetBytes(body, "stream").Bool())
			},
		},
		{
			name:              "responses",
			protocol:          APIProtocolResponses,
			baseURL:           "https://deepseek.test",
			model:             "deepseek-v4-flash",
			stream:            "data: {\"type\":\"response.output_text.delta\",\"delta\":\"ok\"}\n\ndata: {\"type\":\"response.completed\"}\n\n",
			wantURL:           "https://deepseek.test/responses",
			wantAuthorization: "Bearer ds-test-key",
			assertPayload: func(t *testing.T, body []byte) {
				t.Helper()
				require.Equal(t, "deepseek-v4-flash", gjson.GetBytes(body, "model").String())
				require.Equal(t, "user", gjson.GetBytes(body, "input.0.role").String())
				require.Equal(t, "input_text", gjson.GetBytes(body, "input.0.content.0.type").String())
				require.True(t, gjson.GetBytes(body, "stream").Bool())
				require.False(t, gjson.GetBytes(body, "store").Bool())
			},
		},
		{
			name:        "anthropic",
			protocol:    APIProtocolAnthropic,
			baseURL:     "https://deepseek.test/anthropic",
			model:       "deepseek-v4-pro",
			stream:      "data: {\"type\":\"content_block_delta\",\"delta\":{\"text\":\"ok\"}}\n\ndata: {\"type\":\"message_stop\"}\n\n",
			wantURL:     "https://deepseek.test/anthropic/v1/messages",
			wantXAPIKey: "ds-test-key",
			assertPayload: func(t *testing.T, body []byte) {
				t.Helper()
				require.Equal(t, "deepseek-v4-pro", gjson.GetBytes(body, "model").String())
				require.Equal(t, "user", gjson.GetBytes(body, "messages.0.role").String())
				require.Equal(t, "text", gjson.GetBytes(body, "messages.0.content.0.type").String())
				require.True(t, gjson.GetBytes(body, "stream").Bool())
			},
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			account := &Account{
				ID:          int64(i + 1),
				Platform:    PlatformDeepseek,
				Type:        AccountTypeAPIKey,
				Concurrency: 1,
				Credentials: map[string]any{
					"api_key":      "ds-test-key",
					"api_protocol": tt.protocol,
					"base_url":     tt.baseURL,
				},
			}
			repo := &mockAccountRepoForGemini{accountsByID: map[int64]*Account{account.ID: account}}
			upstream := &httpUpstreamRecorder{resp: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
				Body:       io.NopCloser(strings.NewReader(tt.stream)),
			}}
			svc := &AccountTestService{
				accountRepo:  repo,
				httpUpstream: upstream,
				cfg:          &config.Config{},
			}

			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/test", nil)

			require.NoError(t, svc.TestAccountConnection(c, account.ID, tt.model, tt.prompt, AccountTestModeDefault))
			require.NotNil(t, upstream.lastReq)
			require.Equal(t, tt.wantURL, upstream.lastReq.URL.String())
			require.Empty(t, upstream.lastReq.URL.RawQuery)
			require.Equal(t, tt.wantAuthorization, upstream.lastReq.Header.Get("Authorization"))
			require.Equal(t, tt.wantXAPIKey, upstream.lastReq.Header.Get("x-api-key"))
			tt.assertPayload(t, upstream.lastBody)
			require.Contains(t, recorder.Body.String(), `"type":"content"`)
			require.Contains(t, recorder.Body.String(), `"text":"ok"`)
			require.Contains(t, recorder.Body.String(), `"type":"test_complete","success":true`)
		})
	}
}
