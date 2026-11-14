//go:build integration

package repository

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func chatRunFixture(t *testing.T, repo service.ChatRepository, ctx context.Context, owner int64) (*service.ChatRunRecord, json.RawMessage) {
	t.Helper()
	id := uuid.New()
	message := json.RawMessage(`{"id":"reply","role":"assistant","parentId":"question","blocks":[],"status":"pending","prefix":true}`)
	raw, err := json.Marshal(map[string]any{"id": id.String(), "title": "初始", "rootMessageId": "question", "messages": []any{map[string]any{"id": "question", "role": "user", "content": "hello", "parentId": nil, "selectedChildId": "reply"}, message, map[string]any{"id": "old", "role": "assistant", "parentId": "question", "status": "done", "blocks": []any{map[string]any{"type": "text", "text": "old branch"}}}}})
	require.NoError(t, err)
	snapshot, err := service.DecodeChatSnapshot(id, raw)
	require.NoError(t, err)
	_, err = repo.SaveSession(ctx, owner, snapshot, 0)
	require.NoError(t, err)
	return &service.ChatRunRecord{ID: uuid.New(), UserID: owner, SessionID: id, AssistantMessageID: "reply", PresetID: uuid.New(), GroupID: 1, Model: "test-model", RequestID: uuid.New(), Operation: "send", InstanceID: uuid.New(), Options: json.RawMessage(`{}`)}, message
}

func TestChatRunOwnershipLeaseAndCanonicalMerge(t *testing.T) {
	tx := testEntTx(t)
	ctx := dbent.NewTxContext(context.Background(), tx)
	client := tx.Client()
	repo := NewChatRepository(integrationEntClient)
	a := mustCreateUser(t, client, &service.User{})
	b := mustCreateUser(t, client, &service.User{})
	run, initial := chatRunFixture(t, repo, ctx, a.ID)
	claimed, err := repo.ClaimGeneration(ctx, run, 1, initial)
	require.NoError(t, err)
	require.Equal(t, int64(2), claimed.Revision)
	_, err = repo.ClaimGeneration(ctx, run, claimed.Revision, initial)
	require.ErrorIs(t, err, service.ErrChatRunExists)
	_, err = repo.GetGeneration(ctx, b.ID, run.SessionID, run.ID)
	require.ErrorIs(t, err, service.ErrChatNotFound)
	require.ErrorIs(t, repo.StopGeneration(ctx, b.ID, run.SessionID, run.ID), service.ErrChatNotFound)
	require.ErrorIs(t, repo.DeleteSessions(ctx, b.ID, &run.SessionID), service.ErrChatNotFound)
	snapshot, err := service.DecodeChatSnapshot(run.SessionID, claimed.Snapshot)
	require.NoError(t, err)
	_, err = repo.SaveSession(ctx, a.ID, snapshot, claimed.Revision)
	require.ErrorIs(t, err, service.ErrChatSessionBusy)
	renamed, err := repo.RenameSession(ctx, a.ID, run.SessionID, "手动标题", claimed.Revision)
	require.NoError(t, err)
	require.Equal(t, claimed.Revision+1, renamed.Revision)
	require.NoError(t, repo.AddGenerationRequest(ctx, run, uuid.NewString()))
	auxiliaryID := "chat_auxiliary:" + uuid.NewString()
	require.NoError(t, repo.AddGenerationRequest(ctx, run, auxiliaryID))
	for _, invalid := range []string{"chat_auxiliary:", "other:" + uuid.NewString(), "chat_auxiliary:urn:uuid:" + uuid.NewString(), "chat_auxiliary:" + uuid.Nil.String()} {
		require.ErrorIs(t, repo.AddGenerationRequest(ctx, run, invalid), service.ErrChatInvalid)
	}
	reply := json.RawMessage(`{"id":"reply","role":"assistant","parentId":"question","status":"streaming","blocks":[{"type":"text","text":"部分中文","citations":[{"url":"https://example.com"}]}]}`)
	saved, err := repo.SaveGeneration(ctx, run, reply, "模型标题", "running")
	require.NoError(t, err)
	require.Equal(t, "手动标题", saved.Title)
	require.Contains(t, string(saved.Snapshot), "old branch")
	require.Contains(t, string(saved.Snapshot), "citations")
	foreign := *run
	foreign.InstanceID = uuid.New()
	require.ErrorIs(t, repo.RenewGeneration(ctx, &foreign), service.ErrChatLeaseLost)
	_, err = repo.SaveGeneration(ctx, &foreign, reply, "", "done")
	require.ErrorIs(t, err, service.ErrChatLeaseLost)
	require.NoError(t, repo.StopGeneration(ctx, a.ID, run.SessionID, run.ID))
	terminal, err := repo.GetGeneration(ctx, a.ID, run.SessionID, run.ID)
	require.NoError(t, err)
	require.Equal(t, "stopped", terminal.Status)
	require.Len(t, terminal.GatewayRequestIDs, 2)
	require.Contains(t, terminal.GatewayRequestIDs, auxiliaryID)
	latest, err := repo.LastGeneration(ctx, a.ID, run.SessionID, run.AssistantMessageID)
	require.NoError(t, err)
	require.Equal(t, run.ID, latest.ID)
	_, err = repo.LastGeneration(ctx, b.ID, run.SessionID, run.AssistantMessageID)
	require.ErrorIs(t, err, service.ErrChatNotFound)
	_, err = repo.SaveGeneration(ctx, run, reply, "", "done")
	require.ErrorIs(t, err, service.ErrChatLeaseLost)
	require.NoError(t, repo.DeleteSessions(ctx, a.ID, &run.SessionID))
	require.NoError(t, repo.DeleteSessions(ctx, a.ID, &run.SessionID))
}

