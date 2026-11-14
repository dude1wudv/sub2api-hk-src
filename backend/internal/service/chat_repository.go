package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type ChatSessionRecord struct {
	ID          uuid.UUID       `json:"id"`
	UserID      int64           `json:"-"`
	Title       string          `json:"title"`
	Snapshot    json.RawMessage `json:"session"`
	Revision    int64           `json:"revision"`
	TitleManual bool            `json:"-"`
	CreatedAt   time.Time       `json:"-"`
	UpdatedAt   time.Time       `json:"-"`
}

type ChatSessionSummary struct {
	ID        uuid.UUID `json:"id"`
	Title     string    `json:"title"`
	CreatedAt int64     `json:"createdAt"`
	UpdatedAt int64     `json:"updatedAt"`
	Revision  int64     `json:"revision"`
}

type ChatSessionPage struct {
	Items      []ChatSessionSummary `json:"items"`
	NextCursor string               `json:"next_cursor"`
}

type ChatRepository interface {
	ChatGenerationRepository
	ChatAttachmentRepository
	ChatRemoteFileRepository
	ChatParseRepository
	ChatImportRepository
	ChatCleanupRepository
	ListSessions(context.Context, int64, string, int) (*ChatSessionPage, error)
	GetSession(context.Context, int64, uuid.UUID) (*ChatSessionRecord, error)
	SaveSession(context.Context, int64, *ChatSnapshot, int64) (*ChatSessionRecord, error)
	RenameSession(context.Context, int64, uuid.UUID, string, int64) (*ChatSessionRecord, error)
	GetPreferences(context.Context, int64) (json.RawMessage, int64, error)
	SavePreferences(context.Context, int64, json.RawMessage) error
}
