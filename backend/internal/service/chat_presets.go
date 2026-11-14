package service

import (
	"context"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/google/uuid"
	"strings"
	"unicode/utf8"
)

var ErrChatPresetDuplicate = infraerrors.Conflict("CHAT_PRESET_DUPLICATE", "该分组的模型已在个人列表中")

type ChatModelPreset struct {
	ID                      uuid.UUID `json:"id"`
	UserID                  int64     `json:"-"`
	GroupID                 int64     `json:"group_id"`
	Model                   string    `json:"model"`
	DisplayName             string    `json:"display_name"`
	SortOrder               int       `json:"sort_order"`
	IsDefault               bool      `json:"is_default"`
	NormalReasoningEffort   string    `json:"normal_reasoning_effort"`
	ThinkingReasoningEffort string    `json:"thinking_reasoning_effort"`
}

type ChatPresetPatch struct {
	GroupID                 *int64  `json:"group_id"`
	Model                   *string `json:"model"`
	DisplayName             *string `json:"display_name"`
	IsDefault               *bool   `json:"is_default"`
	NormalReasoningEffort   *string `json:"normal_reasoning_effort"`
	ThinkingReasoningEffort *string `json:"thinking_reasoning_effort"`
}

type ChatPresetRepository interface {
	ListPresets(context.Context, int64) ([]ChatModelPreset, error)
	GetPreset(context.Context, int64, uuid.UUID) (*ChatModelPreset, error)
	CreatePreset(context.Context, int64, int64, string, string) (*ChatModelPreset, error)
	UpdatePreset(context.Context, int64, uuid.UUID, ChatPresetPatch) (*ChatModelPreset, error)
	OrderPresets(context.Context, int64, []uuid.UUID) error
	DeletePreset(context.Context, int64, uuid.UUID) error
}

func validateChatPreset(groupID int64, model, name string) error {
	if groupID <= 0 || model == "" || model != strings.TrimSpace(model) || len(model) > 256 || !utf8.ValidString(model) || strings.ContainsAny(model, "*?\r\n\x00") || !utf8.ValidString(name) || utf8.RuneCountInString(name) > 200 {
		return ErrChatInvalid
	}
	return nil
}

func validChatReasoningEffort(effort string) bool {
	return effort == "low" || effort == "medium" || effort == "high"
}

func (p *ChatModelPreset) reasoningEffort(thinking bool) string {
	effort, fallback := p.NormalReasoningEffort, "low"
	if thinking {
		effort, fallback = p.ThinkingReasoningEffort, "high"
	}
	if validChatReasoningEffort(effort) {
		return effort
	}
	return fallback
}
