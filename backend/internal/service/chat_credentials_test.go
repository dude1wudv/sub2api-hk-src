//go:build unit

package service

import (
	"context"
	"errors"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type chatCredentialRepoStub struct{ binding *ChatCredentialRecord }

func (r *chatCredentialRepoStub) WithCredential(ctx context.Context, _ int64, _ int64, fn func(context.Context, *ChatCredentialRecord) (*APIKey, error)) (*APIKey, error) {
	key, err := fn(ctx, r.binding)
	if err == nil {
		r.binding = &ChatCredentialRecord{APIKeyID: key.ID, Key: key}
	}
	return key, err
}

type chatCredentialKeysStub struct {
	groups  []Group
	calls   int
	request CreateAPIKeyRequest
	err     error
}

func (s *chatCredentialKeysStub) GetAvailableGroups(context.Context, int64) ([]Group, error) {
	return s.groups, nil
}
func (s *chatCredentialKeysStub) Create(_ context.Context, user int64, request CreateAPIKeyRequest) (*APIKey, error) {
	s.calls++
	s.request = request
	if s.err != nil {
		return nil, s.err
	}
	return &APIKey{ID: int64(s.calls + 10), UserID: user, GroupID: request.GroupID, Status: StatusActive, Key: "synthetic-chat-key"}, nil
}
func TestChatCredentialOrdinaryCreateAndStableReuse(t *testing.T) {
	repo := &chatCredentialRepoStub{}
	keys := &chatCredentialKeysStub{groups: []Group{{ID: 11}}}
	s := &ChatCredentialService{repo: repo, keys: keys}
	a, err := s.Ensure(context.Background(), 7, 11)
	require.NoError(t, err)
	b, err := s.Ensure(context.Background(), 7, 11)
	require.NoError(t, err)
	require.Equal(t, a.ID, b.ID)
	require.Equal(t, 1, keys.calls)
	require.Equal(t, "Web Chat · 11", keys.request.Name)
	require.Equal(t, int64(11), *keys.request.GroupID)
	require.Empty(t, keys.request.RoutingGroupIDs)
	require.NoError(t, s.Reconnect(context.Background(), 7, 11))
	require.Equal(t, 1, keys.calls)
	keys.groups = nil
	_, err = s.Ensure(context.Background(), 7, 11)
	require.ErrorIs(t, err, ErrChatNotFound)
	require.Equal(t, 1, keys.calls)
}
func TestChatCredentialInvalidBindingsNeverAutoReplace(t *testing.T) {
	groupID := int64(11)
	past := time.Now().Add(-time.Minute)
	for _, tc := range []struct {
		name   string
		mutate func(*APIKey)
	}{
		{"disabled", func(k *APIKey) { k.Status = StatusAPIKeyDisabled }},
		{"expired", func(k *APIKey) { k.ExpiresAt = &past }},
		{"quota", func(k *APIKey) { k.Quota = 1; k.QuotaUsed = 1 }},
		{"moved", func(k *APIKey) { other := int64(12); k.GroupID = &other }},
		{"owner", func(k *APIKey) { k.UserID = 8 }},
		{"routing", func(k *APIKey) { k.RoutingGroupIDs = []int64{11, 12} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			key := &APIKey{ID: 5, UserID: 7, GroupID: &groupID, Status: StatusActive, Key: "synthetic"}
			tc.mutate(key)
			repo := &chatCredentialRepoStub{binding: &ChatCredentialRecord{APIKeyID: 5, Key: key}}
			keys := &chatCredentialKeysStub{groups: []Group{{ID: 11}}}
			s := &ChatCredentialService{repo: repo, keys: keys}
			_, err := s.Ensure(context.Background(), 7, 11)
			require.ErrorIs(t, err, ErrChatCredentialUnavailable)
			require.Zero(t, keys.calls)
			keys.err = errors.New("create limit")
			require.ErrorIs(t, s.Reconnect(context.Background(), 7, 11), keys.err)
			require.Equal(t, int64(5), repo.binding.APIKeyID)
			keys.err = nil
			require.NoError(t, s.Reconnect(context.Background(), 7, 11))
			require.NotEqual(t, int64(5), repo.binding.APIKeyID)
		})
	}
	repo := &chatCredentialRepoStub{binding: &ChatCredentialRecord{APIKeyID: 5}}
	keys := &chatCredentialKeysStub{groups: []Group{{ID: 11}}}
	s := &ChatCredentialService{repo: repo, keys: keys}
	_, err := s.Ensure(context.Background(), 7, 11)
	require.ErrorIs(t, err, ErrChatCredentialUnavailable)
	require.Zero(t, keys.calls)
}
