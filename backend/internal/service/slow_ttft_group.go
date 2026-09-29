package service

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
)

type SlowTTFTGroupRepository interface {
	RecoverSlowTTFTGroup(context.Context, int64) (*time.Time, bool, error)
	SlowTTFTGroupExemption(context.Context, int64) (*time.Time, error)
}
type slowTTFTGroupKey struct{}
type slowTTFTGroupState struct {
	ID        int64
	Until     *time.Time
	Recovered bool
}

func slowTTFTGroupID(ctx context.Context) int64 {
	if state, ok := ctx.Value(slowTTFTGroupKey{}).(slowTTFTGroupState); ok {
		return state.ID
	}
	if group, ok := ctx.Value(ctxkey.Group).(*Group); ok && group != nil {
		return group.ID
	}
	return 0
}
func withSlowTTFTGroup(ctx context.Context, repo AccountRepository, id *int64) (context.Context, bool) {
	if id == nil {
		return context.WithValue(ctx, slowTTFTGroupKey{}, slowTTFTGroupState{}), false
	}
	r, ok := repo.(SlowTTFTGroupRepository)
	if !ok {
		return context.WithValue(ctx, slowTTFTGroupKey{}, slowTTFTGroupState{ID: *id}), false
	}
	if state, ok := ctx.Value(slowTTFTGroupKey{}).(slowTTFTGroupState); ok && state.ID == *id && state.Until != nil && time.Now().Before(*state.Until) {
		return ctx, state.Recovered
	}
	until, recovered, err := r.RecoverSlowTTFTGroup(ctx, *id)
	if err != nil {
		return context.WithValue(ctx, slowTTFTGroupKey{}, slowTTFTGroupState{ID: *id}), false
	}
	return context.WithValue(ctx, slowTTFTGroupKey{}, slowTTFTGroupState{*id, until, recovered}), recovered
}
func slowTTFTContextExempt(ctx context.Context) bool {
	state, ok := ctx.Value(slowTTFTGroupKey{}).(slowTTFTGroupState)
	return ok && state.Until != nil && time.Now().Before(*state.Until)
}
func slowTTFTGroupRecovered(ctx context.Context) bool {
	state, ok := ctx.Value(slowTTFTGroupKey{}).(slowTTFTGroupState)
	return ok && state.Recovered
}
func (s *RateLimitService) slowTTFTGroupExempt(ctx context.Context) bool {
	if slowTTFTContextExempt(ctx) {
		return true
	}
	id := slowTTFTGroupID(ctx)
	if id <= 0 || s == nil {
		return false
	}
	if r, ok := s.accountRepo.(SlowTTFTGroupRepository); ok {
		until, err := r.SlowTTFTGroupExemption(ctx, id)
		return err == nil && until != nil && time.Now().Before(*until)
	}
	return false
}
func accountForSlowTTFTContext(ctx context.Context, a *Account) *Account {
	if a == nil || !slowTTFTContextExempt(ctx) {
		return a
	}
	copy := *a
	copy.SlowTTFTUntil = nil
	copy.SlowTTFTReason = ""
	return &copy
}
func accountsForSlowTTFTContext(ctx context.Context, accounts []Account) []Account {
	if !slowTTFTContextExempt(ctx) {
		return accounts
	}
	out := append([]Account(nil), accounts...)
	for i := range out {
		out[i].SlowTTFTUntil = nil
		out[i].SlowTTFTReason = ""
	}
	return out
}

// WithSlowTTFTContext carries the actual scheduling group across fallback
// boundaries without replacing the authentication/billing group.
func (s *AccountSelectionResult) WithSlowTTFTContext(ctx context.Context) context.Context {
	if s == nil || s.slowTTFTGroup == nil {
		return ctx
	}
	return context.WithValue(ctx, slowTTFTGroupKey{}, *s.slowTTFTGroup)
}

// CopySlowTTFTUsageContext retains the actual scheduling group when a usage
// worker detaches from the request. Do not substitute the billing group for a
// fallback/composite selection, or inherit unrelated request cancellation.
func CopySlowTTFTUsageContext(dst, src context.Context) context.Context {
	if state, ok := src.Value(slowTTFTGroupKey{}).(slowTTFTGroupState); ok {
		return context.WithValue(dst, slowTTFTGroupKey{}, state)
	}
	return context.WithValue(dst, slowTTFTGroupKey{}, slowTTFTGroupState{ID: slowTTFTGroupID(src)})
}
