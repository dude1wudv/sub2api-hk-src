package repository

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestAPIKeyCacheTTLPersistenceAndAuthProjection(t *testing.T) {
	repo, client := newAPIKeyRepoSQLite(t)
	ctx := context.Background()
	user := mustCreateAPIKeyRepoUser(t, ctx, client, "cache-ttl@example.test")
	for _, mode := range []string{"", "inherit", "off", "5m", "1h"} {
		key := &service.APIKey{UserID: user.ID, Key: "sk-cache-ttl-" + mode, Name: "cache ttl", Status: service.StatusActive, AnthropicCacheTTLMode: mode}
		require.NoError(t, repo.Create(ctx, key))
		want := mode
		if want == "" {
			want = "inherit"
		}
		loaded, err := repo.GetByKeyForAuth(ctx, key.Key)
		require.NoError(t, err)
		require.Equal(t, want, loaded.AnthropicCacheTTLMode)
		_, err = client.APIKey.UpdateOneID(key.ID).SetQuotaUsed(43).SetUsage5h(12).Save(ctx)
		require.NoError(t, err)
		key.AnthropicCacheTTLMode = "off"
		require.NoError(t, repo.Update(ctx, key, service.APIKeyUpdateFields{AnthropicCacheTTLMode: true}))
		updated, err := repo.GetByID(ctx, key.ID)
		require.NoError(t, err)
		require.Equal(t, "off", updated.AnthropicCacheTTLMode)
		require.Equal(t, float64(43), updated.QuotaUsed, "editing TTL must not overwrite live billing counters")
		require.Equal(t, float64(12), updated.Usage5h)
		key.Name = "renamed"
		key.AnthropicCacheTTLMode = "1h"
		require.NoError(t, repo.Update(ctx, key, service.APIKeyUpdateFields{Name: true}))
		updated, err = repo.GetByID(ctx, key.ID)
		require.NoError(t, err)
		require.Equal(t, "off", updated.AnthropicCacheTTLMode)
	}
}
