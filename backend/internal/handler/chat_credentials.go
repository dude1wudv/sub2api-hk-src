package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

func (h *ChatHandler) ReconnectPreset(c *gin.Context) {
	userID, ok := chatOwner(c)
	if !ok {
		return
	}
	id, ok := chatID(c)
	if !ok {
		return
	}
	if err := h.service.ReconnectPreset(c.Request.Context(), userID, id); err != nil {
		chatError(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}
