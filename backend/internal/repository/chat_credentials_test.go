//go:build integration

package repository

import (
	"context"
	"sync"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/apikey"
	"github.com/Wei-Shaw/sub2api/ent/chatcredential"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func chatCredentialIntegration(t *testing.T) (*service.ChatCredentialService, *service.APIKeyService, *dbent.Client, int64, int64) {
	t.Helper()
	client := testEntClient(t)
	owner := mustCreateUser(t, client, &service.User{})
	group, err := client.Group.Create().SetName("chat-credentials-" + uuid.NewString()).SetPlatform(service.PlatformOpenAI).Save(context.Background())
	require.NoError(t, err)
	keys := service.NewAPIKeyService(NewAPIKeyRepository(client, integrationDB), NewUserRepository(client, integrationDB), NewGroupRepository(client, integrationDB), NewUserSubscriptionRepository(client), NewUserGroupRateRepository(integrationDB), nil, &config.Config{APIKeyCreate: config.APIKeyCreateConfig{MaxActivePerUser: 20}})
	return service.NewChatCredentialService(NewChatCredentialRepository(client), keys), keys, client, owner.ID, group.ID
}

func TestChatCredentialSingleConnectionTransaction(t *testing.T) {
	s, _, _, user, group := chatCredentialIntegration(t)
	previous := integrationDB.Stats().MaxOpenConnections
	integrationDB.SetMaxOpenConns(1)
	defer integrationDB.SetMaxOpenConns(previous)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.Ensure(ctx, user, group)
	require.NoError(t, err, "permission reads and ordinary creation must share the locked transaction connection")
}

func TestChatCredentialConcurrentFirstUseAndDeletedBinding(t *testing.T) {
	s, _, client, user, group := chatCredentialIntegration(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	const attempts = 8
	ids := make(chan int64, attempts)
	errs := make(chan error, attempts)
	var wg sync.WaitGroup
	for range attempts {
		wg.Add(1)
		go func() {
			defer wg.Done()
			key, err := s.Ensure(ctx, user, group)
			errs <- err
			if key != nil {
				ids <- key.ID
			}
		}()
	}
	wg.Wait()
	close(errs)
	close(ids)
	for err := range errs {
		require.NoError(t, err)
	}
	var winner int64
	for id := range ids {
		if winner == 0 {
			winner = id
		}
		require.Equal(t, winner, id)
	}
	require.NotZero(t, winner)
	count, err := client.APIKey.Query().Where(apikey.UserIDEQ(user)).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)
	count, err = client.ChatCredential.Query().Where(chatcredential.UserIDEQ(user)).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, count)
	require.NoError(t, NewAPIKeyRepository(client, integrationDB).Delete(ctx, winner))
	_, err = s.Ensure(ctx, user, group)
	require.ErrorIs(t, err, service.ErrChatCredentialUnavailable)
	require.NoError(t, s.Reconnect(ctx, user, group))
	next, err := s.Ensure(ctx, user, group)
	require.NoError(t, err)
	require.NotEqual(t, winner, next.ID)
}

func TestChatAPIKeyCreateSharesTransactionAndRollsBack(t *testing.T) {
	_, keys, client, user, group := chatCredentialIntegration(t)
	tx, err := client.Tx(context.Background())
	require.NoError(t, err)
	key, err := keys.Create(dbent.NewTxContext(context.Background(), tx), user, service.CreateAPIKeyRequest{Name: "chat-rollback", GroupID: &group})
	require.NoError(t, err)
	require.NoError(t, tx.Rollback())
	exists, err := client.APIKey.Query().Where(apikey.IDEQ(key.ID)).Exist(context.Background())
	require.NoError(t, err)
	require.False(t, exists)
	ordinary, err := keys.Create(context.Background(), user, service.CreateAPIKeyRequest{Name: "chat-ordinary", GroupID: &group})
	require.NoError(t, err)
	exists, err = client.APIKey.Query().Where(apikey.IDEQ(ordinary.ID)).Exist(context.Background())
	require.NoError(t, err)
	require.True(t, exists)
}
