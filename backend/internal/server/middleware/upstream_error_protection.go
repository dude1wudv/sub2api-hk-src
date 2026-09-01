package middleware

import (
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// UpstreamErrorProtection binds the configured client-facing upstream error policy.
func UpstreamErrorProtection(cfg *config.Config) gin.HandlerFunc {
	enabled := true
	if cfg != nil {
		enabled = cfg.Gateway.UpstreamErrorProtection
	}
	return func(c *gin.Context) {
		service.SetUpstreamErrorProtectionEnabled(c, enabled)
		c.Next()
	}
}
