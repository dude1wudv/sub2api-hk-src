package service

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/google/uuid"
)

const (
	ChatMaxSessionBytes               = 8 << 20
	ChatMaxPreferenceBytes            = 64 << 10
	ChatMaxFileBytes            int64 = 64 << 20
	ChatMaxUserDataBytes        int64 = 32 << 20
	ChatDefaultUserStorageBytes int64 = 2 << 30
	ChatMaxMessageAttachments         = 50
)

var (
	ErrChatNotFound              = infraerrors.NotFound("CHAT_NOT_FOUND", "chat object not found")
	ErrChatInvalid               = infraerrors.BadRequest("CHAT_INVALID_REQUEST", "invalid chat request")
	ErrChatRevisionConflict      = infraerrors.Conflict("CHAT_REVISION_CONFLICT", "此会话已在其他设备更新，请重新载入")
	ErrChatSessionBusy           = infraerrors.Conflict("CHAT_SESSION_BUSY", "此会话正在生成，请先停止")
	ErrChatTooLarge              = infraerrors.New(http.StatusRequestEntityTooLarge, "CHAT_TOO_LARGE", "chat content exceeds the size limit")
	ErrChatStorageFull           = infraerrors.Conflict("CHAT_STORAGE_FULL", "聊天存储空间已满，请删除不再需要的附件或会话")
	ErrChatCredentialUnavailable = infraerrors.Conflict("CHAT_CREDENTIAL_UNAVAILABLE", "聊天凭据不可用，请在聊天模型配置页重新连接")
	ErrChatUnavailable           = infraerrors.ServiceUnavailable("CHAT_UNAVAILABLE", "chat service unavailable")
)

// The original JSON is authoritative. Parsed views are for validation/request
// construction only, so new upstream block fields and nonselected branches survive.
type ChatSnapshot struct {
	Raw           json.RawMessage
	ID            uuid.UUID
	Title         string
	Messages      []ChatMessage
	RootID        *string
	Linear        bool
	AttachmentIDs []uuid.UUID
}

type ChatMessage struct {
	ID              string              `json:"id"`
	Role            string              `json:"role"`
	ParentID        *string             `json:"parentId"`
	SelectedChildID *string             `json:"selectedChildId"`
	Content         string              `json:"content"`
	Files           []ChatFileReference `json:"files"`
	Blocks          []json.RawMessage   `json:"blocks"`
	Model           *string             `json:"model"`
	GroupID         int64               `json:"groupId,omitempty"`
	PresetID        string              `json:"presetId,omitempty"`
	Thinking        bool                `json:"thinking,omitempty"`
	Search          bool                `json:"search,omitempty"`
	Status          string              `json:"status"`
	Prefix          bool                `json:"prefix"`
}

type ChatFileReference struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Size        int64  `json:"size"`
	IsImage     bool   `json:"isImage"`
	MIMEType    string `json:"mimeType"`
	Unavailable bool   `json:"unavailable,omitempty"`
}

