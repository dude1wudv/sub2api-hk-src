package repository

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/chatattachment"
	"github.com/Wei-Shaw/sub2api/ent/chatgenerationrun"
	"github.com/Wei-Shaw/sub2api/ent/chatpreference"
	"github.com/Wei-Shaw/sub2api/ent/chatsession"
	"github.com/Wei-Shaw/sub2api/ent/chatsessionattachment"
	"github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
)

type chatRepository struct{ client *dbent.Client }

func NewChatRepository(client *dbent.Client) service.ChatRepository {
	return &chatRepository{client: client}
}

// All chat mutations take the owner lock before session/attachment locks.
// Never hold this lock during inference or upstream IO. Atomic import commits
// read verified local staging incrementally to keep memory bounded.
func (r *chatRepository) withUser(ctx context.Context, userID int64, fn func(context.Context, *dbent.Client) error) error {
	return r.withOwner(ctx, userID, false, fn)
}

// Cleanup may lock a soft-deleted owner; public operations never use that mode.
func (r *chatRepository) withOwner(ctx context.Context, userID int64, cleanup bool, fn func(context.Context, *dbent.Client) error) error {
	if userID <= 0 {
		return service.ErrChatNotFound
	}
	locked := func(ctx context.Context, client *dbent.Client) error {
		query := client.User.Query().Where(user.IDEQ(userID))
		if !cleanup {
			query.Where(user.DeletedAtIsNil())
		}
		owner, err := query.ForUpdate().Only(ctx)
		if err != nil {
			return translatePersistenceError(err, service.ErrChatNotFound, nil)
		}
		if !cleanup && owner.Status != service.StatusActive {
			return service.ErrChatNotFound
		}
		return fn(ctx, client)
	}
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return locked(ctx, tx.Client())
	}
	tx, err := r.client.Tx(ctx)
	if errors.Is(err, dbent.ErrTxStarted) {
		return locked(ctx, r.client)
	}
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := locked(dbent.NewTxContext(ctx, tx), tx.Client()); err != nil {
		return err
	}
	return tx.Commit()
}

