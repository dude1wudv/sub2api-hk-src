package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *ChatHandler) SpeechSynthesis(c *gin.Context) {
	user, ok := chatOwner(c)
	if !ok {
		return
	}
	var input service.ChatSpeechRequest
	if err := decodeChatJSON(c, &input, service.ChatMaxSpeechJSONBytes+1024); err != nil {
		chatError(c, err)
		return
	}
	tts := service.ChatTTSInput{Text: input.Text, VoiceID: input.VoiceID, FilterMarkdown: input.FilterMarkdown}
	if _, err := service.ValidateChatTTS(&tts); err != nil {
		chatError(c, err)
		return
	}
	raw, err := json.Marshal(tts)
	if err != nil {
		chatError(c, err)
		return
	}
	err = h.service.Speech(c.Request.Context(), user, input.PresetID, input.SessionID, "tts", ip.GetClientIP(c), bytes.NewReader(raw), "application/json", c.Writer)
	if err != nil && !c.Writer.Written() {
		chatError(c, err)
	}
}

func (h *ChatHandler) SpeechRecognition(c *gin.Context) {
	user, ok := chatOwner(c)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, service.ChatMaxSpeechFileBytes+(64<<10))
	reader, err := c.Request.MultipartReader()
	if err != nil {
		chatError(c, service.ErrChatInvalid)
		return
	}
	fields := map[string]bool{}
	preset, session := uuid.Nil, uuid.Nil
	var audio []byte
	for {
		part, nextErr := reader.NextPart()
		if nextErr == io.EOF {
			break
		}
		if nextErr != nil {
			chatError(c, service.ErrChatInvalid)
			return
		}
		name := part.FormName()
		if fields[name] {
			chatError(c, service.ErrChatInvalid)
			return
		}
		fields[name] = true
		limit := int64(128)
		if name == "file" {
			limit = service.ChatMaxSpeechFileBytes
		}
		value, readErr := io.ReadAll(io.LimitReader(part, limit+1))
		_ = part.Close()
		if readErr != nil || int64(len(value)) > limit {
			chatError(c, service.ErrChatTooLarge)
			return
		}
		switch name {
		case "preset_id":
			preset, err = uuid.Parse(string(value))
		case "session_id":
			session, err = uuid.Parse(string(value))
		case "file":
			if part.FileName() == "" {
				chatError(c, service.ErrChatInvalid)
				return
			}
			audio = value
		default:
			chatError(c, service.ErrChatInvalid)
			return
		}
		if err != nil {
			chatError(c, service.ErrChatInvalid)
			return
		}
	}
	if preset == uuid.Nil || len(audio) == 0 {
		chatError(c, service.ErrChatInvalid)
		return
	}
	format, err := service.ValidateChatSpeechAudio(audio)
	if err != nil {
		chatError(c, err)
		return
	}
	// Build a fresh upstream multipart body; browser identity fields stay in BFF.
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	file, err := writer.CreateFormFile("file", "speech."+format)
	if err != nil {
		chatError(c, err)
		return
	}
	if _, err = file.Write(audio); err != nil {
		chatError(c, err)
		return
	}
	if err = writer.Close(); err != nil {
		chatError(c, err)
		return
	}
	err = h.service.Speech(c.Request.Context(), user, preset, session, "asr", ip.GetClientIP(c), &body, writer.FormDataContentType(), c.Writer)
	if err != nil && !c.Writer.Written() {
		chatError(c, err)
	}
}
