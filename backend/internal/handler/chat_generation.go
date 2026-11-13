package handler

import (
	"errors"
	"mime"
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *ChatHandler) Generate(c *gin.Context) {
	owner, ok := chatOwner(c)
	if !ok {
		return
	}
	id, ok := chatID(c)
	if !ok {
		return
	}
	var req service.ChatGenerateRequest
	if err := decodeChatJSON(c, &req, 8192); err != nil {
		chatError(c, err)
		return
	}
	if err := h.service.Generate(c.Request.Context(), owner, id, req, ip.GetClientIP(c), c.Writer); err != nil && !c.Writer.Written() {
		chatError(c, err)
	}
}

func (h *ChatHandler) StopGeneration(c *gin.Context) {
	owner, ok := chatOwner(c)
	if !ok {
		return
	}
	id, ok := chatID(c)
	if !ok {
		return
	}
	var req struct {
		RunID uuid.UUID `json:"run_id"`
	}
	if err := decodeChatJSON(c, &req, 1024); err != nil {
		chatError(c, err)
		return
	}
	if req.RunID == uuid.Nil {
		chatError(c, service.ErrChatNotFound)
		return
	}
	if err := h.service.StopGeneration(c.Request.Context(), owner, id, req.RunID); err != nil {
		chatError(c, err)
		return
	}
	response.Success(c, gin.H{})
}

func (h *ChatHandler) DeleteSessions(c *gin.Context) {
	owner, ok := chatOwner(c)
	if !ok {
		return
	}
	var id *uuid.UUID
	if c.Param("id") != "" {
		value, ok := chatID(c)
		if !ok {
			return
		}
		id = &value
	}
	if err := h.service.DeleteSessions(c.Request.Context(), owner, id); err != nil {
		chatError(c, err)
		return
	}
	response.Success(c, gin.H{})
}

func (h *ChatHandler) UploadAttachment(c *gin.Context) {
	owner, ok := chatOwner(c)
	if !ok {
		return
	}
 var importID *uuid.UUID
 if c.Param("id") != "" {
  id, ok := chatID(c); if !ok { return }
  if _, err := h.service.GetImport(c.Request.Context(), owner, id); err != nil { chatError(c, err); return }
  importID = &id
 }
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, service.ChatMaxFileBytes+(1<<20))
	if err := c.Request.ParseMultipartForm(1 << 20); err != nil {
		var large *http.MaxBytesError
		if errors.As(err, &large) {
			chatError(c, service.ErrChatTooLarge)
		} else {
			chatError(c, service.ErrChatInvalid)
		}
		return
	}
	defer c.Request.MultipartForm.RemoveAll()
	if len(c.Request.MultipartForm.File) != 1 || len(c.Request.MultipartForm.File["file"]) != 1 {
		chatError(c, service.ErrChatInvalid)
		return
	}
	header := c.Request.MultipartForm.File["file"][0]
	_, params, err := mime.ParseMediaType(header.Header.Get("Content-Disposition"))
	if err != nil {
		chatError(c, service.ErrChatInvalid)
		return
	}
	source, err := header.Open()
	if err != nil {
		chatError(c, err)
		return
	}
	defer source.Close()
 record, err := h.service.UploadAttachment(c.Request.Context(), owner, params["filename"], c.Request.FormValue("purpose"), source, header.Size, importID)
	if err != nil {
		chatError(c, err)
		return
	}
	response.Success(c, record)
}

func (h *ChatHandler) DownloadAttachment(c *gin.Context) {
	owner, ok := chatOwner(c)
	if !ok {
		return
	}
	id, ok := chatID(c)
	if !ok {
		return
	}
	record, file, err := h.service.OpenAttachment(c.Request.Context(), owner, id)
	if err != nil {
		chatError(c, err)
		return
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		chatError(c, err)
		return
	}
	c.Header("Content-Type", record.MIME)
	c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": record.Name}))
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Security-Policy", "default-src 'none'; sandbox")
	c.Header("Cross-Origin-Resource-Policy", "same-origin")
	http.ServeContent(c.Writer, c.Request, record.Name, stat.ModTime(), file)
}
