package repository

import (
	"context"
	"encoding/json"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/chatattachment"
	"github.com/Wei-Shaw/sub2api/ent/chatattachmentparse"
	"github.com/Wei-Shaw/sub2api/ent/chatattachmentupload"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
)

func chatParseRecord(row *dbent.ChatAttachmentParse) *service.ChatParseRecord {
	return &service.ChatParseRecord{ID: row.ID, UserID: row.UserID, AttachmentID: row.AttachmentID, AccountID: row.AccountID, OperationID: row.OperationID, Status: string(row.Status), Result: row.Result, PageCount: row.PageCount, RequestID: row.RequestID, Receipt: row.BillingReceipt, BillingApplied: row.BillingApplied}
}

func (r *chatRepository) ClaimAttachmentParse(ctx context.Context, user int64, attachment uuid.UUID, account, limit int64, retry bool) (*service.ChatParseRecord, bool, error) {
	var record *service.ChatParseRecord
	acquired := false
	err := r.withUser(ctx, user, func(ctx context.Context, c *dbent.Client) error {
		exists, err := c.ChatAttachment.Query().Where(chatattachment.IDEQ(attachment), chatattachment.UserIDEQ(user), chatattachment.StatusEQ(chatattachment.StatusReady), chatattachment.ImportIDIsNil()).Exist(ctx)
		if err != nil {
			return err
		}
		if !exists {
			return service.ErrChatNotFound
		}
		row, err := c.ChatAttachmentParse.Query().Where(chatattachmentparse.UserIDEQ(user), chatattachmentparse.AttachmentIDEQ(attachment), chatattachmentparse.AccountIDEQ(account)).Only(ctx)
		if err != nil && !dbent.IsNotFound(err) {
			return err
		}
		fresh := dbent.IsNotFound(err)
		if !fresh {
			if row.Status == chatattachmentparse.StatusProcessing && row.UpdatedAt.Before(time.Now().Add(-time.Minute)) {
				row, err = c.ChatAttachmentParse.UpdateOne(row).SetStatus(chatattachmentparse.StatusUnknown).SetReservedBytes(0).Save(ctx)
				if err != nil {
					return err
				}
			}
			if row.Status != chatattachmentparse.StatusFailed || !retry {
				record = chatParseRecord(row)
				return nil
			}
			if len(row.BillingReceipt) != 0 {
				return service.ErrChatParseBusy
			}
		}
		used, err := chatReservedStorage(ctx, c, user, nil)
		if err != nil {
			return err
		}
		if limit < service.ChatMaxParseBytes || used > limit-service.ChatMaxParseBytes {
			return service.ErrChatStorageFull
		}
		operation := uuid.New()
		request := "chat_auxiliary:" + operation.String()
		if fresh {
			row, err = c.ChatAttachmentParse.Create().SetUserID(user).SetAttachmentID(attachment).SetAccountID(account).SetOperationID(operation).SetRequestID(request).SetReservedBytes(service.ChatMaxParseBytes).Save(ctx)
		} else {
			row, err = c.ChatAttachmentParse.UpdateOne(row).SetOperationID(operation).SetRequestID(request).SetStatus(chatattachmentparse.StatusProcessing).SetReservedBytes(service.ChatMaxParseBytes).SetResultBytes(0).ClearResult().ClearPageCount().Save(ctx)
		}
		if err != nil {
			return err
		}
		record = chatParseRecord(row)
		acquired = true
		return nil
	})
	return record, acquired, err
}

func (r *chatRepository) GetAttachmentParse(ctx context.Context, user int64, attachment uuid.UUID, account int64) (*service.ChatParseRecord, error) {
	row, err := clientFromContext(ctx, r.client).ChatAttachmentParse.Query().Where(chatattachmentparse.UserIDEQ(user), chatattachmentparse.AttachmentIDEQ(attachment), chatattachmentparse.AccountIDEQ(account)).Only(ctx)
	if dbent.IsNotFound(err) {
		return nil, service.ErrChatParseRequired
	}
	if err != nil {
		return nil, err
	}
	return chatParseRecord(row), nil
}

