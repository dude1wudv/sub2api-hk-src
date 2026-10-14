//go:build unit

package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGetUpstreamEndpointGeminiUsesActualChatCompletionsEndpointWhenSet(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1beta/models/gemini-4-argon:generateContent", nil)
	c.Set(ctxKeyInboundEndpoint, EndpointGeminiModels)

	service.SetActualOpenAIUpstreamEndpoint(c, EndpointChatCompletions)
	require.Equal(t, EndpointChatCompletions, GetUpstreamEndpoint(c, service.PlatformGemini))

	service.ClearActualOpenAIUpstreamEndpoint(c)
	require.Equal(t, EndpointGeminiModels, GetUpstreamEndpoint(c, service.PlatformGemini), "clearing an attempt must restore native Gemini routing")
}
