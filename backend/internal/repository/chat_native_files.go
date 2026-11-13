package repository

import (
	"context"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/chatattachment"
	"github.com/Wei-Shaw/sub2api/ent/chatattachmentupload"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
)

func chatRemoteFileRecord(row *dbent.ChatAttachmentUpload) *service.ChatRemoteFileRecord {
	return &service.ChatRemoteFileRecord{ID: row.ID, UserID: row.UserID, AttachmentID: row.AttachmentID, AccountID: row.AccountID, RemoteID: row.RemoteFileID, Status: string(row.Status), UpdatedAt: row.UpdatedAt, CleanupAttempts: row.CleanupAttempts}
}

// User-row locking serializes claims across processes; no database lock spans IO.
func (r *chatRepository) ClaimRemoteFile(ctx context.Context, userID int64, attachmentID uuid.UUID, accountID int64) (*service.ChatRemoteFileRecord, bool, error) {
	if attachmentID == uuid.Nil || accountID <= 0 {
		return nil, false, service.ErrChatInvalid
	}
	var record *service.ChatRemoteFileRecord
	acquired := false
	err := r.withUser(ctx, userID, func(ctx context.Context, c *dbent.Client) error {
		exists, err := c.ChatAttachment.Query().Where(chatattachment.UserIDEQ(userID), chatattachment.IDEQ(attachmentID), chatattachment.StatusEQ(chatattachment.StatusReady), chatattachment.ImportIDIsNil()).Exist(ctx)
		if err != nil {
			return err
		}
		if !exists {
			return service.ErrChatNotFound
		}
		row, err := c.ChatAttachmentUpload.Query().Where(chatattachmentupload.UserIDEQ(userID), chatattachmentupload.AttachmentIDEQ(attachmentID), chatattachmentupload.AccountIDEQ(accountID)).Only(ctx)
		now := time.Now().UTC().Truncate(time.Microsecond)
		switch {
		case dbent.IsNotFound(err):
			row, err = c.ChatAttachmentUpload.Create().SetUserID(userID).SetAttachmentID(attachmentID).SetAccountID(accountID).SetUpdatedAt(now).Save(ctx)
			acquired = err == nil
		case err != nil:
			return err
		case row.Status == chatattachmentupload.StatusInvalid:
			row, err = c.ChatAttachmentUpload.UpdateOne(row).SetStatus(chatattachmentupload.StatusUploading).SetRemoteFileID("").SetUpdatedAt(now).Save(ctx)
			acquired = err == nil
		case row.Status == chatattachmentupload.StatusUploading && row.UpdatedAt.Before(now.Add(-2*time.Minute)):
			// A crashed or timed-out upload has an unknown outcome. Never replay it.
			row, err = c.ChatAttachmentUpload.UpdateOne(row).SetStatus(chatattachmentupload.StatusError).Save(ctx)
		}
		if err != nil {
			return err
		}
		record = chatRemoteFileRecord(row)
		return nil
	})
	if err == nil && record != nil && (record.Status == "error" || record.Status == "deleting") {
		err = service.ErrChatRemoteFile
	}
	return record, acquired, err
}

func (r *chatRepository) FinishRemoteFile(ctx context.Context, claim *service.ChatRemoteFileRecord, remoteID string) error {
	if claim == nil {
		return service.ErrChatInvalid
	}
	status := chatattachmentupload.StatusReady
	if remoteID == "" {
		status = chatattachmentupload.StatusError
	}
	count, err := clientFromContext(ctx, r.client).ChatAttachmentUpload.Update().Where(chatattachmentupload.IDEQ(claim.ID), chatattachmentupload.UserIDEQ(claim.UserID), chatattachmentupload.StatusEQ(chatattachmentupload.StatusUploading), chatattachmentupload.UpdatedAtEQ(claim.UpdatedAt)).SetRemoteFileID(remoteID).SetStatus(status).Save(ctx)
	if err != nil {
		return err
	}
	if count != 1 {
		return service.ErrChatLeaseLost
	}
	return nil
}

func (r *chatRepository) InvalidateRemoteFiles(ctx context.Context, userID, accountID int64, attachmentIDs []uuid.UUID) error {
	if len(attachmentIDs) == 0 {
		return nil
	}
	_, err := clientFromContext(ctx, r.client).ChatAttachmentUpload.Update().Where(chatattachmentupload.UserIDEQ(userID), chatattachmentupload.AccountIDEQ(accountID), chatattachmentupload.AttachmentIDIn(attachmentIDs...), chatattachmentupload.StatusEQ(chatattachmentupload.StatusReady)).SetStatus(chatattachmentupload.StatusInvalid).Save(ctx)
	return err
}