func (r *chatRepository) RenewAttachmentParse(ctx context.Context, claim *service.ChatParseRecord) error {
	if claim == nil {
		return service.ErrChatLeaseLost
	}
	changed, err := clientFromContext(ctx, r.client).ChatAttachmentParse.Update().Where(chatattachmentparse.IDEQ(claim.ID), chatattachmentparse.UserIDEQ(claim.UserID), chatattachmentparse.OperationIDEQ(claim.OperationID), chatattachmentparse.StatusEQ(chatattachmentparse.StatusProcessing), chatattachmentparse.UpdatedAtGT(time.Now().Add(-time.Minute))).SetUpdatedAt(time.Now()).Save(ctx)
	if err != nil {
		return err
	}
	if changed != 1 {
		return service.ErrChatLeaseLost
	}
	return nil
}

func (r *chatRepository) FinishAttachmentParse(ctx context.Context, claim *service.ChatParseRecord, result json.RawMessage, pages int, receipt json.RawMessage) error {
	if claim == nil || pages < 0 || pages > 1000 || !json.Valid(result) || !json.Valid(receipt) {
		return service.ErrChatInvalid
	}
	return r.withUser(ctx, claim.UserID, func(ctx context.Context, c *dbent.Client) error {
		rows, err := c.QueryContext(ctx, "SELECT octet_length($1::jsonb::text)", string(result))
		if err != nil {
			return err
		}
		var size int64
		if !rows.Next() {
			_ = rows.Close()
			return service.ErrChatInvalid
		}
		err = rows.Scan(&size)
		_ = rows.Close()
		if err != nil {
			return err
		}
		update := c.ChatAttachmentParse.Update().Where(chatattachmentparse.IDEQ(claim.ID), chatattachmentparse.UserIDEQ(claim.UserID), chatattachmentparse.OperationIDEQ(claim.OperationID), chatattachmentparse.StatusEQ(chatattachmentparse.StatusProcessing), chatattachmentparse.BillingReceiptIsNil(), chatattachmentparse.UpdatedAtGT(time.Now().Add(-time.Minute))).SetPageCount(pages).SetBillingReceipt(receipt).SetReservedBytes(0)
		if size > service.ChatMaxParseBytes {
			// Confirmed usage must survive an unusable body and remain recoverable.
			// Unknown prevents a billed operation from being automatically replayed.
			update.ClearResult().SetResultBytes(0).SetStatus(chatattachmentparse.StatusUnknown)
		} else {
			update.SetResult(result).SetResultBytes(size).SetStatus(chatattachmentparse.StatusReady)
		}
		changed, err := update.Save(ctx)
		if err != nil {
			return err
		}
		if changed != 1 {
			return service.ErrChatLeaseLost
		}
		return nil
	})
}

func (r *chatRepository) FailAttachmentParse(ctx context.Context, claim *service.ChatParseRecord, known bool) error {
	if claim == nil {
		return service.ErrChatInvalid
	}
	status := chatattachmentparse.StatusUnknown
	if known {
		status = chatattachmentparse.StatusFailed
	}
	_, err := clientFromContext(ctx, r.client).ChatAttachmentParse.Update().Where(chatattachmentparse.IDEQ(claim.ID), chatattachmentparse.UserIDEQ(claim.UserID), chatattachmentparse.OperationIDEQ(claim.OperationID), chatattachmentparse.StatusEQ(chatattachmentparse.StatusProcessing), chatattachmentparse.BillingReceiptIsNil()).SetStatus(status).SetReservedBytes(0).SetResultBytes(0).ClearResult().Save(ctx)
	return err
}

