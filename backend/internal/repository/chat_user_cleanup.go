package repository

import (
	"context"
	"encoding/json"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/chatattachment"
	"github.com/Wei-Shaw/sub2api/ent/chatattachmentupload"
	"github.com/Wei-Shaw/sub2api/ent/chatcredential"
	"github.com/Wei-Shaw/sub2api/ent/chatimport"
	"github.com/Wei-Shaw/sub2api/ent/chatmodelpreset"
	"github.com/Wei-Shaw/sub2api/ent/chatpreference"
	"github.com/Wei-Shaw/sub2api/ent/chatsession"
)

// Called inside the existing user-deletion transaction. Files remain registered
// until the chat sweeper removes them after commit; rollbacks never unlink data.
func deleteChatUser(ctx context.Context, client *dbent.Client, userID int64) error {
	if err := chatDiscardSettledParseBodies(ctx, client, userID, nil); err != nil {
		return err
	}
	if _, err := client.ChatSession.Delete().Where(chatsession.UserIDEQ(userID)).Exec(ctx); err != nil {
		return err
	}
	if _, err := client.ChatCredential.Delete().Where(chatcredential.UserIDEQ(userID)).Exec(ctx); err != nil {
		return err
	}
	if _, err := client.ChatModelPreset.Delete().Where(chatmodelpreset.UserIDEQ(userID)).Exec(ctx); err != nil {
		return err
	}
	if _, err := client.ChatPreference.Delete().Where(chatpreference.UserIDEQ(userID)).Exec(ctx); err != nil {
		return err
	}
	if _, err := client.ChatAttachmentUpload.Update().Where(chatattachmentupload.UserIDEQ(userID), chatattachmentupload.StatusNEQ(chatattachmentupload.StatusUploading)).SetStatus(chatattachmentupload.StatusDeleting).Save(ctx); err != nil {
		return err
	}
	if _, err := client.ChatAttachment.Update().Where(chatattachment.UserIDEQ(userID)).SetStatus(chatattachment.StatusDeleting).Save(ctx); err != nil {
		return err
	}
	_, err := client.ChatImport.Update().Where(chatimport.UserIDEQ(userID)).SetStatus(chatimport.StatusFailed).SetExpiresAt(time.Now()).SetManifest(json.RawMessage(`{}`)).Save(ctx)
	return err
}
