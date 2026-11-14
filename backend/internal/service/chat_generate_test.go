//go:build unit

package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type chatGenerationRepoFake struct {
	ChatRepository
	mu         sync.Mutex
	record     ChatSessionRecord
	run        *ChatRunRecord
	requestIDs []string
	finalErr   error
	finalized  bool
}

func (r *chatGenerationRepoFake) GetSession(_ context.Context, user int64, id uuid.UUID) (*ChatSessionRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if user != r.record.UserID || id != r.record.ID {
		return nil, ErrChatNotFound
	}
	copy := r.record
	return &copy, nil
}
func (r *chatGenerationRepoFake) GetPreferences(context.Context, int64) (json.RawMessage, int64, error) {
	return json.RawMessage(`{}`), 0, nil
}
func (r *chatGenerationRepoFake) SaveSession(ctx context.Context, user int64, snapshot *ChatSnapshot, expected int64) (*ChatSessionRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.run != nil && r.run.Status == "running" {
		return nil, ErrChatSessionBusy
	}
	if r.record.Revision != expected {
		return nil, ErrChatRevisionConflict
	}
	r.record.Snapshot = snapshot.Raw
	r.record.Revision++
	copy := r.record
	return &copy, nil
}
func (r *chatGenerationRepoFake) ClaimGeneration(ctx context.Context, run *ChatRunRecord, expected int64, message json.RawMessage) (*ChatSessionRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.run != nil && r.run.RequestID == run.RequestID {
		return nil, ErrChatRunExists
	}
	if r.run != nil && r.run.Status == "running" {
		return nil, ErrChatSessionBusy
	}
	if r.record.Revision != expected {
		return nil, ErrChatRevisionConflict
	}
	raw, title, err := MergeChatAssistant(&r.record, run.AssistantMessageID, message, "")
	if err != nil {
		return nil, err
	}
	copyRun := *run
	r.run = &copyRun
	r.record.Snapshot = raw
	r.record.Title = title
	r.record.Revision++
	copy := r.record
	return &copy, nil
}
func (r *chatGenerationRepoFake) SaveGeneration(ctx context.Context, run *ChatRunRecord, message json.RawMessage, title, status string) (*ChatSessionRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if status != "running" && r.finalErr != nil {
		return nil, r.finalErr
	}
	raw, title, err := MergeChatAssistant(&r.record, run.AssistantMessageID, message, title)
	if err != nil {
		return nil, err
	}
	r.record.Snapshot = raw
	r.record.Title = title
	r.record.Revision++
	r.run.Status = status
	r.finalized = status != "running"
	copy := r.record
	return &copy, nil
}
func (r *chatGenerationRepoFake) AddGenerationRequest(_ context.Context, _ *ChatRunRecord, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.requestIDs = append(r.requestIDs, id)
	return nil
}
func (r *chatGenerationRepoFake) RenewGeneration(context.Context, *ChatRunRecord) error { return nil }
func (r *chatGenerationRepoFake) LastGeneration(context.Context, int64, uuid.UUID, string) (*ChatRunRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.run == nil {
		return nil, ErrChatNotFound
	}
	copy := *r.run
	return &copy, nil
}

type chatGenerationPresets struct {
	ChatPresetRepository
	preset ChatModelPreset
}

func (r *chatGenerationPresets) GetPreset(_ context.Context, user int64, id uuid.UUID) (*ChatModelPreset, error) {
	if user != r.preset.UserID || id != r.preset.ID {
		return nil, ErrChatNotFound
	}
	copy := r.preset
	return &copy, nil
}

