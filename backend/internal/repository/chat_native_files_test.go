//go:build integration

package repository

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/chatattachment"
	"github.com/Wei-Shaw/sub2api/ent/chatattachmentupload"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestChatNativeFileClaimsPreserveOwnerAndAccount(t *testing.T) {
	tx := testEntTx(t)
	ctx := dbent.NewTxContext(context.Background(), tx)
	client := tx.Client()
	repo := NewChatRepository(integrationEntClient)
	owner := mustCreateUser(t, client, &service.User{})
	other := mustCreateUser(t, client, &service.User{})
	accountA := mustCreateAccount(t, client, &service.Account{Name: "chat-file-a"}).ID
	accountB := mustCreateAccount(t, client, &service.Account{Name: "chat-file-b"}).ID
	accountC := mustCreateAccount(t, client, &service.Account{Name: "chat-file-c"}).ID
	id := uuid.New()
	_, err := client.ChatAttachment.Create().SetID(id).SetUserID(owner.ID).SetStorageKey(fmt.Sprintf("files/%d/%s", owner.ID, id)).SetOriginalName("test.txt").SetByteSize(1).SetPurpose(chatattachment.PurposeUserData).SetStatus(chatattachment.StatusReady).Save(ctx)
	require.NoError(t, err)
	first, claimed, err := repo.ClaimRemoteFile(ctx, owner.ID, id, accountA)
	require.NoError(t, err)
	require.True(t, claimed)
	same, claimed, err := repo.ClaimRemoteFile(ctx, owner.ID, id, accountA)
	require.NoError(t, err)
	require.False(t, claimed)
	require.Equal(t, first.ID, same.ID)
	_, _, err = repo.ClaimRemoteFile(ctx, other.ID, id, accountA)
	require.ErrorIs(t, err, service.ErrChatNotFound)
	require.NoError(t, repo.FinishRemoteFile(ctx, first, "file_account_a"))
	cached, claimed, err := repo.ClaimRemoteFile(ctx, owner.ID, id, accountA)
	require.NoError(t, err)
	require.False(t, claimed)
	require.Equal(t, "file_account_a", cached.RemoteID)
	second, claimed, err := repo.ClaimRemoteFile(ctx, owner.ID, id, accountB)
	require.NoError(t, err)
	require.True(t, claimed)
	require.NotEqual(t, first.ID, second.ID)
	require.NoError(t, repo.InvalidateRemoteFiles(ctx, other.ID, accountA, []uuid.UUID{id}))
	cached, _, err = repo.ClaimRemoteFile(ctx, owner.ID, id, accountA)
	require.NoError(t, err)
	require.Equal(t, "ready", cached.Status)
	require.NoError(t, repo.InvalidateRemoteFiles(ctx, owner.ID, accountA, []uuid.UUID{id}))
	replacement, claimed, err := repo.ClaimRemoteFile(ctx, owner.ID, id, accountA)
	require.NoError(t, err)
	require.True(t, claimed)
	require.ErrorIs(t, repo.FinishRemoteFile(ctx, first, "late_old_writer"), service.ErrChatLeaseLost)
	require.NoError(t, repo.FinishRemoteFile(ctx, replacement, "file_replacement"))
	require.NoError(t, repo.FinishRemoteFile(ctx, second, ""))
	_, _, err = repo.ClaimRemoteFile(ctx, owner.ID, id, accountB)
	require.ErrorIs(t, err, service.ErrChatRemoteFile)
	third, _, err := repo.ClaimRemoteFile(ctx, owner.ID, id, accountC)
	require.NoError(t, err)
	_, err = client.ChatAttachmentUpload.UpdateOneID(third.ID).SetUpdatedAt(time.Now().Add(-3 * time.Minute)).Save(ctx)
	require.NoError(t, err)
	_, _, err = repo.ClaimRemoteFile(ctx, owner.ID, id, accountC)
	require.ErrorIs(t, err, service.ErrChatRemoteFile)
	state, err := client.ChatAttachmentUpload.Get(ctx, third.ID)
	require.NoError(t, err)
	require.Equal(t, chatattachmentupload.StatusError, state.Status)
}

func TestChatNativeFileConcurrentDatabaseClaims(t *testing.T) {
	ctx := context.Background()
	owner := mustCreateUser(t, integrationEntClient, &service.User{})
	accountID := mustCreateAccount(t, integrationEntClient, &service.Account{Name: "chat-file-concurrent"}).ID
	repo := NewChatRepository(integrationEntClient)
	id := uuid.New()
	_, err := integrationEntClient.ChatAttachment.Create().SetID(id).SetUserID(owner.ID).SetStorageKey(fmt.Sprintf("files/%d/%s", owner.ID, id)).SetOriginalName("test.txt").SetByteSize(1).SetPurpose(chatattachment.PurposeUserData).SetStatus(chatattachment.StatusReady).Save(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = integrationEntClient.ChatAttachmentUpload.Delete().Where(chatattachmentupload.UserIDEQ(owner.ID)).Exec(ctx)
		_, _ = integrationEntClient.ChatAttachment.Delete().Where(chatattachment.UserIDEQ(owner.ID)).Exec(ctx)
		_ = integrationEntClient.User.DeleteOneID(owner.ID).Exec(ctx)
		_ = integrationEntClient.Account.DeleteOneID(accountID).Exec(ctx)
	})
	type outcome struct {
		claimed bool
		err     error
	}
	results := make(chan outcome, 4)
	var workers sync.WaitGroup
	for range 4 {
		workers.Go(func() {
			_, claimed, err := repo.ClaimRemoteFile(ctx, owner.ID, id, accountID)
			results <- outcome{claimed, err}
		})
	}
	workers.Wait()
	close(results)
	acquired := 0
	for result := range results {
		require.NoError(t, result.err)
		if result.claimed {
			acquired++
		}
	}
	require.Equal(t, 1, acquired)
}
