package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type GroupSchedulingAccount struct {
	AccountID   int64  `json:"account_id"`
	Name        string `json:"name"`
	Priority    int    `json:"priority"`
	LoadFactor  int    `json:"load_factor"`
	Concurrency int    `json:"concurrency"`
}
type GroupScheduling struct {
	Version  string                   `json:"version"`
	Accounts []GroupSchedulingAccount `json:"accounts"`
}
type GroupSchedulingRepository interface {
	GetGroupScheduling(context.Context, int64) (*GroupScheduling, error)
	SaveGroupScheduling(context.Context, int64, *GroupScheduling) (*GroupScheduling, error)
}

var ErrSchedulingConflict = infraerrors.Conflict("GROUP_SCHEDULING_CONFLICT", "group membership or scheduling changed; reload and retry")

func (s *adminServiceImpl) GetGroupScheduling(ctx context.Context, id int64) (*GroupScheduling, error) {
	r, ok := s.groupRepo.(GroupSchedulingRepository)
	if !ok {
		return nil, fmt.Errorf("group scheduling repository unavailable")
	}
	return r.GetGroupScheduling(ctx, id)
}
func (s *adminServiceImpl) SaveGroupScheduling(ctx context.Context, id int64, in *GroupScheduling) (*GroupScheduling, error) {
	r, ok := s.groupRepo.(GroupSchedulingRepository)
	if !ok {
		return nil, fmt.Errorf("group scheduling repository unavailable")
	}
	seen := make(map[int64]bool, len(in.Accounts))
	for _, a := range in.Accounts {
		if a.AccountID <= 0 || a.Priority < 0 || a.Priority > 1000000 || seen[a.AccountID] {
			return nil, infraerrors.BadRequest("INVALID_GROUP_PRIORITY", "priorities must be integers from 0 to 1000000 and account IDs must be unique")
		}
		seen[a.AccountID] = true
	}
	return r.SaveGroupScheduling(ctx, id, in)
}

// The same global slot keys are used by legacy and independent scheduling.
type SchedulingCandidate struct {
	ID                                int64
	Priority, LoadFactor, Concurrency int
	PauseGeneration                   string
	IgnoreSlowTTFT                    bool
}
type IndependentSchedulingCache interface {
	AcquireScheduledAccount(context.Context, []SchedulingCandidate, string) (int64, error)
}

func (a *Account) PriorityInGroup(groupID int64) int {
	for _, g := range a.AccountGroups {
		if g.GroupID == groupID {
			return g.Priority
		}
	}
	return a.Priority
}
func (s *ConcurrencyService) AcquireScheduledAccount(ctx context.Context, accounts []*Account, groupID int64) (*Account, *AcquireResult, error) {
	if s == nil {
		return nil, nil, fmt.Errorf("concurrency service unavailable")
	}
	c, ok := s.cache.(IndependentSchedulingCache)
	if !ok {
		return nil, nil, fmt.Errorf("independent scheduling cache unavailable")
	}
	candidates := make([]SchedulingCandidate, 0, len(accounts))
	for _, a := range accounts {
		candidates = append(candidates, SchedulingCandidate{ID: a.ID, Priority: a.PriorityInGroup(groupID), LoadFactor: a.EffectiveLoadFactor(), Concurrency: a.Concurrency, PauseGeneration: a.SlowTTFTConfig().Generation, IgnoreSlowTTFT: slowTTFTContextExempt(ctx) || !a.SlowTTFTConfig().Enabled})
	}
	requestID := generateRequestID()
	id, err := c.AcquireScheduledAccount(ctx, candidates, requestID)
	if err != nil || id == 0 {
		return nil, nil, err
	}
	for _, a := range accounts {
		if a.ID == id {
			var once sync.Once
			return a, &AcquireResult{Acquired: true, ReleaseFunc: func() {
				once.Do(func() {
					bg, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					_ = s.cache.ReleaseAccountSlot(bg, id, requestID)
				})
			}}, nil
		}
	}
	return nil, nil, fmt.Errorf("scheduler returned unknown account")
}