func TestChatRunExpiresWithoutReplayAndRejectsOldWriter(t *testing.T) {
	tx := testEntTx(t)
	ctx := dbent.NewTxContext(context.Background(), tx)
	client := tx.Client()
	repo := NewChatRepository(integrationEntClient)
	a := mustCreateUser(t, client, &service.User{})
	run, initial := chatRunFixture(t, repo, ctx, a.ID)
	_, err := repo.ClaimGeneration(ctx, run, 1, initial)
	require.NoError(t, err)
	require.NoError(t, client.ChatGenerationRun.UpdateOneID(run.ID).SetLeaseExpiresAt(time.Now().Add(-time.Second)).Exec(ctx))
	require.ErrorIs(t, repo.RenewGeneration(ctx, run), service.ErrChatLeaseLost)
	require.NoError(t, repo.RecoverExpiredGenerations(ctx))
	record, err := repo.GetGeneration(ctx, a.ID, run.SessionID, run.ID)
	require.NoError(t, err)
	require.Equal(t, "stopped", record.Status)
	require.Empty(t, record.GatewayRequestIDs)
	session, err := repo.GetSession(ctx, a.ID, run.SessionID)
	require.NoError(t, err)
	require.Contains(t, string(session.Snapshot), `"status": "stopped"`)
	_, err = repo.ClaimGeneration(ctx, run, session.Revision, initial)
	require.ErrorIs(t, err, service.ErrChatRunExists)
}

func TestChatRunConcurrentClaimOneWinner(t *testing.T) {
	client := testEntClient(t)
	repo := NewChatRepository(client)
	a := mustCreateUser(t, client, &service.User{})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	run, initial := chatRunFixture(t, repo, ctx, a.ID)
	errors := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() { _, err := repo.ClaimGeneration(ctx, run, 1, initial); errors <- err })
	}
	wg.Wait()
	close(errors)
	won, duplicates := 0, 0
	for err := range errors {
		if err == nil {
			won++
		} else {
			require.ErrorIs(t, err, service.ErrChatRunExists)
			duplicates++
		}
	}
	require.Equal(t, 1, won)
	require.Equal(t, 1, duplicates)
	require.NoError(t, repo.DeleteSessions(ctx, a.ID, &run.SessionID))
}
