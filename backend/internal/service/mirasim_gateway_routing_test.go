//go:build unit

package service

import (
	"context"
	"crypto/sha256"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/mirasim"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestMirasimModelsRouteAllIngressesToChatCompletions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	privateKey, err := mirasim.GenerateDeviceKey()
	require.NoError(t, err)
	privateKeyPEM, err := mirasim.MarshalPrivateKeyPEM(privateKey)
	require.NoError(t, err)
	const refreshToken = "mirasim-test-refresh"

	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth/refresh" {
			http.NotFound(w, r)
			return
		}
		_, _ = io.WriteString(w, `{"access_token":"access-test","refresh_token":"mirasim-test-refresh","expires_in":3600}`)
	}))
	defer authServer.Close()
	relayServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/device/session" {
			http.NotFound(w, r)
			return
		}
		_, _ = io.WriteString(w, `{"ticket":"ticket-test","expires_in":900}`)
	}))
	defer relayServer.Close()

	testClient, err := mirasim.NewClient(refreshToken, privateKey, relayServer.URL, authServer.URL, "", "", "")
	require.NoError(t, err)
	const accountID = int64(928731)
	session := &mirasimSession{
		client:         testClient,
		initialToken:   refreshToken,
		currentToken:   refreshToken,
		persistedToken: refreshToken,
		keyHash:        sha256.Sum256([]byte(privateKeyPEM)),
	}
	mirasimSessions.Store(accountID, session)
	defer mirasimSessions.Delete(accountID)

	chatResponse := `{"id":"chatcmpl-mirasim","object":"chat.completion","model":"mapped-model","choices":[{"index":0,"message":{"role":"assistant","content":null,"tool_calls":[{"id":"call-1","type":"function","function":{"name":"lookup","arguments":"{\"q\":\"x\"}"}}]},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":11,"completion_tokens":7,"total_tokens":18}}`
	cases := []struct {
		name      string
		model     string
		path      string
		body      string
		forward   func(*OpenAIGatewayService, *gin.Context, *Account, []byte) (*OpenAIForwardResult, error)
		wantTools bool
	}{
		{
			name:  "chat ingress GLM",
			model: "glm-5.3-flash", path: "/v1/chat/completions",
			body: `{"model":"glm-5.3-flash","messages":[{"role":"user","content":"hi"}],"tools":[{"type":"function","function":{"name":"lookup","parameters":{"type":"object"}}}],"stream":false}`,
			forward: func(s *OpenAIGatewayService, c *gin.Context, a *Account, b []byte) (*OpenAIForwardResult, error) {
				return s.ForwardAsChatCompletions(context.Background(), c, a, b, "", "")
			}, wantTools: true,
		},
		{
			name:  "Responses ingress GLM",
			model: "glm-5.3-flash", path: "/v1/responses",
			body: `{"model":"glm-5.3-flash","input":"hi","tools":[{"type":"function","name":"lookup","parameters":{"type":"object"}}],"stream":false}`,
			forward: func(s *OpenAIGatewayService, c *gin.Context, a *Account, b []byte) (*OpenAIForwardResult, error) {
				return s.Forward(context.Background(), c, a, b)
			}, wantTools: true,
		},
		{
			name:  "Messages ingress Kimi",
			model: "kimi-k3", path: "/v1/messages",
			body: `{"model":"kimi-k3","max_tokens":64,"messages":[{"role":"user","content":"hi"}],"tools":[{"name":"lookup","input_schema":{"type":"object","properties":{"q":{"type":"string"}}}}],"stream":false}`,
			forward: func(s *OpenAIGatewayService, c *gin.Context, a *Account, b []byte) (*OpenAIForwardResult, error) {
				return s.ForwardAsAnthropic(context.Background(), c, a, b, "", "")
			}, wantTools: true,
		},
	}
	for i, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			account := &Account{
				ID: accountID, Name: "Mirasim test", Platform: PlatformMirasim, Type: AccountTypeAPIKey, Concurrency: 1,
				Credentials: map[string]any{
					"refresh_token": refreshToken, "private_key": privateKeyPEM,
					"api_protocol": APIProtocolAdaptive,
					"base_url":     "https://relay.mirasim.ai/v1",
					"api_base_urls": map[string]any{
						APIProtocolChatCompletions: "https://relay.mirasim.ai/v1",
						APIProtocolAnthropic:       "https://relay.mirasim.ai",
						APIProtocolResponses:       "https://relay.mirasim.ai/v1",
					},
					"model_mapping": map[string]any{tt.model: "mapped-model"},
				},
			}
			// Separate credentials session state per case while sharing the local fake relay.
			mirasimSessions.Store(accountID+int64(i), session)
			defer mirasimSessions.Delete(accountID + int64(i))
			account.ID = accountID + int64(i)
			upstream := &httpUpstreamRecorder{resp: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(chatResponse)),
			}}
			svc := &OpenAIGatewayService{cfg: rawChatCompletionsTestConfig(), httpUpstream: upstream}
			body := []byte(tt.body)
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader(tt.body))
			ctx.Request.Header.Set("Content-Type", "application/json")
			result, callErr := tt.forward(svc, ctx, account, body)
			require.NoError(t, callErr)
			require.NotNil(t, upstream.lastReq)
			require.Equal(t, "/v1/chat/completions", upstream.lastReq.URL.Path)
			require.Equal(t, "mapped-model", gjson.GetBytes(upstream.lastBody, "model").String())
			if tt.wantTools {
				require.Len(t, gjson.GetBytes(upstream.lastBody, "tools").Array(), 1)
			}
			require.NotNil(t, result)
			require.Equal(t, 11, result.Usage.InputTokens)
			require.Equal(t, 7, result.Usage.OutputTokens)
			if tt.name == "Responses ingress GLM" {
				require.Contains(t, recorder.Body.String(), "function_call")
			} else if tt.name == "Messages ingress Kimi" {
				require.Contains(t, recorder.Body.String(), "tool_use")
			}
		})
	}
}
