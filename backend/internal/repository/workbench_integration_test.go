//go:build integration

package repository

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sync"
	"testing"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/apikey"
	"github.com/Wei-Shaw/sub2api/ent/workbenchcredential"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func workbenchIntegrationUser(t *testing.T, client *dbent.Client) *service.User {
	t.Helper()
	return mustCreateUser(t, client, &service.User{
		Email: fmt.Sprintf("workbench-%d@example.com", time.Now().UnixNano()),
	})
}

func workbenchIntegrationGroup(t *testing.T, client *dbent.Client) int64 {
	t.Helper()
	group, err := client.Group.Create().
		SetName(fmt.Sprintf("workbench-group-%d", time.Now().UnixNano())).
		SetPlatform(service.PlatformOpenAI).
		SetAllowImageGeneration(true).
		Save(context.Background())
	require.NoError(t, err)
	return group.ID
}

func TestWorkbenchRepositoryEnsureCredentialFirstAndRepeatedReuse(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewWorkbenchIntegrationRepository(client)
	user := workbenchIntegrationUser(t, client)
	groupID := workbenchIntegrationGroup(t, client)

	first, err := repo.EnsureCredential(ctx, user.ID, service.WorkbenchHeliosGen, "sk-workbench-first", groupID)
	require.NoError(t, err)
	require.NotZero(t, first.ID)
	require.NotZero(t, first.APIKeyID)
	repeatedGroupID := workbenchIntegrationGroup(t, client)

	repeated, err := repo.EnsureCredential(ctx, user.ID, service.WorkbenchHeliosGen, "sk-workbench-second", repeatedGroupID)
	require.NoError(t, err)
	require.Equal(t, first.ID, repeated.ID)
	require.Equal(t, first.APIKeyID, repeated.APIKeyID)
	key, err := client.APIKey.Get(ctx, first.APIKeyID)
	require.NoError(t, err)
	require.NotNil(t, key.GroupID)
	require.Equal(t, repeatedGroupID, *key.GroupID)

	keyCount, err := client.APIKey.Query().Where(
		apikey.UserIDEQ(user.ID),
		apikey.NameEQ(service.WorkbenchKeyName),
		apikey.DeletedAtIsNil(),
	).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, keyCount)
	bindingCount, err := client.WorkbenchCredential.Query().Where(
		workbenchcredential.UserIDEQ(user.ID),
		workbenchcredential.WorkbenchEQ(service.WorkbenchHeliosGen),
	).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, bindingCount)
}

func TestWorkbenchRepositoryEnsureCredentialConcurrentFirstUseLeavesOneKey(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewWorkbenchIntegrationRepository(client)
	user := workbenchIntegrationUser(t, client)
	groupID := workbenchIntegrationGroup(t, client)

	const attempts = 8
	results := make(chan *service.WorkbenchCredential, attempts)
	errs := make(chan error, attempts)
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			credential, err := repo.EnsureCredential(ctx, user.ID, service.WorkbenchHeliosGen, fmt.Sprintf("sk-workbench-race-%d", i), groupID)
			results <- credential
			errs <- err
		}(i)
	}
	wg.Wait()
	close(results)
	close(errs)

	var winner *service.WorkbenchCredential
	for credential := range results {
		if winner == nil && credential != nil {
			winner = credential
		}
	}
	require.NotNil(t, winner)
	for err := range errs {
		require.NoError(t, err)
	}
	keyCount, err := client.APIKey.Query().Where(
		apikey.UserIDEQ(user.ID),
		apikey.NameEQ(service.WorkbenchKeyName),
		apikey.DeletedAtIsNil(),
	).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, keyCount)
}

