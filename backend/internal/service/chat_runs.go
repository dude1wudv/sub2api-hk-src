package service

import (
	"context"
	"encoding/json"
	"time"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/google/uuid"
)

const ChatRunLease = 60 * time.Second

var (
	ErrChatRunExists       = infraerrors.Conflict("CHAT_GENERATION_EXISTS", "此请求已被接收，请重新载入会话，不要重复发送")
	ErrChatLeaseLost       = infraerrors.Conflict("CHAT_LEASE_LOST", "生成已停止或租约已失效")
	ErrChatCapability      = infraerrors.BadRequest("CHAT_CAPABILITY_UNAVAILABLE", "所选模型或渠道不支持此功能")
	ErrChatProviderContext = infraerrors.Conflict("CHAT_PROVIDER_CONTEXT", "此会话含原渠道专用上下文，请开启新会话")
)

type ChatGenerateRequest struct {
	PresetID           uuid.UUID `json:"preset_id"`
	AssistantMessageID string    `json:"assistant_message_id"`
	ExpectedRevision   *int64    `json:"expected_revision"`
	Operation          string    `json:"operation"`
	RequestID          uuid.UUID `json:"request_id"`
}

type ChatRunOptions struct {
	Thinking        bool          `json:"thinking"`
	ReasoningEffort string        `json:"reasoning_effort,omitempty"`
	Search          bool          `json:"search"`
	Location        *ChatLocation `json:"location,omitempty"`
	Transport       string        `json:"transport"`
	Locale          string        `json:"locale,omitempty"`
	LatestReminder  string        `json:"latest_reminder,omitempty"`
}

type ChatRunRecord struct {
	ID                 uuid.UUID       `json:"id"`
	UserID             int64           `json:"-"`
	SessionID          uuid.UUID       `json:"session_id"`
	AssistantMessageID string          `json:"assistant_message_id"`
	PresetID           uuid.UUID       `json:"preset_id"`
	GroupID            int64           `json:"group_id"`
	Model              string          `json:"model"`
	RequestID          uuid.UUID       `json:"request_id"`
	Operation          string          `json:"operation"`
	Status             string          `json:"status"`
	InstanceID         uuid.UUID       `json:"-"`
	LeaseExpiresAt     time.Time       `json:"-"`
	Options            json.RawMessage `json:"-"`
	GatewayRequestIDs  []string        `json:"gateway_request_ids"`
}

type ChatGenerationRepository interface {
	ClaimGeneration(context.Context, *ChatRunRecord, int64, json.RawMessage) (*ChatSessionRecord, error)
	SaveGeneration(context.Context, *ChatRunRecord, json.RawMessage, string, string) (*ChatSessionRecord, error)
	RenewGeneration(context.Context, *ChatRunRecord) error
	AddGenerationRequest(context.Context, *ChatRunRecord, string) error
	GetGeneration(context.Context, int64, uuid.UUID, uuid.UUID) (*ChatRunRecord, error)
	LastGeneration(context.Context, int64, uuid.UUID, string) (*ChatRunRecord, error)
	StopGeneration(context.Context, int64, uuid.UUID, uuid.UUID) error
	RecoverExpiredGenerations(context.Context) error
	DeleteSessions(context.Context, int64, *uuid.UUID) error
}

// MergeChatAssistant changes only the designated reply and server timestamps.
// Other branches, unknown fields and manual titles remain authoritative in SQL.
func MergeChatAssistant(record *ChatSessionRecord, assistantID string, message json.RawMessage, title string) (json.RawMessage, string, error) {
	var tree map[string]json.RawMessage
	var messages []json.RawMessage
	var reply struct {
		ID   string `json:"id"`
		Role string `json:"role"`
	}
	if json.Unmarshal(record.Snapshot, &tree) != nil || json.Unmarshal(tree["messages"], &messages) != nil || json.Unmarshal(message, &reply) != nil || reply.ID != assistantID || reply.Role != "assistant" {
		return nil, "", ErrChatInvalid
	}
	found := false
	for i, raw := range messages {
		var item struct {
			ID string `json:"id"`
		}
		if json.Unmarshal(raw, &item) != nil {
			return nil, "", ErrChatInvalid
		}
		if item.ID == assistantID {
			messages[i] = message
			found = true
			break
		}
	}
	if !found {
		return nil, "", ErrChatInvalid
	}
	tree["messages"], _ = json.Marshal(messages)
	tree["updatedAt"], _ = json.Marshal(time.Now().UnixMilli())
	if record.TitleManual || title == "" {
		title = record.Title
	}
	if !utf8.ValidString(title) || utf8.RuneCountInString(title) > 500 {
		return nil, "", ErrChatInvalid
	}
	tree["title"], _ = json.Marshal(title)
	raw, err := json.Marshal(tree)
	if err != nil {
		return nil, "", err
	}
	if _, err = DecodeChatSnapshot(record.ID, raw); err != nil {
		return nil, "", err
	}
	return raw, title, nil
}

func ChatAssistantJSON(raw json.RawMessage, assistantID string) (map[string]any, error) {
	var tree struct {
		Messages []json.RawMessage `json:"messages"`
	}
	if json.Unmarshal(raw, &tree) != nil {
		return nil, ErrChatInvalid
	}
	for _, item := range tree.Messages {
		var message map[string]any
		if json.Unmarshal(item, &message) != nil {
			return nil, ErrChatInvalid
		}
		if message["id"] == assistantID && message["role"] == "assistant" {
			return message, nil
		}
	}
	return nil, ErrChatInvalid
}