func chatGenerationFixture(t *testing.T) (*ChatService, *chatGenerationRepoFake, ChatGenerateRequest) {
	t.Helper()
	id := uuid.New()
	preset := ChatModelPreset{ID: uuid.New(), UserID: 7, GroupID: 11, Model: "shared-model"}
	raw, err := json.Marshal(map[string]any{"id": id.String(), "title": "", "rootMessageId": "question", "messages": []any{map[string]any{"id": "question", "role": "user", "content": "你好", "parentId": nil, "selectedChildId": "reply"}, map[string]any{"id": "reply", "role": "assistant", "status": "pending", "blocks": []any{}, "prefix": true, "parentId": "question"}}})
	require.NoError(t, err)
	repo := &chatGenerationRepoFake{record: ChatSessionRecord{ID: id, UserID: 7, Snapshot: raw}}
	models, groups, _, _ := chatModelFixture()
	// Lease/persistence cases deliberately cover the still-supported explicit adapter.
	groups.groups[0].ChatModelPolicies = map[string]ChatModelPolicy{"shared-model": {Transport: "chat_completions"}}
	models.repo = &chatGenerationPresets{preset: preset}
	credentials := &ChatCredentialService{repo: &chatCredentialRepoStub{}, keys: &chatCredentialKeysStub{groups: []Group{{ID: 11}}}}
	storage, err := NewChatStorage(t.TempDir())
	require.NoError(t, err)
	s, err := NewChatService(repo, storage, &config.Config{}, models, credentials)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, s.Stop()) })
	revision := int64(0)
	return s, repo, ChatGenerateRequest{PresetID: preset.ID, AssistantMessageID: "reply", ExpectedRevision: &revision, Operation: "send", RequestID: uuid.New()}
}

type chatCompletionObserver struct {
	*httptest.ResponseRecorder
	beforeWrite func([]byte)
}

func (w *chatCompletionObserver) Write(raw []byte) (int, error) {
	w.beforeWrite(raw)
	return w.ResponseRecorder.Write(raw)
}

func TestChatGeneratePersistsBeforeCompletionAndNeverReplays(t *testing.T) {
	s, repo, req := chatGenerationFixture(t)
	calls := 0
	s.dispatch = func(w http.ResponseWriter, r *http.Request) {
		calls++
		require.Len(t, repo.requestIDs, 1)
		require.Equal(t, "/v1/chat/completions", r.URL.Path)
		_, _ = w.Write([]byte(chatTestFrames(`{"choices":[{"delta":{"content":"回答🌍"}}]}`, `{"choices":[{"delta":{},"finish_reason":"stop"}]}`, `[DONE]`)))
	}
	recorder := httptest.NewRecorder()
	observer := &chatCompletionObserver{ResponseRecorder: recorder, beforeWrite: func(raw []byte) {
		if strings.Contains(string(raw), "event: message_stop") {
			require.True(t, repo.finalized)
		}
	}}
	require.NoError(t, s.Generate(context.Background(), 7, repo.record.ID, req, "192.0.2.1", observer))
	require.Equal(t, 1, calls)
	require.Equal(t, "done", repo.run.Status)
	require.Contains(t, string(repo.record.Snapshot), "回答🌍")
	require.Contains(t, string(repo.record.Snapshot), "chatOptions")
	body := recorder.Body.String()
	require.Less(t, strings.Index(body, "event: message_stop"), strings.Index(body, "event: chat_saved"))
	require.ErrorIs(t, s.Generate(context.Background(), 7, repo.record.ID, req, "192.0.2.1", httptest.NewRecorder()), ErrChatRevisionConflict)
	require.Equal(t, 1, calls)
}

func TestChatGeneratePreflightFailureIsSavedWithoutGateway(t *testing.T) {
	s, repo, req := chatGenerationFixture(t)
	req.PresetID = uuid.New()
	calls := 0
	s.dispatch = func(http.ResponseWriter, *http.Request) { calls++ }
	require.ErrorIs(t, s.Generate(context.Background(), 7, repo.record.ID, req, "192.0.2.1", httptest.NewRecorder()), ErrChatNotFound)
	require.Zero(t, calls)
	require.Nil(t, repo.run)
	message, err := ChatAssistantJSON(repo.record.Snapshot, "reply")
	require.NoError(t, err)
	require.Equal(t, "error", message["status"])
	require.Equal(t, int64(1), repo.record.Revision)
}