func DecodeChatSnapshot(id uuid.UUID, raw json.RawMessage) (*ChatSnapshot, error) {
	if len(raw) > ChatMaxSessionBytes {
		return nil, ErrChatTooLarge
	}
	if id == uuid.Nil || !utf8.Valid(raw) {
		return nil, ErrChatInvalid
	}
	var wire struct {
		ID       string          `json:"id"`
		Title    string          `json:"title"`
		Messages []ChatMessage   `json:"messages"`
		Root     json.RawMessage `json:"rootMessageId"`
	}
	if err := json.Unmarshal(raw, &wire); err != nil || wire.ID != id.String() || wire.Messages == nil || utf8.RuneCountInString(wire.Title) > 500 {
		return nil, ErrChatInvalid
	}
	snapshot := &ChatSnapshot{Raw: bytes.Clone(raw), ID: id, Title: wire.Title, Messages: wire.Messages, Linear: len(wire.Root) == 0}
	if len(wire.Root) != 0 && json.Unmarshal(wire.Root, &snapshot.RootID) != nil {
		return nil, ErrChatInvalid
	}
	byID := make(map[string]*ChatMessage, len(wire.Messages))
	files := make(map[uuid.UUID]bool)
	for i := range snapshot.Messages {
		m := &snapshot.Messages[i]
		if m.ID == "" || len(m.ID) > 200 || byID[m.ID] != nil || strings.ContainsAny(m.ID, "\r\n\x00") {
			return nil, ErrChatInvalid
		}
		if m.Role != "user" && m.Role != "assistant" {
			return nil, ErrChatInvalid
		}
		if m.Role == "assistant" {
			switch m.Status {
			case "pending", "streaming", "done", "stopped", "error":
			default:
				return nil, ErrChatInvalid
			}
		}
		if len(m.Files) > ChatMaxMessageAttachments || (m.Role != "user" && len(m.Files) != 0) {
			return nil, ErrChatInvalid
		}
		messageFiles := make(map[uuid.UUID]bool)
		for _, f := range m.Files {
			fid, err := uuid.Parse(f.ID)
			if err != nil || fid == uuid.Nil || messageFiles[fid] || f.Size < 0 || f.Size > ChatMaxFileBytes {
				return nil, ErrChatInvalid
			}
			messageFiles[fid] = true
			if f.Unavailable {
				continue
			}
			if !files[fid] {
				snapshot.AttachmentIDs = append(snapshot.AttachmentIDs, fid)
				files[fid] = true
			}
		}
		byID[m.ID] = m
	}
	if snapshot.Linear {
		for _, m := range snapshot.Messages {
			if m.ParentID != nil || m.SelectedChildID != nil {
				return nil, ErrChatInvalid
			}
		}
		return snapshot, nil
	}
	if snapshot.RootID == nil {
		if len(snapshot.Messages) != 0 {
			return nil, ErrChatInvalid
		}
	} else if root := byID[*snapshot.RootID]; root == nil || root.ParentID != nil {
		return nil, ErrChatInvalid
	}
	for _, m := range snapshot.Messages {
		if m.ParentID != nil && byID[*m.ParentID] == nil {
			return nil, ErrChatInvalid
		}
		if m.SelectedChildID != nil {
			child := byID[*m.SelectedChildID]
			if child == nil || child.ParentID == nil || *child.ParentID != m.ID {
				return nil, ErrChatInvalid
			}
		}
	}
	// Iterative color traversal checks every branch in O(n), including hidden ones,
	// without allowing a deeply nested JSON tree to exhaust the Go call stack.
	colors := make(map[string]uint8, len(byID))
	for _, m := range snapshot.Messages {
		path := make([]string, 0)
		current := byID[m.ID]
		for current != nil && colors[current.ID] == 0 {
			colors[current.ID] = 1
			path = append(path, current.ID)
			if current.ParentID == nil {
				current = nil
			} else {
				current = byID[*current.ParentID]
			}
		}
		if current != nil && colors[current.ID] == 1 {
			return nil, ErrChatInvalid
		}
		for _, messageID := range path {
			colors[messageID] = 2
		}
	}
	return snapshot, nil
}

func (s *ChatSnapshot) VisibleMessages() []ChatMessage {
	if s.Linear {
		return append([]ChatMessage(nil), s.Messages...)
	}
	byID := make(map[string]ChatMessage, len(s.Messages))
	for _, m := range s.Messages {
		byID[m.ID] = m
	}
	var result []ChatMessage
	for id := s.RootID; id != nil; {
		m, ok := byID[*id]
		if !ok {
			break
		}
		delete(byID, *id)
		result = append(result, m)
		id = m.SelectedChildID
	}
	return result
}

// Noncredential settings retain the upstream names. Unknown fields, including
// apiKey/apiOrigin, are rejected rather than persisted and accidentally exported.
type ChatPreferences struct {
	Locale          string        `json:"locale,omitempty"`
	ThemePreference string        `json:"themePreference,omitempty"`
	Location        *ChatLocation `json:"location,omitempty"`
	TTSVoice        string        `json:"ttsVoice,omitempty"`
	Thinking        *bool         `json:"thinking,omitempty"`
	Search          *bool         `json:"search,omitempty"`
	SiderCollapsed  *bool         `json:"siderCollapsed,omitempty"`
}

type ChatLocation struct {
	Region   string `json:"region"`
	Province string `json:"province,omitempty"`
	City     string `json:"city,omitempty"`
}

func ValidateChatPreferences(raw json.RawMessage) (json.RawMessage, error) {
	if len(raw) > ChatMaxPreferenceBytes {
		return nil, ErrChatTooLarge
	}
	if !utf8.Valid(raw) || len(bytes.TrimSpace(raw)) == 0 || bytes.TrimSpace(raw)[0] != '{' {
		return nil, ErrChatInvalid
	}
	var p ChatPreferences
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(&p) != nil {
		return nil, ErrChatInvalid
	}
	var trailing any
	if d.Decode(&trailing) != io.EOF {
		return nil, ErrChatInvalid
	}
	if p.Locale != "" && p.Locale != "zh-CN" && p.Locale != "en-US" {
		return nil, ErrChatInvalid
	}
	if p.ThemePreference != "" && p.ThemePreference != "light" && p.ThemePreference != "dark" && p.ThemePreference != "system" {
		return nil, ErrChatInvalid
	}
	if p.TTSVoice != "" && p.TTSVoice != "stella" && p.TTSVoice != "tide" {
		return nil, ErrChatInvalid
	}
	if p.Location != nil && (len(p.Location.Region) > 64 || len(p.Location.Province) > 128 || len(p.Location.City) > 128) {
		return nil, ErrChatInvalid
	}
	encoded, err := json.Marshal(p)
	if err != nil {
		return nil, ErrChatInvalid
	}
	return encoded, nil
}
