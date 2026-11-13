package routes

import (
	"github.com/Wei-Shaw/sub2api/internal/handler"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func RegisterChatRoutes(v1 *gin.RouterGroup, h *handler.Handlers, jwtAuth servermiddleware.JWTAuthMiddleware, audit servermiddleware.AuditLogMiddleware, settings *service.SettingService, limiter *servermiddleware.PanelRateLimiter) {
	if h.Chat == nil {
		return
	}
	chat := v1.Group("/chat")
	chat.Use(gin.HandlerFunc(jwtAuth))
	chat.Use(servermiddleware.BackendModeUserGuard(settings))
	chat.Use(limiter.Global())
	chat.Use(gin.HandlerFunc(audit))
	chat.GET("/catalog", h.Chat.Catalog)
	chat.GET("/presets", h.Chat.ListPresets)
	chat.POST("/presets", h.Chat.CreatePreset)
	chat.PUT("/presets/order", h.Chat.OrderPresets)
	chat.PATCH("/presets/:id", h.Chat.UpdatePreset)
	chat.DELETE("/presets/:id", h.Chat.DeletePreset)
	chat.POST("/presets/:id/reconnect", h.Chat.ReconnectPreset)
	chat.GET("/sessions", h.Chat.ListSessions)
	chat.GET("/sessions/:id", h.Chat.GetSession)
	chat.PUT("/sessions/:id", h.Chat.SaveSession)
	chat.PATCH("/sessions/:id", h.Chat.RenameSession)
	chat.POST("/sessions/:id/generate", h.Chat.Generate)
	chat.POST("/sessions/:id/stop", h.Chat.StopGeneration)
	chat.DELETE("/sessions/:id", h.Chat.DeleteSessions)
	chat.DELETE("/sessions", h.Chat.DeleteSessions)
	chat.POST("/attachments", h.Chat.UploadAttachment)
	chat.GET("/attachments/:id/content", h.Chat.DownloadAttachment)
 chat.POST("/imports", h.Chat.BeginImport)
 chat.PUT("/imports/:id/sessions/:session_id", h.Chat.StageImportSession)
 chat.POST("/imports/:id/attachments", h.Chat.UploadAttachment)
 chat.POST("/imports/:id/commit", h.Chat.CommitImport)
 chat.DELETE("/imports/:id", h.Chat.AbortImport)
	chat.GET("/preferences", h.Chat.GetPreferences)
	chat.PUT("/preferences", h.Chat.SavePreferences)
}
