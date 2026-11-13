package service

import (
	"context"
	"fmt"
)

type ChatCredentialRecord struct {
	APIKeyID int64
	Key      *APIKey
}

// The callback executes under the user's row lock and shares its transaction.
// A nonnil binding with a nil Key is a deleted credential, not a new binding.
type ChatCredentialRepository interface {
	WithCredential(context.Context, int64, int64, func(context.Context, *ChatCredentialRecord) (*APIKey, error)) (*APIKey, error)
}

type chatCredentialKeys interface {
	GetAvailableGroups(context.Context, int64) ([]Group, error)
	Create(context.Context, int64, CreateAPIKeyRequest) (*APIKey, error)
}

type ChatCredentialService struct {
	repo ChatCredentialRepository
	keys chatCredentialKeys
}

func NewChatCredentialService(repo ChatCredentialRepository, keys *APIKeyService) *ChatCredentialService {
	return &ChatCredentialService{repo: repo, keys: keys}
}

func (s *ChatCredentialService) Ensure(ctx context.Context, userID, groupID int64) (*APIKey, error) {
	return s.ensure(ctx, userID, groupID, false)
}

// Reconnect never rotates a valid key. Only an explicit user operation may
// replace an invalid binding, through the ordinary create limits and policies.
func (s *ChatCredentialService) Reconnect(ctx context.Context, userID, groupID int64) error {
	_, err := s.ensure(ctx, userID, groupID, true)
	return err
}

func (s *ChatCredentialService) ensure(ctx context.Context, userID, groupID int64, reconnect bool) (*APIKey, error) {
	if userID <= 0 || groupID <= 0 {
		return nil, ErrChatNotFound
	}
	return s.repo.WithCredential(ctx, userID, groupID, func(tx context.Context, binding *ChatCredentialRecord) (*APIKey, error) {
		groups, err := s.keys.GetAvailableGroups(tx, userID)
		if err != nil {
			return nil, err
		}
		allowed := false
		for _, group := range groups {
			if group.ID == groupID {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, ErrChatNotFound
		}
		if binding != nil {
			if chatCredentialValid(binding.Key, userID, groupID) {
				return binding.Key, nil
			}
			if !reconnect {
				return nil, ErrChatCredentialUnavailable
			}
		}
		return s.keys.Create(tx, userID, CreateAPIKeyRequest{Name: fmt.Sprintf("Web Chat · %d", groupID), GroupID: &groupID})
	})
}

func chatCredentialValid(key *APIKey, userID, groupID int64) bool {
	return key != nil && key.UserID == userID && key.GroupID != nil && *key.GroupID == groupID && len(key.RoutingGroupIDs) == 0 && key.Key != "" && key.IsActive() && !key.IsExpired() && !key.IsQuotaExhausted()
}
