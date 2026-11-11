//go:build unit

package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAPIKeyCacheTTLValidationAndAuthSnapshot(t *testing.T) {
	svc := &APIKeyService{}
	for _, mode := range []string{"inherit", "off", "5m", "1h"} {
		t.Run(mode, func(t *testing.T) {
			require.NoError(t, validateCreateAPIKeyRequest(CreateAPIKeyRequest{AnthropicCacheTTLMode: mode}))
			require.NoError(t, validateUpdateAPIKeyRequest(UpdateAPIKeyRequest{AnthropicCacheTTLMode: &mode}))
			key := &APIKey{ID: 1, UserID: 2, AnthropicCacheTTLMode: mode, User: &User{ID: 2}}
			snapshot := svc.snapshotFromAPIKey(context.Background(), key)
			wire, err := json.Marshal(snapshot)
			require.NoError(t, err)
			var restored APIKeyAuthSnapshot
			require.NoError(t, json.Unmarshal(wire, &restored))
			require.Equal(t, mode, svc.snapshotToAPIKey("sk-test", &restored).AnthropicCacheTTLMode)
		})
	}
	for _, mode := range []string{"1H", "5M", "2h", "invalid", " inherit "} {
		require.Error(t, validateCreateAPIKeyRequest(CreateAPIKeyRequest{AnthropicCacheTTLMode: mode}))
		require.Error(t, validateUpdateAPIKeyRequest(UpdateAPIKeyRequest{AnthropicCacheTTLMode: &mode}))
	}
	empty := ""
	require.Error(t, validateUpdateAPIKeyRequest(UpdateAPIKeyRequest{AnthropicCacheTTLMode: &empty}))
	require.NoError(t, validateCreateAPIKeyRequest(CreateAPIKeyRequest{}))
	_, used, err := svc.applyAuthCacheEntry("sk-test", &APIKeyAuthCacheEntry{Snapshot: &APIKeyAuthSnapshot{Version: 28}})
	require.NoError(t, err)
	require.False(t, used, "old auth caches cannot silently lose a configured key policy")
}

func TestAPIKeyCacheTTLCreateAndUpdate(t *testing.T) {
	for _, mode := range []string{"", "off", "5m", "1h"} {
		t.Run("create "+mode, func(t *testing.T) {
			repo := &createLimitAPIKeyRepoStub{}
			svc := newCreateLimitService(repo, nil, 0, 0)
			svc.cache = nil // Avoid assigning a typed nil cache stub to its interface.
			key, err := svc.Create(context.Background(), 7, CreateAPIKeyRequest{Name: "ttl-test", AnthropicCacheTTLMode: mode})
			require.NoError(t, err)
			require.Len(t, repo.created, 1)
			want := mode
			if want == "" {
				want = "inherit"
			}
			require.Equal(t, want, key.AnthropicCacheTTLMode)
		})
	}
	key := &APIKey{ID: 1, UserID: 7, Key: "sk-test", Status: StatusActive, AnthropicCacheTTLMode: "1h", QuotaUsed: 42, Usage5h: 12}
	svc, repo := newUpdateFieldsAPIKeyService(key)
	name := "renamed"
	got, err := svc.Update(context.Background(), 1, 7, UpdateAPIKeyRequest{Name: &name})
	require.NoError(t, err)
	require.Equal(t, "1h", got.AnthropicCacheTTLMode, "omitted field preserves policy")
	require.Equal(t, APIKeyUpdateFields{Name: true}, repo.updateFields[0])
	mode := "off"
	got, err = svc.Update(context.Background(), 1, 7, UpdateAPIKeyRequest{AnthropicCacheTTLMode: &mode})
	require.NoError(t, err)
	require.Equal(t, "off", got.AnthropicCacheTTLMode)
	require.Equal(t, APIKeyUpdateFields{AnthropicCacheTTLMode: true}, repo.updateFields[1])
	require.Equal(t, float64(42), got.QuotaUsed)
	require.Equal(t, float64(12), got.Usage5h)
}
