//go:build integration

package repository

import (
	"context"
	"encoding/json"
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/chatattachment"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func chatIntegrationSnapshot(t *testing.T, id uuid.UUID, files ...uuid.UUID) *service.ChatSnapshot {
	t.Helper()
	refs := []any{}
	for _, file := range files {
		refs = append(refs, map[string]any{"id": file.String(), "name": "image.png", "size": 4, "isImage": true})
	}
	raw, err := json.Marshal(map[string]any{"id": id.String(), "title": "原始标题", "rootMessageId": "question", "messages": []any{map[string]any{"id": "question", "role": "user", "parentId": nil, "content": "私有问题", "files": refs}}})
	require.NoError(t, err)
	snapshot, err := service.DecodeChatSnapshot(id, raw)
	require.NoError(t, err)
	return snapshot
}

func TestChatSessionOwnershipRevisionAndReferences(t *testing.T) {
	tx := testEntTx(t)
	ctx := dbent.NewTxContext(context.Background(), tx)
	client := tx.Client()
	repo := NewChatRepository(integrationEntClient)
	a := mustCreateUser(t, client, &service.User{})
	b := mustCreateUser(t, client, &service.User{})
	file, err := client.ChatAttachment.Create().SetUserID(a.ID).SetStorageKey(uuid.NewString()).SetOriginalName("image.png").SetByteSize(4).SetPurpose(chatattachment.PurposeImage).SetStatus(chatattachment.StatusReady).Save(ctx)
	require.NoError(t, err)
	first := chatIntegrationSnapshot(t, uuid.New(), file.ID)
	saved, err := repo.SaveSession(ctx, a.ID, first, 0)
	require.NoError(t, err)
	require.Equal(t, int64(1), saved.Revision)
	_, err = repo.GetSession(ctx, b.ID, first.ID)
	require.ErrorIs(t, err, service.ErrChatNotFound)
	_, err = repo.SaveSession(ctx, b.ID, chatIntegrationSnapshot(t, uuid.New(), file.ID), 0)
	require.ErrorIs(t, err, service.ErrChatNotFound)
	updated, err := repo.SaveSession(ctx, a.ID, first, saved.Revision)
	require.NoError(t, err)
	require.Equal(t, saved.Revision+1, updated.Revision)
	_, err = repo.SaveSession(ctx, a.ID, first, saved.Revision)
	require.ErrorIs(t, err, service.ErrChatRevisionConflict)
	renamed, err := repo.RenameSession(ctx, a.ID, first.ID, "手动命名", updated.Revision)
	require.NoError(t, err)
	require.Equal(t, updated.Revision+1, renamed.Revision)
	require.True(t, renamed.TitleManual)
	_, historyRevision, err := repo.GetPreferences(ctx, a.ID)
	require.NoError(t, err)
	require.Equal(t, int64(3), historyRevision)
	second := chatIntegrationSnapshot(t, uuid.New(), file.ID)
	secondSaved, err := repo.SaveSession(ctx, a.ID, second, 0)
	require.NoError(t, err)
	_, err = repo.SaveSession(ctx, a.ID, chatIntegrationSnapshot(t, first.ID), renamed.Revision)
	require.NoError(t, err)
	stillUsed, err := client.ChatAttachment.Get(ctx, file.ID)
	require.NoError(t, err)
	require.Equal(t, chatattachment.StatusReady, stillUsed.Status)
	_, err = repo.SaveSession(ctx, a.ID, chatIntegrationSnapshot(t, second.ID), secondSaved.Revision)
	require.NoError(t, err)
	unused, err := client.ChatAttachment.Get(ctx, file.ID)
	require.NoError(t, err)
	require.Equal(t, chatattachment.StatusDeleting, unused.Status)
	page, err := repo.ListSessions(ctx, a.ID, "", 1)
	require.NoError(t, err)
	require.Len(t, page.Items, 1)
	require.NotEmpty(t, page.NextCursor)
	next, err := repo.ListSessions(ctx, a.ID, page.NextCursor, 1)
	require.NoError(t, err)
	require.Len(t, next.Items, 1)
	require.NotEqual(t, page.Items[0].ID, next.Items[0].ID)
	theirs, err := repo.ListSessions(ctx, b.ID, "", 100)
	require.NoError(t, err)
	require.Empty(t, theirs.Items)
	encoded, err := json.Marshal(page)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "messages")
	require.NotContains(t, string(encoded), "私有问题")
}

func TestChatUserDeleteRollbackKeepsHistoryAndFiles(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewChatRepository(client)
	users := NewUserRepository(client, integrationDB)
	owner := mustCreateUser(t, client, &service.User{})
	t.Cleanup(func() {
		_, _ = integrationDB.Exec("DELETE FROM chat_sessions WHERE user_id=$1", owner.ID)
		_, _ = integrationDB.Exec("DELETE FROM chat_attachments WHERE user_id=$1", owner.ID)
		_, _ = integrationDB.Exec("DELETE FROM users WHERE id=$1", owner.ID)
	})
	file, err := client.ChatAttachment.Create().SetUserID(owner.ID).SetStorageKey(uuid.NewString()).SetOriginalName("image.png").SetByteSize(4).SetPurpose(chatattachment.PurposeImage).SetStatus(chatattachment.StatusReady).Save(ctx)
	require.NoError(t, err)
	snapshot := chatIntegrationSnapshot(t, uuid.New(), file.ID)
	_, err = repo.SaveSession(ctx, owner.ID, snapshot, 0)
	require.NoError(t, err)
	tx, err := client.Tx(ctx)
	require.NoError(t, err)
	require.NoError(t, users.Delete(dbent.NewTxContext(ctx, tx), owner.ID))
	require.NoError(t, tx.Rollback())
	_, err = repo.GetSession(ctx, owner.ID, snapshot.ID)
	require.NoError(t, err)
	preserved, err := client.ChatAttachment.Get(ctx, file.ID)
	require.NoError(t, err)
	require.Equal(t, chatattachment.StatusReady, preserved.Status)
	require.NoError(t, users.Delete(ctx, owner.ID))
	_, err = repo.GetSession(ctx, owner.ID, snapshot.ID)
	require.ErrorIs(t, err, service.ErrChatNotFound)
	queued, err := client.ChatAttachment.Get(ctx, file.ID)
	require.NoError(t, err)
	require.Equal(t, chatattachment.StatusDeleting, queued.Status)
	require.Equal(t, preserved.StorageKey, queued.StorageKey)
}
