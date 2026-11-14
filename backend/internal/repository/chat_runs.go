package repository

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/chatgenerationrun"
	"github.com/Wei-Shaw/sub2api/ent/chatsession"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
)

func chatRunRecord(row *dbent.ChatGenerationRun) *service.ChatRunRecord {
	return &service.ChatRunRecord{ID: row.ID, UserID: row.UserID, SessionID: row.SessionID, AssistantMessageID: row.AssistantMessageID, PresetID: row.PresetID, GroupID: row.GroupID, Model: row.Model, RequestID: row.RequestID, Operation: string(row.Operation), Status: string(row.Status), InstanceID: row.OwnerInstanceID, LeaseExpiresAt: row.LeaseExpiresAt, Options: row.Options, GatewayRequestIDs: row.GatewayRequestIds}
}

func (r *chatRepository) ClaimGeneration(ctx context.Context, run *service.ChatRunRecord, expected int64, message json.RawMessage) (*service.ChatSessionRecord, error) {
	if run == nil || run.ID == uuid.Nil || run.InstanceID == uuid.Nil || run.RequestID == uuid.Nil || expected < 0 {
		return nil, service.ErrChatInvalid
	}
	var result *service.ChatSessionRecord
	err := r.withUser(ctx, run.UserID, func(ctx context.Context, c *dbent.Client) error {
		duplicate, err := c.ChatGenerationRun.Query().Where(chatgenerationrun.UserIDEQ(run.UserID), chatgenerationrun.RequestIDEQ(run.RequestID)).Exist(ctx)
		if err != nil {
			return err
		}
		if duplicate {
			return service.ErrChatRunExists
		}
		row, err := c.ChatSession.Query().Where(chatsession.UserIDEQ(run.UserID), chatsession.IDEQ(run.SessionID)).ForUpdate().Only(ctx)
		if err != nil {
			return translatePersistenceError(err, service.ErrChatNotFound, nil)
		}
		busy, err := c.ChatGenerationRun.Query().Where(chatgenerationrun.UserIDEQ(run.UserID), chatgenerationrun.SessionIDEQ(run.SessionID), chatgenerationrun.StatusEQ(chatgenerationrun.StatusRunning)).Exist(ctx)
		if err != nil {
			return err
		}
		if busy {
			return service.ErrChatSessionBusy
		}
		if row.Revision != expected {
			return service.ErrChatRevisionConflict
		}
		raw, title, err := service.MergeChatAssistant(chatSessionRecord(row), run.AssistantMessageID, message, "")
		if err != nil {
			return err
		}
		now := time.Now()
		_, err = c.ChatGenerationRun.Create().SetID(run.ID).SetUserID(run.UserID).SetSessionID(run.SessionID).SetAssistantMessageID(run.AssistantMessageID).SetPresetID(run.PresetID).SetGroupID(run.GroupID).SetModel(run.Model).SetOptions(run.Options).SetRequestID(run.RequestID).SetOperation(chatgenerationrun.Operation(run.Operation)).SetStatus(chatgenerationrun.StatusRunning).SetOwnerInstanceID(run.InstanceID).SetLeaseExpiresAt(now.Add(service.ChatRunLease)).SetStartedAt(now).SetHeartbeatAt(now).SetGatewayRequestIds([]string{}).Save(ctx)
		if err != nil {
			return err
		}
		row, err = c.ChatSession.UpdateOneID(row.ID).SetSnapshot(raw).SetTitle(title).AddRevision(1).Save(ctx)
		if err != nil {
			return err
		}
		if err = chatBumpHistory(ctx, c, run.UserID); err != nil {
			return err
		}
		result = chatSessionRecord(row)
		return nil
	})
	return result, err
}