func chatSessionRecord(row *dbent.ChatSession) *service.ChatSessionRecord {
	return &service.ChatSessionRecord{ID: row.ID, UserID: row.UserID, Title: row.Title, Snapshot: row.Snapshot, Revision: row.Revision, TitleManual: row.TitleManual, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

type chatSessionCursor struct {
	At time.Time `json:"at"`
	ID uuid.UUID `json:"id"`
}

func (r *chatRepository) ListSessions(ctx context.Context, userID int64, cursor string, limit int) (*service.ChatSessionPage, error) {
	if userID <= 0 {
		return nil, service.ErrChatNotFound
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	q := clientFromContext(ctx, r.client).ChatSession.Query().Where(chatsession.UserIDEQ(userID))
	if cursor != "" {
		if len(cursor) > 512 {
			return nil, service.ErrChatInvalid
		}
		raw, err := base64.RawURLEncoding.DecodeString(cursor)
		var c chatSessionCursor
		if err != nil || json.Unmarshal(raw, &c) != nil || c.At.IsZero() || c.ID == uuid.Nil {
			return nil, service.ErrChatInvalid
		}
		q.Where(chatsession.Or(chatsession.UpdatedAtLT(c.At), chatsession.And(chatsession.UpdatedAtEQ(c.At), chatsession.IDLT(c.ID))))
	}
	rows, err := q.Select(chatsession.FieldID, chatsession.FieldUserID, chatsession.FieldTitle, chatsession.FieldRevision, chatsession.FieldCreatedAt, chatsession.FieldUpdatedAt).Order(dbent.Desc(chatsession.FieldUpdatedAt), dbent.Desc(chatsession.FieldID)).Limit(limit + 1).All(ctx)
	if err != nil {
		return nil, err
	}
	page := &service.ChatSessionPage{Items: make([]service.ChatSessionSummary, 0, limit)}
	if len(rows) > limit {
		rows = rows[:limit]
		last := rows[len(rows)-1]
		raw, err := json.Marshal(chatSessionCursor{At: last.UpdatedAt, ID: last.ID})
		if err != nil {
			return nil, err
		}
		page.NextCursor = base64.RawURLEncoding.EncodeToString(raw)
	}
	for _, row := range rows {
		page.Items = append(page.Items, service.ChatSessionSummary{ID: row.ID, Title: row.Title, CreatedAt: row.CreatedAt.UnixMilli(), UpdatedAt: row.UpdatedAt.UnixMilli(), Revision: row.Revision})
	}
	return page, nil
}

func (r *chatRepository) GetSession(ctx context.Context, userID int64, id uuid.UUID) (*service.ChatSessionRecord, error) {
	row, err := clientFromContext(ctx, r.client).ChatSession.Query().Where(chatsession.UserIDEQ(userID), chatsession.IDEQ(id)).Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrChatNotFound, nil)
	}
	return chatSessionRecord(row), nil
}

func chatBumpHistory(ctx context.Context, client *dbent.Client, userID int64) error {
	row, err := client.ChatPreference.Query().Where(chatpreference.UserIDEQ(userID)).Only(ctx)
	if dbent.IsNotFound(err) {
		_, err = client.ChatPreference.Create().SetUserID(userID).SetPreferences(json.RawMessage(`{}`)).SetHistoryRevision(1).Save(ctx)
		return err
	}
	if err != nil {
		return err
	}
	return client.ChatPreference.UpdateOneID(row.ID).AddHistoryRevision(1).Exec(ctx)
}

func chatCheckAttachments(ctx context.Context, client *dbent.Client, userID int64, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	count, err := client.ChatAttachment.Query().Where(chatattachment.UserIDEQ(userID), chatattachment.IDIn(ids...), chatattachment.StatusEQ(chatattachment.StatusReady), chatattachment.ImportIDIsNil()).Count(ctx)
	if err != nil {
		return err
	}
	if count != len(ids) {
		return service.ErrChatNotFound
	}
	return nil
}

func chatReplaceAttachmentRefs(ctx context.Context, client *dbent.Client, userID int64, sessionID uuid.UUID, ids []uuid.UUID) error {
	old, err := client.ChatSessionAttachment.Query().Where(chatsessionattachment.UserIDEQ(userID), chatsessionattachment.SessionIDEQ(sessionID)).All(ctx)
	if err != nil {
		return err
	}
	if _, err = client.ChatSessionAttachment.Delete().Where(chatsessionattachment.UserIDEQ(userID), chatsessionattachment.SessionIDEQ(sessionID)).Exec(ctx); err != nil {
		return err
	}
	for _, id := range ids {
		if _, err = client.ChatSessionAttachment.Create().SetUserID(userID).SetSessionID(sessionID).SetAttachmentID(id).Save(ctx); err != nil {
			return err
		}
	}
	for _, ref := range old {
		used, err := client.ChatSessionAttachment.Query().Where(chatsessionattachment.UserIDEQ(userID), chatsessionattachment.AttachmentIDEQ(ref.AttachmentID)).Exist(ctx)
		if err != nil {
			return err
		}
		if !used {
			if err = chatDiscardSettledParseBodies(ctx, client, userID, []uuid.UUID{ref.AttachmentID}); err != nil {
				return err
			}
			if _, err = client.ChatAttachment.Update().Where(chatattachment.UserIDEQ(userID), chatattachment.IDEQ(ref.AttachmentID)).SetStatus(chatattachment.StatusDeleting).Save(ctx); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *chatRepository) SaveSession(ctx context.Context, userID int64, snapshot *service.ChatSnapshot, expected int64) (*service.ChatSessionRecord, error) {
	if snapshot == nil || expected < 0 {
		return nil, service.ErrChatInvalid
	}
	var result *service.ChatSessionRecord
	err := r.withUser(ctx, userID, func(ctx context.Context, client *dbent.Client) error {
		row, err := client.ChatSession.Query().Where(chatsession.UserIDEQ(userID), chatsession.IDEQ(snapshot.ID)).ForUpdate().Only(ctx)
		if err != nil && !dbent.IsNotFound(err) {
			return err
		}
		if row == nil && expected != 0 {
			return service.ErrChatRevisionConflict
		}
		if row != nil {
			busy, err := client.ChatGenerationRun.Query().Where(chatgenerationrun.UserIDEQ(userID), chatgenerationrun.SessionIDEQ(snapshot.ID), chatgenerationrun.StatusEQ(chatgenerationrun.StatusRunning)).Exist(ctx)
			if err != nil {
				return err
			}
			if busy {
				return service.ErrChatSessionBusy
			}
			if row.Revision != expected {
				return service.ErrChatRevisionConflict
			}
		}
		if err = chatCheckAttachments(ctx, client, userID, snapshot.AttachmentIDs); err != nil {
			return err
		}
		if row == nil {
			// Zero denotes an unsaved draft only. A persisted revision never
			// repeats after history replacement or recreation of the same UUID.
			historyRevision, historyErr := chatHistoryRevision(ctx, client, userID)
			if historyErr != nil {
				return historyErr
			}
			row, err = client.ChatSession.Create().SetID(snapshot.ID).SetUserID(userID).SetTitle(snapshot.Title).SetSnapshot(snapshot.Raw).SetRevision(historyRevision + 1).Save(ctx)
			if dbent.IsConstraintError(err) {
				return service.ErrChatNotFound
			}
		} else {
			changed, updateErr := client.ChatSession.Update().Where(chatsession.UserIDEQ(userID), chatsession.IDEQ(snapshot.ID), chatsession.RevisionEQ(expected)).SetTitle(snapshot.Title).SetSnapshot(snapshot.Raw).AddRevision(1).Save(ctx)
			if updateErr != nil {
				return updateErr
			}
			if changed != 1 {
				return service.ErrChatRevisionConflict
			}
			row, err = client.ChatSession.Query().Where(chatsession.UserIDEQ(userID), chatsession.IDEQ(snapshot.ID)).Only(ctx)
		}
		if err != nil {
			return err
		}
		if err = chatReplaceAttachmentRefs(ctx, client, userID, snapshot.ID, snapshot.AttachmentIDs); err != nil {
			return err
		}
		if err = chatBumpHistory(ctx, client, userID); err != nil {
			return err
		}
		result = chatSessionRecord(row)
		return nil
	})
	return result, err
}

func (r *chatRepository) RenameSession(ctx context.Context, userID int64, id uuid.UUID, title string, expected int64) (*service.ChatSessionRecord, error) {
	var result *service.ChatSessionRecord
	err := r.withUser(ctx, userID, func(ctx context.Context, client *dbent.Client) error {
		row, err := client.ChatSession.Query().Where(chatsession.UserIDEQ(userID), chatsession.IDEQ(id)).ForUpdate().Only(ctx)
		if err != nil {
			return translatePersistenceError(err, service.ErrChatNotFound, nil)
		}
		if row.Revision != expected {
			return service.ErrChatRevisionConflict
		}
		var snapshot map[string]json.RawMessage
		if err = json.Unmarshal(row.Snapshot, &snapshot); err != nil {
			return err
		}
		snapshot["title"], err = json.Marshal(title)
		if err != nil {
			return err
		}
		raw, err := json.Marshal(snapshot)
		if err != nil {
			return err
		}
		changed, err := client.ChatSession.Update().Where(chatsession.UserIDEQ(userID), chatsession.IDEQ(id), chatsession.RevisionEQ(expected)).SetTitle(title).SetTitleManual(true).SetSnapshot(raw).AddRevision(1).Save(ctx)
		if err != nil {
			return err
		}
		if changed != 1 {
			return service.ErrChatRevisionConflict
		}
		if err = chatBumpHistory(ctx, client, userID); err != nil {
			return err
		}
		row, err = client.ChatSession.Query().Where(chatsession.UserIDEQ(userID), chatsession.IDEQ(id)).Only(ctx)
		if err != nil {
			return err
		}
		result = chatSessionRecord(row)
		return nil
	})
	return result, err
}

func (r *chatRepository) GetPreferences(ctx context.Context, userID int64) (json.RawMessage, int64, error) {
	row, err := clientFromContext(ctx, r.client).ChatPreference.Query().Where(chatpreference.UserIDEQ(userID)).Only(ctx)
	if dbent.IsNotFound(err) {
		return json.RawMessage(`{}`), 0, nil
	}
	if err != nil {
		return nil, 0, err
	}
	return row.Preferences, row.HistoryRevision, nil
}

func (r *chatRepository) SavePreferences(ctx context.Context, userID int64, raw json.RawMessage) error {
	return r.withUser(ctx, userID, func(ctx context.Context, client *dbent.Client) error {
		row, err := client.ChatPreference.Query().Where(chatpreference.UserIDEQ(userID)).Only(ctx)
		if dbent.IsNotFound(err) {
			_, err = client.ChatPreference.Create().SetUserID(userID).SetPreferences(raw).Save(ctx)
			return err
		}
		if err != nil {
			return err
		}
		return client.ChatPreference.UpdateOneID(row.ID).SetPreferences(raw).Exec(ctx)
	})
}
