package service

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	ErrChatAuxiliarySlotsFull = errors.New("chat auxiliary concurrency limit reached")
	ErrChatAuxiliaryLeaseLost = errors.New("chat auxiliary concurrency lease lost")
)

// The handler acquires the separate API-key lease before these renewable slots.
type ChatAuxiliaryLeaseCache interface {
	AcquireChatAuxiliaryKindSlot(context.Context, int64, string, int, string) (bool, error)
	RefreshChatAuxiliarySlots(context.Context, int64, int64, string, bool, string) (bool, error)
	ReleaseChatAuxiliaryKindSlot(context.Context, int64, string, string) error
}

type ChatAuxiliarySlotLease struct {
	ctx               context.Context
	cancel            context.CancelCauseFunc
	cache             ConcurrencyCache
	leases            ChatAuxiliaryLeaseCache
	userID, accountID int64
	kind, requestID   string
	kindHeld          bool
	once              sync.Once
	done              chan struct{}
}

func (l *ChatAuxiliarySlotLease) Context() context.Context { return l.ctx }

func (l *ChatAuxiliarySlotLease) releaseSlots() {
	l.once.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		defer cancel()
		if l.kindHeld {
			_ = l.leases.ReleaseChatAuxiliaryKindSlot(ctx, l.accountID, l.kind, l.requestID)
		}
		_ = l.cache.ReleaseAccountSlot(ctx, l.accountID, l.requestID)
		_ = l.cache.ReleaseUserSlot(ctx, l.userID, l.requestID)
	})
}

func (l *ChatAuxiliarySlotLease) Release() {
	if l == nil {
		return
	}
	l.cancel(context.Canceled)
	<-l.done
}

func (l *ChatAuxiliarySlotLease) refresh() {
	defer close(l.done)
	defer l.releaseSlots()
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-l.ctx.Done():
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(l.ctx, 2*time.Second)
			ok, err := l.leases.RefreshChatAuxiliarySlots(ctx, l.userID, l.accountID, l.kind, l.kindHeld, l.requestID)
			cancel()
			if err != nil || !ok {
				l.cancel(ErrChatAuxiliaryLeaseLost)
				return
			}
		}
	}
}

func (s *ConcurrencyService) AcquireChatAuxiliarySlots(ctx context.Context, userID int64, userLimit int, accountID int64, accountLimit int, kind string, kindLimit int) (*ChatAuxiliarySlotLease, error) {
	if s == nil || s.cache == nil || userID <= 0 || accountID <= 0 || !chatAuxiliaryKind(kind) {
		return nil, ErrChatAuxiliaryUnavailable
	}
	leases, ok := s.cache.(ChatAuxiliaryLeaseCache)
	if !ok {
		return nil, ErrChatAuxiliaryUnavailable
	}
	// Even an unlimited user/account occupies a renewable member, so ordinary
	// model traffic observes it when a limit is subsequently configured.
	if userLimit <= 0 {
		userLimit = int(^uint(0) >> 1)
	}
	if accountLimit <= 0 {
		accountLimit = int(^uint(0) >> 1)
	}
	switch kind {
	case "asr", "tts":
		if kindLimit <= 0 || kindLimit > 50 {
			kindLimit = 50
		}
	case "ocr":
		if kindLimit <= 0 || kindLimit > 15 {
			kindLimit = 15
		}
	default:
		kindLimit = 0
	}
	leaseCtx, cancel := context.WithCancelCause(ctx)
	l := &ChatAuxiliarySlotLease{ctx: leaseCtx, cancel: cancel, cache: s.cache, leases: leases, userID: userID, accountID: accountID, kind: kind, requestID: generateRequestID(), done: make(chan struct{})}
	// On ambiguous Redis failures remove this operation's member too. Never
	// retry acquisition: doing so could create a slot after cancellation.
	acquireCtx, acquireCancel := context.WithTimeout(ctx, 6*time.Second)
	defer acquireCancel()
	acquired, err := s.cache.AcquireUserSlot(acquireCtx, userID, userLimit, l.requestID)
	if err == nil && acquired {
		acquired, err = s.cache.AcquireAccountSlot(acquireCtx, accountID, accountLimit, l.requestID)
	}
	if err == nil && acquired && kindLimit > 0 {
		l.kindHeld = true
		acquired, err = leases.AcquireChatAuxiliaryKindSlot(acquireCtx, accountID, kind, kindLimit, l.requestID)
	}
	if err != nil || !acquired || ctx.Err() != nil {
		cancel(ErrChatAuxiliaryLeaseLost)
		l.releaseSlots()
		if err != nil {
			return nil, ErrChatAuxiliaryUnavailable
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, ErrChatAuxiliarySlotsFull
	}
	go l.refresh()
	return l, nil
}
