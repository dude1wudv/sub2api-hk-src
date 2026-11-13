package service

import (
	"context"
	"errors"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/google/uuid"
)

type chatActiveRun struct {
	userID    int64
	sessionID uuid.UUID
	cancel    context.CancelFunc
	done      chan struct{}
}

func (s *ChatService) startLifecycle() {
	if s.repo == nil {
		return
	}
	s.runMu.Lock()
	if s.closed || s.lifecycleCancel != nil {
		s.runMu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.lifecycleCancel = cancel
	s.lifecycleDone = make(chan struct{})
	done := s.lifecycleDone
	s.runMu.Unlock()
	go func() {
		defer close(done)
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			sweep, cancel := context.WithTimeout(ctx, 5*time.Second)
			err := s.repo.RecoverExpiredGenerations(sweep)
			cancel()
			if err != nil && ctx.Err() == nil {
				logger.LegacyPrintf("service.chat", "chat lease recovery failed: %v", err)
			}
			cleanup, finishCleanup := context.WithTimeout(ctx, 20*time.Second)
			err = s.SweepChat(cleanup)
			finishCleanup()
			if err != nil && ctx.Err() == nil {
				logger.LegacyPrintf("service.chat", "chat file cleanup failed: %v", err)
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (s *ChatService) registerRun(run *ChatRunRecord, cancel context.CancelFunc) (*chatActiveRun, error) {
	s.runMu.Lock()
	defer s.runMu.Unlock()
	if s.closed {
		return nil, ErrChatUnavailable
	}
	if s.active == nil {
		s.active = make(map[uuid.UUID]*chatActiveRun)
	}
	active := &chatActiveRun{userID: run.UserID, sessionID: run.SessionID, cancel: cancel, done: make(chan struct{})}
	s.active[run.ID] = active
	return active, nil
}

func (s *ChatService) finishRun(runID uuid.UUID, active *chatActiveRun) {
	s.runMu.Lock()
	defer s.runMu.Unlock()
	delete(s.active, runID)
	close(active.done)
}

func (s *ChatService) StopGeneration(ctx context.Context, userID int64, sessionID, runID uuid.UUID) error {
	if _, err := s.repo.GetGeneration(ctx, userID, sessionID, runID); err != nil {
		return err
	}
	s.runMu.Lock()
	active := s.active[runID]
	if active != nil {
		active.cancel()
	}
	s.runMu.Unlock()
	if active != nil {
		select {
		case <-active.done:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return s.repo.StopGeneration(ctx, userID, sessionID, runID)
}

func (s *ChatService) DeleteSessions(ctx context.Context, userID int64, sessionID *uuid.UUID) error {
	s.runMu.Lock()
	pending := make([]*chatActiveRun, 0)
	for _, active := range s.active {
		if active.userID == userID && (sessionID == nil || active.sessionID == *sessionID) {
			active.cancel()
			pending = append(pending, active)
		}
	}
	s.runMu.Unlock()
	for _, active := range pending {
		select {
		case <-active.done:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return s.repo.DeleteSessions(ctx, userID, sessionID)
}

func (s *ChatService) stopLifecycle() error {
	s.runMu.Lock()
	s.closed = true
	pending := make([]<-chan struct{}, 0, len(s.active)+1)
	for _, active := range s.active {
		active.cancel()
		pending = append(pending, active.done)
	}
	if s.lifecycleCancel != nil {
		s.lifecycleCancel()
		pending = append(pending, s.lifecycleDone)
	}
	s.runMu.Unlock()
	timeout := time.NewTimer(10 * time.Second)
	defer timeout.Stop()
	for _, done := range pending {
		select {
		case <-done:
		case <-timeout.C:
			return errors.New("chat shutdown timed out")
		}
	}
	return nil
}
