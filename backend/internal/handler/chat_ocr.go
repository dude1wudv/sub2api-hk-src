package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *ChatHandler) ParseAttachment(c *gin.Context) {
	user, ok := chatOwner(c)
	if !ok {
		return
	}
	id, ok := chatID(c)
	if !ok {
		return
	}
	var input struct {
		PresetID  uuid.UUID `json:"preset_id"`
		SessionID uuid.UUID `json:"session_id"`
		Retry     bool      `json:"retry"`
	}
	if err := decodeChatJSON(c, &input, 1024); err != nil {
		chatError(c, err)
		return
	}
	if input.PresetID == uuid.Nil {
		chatError(c, service.ErrChatInvalid)
		return
	}
	record, err := h.service.ParseAttachment(c.Request.Context(), user, id, input.PresetID, input.SessionID, ip.GetClientIP(c), input.Retry)
	if err != nil {
		chatError(c, err)
		return
	}
	response.Success(c, record)
}

func (h *ChatHandler) GetAttachmentParse(c *gin.Context) {
	user, ok := chatOwner(c)
	if !ok {
		return
	}
	id, ok := chatID(c)
	if !ok {
		return
	}
	preset, err := uuid.Parse(c.Query("preset_id"))
	if err != nil || preset == uuid.Nil {
		chatError(c, service.ErrChatInvalid)
		return
	}
	record, err := h.service.GetParsedAttachment(c.Request.Context(), user, id, preset)
	if err != nil {
		chatError(c, err)
		return
	}
	response.Success(c, record)
}
