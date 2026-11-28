package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

// RegisterCharityRoutes 注册 free.xjtu.shop 公益站公开路由（免登录，按安全客户端 IP 限流）。
func RegisterCharityRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	panelRateLimiter *middleware.PanelRateLimiter,
) {
	if h == nil || h.Charity == nil {
		return
	}
	charity := v1.Group("/charity")
	if panelRateLimiter != nil {
		charity.Use(panelRateLimiter.PublicIP())
	}
	{
		charity.GET("/overview", h.Charity.GetOverview)
		charity.POST("/key-status", h.Charity.LookupKeyStatus)
	}
}
