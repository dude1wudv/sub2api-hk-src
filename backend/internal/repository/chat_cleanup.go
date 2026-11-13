package repository

import (
	"context"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/chatattachment"
	"github.com/Wei-Shaw/sub2api/ent/chatattachmentupload"
	"github.com/Wei-Shaw/sub2api/ent/chatimport"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
)

func (r *chatRepository) FindChatGarbage(ctx context.Context) (*service.ChatCleanupBatch, error) {
	batch := &service.ChatCleanupBatch{}
	now := time.Now()
	c := clientFromContext(ctx, r.client)
	imports, err := c.ChatImport.Query().Where(chatimport.Or(chatimport.And(chatimport.StatusEQ(chatimport.StatusPending), chatimport.ExpiresAtLTE(now)), chatimport.And(chatimport.StatusNEQ(chatimport.StatusPending), chatimport.UpdatedAtLTE(now.Add(-2*time.Minute))))).Order(dbent.Asc(chatimport.FieldUpdatedAt)).Limit(20).All(ctx)
	if err != nil {
		return nil, err
	}
	for _, candidate := range imports {
		err = r.withOwner(ctx, candidate.UserID, true, func(ctx context.Context, c *dbent.Client) error {
			row, err := c.ChatImport.Query().Where(chatimport.UserIDEQ(candidate.UserID), chatimport.IDEQ(candidate.ID)).Only(ctx)
			if dbent.IsNotFound(err) {
				return nil
			}
			if err != nil {
				return err
			}
			if row.Status == chatimport.StatusPending {
				if row.ExpiresAt.After(now) {
					return nil
				}
				if _, err = c.ChatAttachment.Update().Where(chatattachment.UserIDEQ(row.UserID), chatattachment.ImportIDEQ(row.ID)).SetStatus(chatattachment.StatusDeleting).Save(ctx); err != nil {
					return err
				}
				return c.ChatImport.UpdateOne(row).SetStatus(chatimport.StatusFailed).Exec(ctx)
			}
			if !row.UpdatedAt.After(now.Add(-2 * time.Minute)) {
				batch.Imports = append(batch.Imports, chatImportRecord(row))
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	files, err := c.ChatAttachment.Query().Where(chatattachment.Not(chatattachment.HasSessionRefs()), chatattachment.Or(chatattachment.StatusEQ(chatattachment.StatusDeleting), chatattachment.UpdatedAtLTE(now.Add(-24*time.Hour)))).Order(dbent.Asc(chatattachment.FieldUpdatedAt)).Limit(50).All(ctx)
	if err != nil {
		return nil, err
	}
	for _, candidate := range files {
		err = r.withOwner(ctx, candidate.UserID, true, func(ctx context.Context, c *dbent.Client) error {
			row, err := c.ChatAttachment.Query().Where(chatattachment.UserIDEQ(candidate.UserID), chatattachment.IDEQ(candidate.ID), chatattachment.Not(chatattachment.HasSessionRefs())).Only(ctx)
			if dbent.IsNotFound(err) {
				return nil
			}
			if err != nil {
				return err
			}
			if row.Status != chatattachment.StatusDeleting && row.UpdatedAt.After(now.Add(-24*time.Hour)) {
				return nil
			}
			active, err := c.ChatAttachmentUpload.Query().Where(chatattachmentupload.UserIDEQ(row.UserID), chatattachmentupload.AttachmentIDEQ(row.ID), chatattachmentupload.StatusEQ(chatattachmentupload.StatusUploading), chatattachmentupload.UpdatedAtGT(now.Add(-2*time.Minute))).Exist(ctx)
			if err != nil {
				return err
			}
			if active {
				return nil
			}
			if row.Status != chatattachment.StatusDeleting {
				row, err = c.ChatAttachment.UpdateOne(row).SetStatus(chatattachment.StatusDeleting).Save(ctx)
				if err != nil {
					return err
				}
			}
			batch.Files = append(batch.Files, chatAttachmentRecord(row))
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return batch, nil
}

func (r *chatRepository) ClaimRemoteCleanup(ctx context.Context, owner int64, id uuid.UUID) (*service.ChatRemoteFileRecord, error) {
	var claimed *service.ChatRemoteFileRecord
	err := r.withOwner(ctx, owner, true, func(ctx context.Context, c *dbent.Client) error {
		eligible, err := c.ChatAttachment.Query().Where(chatattachment.UserIDEQ(owner), chatattachment.IDEQ(id), chatattachment.StatusEQ(chatattachment.StatusDeleting)).Exist(ctx)
		if err != nil {
			return err
		}
		if !eligible {
			return nil
		}
		row, err := c.ChatAttachmentUpload.Query().Where(chatattachmentupload.UserIDEQ(owner), chatattachmentupload.AttachmentIDEQ(id), chatattachmentupload.RemoteFileIDNEQ(""), chatattachmentupload.CleanupAttemptsLT(3), chatattachmentupload.Or(chatattachmentupload.StatusEQ(chatattachmentupload.StatusReady), chatattachmentupload.StatusEQ(chatattachmentupload.StatusInvalid), chatattachmentupload.And(chatattachmentupload.StatusIn(chatattachmentupload.StatusDeleting, chatattachmentupload.StatusError), chatattachmentupload.UpdatedAtLTE(time.Now().Add(-2*time.Minute))))).Order(dbent.Asc(chatattachmentupload.FieldUpdatedAt)).First(ctx)
		if dbent.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		row, err = c.ChatAttachmentUpload.UpdateOne(row).SetStatus(chatattachmentupload.StatusDeleting).AddCleanupAttempts(1).SetUpdatedAt(time.Now().UTC().Truncate(time.Microsecond)).Save(ctx)
		if err != nil {
			return err
		}
		claimed = chatRemoteFileRecord(row)
		return nil
	})
	return claimed, err
}
func (r *chatRepository) FinishRemoteCleanup(ctx context.Context, claim *service.ChatRemoteFileRecord, success bool) error {
	if claim == nil {
		return service.ErrChatInvalid
	}
	update := clientFromContext(ctx, r.client).ChatAttachmentUpload.Update().Where(chatattachmentupload.UserIDEQ(claim.UserID), chatattachmentupload.IDEQ(claim.ID), chatattachmentupload.StatusEQ(chatattachmentupload.StatusDeleting), chatattachmentupload.CleanupAttemptsEQ(claim.CleanupAttempts), chatattachmentupload.UpdatedAtEQ(claim.UpdatedAt)).SetStatus(chatattachmentupload.StatusError)
	if success {
		update.SetRemoteFileID("")
	}
	_, err := update.Save(ctx)
	return err
}
func (r *chatRepository) FinishLocalCleanup(ctx context.Context, owner int64, id uuid.UUID) error {
	return r.withOwner(ctx, owner, true, func(ctx context.Context, c *dbent.Client) error {
		row, err := c.ChatAttachment.Query().Where(chatattachment.UserIDEQ(owner), chatattachment.IDEQ(id), chatattachment.StatusEQ(chatattachment.StatusDeleting), chatattachment.Not(chatattachment.HasSessionRefs())).Only(ctx)
		if dbent.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if err = c.ChatAttachment.UpdateOne(row).SetByteSize(0).Exec(ctx); err != nil {
			return err
		}
		pending, err := c.ChatAttachmentUpload.Query().Where(chatattachmentupload.UserIDEQ(owner), chatattachmentupload.AttachmentIDEQ(id), chatattachmentupload.RemoteFileIDNEQ(""), chatattachmentupload.Or(chatattachmentupload.CleanupAttemptsLT(3), chatattachmentupload.And(chatattachmentupload.StatusEQ(chatattachmentupload.StatusDeleting), chatattachmentupload.UpdatedAtGT(time.Now().Add(-2*time.Minute))))).Exist(ctx)
		if err != nil {
			return err
		}
		if pending {
			return nil
		}
		return c.ChatAttachment.DeleteOneID(id).Exec(ctx)
	})
}
func (r *chatRepository) FinishImportCleanup(ctx context.Context, owner int64, id uuid.UUID) error {
	return r.withOwner(ctx, owner, true, func(ctx context.Context, c *dbent.Client) error {
		exists, err := c.ChatImport.Query().Where(chatimport.UserIDEQ(owner), chatimport.IDEQ(id), chatimport.StatusNEQ(chatimport.StatusPending), chatimport.UpdatedAtLTE(time.Now().Add(-2*time.Minute))).Exist(ctx)
		if err != nil {
			return err
		}
		if !exists {
			return nil
		}
		if _, err = c.ChatAttachment.Update().Where(chatattachment.UserIDEQ(owner), chatattachment.ImportIDEQ(id)).SetStatus(chatattachment.StatusDeleting).ClearImportID().Save(ctx); err != nil {
			return err
		}
		return c.ChatImport.DeleteOneID(id).Exec(ctx)
	})
}
