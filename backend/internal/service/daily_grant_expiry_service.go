package service

import (
	"context"
	"database/sql"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	// dailyGrantExpiryLeaderLockKey gates the periodic sweep so only one instance
	// marks expired daily-balance grants per cycle in a multi-instance deployment.
	dailyGrantExpiryLeaderLockKey = "daily_grant:expiry:leader"
	// dailyGrantExpiryLeaderLockTTL must exceed the sweep timeout so the lock never
	// expires mid-run.
	dailyGrantExpiryLeaderLockTTL = 3 * time.Minute
	dailyGrantExpiryTimeout       = 30 * time.Second
)

// DailyGrantExpiryService periodically marks expired daily-balance grants
// (expires_at < now) as expired so they stop counting toward usable balance.
//
// Defense-in-depth: this is a backstop. All read/deduct queries already filter
// `expires_at > now` lazily, so a lagging sweep never causes expired grants to be
// spent — the sweep just keeps the status column tidy and the SumActiveRemaining
// fast path accurate.
type DailyGrantExpiryService struct {
	grantRepo DailyGrantRepository
	interval  time.Duration
	stopCh    chan struct{}
	stopOnce  sync.Once
	wg        sync.WaitGroup

	lockCache  LeaderLockCache
	db         *sql.DB
	instanceID string
}

func NewDailyGrantExpiryService(grantRepo DailyGrantRepository, interval time.Duration) *DailyGrantExpiryService {
	return &DailyGrantExpiryService{
		grantRepo:  grantRepo,
		interval:   interval,
		stopCh:     make(chan struct{}),
		instanceID: uuid.NewString(),
	}
}

// SetLeaderLock injects the leader-lock cache and DB used to elect a single
// instance for the periodic sweep. When both are nil the job runs ungated
// (single-instance / test behavior).
func (s *DailyGrantExpiryService) SetLeaderLock(lockCache LeaderLockCache, db *sql.DB) {
	if s == nil {
		return
	}
	s.lockCache = lockCache
	s.db = db
}

func (s *DailyGrantExpiryService) Start() {
	if s == nil || s.grantRepo == nil || s.interval <= 0 {
		return
	}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		s.runOnce()
		for {
			select {
			case <-ticker.C:
				s.runOnce()
			case <-s.stopCh:
				return
			}
		}
	}()
}

func (s *DailyGrantExpiryService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
	s.wg.Wait()
}

func (s *DailyGrantExpiryService) runOnce() {
	// Multi-instance guard: only the leader sweeps per cycle, avoiding redundant
	// bulk UPDATEs across instances.
	lockCtx, lockCancel := context.WithTimeout(context.Background(), 2*time.Second)
	release, ok := tryAcquireSingletonLeaderLock(lockCtx, s.lockCache, s.db, dailyGrantExpiryLeaderLockKey, s.instanceID, dailyGrantExpiryLeaderLockTTL)
	lockCancel()
	if !ok {
		return
	}
	defer release()

	ctx, cancel := context.WithTimeout(context.Background(), dailyGrantExpiryTimeout)
	defer cancel()
	expired, err := s.grantRepo.MarkExpired(ctx, time.Now())
	if err != nil {
		slog.Error("[DailyGrantExpiry] failed to mark expired grants", "error", err)
		return
	}
	if expired > 0 {
		slog.Info("[DailyGrantExpiry] marked expired daily grants", "count", expired)
	}
}
