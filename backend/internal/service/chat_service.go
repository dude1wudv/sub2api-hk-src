package service

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/google/uuid"
)

type ChatService struct {
	repo                ChatRepository
	storage             *ChatStorage
	storageLimit        int64
	stopOnce            sync.Once
	stopErr             error
	models              *ChatModelService
	credentials         *ChatCredentialService
	dispatchMu          sync.RWMutex
	dispatch            ChatGatewayDispatch
	maxRequestBodyBytes int64
	instanceID          uuid.UUID
	runMu               sync.Mutex
	active              map[uuid.UUID]*chatActiveRun
	closed              bool
	lifecycleCancel     context.CancelFunc
	lifecycleDone       chan struct{}
}

func NewChatService(repo ChatRepository, storage *ChatStorage, cfg *config.Config, models *ChatModelService, credentials *ChatCredentialService) (*ChatService, error) {
	if repo == nil || storage == nil || cfg == nil || cfg.Chat.UserStorageMaxBytes < 0 {
		return nil, ErrChatUnavailable
	}
	limit := cfg.Chat.UserStorageMaxBytes
	if limit == 0 {
		limit = ChatDefaultUserStorageBytes
	}
	maxBody := cfg.Server.MaxRequestBodySize
	if maxBody <= 0 {
		maxBody = cfg.Gateway.MaxBodySize
	}
	return &ChatService{repo: repo, storage: storage, storageLimit: limit, models: models, credentials: credentials, maxRequestBodyBytes: maxBody, instanceID: uuid.New(), active: make(map[uuid.UUID]*chatActiveRun)}, nil
}

func (s *ChatService) Stop() error {
	s.stopOnce.Do(func() { s.stopErr = errors.Join(s.stopLifecycle(), s.storage.Close()) })
	return s.stopErr
}

func (s *ChatService) ListSessions(ctx context.Context, userID int64, cursor string, limit int) (*ChatSessionPage, error) {
	return s.repo.ListSessions(ctx, userID, cursor, limit)
}

func (s *ChatService) GetSession(ctx context.Context, userID int64, id uuid.UUID) (*ChatSessionRecord, error) {
	return s.repo.GetSession(ctx, userID, id)
}

func (s *ChatService) SaveSession(ctx context.Context, userID int64, id uuid.UUID, raw json.RawMessage, expected int64) (*ChatSessionRecord, error) {
	snapshot, err := DecodeChatSnapshot(id, raw)
	if err != nil {
		return nil, err
	}
	return s.repo.SaveSession(ctx, userID, snapshot, expected)
}

func (s *ChatService) RenameSession(ctx context.Context, userID int64, id uuid.UUID, title string, expected int64) (*ChatSessionRecord, error) {
	if !utf8.ValidString(title) || utf8.RuneCountInString(title) > 500 || expected < 0 {
		return nil, ErrChatInvalid
	}
	return s.repo.RenameSession(ctx, userID, id, title, expected)
}

func (s *ChatService) GetPreferences(ctx context.Context, userID int64) (json.RawMessage, int64, error) {
	return s.repo.GetPreferences(ctx, userID)
}

func (s *ChatService) SavePreferences(ctx context.Context, userID int64, raw json.RawMessage) error {
	validated, err := ValidateChatPreferences(raw)
	if err != nil {
		return err
	}
	return s.repo.SavePreferences(ctx, userID, validated)
}
