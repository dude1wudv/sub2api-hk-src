//go:build unit

package service

import (
	"context"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestChatPolicyAuthSnapshotIsolated(t *testing.T) {
	policy := ChatModelPolicy{Transport: "messages", ImageInput: true, DocumentInput: true, NativeFiles: true}
	key := &APIKey{ID: 4, UserID: 7, User: &User{ID: 7}, Group: &Group{ID: 9, ChatModelPolicies: map[string]ChatModelPolicy{"exact-model": policy}}}
	service := &APIKeyService{}
	snapshot := service.snapshotFromAPIKey(context.Background(), key)
	require.NotNil(t, snapshot)
	require.Equal(t, policy, snapshot.Group.ChatModelPolicies["exact-model"])
	delete(key.Group.ChatModelPolicies, "exact-model")
	require.Len(t, snapshot.Group.ChatModelPolicies, 1)
	restored := service.snapshotToAPIKey("test-only", snapshot)
	require.Equal(t, policy, restored.Group.ChatModelPolicies["exact-model"])
	delete(restored.Group.ChatModelPolicies, "exact-model")
	require.Len(t, snapshot.Group.ChatModelPolicies, 1)
}
