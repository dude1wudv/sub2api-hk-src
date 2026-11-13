//go:build integration

package repository

import (
	"context"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestChatPresetsOwnerBindingDefaultsAndOrder(t *testing.T) {
	tx := testEntTx(t)
	ctx := dbent.NewTxContext(context.Background(), tx)
	client := tx.Client()
	a := mustCreateUser(t, client, &service.User{})
	b := mustCreateUser(t, client, &service.User{})
	x, err := client.Group.Create().SetName("chat-preset-x").SetPlatform(service.PlatformOpenAI).Save(ctx)
	require.NoError(t, err)
	y, err := client.Group.Create().SetName("chat-preset-y").SetPlatform(service.PlatformOpenAI).Save(ctx)
	require.NoError(t, err)
	repo := NewChatPresetRepository(integrationEntClient)
	first, err := repo.CreatePreset(ctx, a.ID, x.ID, "shared-model", "first")
	require.NoError(t, err)
	require.True(t, first.IsDefault)
	duplicate, err := repo.CreatePreset(ctx, a.ID, x.ID, "shared-model", "ignored")
	require.NoError(t, err)
	require.Equal(t, first.ID, duplicate.ID)
	require.Equal(t, "first", duplicate.DisplayName)
	second, err := repo.CreatePreset(ctx, a.ID, y.ID, "shared-model", "second")
	require.NoError(t, err)
	require.NotEqual(t, first.ID, second.ID)
	_, err = repo.GetPreset(ctx, b.ID, first.ID)
	require.ErrorIs(t, err, service.ErrChatNotFound)
	require.NoError(t, repo.DeletePreset(ctx, b.ID, first.ID))
	_, err = repo.GetPreset(ctx, a.ID, first.ID)
	require.NoError(t, err)
	yes := true
	_, err = repo.UpdatePreset(ctx, a.ID, second.ID, service.ChatPresetPatch{IsDefault: &yes})
	require.NoError(t, err)
	first, err = repo.GetPreset(ctx, a.ID, first.ID)
	require.NoError(t, err)
	require.False(t, first.IsDefault)
	require.ErrorIs(t, repo.OrderPresets(ctx, a.ID, []uuid.UUID{first.ID, first.ID}), service.ErrChatInvalid)
	require.ErrorIs(t, repo.OrderPresets(ctx, a.ID, []uuid.UUID{first.ID}), service.ErrChatInvalid)
	require.ErrorIs(t, repo.OrderPresets(ctx, a.ID, []uuid.UUID{first.ID, uuid.New()}), service.ErrChatInvalid)
	require.NoError(t, repo.OrderPresets(ctx, a.ID, []uuid.UUID{second.ID, first.ID}))
	items, err := repo.ListPresets(ctx, a.ID)
	require.NoError(t, err)
	require.Equal(t, second.ID, items[0].ID)
	_, err = repo.UpdatePreset(ctx, a.ID, second.ID, service.ChatPresetPatch{GroupID: &x.ID})
	require.ErrorIs(t, err, service.ErrChatPresetDuplicate)
	require.NoError(t, repo.DeletePreset(ctx, a.ID, first.ID))
	require.NoError(t, repo.DeletePreset(ctx, a.ID, first.ID))
}
