package service

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/google/uuid"
)

type ChatCleanupBatch struct {
	Files   []*ChatAttachmentRecord
	Imports []*ChatImportRecord
}
type ChatCleanupRepository interface {
	FindChatGarbage(context.Context) (*ChatCleanupBatch, error)
	ClaimRemoteCleanup(context.Context, int64, uuid.UUID) (*ChatRemoteFileRecord, error)
	FinishRemoteCleanup(context.Context, *ChatRemoteFileRecord, bool) error
	FinishLocalCleanup(context.Context, int64, uuid.UUID) error
	FinishImportCleanup(context.Context, int64, uuid.UUID) error
}

// SweepChat touches only registered UUID paths beneath ChatStorage. Remote
// failures leave invisible, zero-byte metadata for bounded cleanup retries.
func (s *ChatService) SweepChat(ctx context.Context) error {
	batch, err := s.repo.FindChatGarbage(ctx)
	if err != nil {
		return err
	}
	for _, record := range batch.Files {
		if err = ctx.Err(); err != nil {
			return err
		}
		if err = s.storage.RemoveFile(record.UserID, record.ID); err != nil {
			return err
		}
		if err = s.repo.FinishLocalCleanup(ctx, record.UserID, record.ID); err != nil {
			return err
		}
		for {
			remote, err := s.repo.ClaimRemoteCleanup(ctx, record.UserID, record.ID)
			if err != nil {
				return err
			}
			if remote == nil {
				break
			}
			attempt, cancel := context.WithTimeout(ctx, 5*time.Second)
			var deleteErr error
			if s.models == nil || s.models.openai == nil {
				deleteErr = ErrChatUnavailable
			} else {
				deleteErr = s.models.openai.deleteChatNativeFile(attempt, remote.AccountID, remote.RemoteID)
			}
			cancel()
			persist, done := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			err = s.repo.FinishRemoteCleanup(persist, remote, deleteErr == nil)
			done()
			if err != nil {
				return err
			}
			if deleteErr != nil && remote.CleanupAttempts >= 3 {
				logger.LegacyPrintf("service.chat", "remote file cleanup exhausted: user=%d account=%d attachment=%s", remote.UserID, remote.AccountID, remote.AttachmentID)
			}
			if err = ctx.Err(); err != nil {
				return err
			}
		}
		if err = s.repo.FinishLocalCleanup(ctx, record.UserID, record.ID); err != nil {
			return err
		}
	}
	for _, record := range batch.Imports {
		if err = ctx.Err(); err != nil {
			return err
		}
		if err = s.storage.RemoveImport(record.UserID, record.ID); err != nil {
			return err
		}
		if err = s.repo.FinishImportCleanup(ctx, record.UserID, record.ID); err != nil {
			return err
		}
	}
	return nil
}