func (r *chatRepository) MarkAttachmentParseBilled(ctx context.Context, claim *service.ChatParseRecord) error {
	if claim == nil {
		return service.ErrChatInvalid
	}
	c := clientFromContext(ctx, r.client)
	changed, err := c.ChatAttachmentParse.Update().Where(chatattachmentparse.IDEQ(claim.ID), chatattachmentparse.UserIDEQ(claim.UserID), chatattachmentparse.OperationIDEQ(claim.OperationID), chatattachmentparse.BillingReceiptNotNil(), chatattachmentparse.BillingAppliedEQ(false)).SetBillingApplied(true).Save(ctx)
	if err != nil {
		return err
	}
	if changed == 1 {
		return nil
	}
	done, err := c.ChatAttachmentParse.Query().Where(chatattachmentparse.IDEQ(claim.ID), chatattachmentparse.UserIDEQ(claim.UserID), chatattachmentparse.OperationIDEQ(claim.OperationID), chatattachmentparse.BillingAppliedEQ(true)).Exist(ctx)
	if err != nil {
		return err
	}
	if !done {
		return service.ErrChatLeaseLost
	}
	return nil
}

func (r *chatRepository) PendingChatParseReceipts(ctx context.Context) ([]*service.ChatParseRecord, error) {
	c := clientFromContext(ctx, r.client)
	_, err := c.ChatAttachmentParse.Update().Where(chatattachmentparse.StatusEQ(chatattachmentparse.StatusProcessing), chatattachmentparse.BillingReceiptIsNil(), chatattachmentparse.UpdatedAtLT(time.Now().Add(-5*time.Minute))).SetStatus(chatattachmentparse.StatusUnknown).SetReservedBytes(0).Save(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := c.ChatAttachmentParse.Query().Where(chatattachmentparse.BillingReceiptNotNil(), chatattachmentparse.BillingAppliedEQ(false)).Order(dbent.Asc(chatattachmentparse.FieldUpdatedAt)).Limit(20).All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*service.ChatParseRecord, 0, len(rows))
	for _, row := range rows {
		result = append(result, chatParseRecord(row))
	}
	return result, nil
}

// Call under the existing user-row lock before removing the last reference or
// deleting an owner. No cleanup path can erase an active parse or unpaid receipt.
func chatProtectAttachmentWork(ctx context.Context, c *dbent.Client, user int64, ids []uuid.UUID) error {
	parses := c.ChatAttachmentParse.Query().Where(chatattachmentparse.UserIDEQ(user), chatattachmentparse.Or(chatattachmentparse.StatusEQ(chatattachmentparse.StatusProcessing), chatattachmentparse.And(chatattachmentparse.BillingReceiptNotNil(), chatattachmentparse.BillingAppliedEQ(false))))
	uploads := c.ChatAttachmentUpload.Query().Where(chatattachmentupload.UserIDEQ(user), chatattachmentupload.StatusEQ(chatattachmentupload.StatusUploading), chatattachmentupload.LeaseExpiresAtGT(time.Now()))
	if ids != nil {
		if len(ids) == 0 {
			return nil
		}
		parses.Where(chatattachmentparse.AttachmentIDIn(ids...))
		uploads.Where(chatattachmentupload.AttachmentIDIn(ids...))
	}
	blocked, err := parses.Exist(ctx)
	if err != nil {
		return err
	}
	if blocked {
		return service.ErrChatParseBusy
	}
	blocked, err = uploads.Exist(ctx)
	if err != nil {
		return err
	}
	if blocked {
		return service.ErrChatParseBusy
	}
	return nil
}

func chatDiscardSettledParseBodies(ctx context.Context, c *dbent.Client, user int64, ids []uuid.UUID) error {
	if err := chatProtectAttachmentWork(ctx, c, user, ids); err != nil {
		return err
	}
	update := c.ChatAttachmentParse.Update().Where(chatattachmentparse.UserIDEQ(user))
	if ids != nil {
		if len(ids) == 0 {
			return nil
		}
		update.Where(chatattachmentparse.AttachmentIDIn(ids...))
	}
	_, err := update.ClearResult().SetResultBytes(0).SetReservedBytes(0).Save(ctx)
	return err
}
