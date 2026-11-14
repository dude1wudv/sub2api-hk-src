//go:build integration

package repository

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"os"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/chatattachment"
	"github.com/Wei-Shaw/sub2api/ent/chatattachmentupload"
	"github.com/Wei-Shaw/sub2api/ent/chatimport"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestChatImportAtomicReplacementConflictAndOwnership(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewChatRepository(client)
	owner := mustCreateUser(t, client, &service.User{})
	other := mustCreateUser(t, client, &service.User{})
	t.Cleanup(func() {
		_ = client.User.DeleteOneID(owner.ID).Exec(ctx)
		_ = client.User.DeleteOneID(other.ID).Exec(ctx)
	})
	storage, err := service.NewChatStorage(t.TempDir())
	require.NoError(t, err)
	svc, err := service.NewChatService(repo, storage, &config.Config{}, nil, nil, nil)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, svc.Stop()) })
	old := chatIntegrationSnapshot(t, uuid.New())
	_, err = repo.SaveSession(ctx, owner.ID, old, 0)
	require.NoError(t, err)
	_, revision, err := repo.GetPreferences(ctx, owner.ID)
	require.NoError(t, err)
	broken, err := svc.BeginImport(ctx, owner.ID, revision)
	require.NoError(t, err)
	invalid := chatIntegrationSnapshot(t, uuid.New(), uuid.New())
	require.NoError(t, svc.StageImportSession(ctx, owner.ID, broken.ID, invalid.ID, invalid.Raw))
	_, err = svc.CommitImport(ctx, owner.ID, broken.ID, revision)
	require.ErrorIs(t, err, service.ErrChatNotFound)
	preserved, err := repo.GetSession(ctx, owner.ID, old.ID)
	require.NoError(t, err)
	require.JSONEq(t, string(old.Raw), string(preserved.Snapshot))
	_, afterFailure, err := repo.GetPreferences(ctx, owner.ID)
	require.NoError(t, err)
	require.Equal(t, revision, afterFailure)
	require.NoError(t, svc.AbortImport(ctx, owner.ID, broken.ID))
	staged, err := svc.BeginImport(ctx, owner.ID, revision)
	require.NoError(t, err)
	_, err = svc.GetImport(ctx, other.ID, staged.ID)
	require.ErrorIs(t, err, service.ErrChatNotFound)
	require.ErrorIs(t, svc.AbortImport(ctx, other.ID, staged.ID), service.ErrChatNotFound)
	var data bytes.Buffer
	require.NoError(t, png.Encode(&data, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	attachment, err := svc.UploadAttachment(ctx, owner.ID, "image.png", "image", bytes.NewReader(data.Bytes()), int64(data.Len()), &staged.ID)
	require.NoError(t, err)
	_, err = repo.GetAttachment(ctx, owner.ID, attachment.ID)
	require.ErrorIs(t, err, service.ErrChatNotFound)
	imported := chatIntegrationSnapshot(t, uuid.New(), attachment.ID)
	require.NoError(t, svc.StageImportSession(ctx, owner.ID, staged.ID, imported.ID, imported.Raw))
	before, err := repo.ListSessions(ctx, owner.ID, "", 100)
	require.NoError(t, err)
	require.Len(t, before.Items, 1)
	require.Equal(t, old.ID, before.Items[0].ID)
	newRevision, err := svc.CommitImport(ctx, owner.ID, staged.ID, revision)
	require.NoError(t, err)
	require.Equal(t, revision+1, newRevision)
	_, err = repo.GetSession(ctx, owner.ID, old.ID)
	require.ErrorIs(t, err, service.ErrChatNotFound)
	restored, err := repo.GetSession(ctx, owner.ID, imported.ID)
	require.NoError(t, err)
	require.Contains(t, string(restored.Snapshot), attachment.ID.String())
	file, err := repo.GetAttachment(ctx, owner.ID, attachment.ID)
	require.NoError(t, err)
	require.Nil(t, file.ImportID)
	_, err = repo.GetAttachment(ctx, other.ID, attachment.ID)
	require.ErrorIs(t, err, service.ErrChatNotFound)
	conflict, err := svc.BeginImport(ctx, owner.ID, newRevision)
	require.NoError(t, err)
	_, err = repo.RenameSession(ctx, owner.ID, imported.ID, "其他设备的修改", restored.Revision)
	require.NoError(t, err)
	_, err = svc.CommitImport(ctx, owner.ID, conflict.ID, newRevision)
	require.ErrorIs(t, err, service.ErrChatRevisionConflict)
	restored, err = repo.GetSession(ctx, owner.ID, imported.ID)
	require.NoError(t, err)
	require.Equal(t, "其他设备的修改", restored.Title)
}

func TestChatCleanupPreservesSharedFilesAndBoundsRemoteRetry(t *testing.T) {
	tx := testEntTx(t)
	ctx := dbent.NewTxContext(context.Background(), tx)
	client := tx.Client()
	repo := NewChatRepository(integrationEntClient)
	owner := mustCreateUser(t, client, &service.User{})
	account := mustCreateAccount(t, client, &service.Account{Name: "chat-cleanup"})
	storage, err := service.NewChatStorage(t.TempDir())
	require.NoError(t, err)
	svc, err := service.NewChatService(repo, storage, &config.Config{}, nil, nil, nil)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, svc.Stop()) })
	attachment, err := svc.UploadAttachment(ctx, owner.ID, "test.txt", "user_data", bytes.NewReader([]byte("test")), 4, nil)
	require.NoError(t, err)
	first, second := chatIntegrationSnapshot(t, uuid.New(), attachment.ID), chatIntegrationSnapshot(t, uuid.New(), attachment.ID)
	_, err = repo.SaveSession(ctx, owner.ID, first, 0)
	require.NoError(t, err)
	_, err = repo.SaveSession(ctx, owner.ID, second, 0)
	require.NoError(t, err)
	remote, claimed, err := repo.ClaimRemoteFile(ctx, owner.ID, attachment.ID, account.ID)
	require.NoError(t, err)
	require.True(t, claimed)
	require.NoError(t, repo.FinishRemoteFile(ctx, remote, "file_cleanup"))
	require.NoError(t, repo.DeleteSessions(ctx, owner.ID, &first.ID))
	require.NoError(t, svc.SweepChat(ctx))
	open, err := storage.OpenFile(owner.ID, attachment.ID)
	require.NoError(t, err)
	require.NoError(t, open.Close())
	require.NoError(t, repo.DeleteSessions(ctx, owner.ID, &second.ID))
	require.NoError(t, svc.SweepChat(ctx))
	_, err = storage.OpenFile(owner.ID, attachment.ID)
	require.ErrorIs(t, err, os.ErrNotExist)
	pending, err := client.ChatAttachment.Get(ctx, attachment.ID)
	require.NoError(t, err)
	require.Zero(t, pending.ByteSize)
	for attempt := 1; attempt < 3; attempt++ {
		state, err := client.ChatAttachmentUpload.Get(ctx, remote.ID)
		require.NoError(t, err)
		require.Equal(t, attempt, state.CleanupAttempts)
		require.NoError(t, svc.SweepChat(ctx))
		state, err = client.ChatAttachmentUpload.Get(ctx, remote.ID)
		require.NoError(t, err)
		require.Equal(t, attempt, state.CleanupAttempts)
		require.NoError(t, client.ChatAttachmentUpload.UpdateOneID(remote.ID).SetUpdatedAt(time.Now().Add(-3*time.Minute)).Exec(ctx))
		require.NoError(t, svc.SweepChat(ctx))
	}
	_, err = client.ChatAttachmentUpload.Get(ctx, remote.ID)
	require.True(t, dbent.IsNotFound(err))
	_, err = client.ChatAttachment.Get(ctx, attachment.ID)
	require.True(t, dbent.IsNotFound(err))
}