func (r *chatRepository) SaveGeneration(ctx context.Context, run *service.ChatRunRecord, message json.RawMessage, title, status string) (*service.ChatSessionRecord, error) {
	switch status {
	case "running", "done", "stopped", "error":
	default:
		return nil, service.ErrChatInvalid
	}
	var result *service.ChatSessionRecord
	err := r.withUser(ctx, run.UserID, func(ctx context.Context, c *dbent.Client) error {
		active, err := c.ChatGenerationRun.Query().Where(chatgenerationrun.UserIDEQ(run.UserID), chatgenerationrun.IDEQ(run.ID), chatgenerationrun.SessionIDEQ(run.SessionID), chatgenerationrun.OwnerInstanceIDEQ(run.InstanceID), chatgenerationrun.StatusEQ(chatgenerationrun.StatusRunning), chatgenerationrun.LeaseExpiresAtGT(time.Now())).ForUpdate().Only(ctx)
		if dbent.IsNotFound(err) {
			return service.ErrChatLeaseLost
		}
		if err != nil {
			return err
		}
		row, err := c.ChatSession.Query().Where(chatsession.UserIDEQ(run.UserID), chatsession.IDEQ(run.SessionID)).ForUpdate().Only(ctx)
		if err != nil {
			return translatePersistenceError(err, service.ErrChatNotFound, nil)
		}
		raw, nextTitle, err := service.MergeChatAssistant(chatSessionRecord(row), active.AssistantMessageID, message, title)
		if err != nil {
			return err
		}
		row, err = c.ChatSession.UpdateOneID(row.ID).SetSnapshot(raw).SetTitle(nextTitle).AddRevision(1).Save(ctx)
		if err != nil {
			return err
		}
		if status != "running" {
			if err = c.ChatGenerationRun.UpdateOneID(active.ID).SetStatus(chatgenerationrun.Status(status)).SetEndedAt(time.Now()).Exec(ctx); err != nil {
				return err
			}
		}
		if err = chatBumpHistory(ctx, c, run.UserID); err != nil {
			return err
		}
		result = chatSessionRecord(row)
		return nil
	})
	return result, err
}

func (r *chatRepository) RenewGeneration(ctx context.Context, run *service.ChatRunRecord) error {
	now := time.Now()
	changed, err := clientFromContext(ctx, r.client).ChatGenerationRun.Update().Where(chatgenerationrun.UserIDEQ(run.UserID), chatgenerationrun.IDEQ(run.ID), chatgenerationrun.OwnerInstanceIDEQ(run.InstanceID), chatgenerationrun.StatusEQ(chatgenerationrun.StatusRunning), chatgenerationrun.LeaseExpiresAtGT(now)).SetHeartbeatAt(now).SetLeaseExpiresAt(now.Add(service.ChatRunLease)).Save(ctx)
	if err != nil {
		return err
	}
	if changed != 1 {
		return service.ErrChatLeaseLost
	}
	return nil
}

func (r *chatRepository) AddGenerationRequest(ctx context.Context, run *service.ChatRunRecord, requestID string) error {
	id := strings.TrimPrefix(requestID, "chat_auxiliary:")
	parsed, err := uuid.Parse(id)
	if err != nil || parsed == uuid.Nil || parsed.String() != id {
		return service.ErrChatInvalid
	}
	changed, err := clientFromContext(ctx, r.client).ChatGenerationRun.Update().Where(chatgenerationrun.UserIDEQ(run.UserID), chatgenerationrun.IDEQ(run.ID), chatgenerationrun.OwnerInstanceIDEQ(run.InstanceID), chatgenerationrun.StatusEQ(chatgenerationrun.StatusRunning), chatgenerationrun.LeaseExpiresAtGT(time.Now())).AppendGatewayRequestIds([]string{requestID}).Save(ctx)
	if err != nil {
		return err
	}
	if changed != 1 {
		return service.ErrChatLeaseLost
	}
	return nil
}

func (r *chatRepository) GetGeneration(ctx context.Context, userID int64, sessionID, runID uuid.UUID) (*service.ChatRunRecord, error) {
	row, err := clientFromContext(ctx, r.client).ChatGenerationRun.Query().Where(chatgenerationrun.UserIDEQ(userID), chatgenerationrun.SessionIDEQ(sessionID), chatgenerationrun.IDEQ(runID)).Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrChatNotFound, nil)
	}
	return chatRunRecord(row), nil
}

func (r *chatRepository) StopGeneration(ctx context.Context, userID int64, sessionID, runID uuid.UUID) error {
	return r.stopGeneration(ctx, userID, sessionID, runID, false)
}

