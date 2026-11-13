package handler

import (
	"encoding/json"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *ChatHandler) BeginImport(c *gin.Context) {
	owner, ok := chatOwner(c)
	if !ok {
		return
	}
	var request struct {
		Expected *int64 `json:"expected_history_revision"`
	}
	if err := decodeChatJSON(c, &request, 1024); err != nil {
		chatError(c, err)
		return
	}
	if request.Expected == nil {
		chatError(c, service.ErrChatInvalid)
		return
	}
	record, err := h.service.BeginImport(c.Request.Context(), owner, *request.Expected)
	if err != nil {
		chatError(c, err)
		return
	}
	response.Success(c, record)
}

func (h *ChatHandler) StageImportSession(c *gin.Context) {
	owner, ok := chatOwner(c)
	if !ok {
		return
	}
	id, ok := chatID(c)
	if !ok {
		return
	}
	sessionID, err := uuid.Parse(c.Param("session_id"))
	if err != nil || sessionID == uuid.Nil {
		chatError(c, service.ErrChatNotFound)
		return
	}
	var request struct {
		Session json.RawMessage `json:"session"`
	}
	if err = decodeChatJSON(c, &request, service.ChatMaxSessionBytes+1024); err != nil {
		chatError(c, err)
		return
	}
	if err = h.service.StageImportSession(c.Request.Context(), owner, id, sessionID, request.Session); err != nil {
		chatError(c, err)
		return
	}
	response.Success(c, gin.H{})
}

func (h *ChatHandler) CommitImport(c *gin.Context) {
	owner, ok := chatOwner(c)
	if !ok {
		return
	}
	id, ok := chatID(c)
	if !ok {
		return
	}
	var request struct {
		Expected *int64 `json:"expected_history_revision"`
	}
	if err := decodeChatJSON(c, &request, 1024); err != nil {
		chatError(c, err)
		return
	}
	if request.Expected == nil {
		chatError(c, service.ErrChatInvalid)
		return
	}
	revision, err := h.service.CommitImport(c.Request.Context(), owner, id, *request.Expected)
	if err != nil {
		chatError(c, err)
		return
	}
	response.Success(c, gin.H{"history_revision": revision})
}

func (h *ChatHandler) AbortImport(c *gin.Context) {
	owner, ok := chatOwner(c)
	if !ok {
		return
	}
	id, ok := chatID(c)
	if !ok {
		return
	}
	if err := h.service.AbortImport(c.Request.Context(), owner, id); err != nil {
		chatError(c, err)
		return
	}
	response.Success(c, gin.H{})
}