func TestWorkbenchRepositoryEnsureCredentialDisabledKeyReturnsConflict(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewWorkbenchIntegrationRepository(client)
	user := workbenchIntegrationUser(t, client)
	groupID := workbenchIntegrationGroup(t, client)

	credential, err := repo.EnsureCredential(ctx, user.ID, service.WorkbenchHeliosGen, "sk-workbench-disabled", groupID)
	require.NoError(t, err)
	_, err = client.APIKey.UpdateOneID(credential.APIKeyID).
		SetStatus(service.StatusAPIKeyDisabled).
		Save(ctx)
	require.NoError(t, err)

	_, err = repo.EnsureCredential(ctx, user.ID, service.WorkbenchHeliosGen, "sk-workbench-not-reenabled", groupID)
	require.ErrorIs(t, err, service.ErrWorkbenchKeyDisabled)
	keyCount, err := client.APIKey.Query().Where(
		apikey.UserIDEQ(user.ID),
		apikey.NameEQ(service.WorkbenchKeyName),
		apikey.DeletedAtIsNil(),
	).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, keyCount)
}

func TestWorkbenchRepositoryEnsureCredentialReplacesSoftDeletedKey(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewWorkbenchIntegrationRepository(client)
	user := workbenchIntegrationUser(t, client)
	groupID := workbenchIntegrationGroup(t, client)

	first, err := repo.EnsureCredential(ctx, user.ID, service.WorkbenchHeliosGen, "sk-workbench-deleted", groupID)
	require.NoError(t, err)
	require.NoError(t, client.APIKey.DeleteOneID(first.APIKeyID).Exec(ctx))

	replacement, err := repo.EnsureCredential(ctx, user.ID, service.WorkbenchHeliosGen, "sk-workbench-replacement", groupID)
	require.NoError(t, err)
	require.NotEqual(t, first.APIKeyID, replacement.APIKeyID)
	keyCount, err := client.APIKey.Query().Where(
		apikey.UserIDEQ(user.ID),
		apikey.NameEQ(service.WorkbenchKeyName),
		apikey.DeletedAtIsNil(),
	).Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, keyCount)
	binding, err := client.WorkbenchCredential.Query().Where(
		workbenchcredential.UserIDEQ(user.ID),
		workbenchcredential.WorkbenchEQ(service.WorkbenchHeliosGen),
	).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, replacement.APIKeyID, binding.APIKeyID)
}

func workbenchGrantHash(raw string) string {
	digest := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", digest[:])
}

func TestWorkbenchRepositoryGrantExpiryAndAtomicConsume(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := NewWorkbenchIntegrationRepository(client)
	user := workbenchIntegrationUser(t, client)
	groupID := workbenchIntegrationGroup(t, client)
	credential, err := repo.EnsureCredential(ctx, user.ID, service.WorkbenchHeliosGen, "sk-workbench-grants", groupID)
	require.NoError(t, err)

	now := time.Now().UTC()
	expired := &service.WorkbenchLaunchGrant{
		CodeHash: workbenchGrantHash("expired-code"), UserID: user.ID, APIKeyID: credential.APIKeyID,
		ClientID: "heliosgen-web", RedirectURI: "https://canvas.sub.sunmmyapi.xyz/bootstrap", ExpiresAt: now.Add(-time.Second),
	}
	require.NoError(t, repo.CreateGrant(ctx, expired))
	_, err = repo.ConsumeGrant(ctx, expired.CodeHash, now)
	require.ErrorIs(t, err, service.ErrWorkbenchInvalidGrant)

	active := &service.WorkbenchLaunchGrant{
		CodeHash: workbenchGrantHash("active-code"), UserID: user.ID, APIKeyID: credential.APIKeyID,
		ClientID: "heliosgen-web", RedirectURI: "https://canvas.sub.sunmmyapi.xyz/bootstrap", ExpiresAt: now.Add(90 * time.Second),
	}
	require.NoError(t, repo.CreateGrant(ctx, active))

	const attempts = 8
	results := make(chan *service.WorkbenchGrantResolution, attempts)
	errs := make(chan error, attempts)
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resolution, consumeErr := repo.ConsumeGrant(ctx, active.CodeHash, now)
			results <- resolution
			errs <- consumeErr
		}()
	}
	wg.Wait()
	close(results)
	close(errs)

	successes := 0
	for resolution := range results {
		if resolution != nil {
			successes++
		}
	}
	require.Equal(t, 1, successes)
	invalid := 0
	for consumeErr := range errs {
		if consumeErr != nil {
			invalid++
		}
	}
	require.Equal(t, attempts-1, invalid)
}
