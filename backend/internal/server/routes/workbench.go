package routes

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/middleware"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RegisterWorkbenchRoutes wires the authenticated launch endpoint separately
// from the public confidential token endpoint.
func RegisterWorkbenchRoutes(
	v1 *gin.RouterGroup,
	h *handler.Handlers,
	jwtAuth servermiddleware.JWTAuthMiddleware,
	auditLog servermiddleware.AuditLogMiddleware,
	settingService *service.SettingService,
	panelRateLimiter *servermiddleware.PanelRateLimiter,
	redisClient *redis.Client,
) {
	launch := v1.Group("/workbenches/helios")
	launch.Use(gin.HandlerFunc(jwtAuth))
	launch.Use(servermiddleware.BackendModeUserGuard(settingService))
	launch.Use(panelRateLimiter.Global())
	launch.Use(gin.HandlerFunc(auditLog))
	launch.POST("/launch", h.Workbench.Launch)

	token := v1.Group("/workbenches/helios")
	tokenLimiter := middleware.NewRateLimiter(redisClient)
	token.Use(tokenLimiter.LimitWithOptions("workbench-token", 20, time.Minute, middleware.RateLimitOptions{
		FailureMode: middleware.RateLimitFailClose,
	}))
	token.Use(gin.HandlerFunc(auditLog))
	token.POST("/token", h.Workbench.Token)
}
