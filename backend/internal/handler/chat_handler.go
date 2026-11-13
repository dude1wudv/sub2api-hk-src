package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ChatHandler struct {
	service *service.ChatService
	models  *service.ChatModelService
}

func NewChatHandler(chat *service.ChatService, models *service.ChatModelService) *ChatHandler {
	return &ChatHandler{service: chat, models: models}
}

func (h *ChatHandler) Stop() error { return h.service.Stop() }

func chatOwner(c *gin.Context) (int64, bool) {
	c.Header("Cache-Control", "private, no-store")
	subject, ok := servermiddleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "authentication required")
		return 0, false
	}
	return subject.UserID, true
}

func chatID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil || id == uuid.Nil {
		chatError(c, service.ErrChatNotFound)
		return uuid.Nil, false
	}
	return id, true
}

func chatError(c *gin.Context, err error) {
	if !response.ErrorFrom(c, err) {
		response.InternalError(c, "chat operation failed")
	}
}

func decodeChatJSON(c *gin.Context, target any, limit int64) error {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		var large *http.MaxBytesError
		if errors.As(err, &large) {
			return service.ErrChatTooLarge
		}
		return service.ErrChatInvalid
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		var large *http.MaxBytesError
		if errors.As(err, &large) {
			return service.ErrChatTooLarge
		}
		return service.ErrChatInvalid
	}
	return nil
}

func (h *ChatHandler) ListSessions(c *gin.Context) {
	userID, ok := chatOwner(c)
	if !ok {
		return
	}
	limit := 50
	if raw := c.Query("limit"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 1 {
			chatError(c, service.ErrChatInvalid)
			return
		}
		limit = v
	}
	page, err := h.service.ListSessions(c.Request.Context(), userID, c.Query("cursor"), limit)
	if err != nil {
		chatError(c, err)
		return
	}
	response.Success(c, page)
}

func (h *ChatHandler) GetSession(c *gin.Context) {
	userID, ok := chatOwner(c)
	if !ok {
		return
	}
	id, ok := chatID(c)
	if !ok {
		return
	}
	record, err := h.service.GetSession(c.Request.Context(), userID, id)
	if err != nil {
		chatError(c, err)
		return
	}
	response.Success(c, gin.H{"session": record.Snapshot, "revision": record.Revision})
}

func (h *ChatHandler) SaveSession(c *gin.Context) {
	userID, ok := chatOwner(c)
	if !ok {
		return
	}
	id, ok := chatID(c)
	if !ok {
		return
	}
	var req struct {
		Session          json.RawMessage `json:"session"`
		ExpectedRevision *int64          `json:"expected_revision"`
	}
	if err := decodeChatJSON(c, &req, service.ChatMaxSessionBytes+1024); err != nil {
		chatError(c, err)
		return
	}
	if req.ExpectedRevision == nil {
		chatError(c, service.ErrChatInvalid)
		return
	}
	record, err := h.service.SaveSession(c.Request.Context(), userID, id, req.Session, *req.ExpectedRevision)
	if err != nil {
		chatError(c, err)
		return
	}
	response.Success(c, gin.H{"session": record.Snapshot, "revision": record.Revision})
}

func (h *ChatHandler) RenameSession(c *gin.Context) {
	userID, ok := chatOwner(c)
	if !ok {
		return
	}
	id, ok := chatID(c)
	if !ok {
		return
	}
	var req struct {
		Title            *string `json:"title"`
		ExpectedRevision *int64  `json:"expected_revision"`
	}
	if err := decodeChatJSON(c, &req, 8192); err != nil {
		chatError(c, err)
		return
	}
	if req.Title == nil || req.ExpectedRevision == nil {
		chatError(c, service.ErrChatInvalid)
		return
	}
	record, err := h.service.RenameSession(c.Request.Context(), userID, id, *req.Title, *req.ExpectedRevision)
	if err != nil {
		chatError(c, err)
		return
	}
	response.Success(c, gin.H{"session": record.Snapshot, "revision": record.Revision})
}

func (h *ChatHandler) GetPreferences(c *gin.Context) {
	userID, ok := chatOwner(c)
	if !ok {
		return
	}
	preferences, revision, err := h.service.GetPreferences(c.Request.Context(), userID)
	if err != nil {
		chatError(c, err)
		return
	}
	response.Success(c, gin.H{"preferences": preferences, "history_revision": revision})
}

func (h *ChatHandler) SavePreferences(c *gin.Context) {
	userID, ok := chatOwner(c)
	if !ok {
		return
	}
	var raw json.RawMessage
	if err := decodeChatJSON(c, &raw, service.ChatMaxPreferenceBytes); err != nil {
		chatError(c, err)
		return
	}
	if err := h.service.SavePreferences(c.Request.Context(), userID, raw); err != nil {
		chatError(c, err)
		return
	}
	h.GetPreferences(c)
}
