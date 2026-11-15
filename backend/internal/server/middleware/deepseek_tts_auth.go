package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Run before APIKeyAuth and request logging. Never retain or negotiate a bearer
// credential as the selected WebSocket protocol.
func DeepSeekTTSWebSocketAuth(c *gin.Context) {
	var token string
	protocols := []string{}
	for _, line := range c.Request.Header.Values("Sec-WebSocket-Protocol") {
		for _, raw := range strings.Split(line, ",") {
			value := strings.TrimSpace(raw)
			if strings.HasPrefix(value, "bearer.") {
				if token != "" || len(value) <= len("bearer.") {
					c.AbortWithStatus(http.StatusUnauthorized)
					return
				}
				token = strings.TrimPrefix(value, "bearer.")
			} else if value != "" {
				protocols = append(protocols, value)
			}
		}
	}
	c.Request.Header.Del("Sec-WebSocket-Protocol")
	if len(protocols) > 0 {
		c.Request.Header.Set("Sec-WebSocket-Protocol", strings.Join(protocols, ", "))
	}
	if token != "" {
		auth := c.GetHeader("Authorization")
		if auth != "" && auth != "Bearer "+token {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Request.Header.Set("Authorization", "Bearer "+token)
	}
	c.Next()
}
