package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"time"

	"github.com/google/uuid"
)

const ChatMaxArchiveBytes int64 = 2 << 30

type ChatImportSession struct {
	SessionID uuid.UUID `json:"session_id"`
	FileID    uuid.UUID `json:"file_id"`
	Size      int64     `json:"size"`
	SHA256    string    `json:"sha256"`
	Ready     bool      `json:"ready"`
}
type ChatImportManifest struct {
	Sessions map[string]ChatImportSession `json:"sessions"`
	Bytes    int64                        `json:"bytes"`
}
type ChatImportRecord struct {
	ID              uuid.UUID       `json:"id"`
	ExpiresAt       time.Time       `json:"expires_at"`
	UserID          int64           `json:"-"`
	HistoryRevision int64           `json:"-"`
	Status          string          `json:"-"`
	Manifest        json.RawMessage `json:"-"`
}
type ChatImportedSnapshot struct {
	Snapshot  *ChatSnapshot
	CreatedAt time.Time
	UpdatedAt time.Time
}
type ChatImportRepository interface {
	BeginImport(context.Context, int64, int64) (*ChatImportRecord, error)
	GetImport(context.Context, int64, uuid.UUID) (*ChatImportRecord, error)
	ReserveImportSession(context.Context, int64, uuid.UUID, ChatImportSession, int64) error
	FinishImportSession(context.Context, int64, uuid.UUID, uuid.UUID) error
	CommitImport(context.Context, *ChatImportRecord, func(ChatImportSession) (*ChatImportedSnapshot, error)) (int64, error)
	AbortImport(context.Context, int64, uuid.UUID) error
}

func ParseChatImportManifest(raw json.RawMessage) (*ChatImportManifest, error) {
	var manifest ChatImportManifest
	if len(raw) > ChatMaxSessionBytes || json.Unmarshal(raw, &manifest) != nil || manifest.Sessions == nil {
		return nil, ErrChatInvalid
	}
	var total int64
	files := map[uuid.UUID]bool{}
	for key, item := range manifest.Sessions {
		if key != item.SessionID.String() || item.SessionID == uuid.Nil || item.FileID == uuid.Nil || files[item.FileID] || item.Size < 0 || item.Size > ChatMaxSessionBytes || len(item.SHA256) != 64 {
			return nil, ErrChatInvalid
		}
		if _, err := hex.DecodeString(item.SHA256); err != nil {
			return nil, ErrChatInvalid
		}
		files[item.FileID] = true
		total += item.Size
		if total > ChatMaxArchiveBytes {
			return nil, ErrChatTooLarge
		}
	}
	if manifest.Bytes != total {
		return nil, ErrChatInvalid
	}
	return &manifest, nil
}

func chatSanitizeImport(id uuid.UUID, raw json.RawMessage) (*ChatImportedSnapshot, error) {
	parsed, err := DecodeChatSnapshot(id, raw)
	if err != nil {
		return nil, err
	}
	var value map[string]json.RawMessage
	if json.Unmarshal(raw, &value) != nil {
		return nil, ErrChatInvalid
	}
	delete(value, "presetId")
	var messages []map[string]json.RawMessage
	if json.Unmarshal(value["messages"], &messages) != nil {
		return nil, ErrChatInvalid
	}
	for index, message := range messages {
		if parsed.Messages[index].Role == "assistant" {
			delete(message, "groupId")
			delete(message, "presetId")
			delete(message, "chatOptions")
			message["prefix"] = json.RawMessage(`false`)
			if parsed.Messages[index].Status == "streaming" || parsed.Messages[index].Status == "pending" {
				message["status"] = json.RawMessage(`"stopped"`)
			}
		}
		var files []map[string]json.RawMessage
		if rawFiles := message["files"]; len(rawFiles) > 0 {
			if json.Unmarshal(rawFiles, &files) != nil {
				return nil, ErrChatInvalid
			}
			for _, file := range files {
				file["serverId"] = json.RawMessage(`null`)
			}
			message["files"], _ = json.Marshal(files)
		}
	}
	value["messages"], _ = json.Marshal(messages)
	sanitized, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	snapshot, err := DecodeChatSnapshot(id, sanitized)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	dates := map[string]time.Time{"createdAt": now, "updatedAt": now}
	for key := range dates {
		if field, ok := value[key]; ok {
			var milliseconds int64
			if json.Unmarshal(field, &milliseconds) != nil || milliseconds < 0 || milliseconds > 253402300799000 {
				return nil, ErrChatInvalid
			}
			dates[key] = time.UnixMilli(milliseconds).UTC()
		}
	}
	return &ChatImportedSnapshot{Snapshot: snapshot, CreatedAt: dates["createdAt"], UpdatedAt: dates["updatedAt"]}, nil
}

func (s *ChatService) BeginImport(ctx context.Context, userID, expected int64) (*ChatImportRecord, error) {
	if expected < 0 {
		return nil, ErrChatInvalid
	}
	return s.repo.BeginImport(ctx, userID, expected)
}

func (s *ChatService) GetImport(ctx context.Context, userID int64, id uuid.UUID) (*ChatImportRecord, error) {
	return s.repo.GetImport(ctx, userID, id)
}

func (s *ChatService) StageImportSession(ctx context.Context, userID int64, importID, sessionID uuid.UUID, raw json.RawMessage) error {
	imported, err := chatSanitizeImport(sessionID, raw)
	if err != nil {
		return err
	}
	raw = imported.Snapshot.Raw
	sum := sha256.Sum256(raw)
	item := ChatImportSession{SessionID: sessionID, FileID: uuid.New(), Size: int64(len(raw)), SHA256: hex.EncodeToString(sum[:])}
	if err = s.repo.ReserveImportSession(ctx, userID, importID, item, s.storageLimit); err != nil {
		return err
	}
	write, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err = s.storage.StageSession(write, userID, importID, item.FileID, bytes.NewReader(raw)); err != nil {
		return err
	}
	return s.repo.FinishImportSession(ctx, userID, importID, sessionID)
}

func (s *ChatService) CommitImport(ctx context.Context, userID int64, id uuid.UUID, expected int64) (int64, error) {
	record, err := s.repo.GetImport(ctx, userID, id)
	if err != nil {
		return 0, err
	}
	if expected < 0 || record.HistoryRevision != expected {
		return 0, ErrChatRevisionConflict
	}
	manifest, err := ParseChatImportManifest(record.Manifest)
	if err != nil {
		return 0, err
	}
	read := func(item ChatImportSession) (*ChatImportedSnapshot, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if !item.Ready {
			return nil, ErrChatInvalid
		}
		raw, err := s.storage.ReadStagedSession(userID, id, item.FileID)
		if err != nil {
			return nil, ErrChatInvalid
		}
		digest := sha256.Sum256(raw)
		if int64(len(raw)) != item.Size || hex.EncodeToString(digest[:]) != item.SHA256 {
			return nil, ErrChatInvalid
		}
		return chatSanitizeImport(item.SessionID, raw)
	}
	// Validate every file before replacement, then revalidate incrementally inside
	// the transaction to bound memory and close the staging-file TOCTOU window.
	keys := make([]string, 0, len(manifest.Sessions))
	for key := range manifest.Sessions {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if _, err = read(manifest.Sessions[key]); err != nil {
			return 0, err
		}
	}
	return s.repo.CommitImport(ctx, record, read)
}

func (s *ChatService) AbortImport(ctx context.Context, userID int64, id uuid.UUID) error {
	return s.repo.AbortImport(ctx, userID, id)
}