func TestChatAttachmentQuotaAndExpiredImportCleanup(t *testing.T) {
	tx := testEntTx(t)
	ctx := dbent.NewTxContext(context.Background(), tx)
	client := tx.Client()
	repo := NewChatRepository(integrationEntClient)
	owner := mustCreateUser(t, client, &service.User{})
	storage, err := service.NewChatStorage(t.TempDir())
	require.NoError(t, err)
	cfg := &config.Config{}
	var imageData bytes.Buffer
	require.NoError(t, png.Encode(&imageData, image.NewRGBA(image.Rect(0, 0, 2, 2))))
	cfg.Chat.UserStorageMaxBytes = int64(imageData.Len())
	svc, err := service.NewChatService(repo, storage, cfg, nil, nil, nil)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, svc.Stop()) })
	first, err := svc.UploadAttachment(ctx, owner.ID, "image.png", "image", bytes.NewReader(imageData.Bytes()), int64(imageData.Len()), nil)
	require.NoError(t, err)
	_, err = svc.UploadAttachment(ctx, owner.ID, "over.txt", "user_data", bytes.NewReader([]byte("x")), 1, nil)
	require.ErrorIs(t, err, service.ErrChatStorageFull)
	require.NoError(t, client.ChatAttachment.UpdateOneID(first.ID).SetUpdatedAt(time.Now().Add(-25*time.Hour)).Exec(ctx))
	require.NoError(t, svc.SweepChat(ctx))
	_, err = client.ChatAttachment.Get(ctx, first.ID)
	require.True(t, dbent.IsNotFound(err))
	staged, err := svc.BeginImport(ctx, owner.ID, 0)
	require.NoError(t, err)
	file, err := svc.UploadAttachment(ctx, owner.ID, "image.png", "image", bytes.NewReader(imageData.Bytes()), int64(imageData.Len()), &staged.ID)
	require.NoError(t, err)
	require.NoError(t, client.ChatImport.UpdateOneID(staged.ID).SetExpiresAt(time.Now().Add(-time.Minute)).Exec(ctx))
	require.NoError(t, svc.SweepChat(ctx))
	record, err := client.ChatImport.Get(ctx, staged.ID)
	require.NoError(t, err)
	require.Equal(t, chatimport.StatusFailed, record.Status)
	_, err = storage.OpenFile(owner.ID, file.ID)
	require.ErrorIs(t, err, os.ErrNotExist)
	require.NoError(t, client.ChatImport.UpdateOneID(staged.ID).SetUpdatedAt(time.Now().Add(-3*time.Minute)).Exec(ctx))
	require.NoError(t, svc.SweepChat(ctx))
	_, err = client.ChatImport.Get(ctx, staged.ID)
	require.True(t, dbent.IsNotFound(err))
	count, err := client.ChatAttachment.Query().Where(chatattachment.UserIDEQ(owner.ID)).Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count)
	count, err = client.ChatAttachmentUpload.Query().Where(chatattachmentupload.UserIDEQ(owner.ID)).Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestChatImportRejectsPreImportSnapshot(t *testing.T) {
	for _, sameID := range []bool{true, false} {
		t.Run(map[bool]string{true: "same-uuid", false: "removed-session"}[sameID], func(t *testing.T) {
			tx := testEntTx(t)
			ctx := dbent.NewTxContext(context.Background(), tx)
			repo := NewChatRepository(integrationEntClient)
			owner := mustCreateUser(t, tx.Client(), &service.User{})
			storage, err := service.NewChatStorage(t.TempDir())
			require.NoError(t, err)
			svc, err := service.NewChatService(repo, storage, &config.Config{}, nil, nil, nil)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, svc.Stop()) })
			original := chatIntegrationSnapshot(t, uuid.New())
			saved, err := repo.SaveSession(ctx, owner.ID, original, 0)
			require.NoError(t, err)
			_, revision, err := repo.GetPreferences(ctx, owner.ID)
			require.NoError(t, err)
			staged, err := svc.BeginImport(ctx, owner.ID, revision)
			require.NoError(t, err)
			id := uuid.New()
			if sameID {
				id = original.ID
			}
			replacement := chatIntegrationSnapshot(t, id)
			require.NoError(t, svc.StageImportSession(ctx, owner.ID, staged.ID, replacement.ID, replacement.Raw))
			_, err = svc.CommitImport(ctx, owner.ID, staged.ID, revision)
			require.NoError(t, err)
			_, err = repo.SaveSession(ctx, owner.ID, original, saved.Revision)
			require.ErrorIs(t, err, service.ErrChatRevisionConflict)
		})
	}
}
