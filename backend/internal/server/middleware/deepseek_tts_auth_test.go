package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestDeepSeekTTSSubprotocolAuthentication(t *testing.T) {
	for _, tc := range []struct {
		protocol, auth string
		status         int
	}{
		{"tts, bearer.synthetic-client-key", "", 200},
		{"tts", "Bearer synthetic-header-key", 200},
		{"tts, bearer.a, bearer.b", "", 401},
		{"tts, bearer.a", "Bearer b", 401},
	} {
		router := gin.New()
		router.Use(DeepSeekTTSWebSocketAuth)
		router.GET("/tts", func(c *gin.Context) {
			require.Equal(t, "tts", c.GetHeader("Sec-WebSocket-Protocol"))
			require.Contains(t, c.GetHeader("Authorization"), "Bearer synthetic-")
			c.Status(200)
		})
		req := httptest.NewRequest(http.MethodGet, "/tts", nil)
		req.Header.Set("Sec-WebSocket-Protocol", tc.protocol)
		req.Header.Set("Authorization", tc.auth)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		require.Equal(t, tc.status, response.Code)
	}
}
