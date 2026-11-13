package repository

import (
	"context"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/apikey"
	"github.com/Wei-Shaw/sub2api/ent/chatcredential"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func NewChatCredentialRepository(client *dbent.Client) service.ChatCredentialRepository {
	return &chatRepository{client: client}
}

func (r *chatRepository) WithCredential(ctx context.Context, userID, groupID int64, fn func(context.Context, *service.ChatCredentialRecord) (*service.APIKey, error)) (*service.APIKey, error) {
	var result *service.APIKey
	err := r.withUser(ctx, userID, func(tx context.Context, client *dbent.Client) error {
		row, err := client.ChatCredential.Query().Where(chatcredential.UserIDEQ(userID), chatcredential.GroupIDEQ(groupID)).Only(tx)
		if err != nil && !dbent.IsNotFound(err) {
			return err
		}
		var binding *service.ChatCredentialRecord
		if row != nil {
			binding = &service.ChatCredentialRecord{APIKeyID: row.APIKeyID}
			key, readErr := client.APIKey.Query().Where(apikey.IDEQ(row.APIKeyID), apikey.DeletedAtIsNil()).Only(tx)
			if readErr != nil && !dbent.IsNotFound(readErr) {
				return readErr
			}
			if key != nil {
				binding.Key = apiKeyEntityToService(key)
			}
		}
		key, err := fn(tx, binding)
		if err != nil {
			return err
		}
		if key == nil || key.ID <= 0 || key.UserID != userID || key.GroupID == nil || *key.GroupID != groupID {
			return service.ErrChatCredentialUnavailable
		}
		if row == nil {
			_, err = client.ChatCredential.Create().SetUserID(userID).SetGroupID(groupID).SetAPIKeyID(key.ID).Save(tx)
		} else if row.APIKeyID != key.ID {
			_, err = client.ChatCredential.UpdateOneID(row.ID).SetAPIKeyID(key.ID).Save(tx)
		}
		if err != nil {
			return err
		}
		result = key
		return nil
	})
	return result, err
}