func TestChatGenerateDisconnectStopsAndSavesPartialWithDetachedContext(t *testing.T) {
	s, repo, req := chatGenerationFixture(t)
	started := make(chan struct{})
	cancelled := make(chan struct{})
	done := make(chan error, 1)
	s.dispatch = func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(chatTestFrames(`{"choices":[{"delta":{"content":"部分"}}]}`)))
		close(started)
		<-r.Context().Done()
		close(cancelled)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { done <- s.Generate(ctx, 7, repo.record.ID, req, "192.0.2.1", httptest.NewRecorder()) }()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("generation did not start")
	}
	cancel()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("generation did not cancel")
	}
	<-cancelled
	require.True(t, repo.finalized)
	require.Equal(t, "stopped", repo.run.Status)
	require.Contains(t, string(repo.record.Snapshot), "部分")
	require.Len(t, repo.requestIDs, 1)
}

func TestChatGenerateIncompleteStreamPersistsErrorBeforeNotice(t *testing.T) {
	s, repo, req := chatGenerationFixture(t)
	s.dispatch = func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(chatTestFrames(`{"choices":[{"delta":{"content":"partial"}}]}`)))
	}
	recorder := httptest.NewRecorder()
	require.NoError(t, s.Generate(context.Background(), 7, repo.record.ID, req, "192.0.2.1", recorder))
	require.Equal(t, "error", repo.run.Status)
	body := recorder.Body.String()
	require.NotContains(t, body, "event: message_stop")
	require.GreaterOrEqual(t, strings.Index(body, "event: chat_saved"), 0)
	require.Less(t, strings.Index(body, "event: chat_saved"), strings.Index(body, "event: error"))
}

func TestChatGenerateFailedFinalSaveNeverDeclaresCompletion(t *testing.T) {
	s, repo, req := chatGenerationFixture(t)
	repo.finalErr = errors.New("database unavailable")
	s.dispatch = func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(chatTestFrames(`{"choices":[{"delta":{"content":"answer"},"finish_reason":"stop"}]}`, `[DONE]`)))
	}
	recorder := httptest.NewRecorder()
	require.NoError(t, s.Generate(context.Background(), 7, repo.record.ID, req, "192.0.2.1", recorder))
	require.False(t, repo.finalized)
	require.NotContains(t, recorder.Body.String(), "event: message_stop")
	require.NotContains(t, recorder.Body.String(), "event: chat_saved")
}

func TestChatGenerateContinueUsesRecordedOptions(t *testing.T) {
	s, repo, req := chatGenerationFixture(t)
	calls := 0
	s.dispatch = func(w http.ResponseWriter, _ *http.Request) {
		calls++
		_, _ = w.Write([]byte(chatTestFrames(`{"choices":[{"delta":{"content":"answer"},"finish_reason":"stop"}]}`, `[DONE]`)))
	}
	require.NoError(t, s.Generate(context.Background(), 7, repo.record.ID, req, "192.0.2.1", httptest.NewRecorder()))
	message, err := ChatAssistantJSON(repo.record.Snapshot, "reply")
	require.NoError(t, err)
	message["status"], message["thinking"], message["search"] = "pending", true, true
	rawMessage, err := json.Marshal(message)
	require.NoError(t, err)
	raw, _, err := MergeChatAssistant(&repo.record, "reply", rawMessage, "")
	require.NoError(t, err)
	snapshot, err := DecodeChatSnapshot(repo.record.ID, raw)
	require.NoError(t, err)
	saved, err := repo.SaveSession(context.Background(), 7, snapshot, repo.record.Revision)
	require.NoError(t, err)
	req.ExpectedRevision, req.Operation, req.RequestID = &saved.Revision, "continue", uuid.New()
	require.NoError(t, s.Generate(context.Background(), 7, repo.record.ID, req, "192.0.2.1", httptest.NewRecorder()))
	var options ChatRunOptions
	require.NoError(t, json.Unmarshal(repo.run.Options, &options))
	require.False(t, options.Thinking)
	require.False(t, options.Search)
	require.Equal(t, 2, calls)
	message, err = ChatAssistantJSON(repo.record.Snapshot, "reply")
	require.NoError(t, err)
	require.Len(t, message["blocks"], 2)
}
