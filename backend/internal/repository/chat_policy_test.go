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

func TestChatPolicyAuthProjection(t *testing.T) {
	tx := testEntTx(t)
	ctx := dbent.NewTxContext(context.Background(), tx)
	client := tx.Client()
	policies := map[string]service.ChatModelPolicy{"exact-model": {Transport: "messages", DocumentInput: true, NativeFiles: true}}
	group, err := client.Group.Create().SetName(uuid.NewString()).SetPlatform(service.PlatformOpenAI).SetChatModelPolicies(policies).Save(ctx)
	require.NoError(t, err)
	owner := mustCreateUser(t, client, &service.User{})
	repo := NewAPIKeyRepository(client, integrationDB)
	key := &service.APIKey{UserID: owner.ID, GroupID: &group.ID, Key: uuid.NewString(), Name: "chat-policy-test", Status: service.StatusActive}
	require.NoError(t, repo.Create(ctx, key))
	got, err := repo.GetByKeyForAuth(ctx, key.Key)
	require.NoError(t, err)
	require.NotNil(t, got.Group)
	require.Equal(t, policies, got.Group.ChatModelPolicies)
}
