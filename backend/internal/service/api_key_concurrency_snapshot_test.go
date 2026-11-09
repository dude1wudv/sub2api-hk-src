//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAPIKeyConcurrencyLimitSurvivesAuthSnapshot(t *testing.T) {
	svc := &APIKeyService{}
	original := &APIKey{ID: 11, UserID: 7, ConcurrencyLimit: 3, Status: StatusActive, User: &User{ID: 7, Status: StatusActive}}
	snapshot := svc.snapshotFromAPIKey(context.Background(), original)
	require.Equal(t, 3, snapshot.ConcurrencyLimit)
	restored := svc.snapshotToAPIKey("credential", snapshot)
	require.Equal(t, 3, restored.ConcurrencyLimit)
	_, used, err := svc.applyAuthCacheEntry("credential", &APIKeyAuthCacheEntry{Snapshot: &APIKeyAuthSnapshot{Version: 27}})
	require.NoError(t, err)
	require.False(t, used, "pre-limit cached snapshots must not silently disable enforcement")
}

func TestAPIKeyConcurrencyLimitValidation(t *testing.T) {
	negative := -1
	zero := 0
	require.Error(t, validateCreateAPIKeyRequest(CreateAPIKeyRequest{ConcurrencyLimit: -1}))
	require.Error(t, validateUpdateAPIKeyRequest(UpdateAPIKeyRequest{ConcurrencyLimit: &negative}))
	require.NoError(t, validateUpdateAPIKeyRequest(UpdateAPIKeyRequest{ConcurrencyLimit: &zero}))
}
