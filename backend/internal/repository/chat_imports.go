package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"sort"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/chatattachment"
	"github.com/Wei-Shaw/sub2api/ent/chatgenerationrun"
	"github.com/Wei-Shaw/sub2api/ent/chatimport"
	"github.com/Wei-Shaw/sub2api/ent/chatpreference"
	"github.com/Wei-Shaw/sub2api/ent/chatsession"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
)

func chatImportRecord(row *dbent.ChatImport) *service.ChatImportRecord {
	return &service.ChatImportRecord{ID: row.ID, UserID: row.UserID, HistoryRevision: row.HistoryRevision, Status: string(row.Status), ExpiresAt: row.ExpiresAt, Manifest: bytes.Clone(row.Manifest)}
}
func chatHistoryRevision(ctx context.Context, c *dbent.Client, userID int64) (int64, error) {
	row, err := c.ChatPreference.Query().Where(chatpreference.UserIDEQ(userID)).Only(ctx)
	if dbent.IsNotFound(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return row.HistoryRevision, nil
}

// Include reservations and not-yet-reclaimed staging, not just visible files.
func chatReservedStorage(ctx context.Context, c *dbent.Client, userID int64, importID *uuid.UUID) (int64, error) {
	query := `SELECT COALESCE((SELECT SUM(byte_size) FROM chat_attachments WHERE user_id=$1),0)+COALESCE((SELECT SUM((manifest->>'bytes')::bigint) FROM chat_imports WHERE user_id=$1),0)`
	args := []any{userID}
	if importID != nil {
		query = `SELECT COALESCE((SELECT SUM(byte_size) FROM chat_attachments WHERE user_id=$1 AND import_id=$2),0)+COALESCE((SELECT (manifest->>'bytes')::bigint FROM chat_imports WHERE user_id=$1 AND id=$2),0)`
		args = append(args, *importID)
	}
	rows, err := c.QueryContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err = rows.Err(); err != nil {
			return 0, err
		}
		return 0, service.ErrChatUnavailable
	}
	var used int64
	if err = rows.Scan(&used); err != nil {
		return 0, err
	}
	return used, rows.Err()
}

func (r *chatRepository) BeginImport(ctx context.Context, userID, expected int64) (*service.ChatImportRecord, error) {
	var record *service.ChatImportRecord
	err := r.withUser(ctx, userID, func(ctx context.Context, c *dbent.Client) error {
		revision, err := chatHistoryRevision(ctx, c, userID)
		if err != nil {
			return err
		}
		if expected < 0 || revision != expected {
			return service.ErrChatRevisionConflict
		}
		pending, err := c.ChatImport.Query().Where(chatimport.UserIDEQ(userID), chatimport.StatusEQ(chatimport.StatusPending), chatimport.ExpiresAtGT(time.Now())).Count(ctx)
		if err != nil {
			return err
		}
		if pending >= 8 {
			return service.ErrChatStorageFull
		}
		row, err := c.ChatImport.Create().SetUserID(userID).SetHistoryRevision(expected).SetExpiresAt(time.Now().Add(24 * time.Hour)).SetManifest(json.RawMessage(`{"sessions":{},"bytes":0}`)).Save(ctx)
		if err != nil {
			return err
		}
		record = chatImportRecord(row)
		return nil
	})
	return record, err
}
func (r *chatRepository) GetImport(ctx context.Context, userID int64, id uuid.UUID) (*service.ChatImportRecord, error) {
	row, err := clientFromContext(ctx, r.client).ChatImport.Query().Where(chatimport.UserIDEQ(userID), chatimport.IDEQ(id), chatimport.StatusEQ(chatimport.StatusPending), chatimport.ExpiresAtGT(time.Now())).Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrChatNotFound, nil)
	}
	return chatImportRecord(row), nil
}
func (r *chatRepository) ReserveImportSession(ctx context.Context, userID int64, id uuid.UUID, item service.ChatImportSession, limit int64) error {
	return r.withUser(ctx, userID, func(ctx context.Context, c *dbent.Client) error {
		record, err := r.GetImport(ctx, userID, id)
		if err != nil {
			return err
		}
		manifest, err := service.ParseChatImportManifest(record.Manifest)
		if err != nil {
			return err
		}
		if _, exists := manifest.Sessions[item.SessionID.String()]; exists {
			return service.ErrChatInvalid
		}
		if item.Size < 0 || item.Size > service.ChatMaxSessionBytes || item.Ready {
			return service.ErrChatInvalid
		}
		used, err := chatReservedStorage(ctx, c, userID, nil)
		if err != nil {
			return err
		}
		if used > limit || item.Size > limit-used {
			return service.ErrChatStorageFull
		}
		imported, err := chatReservedStorage(ctx, c, userID, &id)
		if err != nil {
			return err
		}
		if imported > service.ChatMaxArchiveBytes || item.Size > service.ChatMaxArchiveBytes-imported {
			return service.ErrChatTooLarge
		}
		manifest.Sessions[item.SessionID.String()] = item
		manifest.Bytes += item.Size
		raw, err := json.Marshal(manifest)
		if err != nil {
			return err
		}
		if _, err = service.ParseChatImportManifest(raw); err != nil {
			return err
		}
		return c.ChatImport.UpdateOneID(id).SetManifest(raw).Exec(ctx)
	})
}
func (r *chatRepository) FinishImportSession(ctx context.Context, userID int64, id, sessionID uuid.UUID) error {
	return r.withUser(ctx, userID, func(ctx context.Context, c *dbent.Client) error {
		record, err := r.GetImport(ctx, userID, id)
		if err != nil {
			return err
		}
		manifest, err := service.ParseChatImportManifest(record.Manifest)
		if err != nil {
			return err
		}
		item, exists := manifest.Sessions[sessionID.String()]
		if !exists {
			return service.ErrChatNotFound
		}
		item.Ready = true
		manifest.Sessions[sessionID.String()] = item
		raw, err := json.Marshal(manifest)
		if err != nil {
			return err
		}
		return c.ChatImport.UpdateOneID(id).SetManifest(raw).Exec(ctx)
	})
}