func (r *chatRepository) stopGeneration(ctx context.Context, userID int64, sessionID, runID uuid.UUID, expiredOnly bool) error {
	return r.withUser(ctx, userID, func(ctx context.Context, c *dbent.Client) error {
		run, err := c.ChatGenerationRun.Query().Where(chatgenerationrun.UserIDEQ(userID), chatgenerationrun.SessionIDEQ(sessionID), chatgenerationrun.IDEQ(runID)).ForUpdate().Only(ctx)
		if err != nil {
			return translatePersistenceError(err, service.ErrChatNotFound, nil)
		}
		if run.Status != chatgenerationrun.StatusRunning || (expiredOnly && run.LeaseExpiresAt.After(time.Now())) {
			return nil
		}
		row, err := c.ChatSession.Query().Where(chatsession.UserIDEQ(userID), chatsession.IDEQ(sessionID)).ForUpdate().Only(ctx)
		if err != nil {
			return err
		}
		message, err := service.ChatAssistantJSON(row.Snapshot, run.AssistantMessageID)
		if err != nil {
			return err
		}
		message["status"] = "stopped"
		rawMessage, err := json.Marshal(message)
		if err != nil {
			return err
		}
		raw, _, err := service.MergeChatAssistant(chatSessionRecord(row), run.AssistantMessageID, rawMessage, "")
		if err != nil {
			return err
		}
		if err = c.ChatSession.UpdateOneID(row.ID).SetSnapshot(raw).AddRevision(1).Exec(ctx); err != nil {
			return err
		}
		if err = c.ChatGenerationRun.UpdateOneID(run.ID).SetStatus(chatgenerationrun.StatusStopped).SetEndedAt(time.Now()).Exec(ctx); err != nil {
			return err
		}
		return chatBumpHistory(ctx, c, userID)
	})
}

func (r *chatRepository) RecoverExpiredGenerations(ctx context.Context) error {
	rows, err := clientFromContext(ctx, r.client).ChatGenerationRun.Query().Where(chatgenerationrun.StatusEQ(chatgenerationrun.StatusRunning), chatgenerationrun.LeaseExpiresAtLTE(time.Now())).Order(dbent.Asc(chatgenerationrun.FieldLeaseExpiresAt)).Limit(100).All(ctx)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if err = r.stopGeneration(ctx, row.UserID, row.SessionID, row.ID, true); err != nil && !errors.Is(err, service.ErrChatNotFound) {
			return err
		}
	}
	return nil
}

func (r *chatRepository) DeleteSessions(ctx context.Context, userID int64, id *uuid.UUID) error {
	return r.withUser(ctx, userID, func(ctx context.Context, c *dbent.Client) error {
		query := c.ChatSession.Query().Where(chatsession.UserIDEQ(userID))
		if id != nil {
			query.Where(chatsession.IDEQ(*id))
		}
		rows, err := query.ForUpdate().All(ctx)
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			if id != nil {
				foreign, err := c.ChatSession.Query().Where(chatsession.IDEQ(*id), chatsession.UserIDNEQ(userID)).Exist(ctx)
				if err != nil {
					return err
				}
				if foreign {
					return service.ErrChatNotFound
				}
			}
			return nil
		}
		for _, row := range rows {
			if err = chatReplaceAttachmentRefs(ctx, c, userID, row.ID, nil); err != nil {
				return err
			}
			if err = c.ChatSession.DeleteOneID(row.ID).Exec(ctx); err != nil {
				return err
			}
		}
		return chatBumpHistory(ctx, c, userID)
	})
}

func (r *chatRepository) LastGeneration(ctx context.Context, userID int64, sessionID uuid.UUID, assistantID string) (*service.ChatRunRecord, error) {
	row, err := clientFromContext(ctx, r.client).ChatGenerationRun.Query().Where(chatgenerationrun.UserIDEQ(userID), chatgenerationrun.SessionIDEQ(sessionID), chatgenerationrun.AssistantMessageIDEQ(assistantID)).Order(dbent.Desc(chatgenerationrun.FieldStartedAt), dbent.Desc(chatgenerationrun.FieldID)).First(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrChatNotFound, nil)
	}
	return chatRunRecord(row), nil
}
