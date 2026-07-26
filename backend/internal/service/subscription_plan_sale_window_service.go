package service

import (
	"context"
	"database/sql"
	"log/slog"
	"sync"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/subscriptionplan"
	"github.com/google/uuid"
)

const (
	subscriptionPlanSaleWindowLeaderLockKey = "subscription:plan:sale-window:leader"
	subscriptionPlanSaleWindowLeaderLockTTL = 2 * time.Minute
)

// SubscriptionPlanSaleWindowService persists automatic unlisting after the configured cutoff.
type SubscriptionPlanSaleWindowService struct {
	entClient  *dbent.Client
	interval   time.Duration
	stopCh     chan struct{}
	stopOnce   sync.Once
	wg         sync.WaitGroup
	lockCache  LeaderLockCache
	db         *sql.DB
	instanceID string
}

func NewSubscriptionPlanSaleWindowService(entClient *dbent.Client, interval time.Duration) *SubscriptionPlanSaleWindowService {
	return &SubscriptionPlanSaleWindowService{
		entClient:  entClient,
		interval:   interval,
		stopCh:     make(chan struct{}),
		instanceID: uuid.NewString(),
	}
}

func (s *SubscriptionPlanSaleWindowService) SetLeaderLock(lockCache LeaderLockCache, db *sql.DB) {
	if s == nil {
		return
	}
	s.lockCache = lockCache
	s.db = db
}

func (s *SubscriptionPlanSaleWindowService) Start() {
	if s == nil || s.entClient == nil || s.interval <= 0 {
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

func (s *SubscriptionPlanSaleWindowService) Stop() {
	if s == nil {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
	s.wg.Wait()
}

func (s *SubscriptionPlanSaleWindowService) runOnce() {
	lockCtx, lockCancel := context.WithTimeout(context.Background(), 2*time.Second)
	release, ok := tryAcquireSingletonLeaderLock(lockCtx, s.lockCache, s.db, subscriptionPlanSaleWindowLeaderLockKey, s.instanceID, subscriptionPlanSaleWindowLeaderLockTTL)
	lockCancel()
	if !ok {
		return
	}
	defer release()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	count, err := s.entClient.SubscriptionPlan.Update().
		Where(
			subscriptionplan.ForSaleEQ(true),
			subscriptionplan.SaleEndsAtNotNil(),
			subscriptionplan.SaleEndsAtLTE(subscriptionBusinessNow()),
		).
		SetForSale(false).
		Save(ctx)
	if err != nil {
		slog.Error("subscription sale-window unlisting failed", "error", err)
		return
	}
	if count > 0 {
		slog.Info("subscription sale-window plans unlisted", "count", count)
	}
}
