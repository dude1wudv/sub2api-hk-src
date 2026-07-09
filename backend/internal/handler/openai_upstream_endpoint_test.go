//go:build unit

package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai_compat"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestResolveOpenAIUpstreamEndpoint_GrokOAuthChatCompletions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	InboundEndpointMiddleware()(c)

	account := &service.Account{
		ID:       590,
		Platform: service.PlatformGrok,
		Type:     service.AccountTypeOAuth,
	}
	require.Equal(t, EndpointChatCompletions, resolveOpenAIUpstreamEndpoint(c, account))
}

func TestResolveOpenAIUpstreamEndpoint_GrokOAuthResponses(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	InboundEndpointMiddleware()(c)

	account := &service.Account{
		ID:       590,
		Platform: service.PlatformGrok,
		Type:     service.AccountTypeOAuth,
	}
	require.Equal(t, EndpointResponses, resolveOpenAIUpstreamEndpoint(c, account))
}

func TestResolveOpenAIUpstreamEndpoint_OpenAIAPIKeyRawChat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	account := &service.Account{
		ID:       1,
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeAPIKey,
		Extra: map[string]any{
			openai_compat.ExtraKeyResponsesMode: string(openai_compat.ResponsesSupportModeForceChatCompletions),
		},
	}
	require.False(t, openai_compat.ShouldUseResponsesAPI(account.Extra))
	require.Equal(t, EndpointChatCompletions, resolveOpenAIUpstreamEndpoint(c, account))
}