func (r *chatRepository) CommitImport(ctx context.Context, expected *service.ChatImportRecord, read func(service.ChatImportSession) (*service.ChatImportedSnapshot, error)) (int64, error) {
	if expected == nil || read == nil {
		return 0, service.ErrChatInvalid
	}
	var revision int64
	err := r.withUser(ctx, expected.UserID, func(ctx context.Context, c *dbent.Client) error {
		owner, id := expected.UserID, expected.ID
		current, err := r.GetImport(ctx, owner, id)
		if err != nil {
			return err
		}
		revision, err = chatHistoryRevision(ctx, c, owner)
		if err != nil {
			return err
		}
		if revision != expected.HistoryRevision || current.HistoryRevision != expected.HistoryRevision || !bytes.Equal(current.Manifest, expected.Manifest) {
			return service.ErrChatRevisionConflict
		}
		manifest, err := service.ParseChatImportManifest(current.Manifest)
		if err != nil {
			return err
		}
		busy, err := c.ChatGenerationRun.Query().Where(chatgenerationrun.UserIDEQ(owner), chatgenerationrun.StatusEQ(chatgenerationrun.StatusRunning)).Exist(ctx)
		if err != nil {
			return err
		}
		if busy {
			return service.ErrChatSessionBusy
		}
		uploading, err := c.ChatAttachment.Query().Where(chatattachment.UserIDEQ(owner), chatattachment.ImportIDEQ(id), chatattachment.StatusEQ(chatattachment.StatusUploading)).Exist(ctx)
		if err != nil {
			return err
		}
		if uploading {
			return service.ErrChatInvalid
		}
		keys := make([]string, 0, len(manifest.Sessions))
		for key, item := range manifest.Sessions {
			if !item.Ready {
				return service.ErrChatInvalid
			}
			foreign, err := c.ChatSession.Query().Where(chatsession.IDEQ(item.SessionID), chatsession.UserIDNEQ(owner)).Exist(ctx)
			if err != nil {
				return err
			}
			if foreign {
				return service.ErrChatNotFound
			}
			keys = append(keys, key)
		}
		sort.Strings(keys)
		// Mark only attachments previously referenced by this history. Unsubmitted
		// drafts from another tab keep their normal 24-hour orphan grace period.
		if _, err = c.ChatAttachment.Update().Where(chatattachment.UserIDEQ(owner), chatattachment.HasSessionRefs()).SetStatus(chatattachment.StatusDeleting).Save(ctx); err != nil {
			return err
		}
		if _, err = c.ChatSession.Delete().Where(chatsession.UserIDEQ(owner)).Exec(ctx); err != nil {
			return err
		}
		if _, err = c.ChatAttachment.Update().Where(chatattachment.UserIDEQ(owner), chatattachment.ImportIDEQ(id), chatattachment.StatusEQ(chatattachment.StatusStaged)).SetStatus(chatattachment.StatusReady).Save(ctx); err != nil {
			return err
		}
		for _, key := range keys {
			imported, err := read(manifest.Sessions[key])
			if err != nil {
				return err
			}
			snapshot := imported.Snapshot
			if snapshot == nil || snapshot.ID != manifest.Sessions[key].SessionID {
				return service.ErrChatInvalid
			}
			if len(snapshot.AttachmentIDs) > 0 {
				count, err := c.ChatAttachment.Query().Where(chatattachment.UserIDEQ(owner), chatattachment.ImportIDEQ(id), chatattachment.StatusEQ(chatattachment.StatusReady), chatattachment.IDIn(snapshot.AttachmentIDs...)).Count(ctx)
				if err != nil {
					return err
				}
				if count != len(snapshot.AttachmentIDs) {
					return service.ErrChatNotFound
				}
			}
			if _, err = c.ChatSession.Create().SetID(snapshot.ID).SetUserID(owner).SetTitle(snapshot.Title).SetTitleManual(true).SetSnapshot(snapshot.Raw).SetRevision(revision + 1).SetCreatedAt(imported.CreatedAt).SetUpdatedAt(imported.UpdatedAt).Save(ctx); err != nil {
				return err
			}
			for _, fileID := range snapshot.AttachmentIDs {
				if _, err = c.ChatSessionAttachment.Create().SetUserID(owner).SetSessionID(snapshot.ID).SetAttachmentID(fileID).Save(ctx); err != nil {
					return err
				}
			}
		}
		if _, err = c.ChatAttachment.Update().Where(chatattachment.UserIDEQ(owner), chatattachment.ImportIDEQ(id), chatattachment.Not(chatattachment.HasSessionRefs())).SetStatus(chatattachment.StatusDeleting).Save(ctx); err != nil {
			return err
		}
		if _, err = c.ChatAttachment.Update().Where(chatattachment.UserIDEQ(owner), chatattachment.ImportIDEQ(id)).ClearImportID().Save(ctx); err != nil {
			return err
		}
		if err = c.ChatImport.UpdateOneID(id).SetStatus(chatimport.StatusCommitted).SetExpiresAt(time.Now()).Exec(ctx); err != nil {
			return err
		}
		if err = chatBumpHistory(ctx, c, owner); err != nil {
			return err
		}
		revision++
		return nil
	})
	return revision, err
}

func (r *chatRepository) AbortImport(ctx context.Context, owner int64, id uuid.UUID) error {
	return r.withUser(ctx, owner, func(ctx context.Context, c *dbent.Client) error {
		row, err := c.ChatImport.Query().Where(chatimport.UserIDEQ(owner), chatimport.IDEQ(id)).Only(ctx)
		if dbent.IsNotFound(err) {
			foreign, err := c.ChatImport.Query().Where(chatimport.IDEQ(id), chatimport.UserIDNEQ(owner)).Exist(ctx)
			if err != nil {
				return err
			}
			if foreign {
				return service.ErrChatNotFound
			}
			return nil
		}
		if err != nil {
			return err
		}
		if row.Status != chatimport.StatusPending {
			return nil
		}
		if _, err = c.ChatAttachment.Update().Where(chatattachment.UserIDEQ(owner), chatattachment.ImportIDEQ(id)).SetStatus(chatattachment.StatusDeleting).Save(ctx); err != nil {
			return err
		}
		return c.ChatImport.UpdateOneID(id).SetStatus(chatimport.StatusFailed).SetExpiresAt(time.Now()).Exec(ctx)
	})
}
