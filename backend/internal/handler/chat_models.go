package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"strconv"
)

func (h *ChatHandler) Catalog(c *gin.Context) {
	userID, ok := chatOwner(c)
	if !ok {
		return
	}
	groupID, err := strconv.ParseInt(c.Query("group_id"), 10, 64)
	if err != nil || groupID <= 0 {
		chatError(c, service.ErrChatInvalid)
		return
	}
	catalog, err := h.models.Catalog(c.Request.Context(), userID, groupID, c.Query("search"))
	if err != nil {
		chatError(c, err)
		return
	}
	response.Success(c, catalog)
}

func (h *ChatHandler) ListPresets(c *gin.Context) {
	userID, ok := chatOwner(c)
	if !ok {
		return
	}
	presets, err := h.models.ListPresets(c.Request.Context(), userID)
	if err != nil {
		chatError(c, err)
		return
	}
	response.Success(c, presets)
}

func (h *ChatHandler) CreatePreset(c *gin.Context) {
	userID, ok := chatOwner(c)
	if !ok {
		return
	}
	var req struct {
		GroupID     int64  `json:"group_id"`
		Model       string `json:"model"`
		DisplayName string `json:"display_name"`
	}
	if err := decodeChatJSON(c, &req, 8192); err != nil {
		chatError(c, err)
		return
	}
	preset, err := h.models.CreatePreset(c.Request.Context(), userID, req.GroupID, req.Model, req.DisplayName)
	if err != nil {
		chatError(c, err)
		return
	}
	response.Success(c, preset)
}

func (h *ChatHandler) UpdatePreset(c *gin.Context) {
	userID, ok := chatOwner(c)
	if !ok {
		return
	}
	id, ok := chatID(c)
	if !ok {
		return
	}
	var req service.ChatPresetPatch
	if err := decodeChatJSON(c, &req, 8192); err != nil {
		chatError(c, err)
		return
	}
	if req.GroupID == nil && req.Model == nil && req.DisplayName == nil && req.IsDefault == nil {
		chatError(c, service.ErrChatInvalid)
		return
	}
	preset, err := h.models.UpdatePreset(c.Request.Context(), userID, id, req)
	if err != nil {
		chatError(c, err)
		return
	}
	response.Success(c, preset)
}

func (h *ChatHandler) OrderPresets(c *gin.Context) {
	userID, ok := chatOwner(c)
	if !ok {
		return
	}
	var req struct {
		IDs []uuid.UUID `json:"ids"`
	}
	if err := decodeChatJSON(c, &req, 65536); err != nil {
		chatError(c, err)
		return
	}
	if req.IDs == nil {
		chatError(c, service.ErrChatInvalid)
		return
	}
	if err := h.models.OrderPresets(c.Request.Context(), userID, req.IDs); err != nil {
		chatError(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *ChatHandler) DeletePreset(c *gin.Context) {
	userID, ok := chatOwner(c)
	if !ok {
		return
	}
	id, ok := chatID(c)
	if !ok {
		return
	}
	if err := h.models.DeletePreset(c.Request.Context(), userID, id); err != nil {
		chatError(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}
