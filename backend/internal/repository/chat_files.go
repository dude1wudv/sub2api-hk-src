package repository

import (
	"context"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/chatattachment"
	"github.com/Wei-Shaw/sub2api/ent/chatimport"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
)

func chatAttachmentRecord(row *dbent.ChatAttachment) *service.ChatAttachmentRecord {
	return &service.ChatAttachmentRecord{ID: row.ID, UserID: row.UserID, Name: row.OriginalName, Size: row.ByteSize, MIME: row.MimeType, IsImage: service.IsChatPreviewMIME(row.MimeType), StorageKey: row.StorageKey, SHA256: row.Sha256, Purpose: string(row.Purpose), Status: string(row.Status), ImportID: row.ImportID}
}

func chatRequireImport(ctx context.Context, c *dbent.Client, userID int64, id uuid.UUID) error {
	exists, err := c.ChatImport.Query().Where(chatimport.UserIDEQ(userID), chatimport.IDEQ(id), chatimport.StatusEQ(chatimport.StatusPending), chatimport.ExpiresAtGT(time.Now())).Exist(ctx)
	if err != nil {
		return err
	}
	if !exists {
		return service.ErrChatNotFound
	}
	return nil
}

func (r *chatRepository) ReserveAttachment(ctx context.Context, file *service.ChatAttachmentRecord, limit int64) error {
	if file == nil || file.ID == uuid.Nil || file.Size < 0 || limit <= 0 {
		return service.ErrChatInvalid
	}
	return r.withUser(ctx, file.UserID, func(ctx context.Context, c *dbent.Client) error {
		if file.ImportID != nil {
			if err := chatRequireImport(ctx, c, file.UserID, *file.ImportID); err != nil {
				return err
			}
		}
		used, err := chatReservedStorage(ctx, c, file.UserID, nil)
		if err != nil {
			return err
		}
		if file.ImportID != nil {
			imported, err := chatReservedStorage(ctx, c, file.UserID, file.ImportID)
			if err != nil {
				return err
			}
			if imported > service.ChatMaxArchiveBytes || file.Size > service.ChatMaxArchiveBytes-imported {
				return service.ErrChatTooLarge
			}
		}
		if used > limit || file.Size > limit-used {
			return service.ErrChatStorageFull
		}
		_, err = c.ChatAttachment.Create().SetID(file.ID).SetUserID(file.UserID).SetStorageKey(file.StorageKey).SetOriginalName(file.Name).SetMimeType(file.MIME).SetByteSize(file.Size).SetPurpose(chatattachment.Purpose(file.Purpose)).SetStatus(chatattachment.StatusUploading).SetNillableImportID(file.ImportID).Save(ctx)
		return err
	})
}

func (r *chatRepository) FinishAttachment(ctx context.Context, file *service.ChatAttachmentRecord) error {
	return r.withUser(ctx, file.UserID, func(ctx context.Context, c *dbent.Client) error {
		if file.ImportID != nil {
			if err := chatRequireImport(ctx, c, file.UserID, *file.ImportID); err != nil {
				return err
			}
		}
		status := chatattachment.StatusReady
		if file.ImportID != nil {
			status = chatattachment.StatusStaged
		}
		changed, err := c.ChatAttachment.Update().Where(chatattachment.UserIDEQ(file.UserID), chatattachment.IDEQ(file.ID), chatattachment.StatusEQ(chatattachment.StatusUploading), chatattachment.ByteSizeEQ(file.Size)).SetSha256(file.SHA256).SetStatus(status).Save(ctx)
		if err != nil {
			return err
		}
		if changed != 1 {
			return service.ErrChatNotFound
		}
		return nil
	})
}

func (r *chatRepository) AbandonAttachment(ctx context.Context, userID int64, id uuid.UUID) error {
	_, err := clientFromContext(ctx, r.client).ChatAttachment.Update().Where(chatattachment.UserIDEQ(userID), chatattachment.IDEQ(id), chatattachment.StatusEQ(chatattachment.StatusUploading)).SetStatus(chatattachment.StatusOrphan).Save(ctx)
	return err
}

func (r *chatRepository) GetAttachment(ctx context.Context, userID int64, id uuid.UUID) (*service.ChatAttachmentRecord, error) {
	row, err := clientFromContext(ctx, r.client).ChatAttachment.Query().Where(chatattachment.UserIDEQ(userID), chatattachment.IDEQ(id), chatattachment.StatusEQ(chatattachment.StatusReady), chatattachment.ImportIDIsNil()).Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrChatNotFound, nil)
	}
	return chatAttachmentRecord(row), nil
}
