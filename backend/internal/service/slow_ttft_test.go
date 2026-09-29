//go:build unit

package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	coderws "github.com/coder/websocket"
	"github.com/stretchr/testify/require"
)

type slowTTFTTestRepo struct {
	AccountRepository
	mu      sync.Mutex
	account *Account
	gets    int
	pauses  int
}

func (r *slowTTFTTestRepo) GetByID(_ context.Context, id int64) (*Account, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.gets++
	if r.account == nil || r.account.ID != id {
		return nil, ErrAccountNotFound
	}
	copy := *r.account
	return &copy, nil
}
func (r *slowTTFTTestRepo) SetSlowTTFTPause(context.Context, int64, SlowTTFTConfig, time.Time, string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pauses++
	return true, nil
}
func (r *slowTTFTTestRepo) ClearSlowTTFTPause(context.Context, int64) error { return nil }
func (r *slowTTFTTestRepo) count() (int, int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.gets, r.pauses
}

type slowTTFTTestCache struct {
	TempUnschedCache
	mu       sync.Mutex
	attempts []string
	slow     []bool
	result   SlowTTFTObservation
}

func (c *slowTTFTTestCache) ObserveSlowTTFT(_ context.Context, _ int64, _ SlowTTFTConfig, attempt string, slow bool) (SlowTTFTObservation, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.attempts = append(c.attempts, attempt)
	c.slow = append(c.slow, slow)
	return c.result, nil
}
func (c *slowTTFTTestCache) snapshot() ([]string, []bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.attempts...), append([]bool(nil), c.slow...)
}
func slowTTFTTestService(enabled bool) (*RateLimitService, *slowTTFTTestRepo, *slowTTFTTestCache) {
	var extra map[string]any
	if enabled {
		cfg := DefaultSlowTTFTConfig()
		cfg.Enabled = true
		extra = map[string]any{SlowTTFTConfigKey: cfg}
	}
	repo := &slowTTFTTestRepo{account: &Account{ID: 71, Extra: extra}}
	cache := &slowTTFTTestCache{}
	return NewRateLimitService(repo, nil, nil, nil, cache), repo, cache
}
func slowTTFTUsageLog(requestID string, firstTokenMs int) *UsageLog {
	return &UsageLog{AccountID: 71, APIKeyID: 5, RequestID: requestID, Stream: true, FirstTokenMs: &firstTokenMs}
}

func TestSlowTTFTUsageFirstTokenSampling(t *testing.T) {
	t.Run("disabled by default", func(t *testing.T) {
		s, repo, cache := slowTTFTTestService(false)
		s.ObserveUsageFirstToken(context.Background(), slowTTFTUsageLog("r1", 60000), repo.account.SlowTTFTConfig())
		attempts, _ := cache.snapshot()
		require.Empty(t, attempts)
	})
	t.Run("strict threshold on the recorded first token", func(t *testing.T) {
		s, repo, cache := slowTTFTTestService(true)
		s.ObserveUsageFirstToken(context.Background(), slowTTFTUsageLog("r1", 15000), repo.account.SlowTTFTConfig())
		s.ObserveUsageFirstToken(context.Background(), slowTTFTUsageLog("r2", 15001), repo.account.SlowTTFTConfig())
		s.ObserveUsageFirstToken(context.Background(), slowTTFTUsageLog("r3", 582), repo.account.SlowTTFTConfig())
		attempts, slow := cache.snapshot()
		require.Equal(t, []string{"5:r1", "5:r2", "5:r3"}, attempts)
		require.Equal(t, []bool{false, true, false}, slow)
	})
	t.Run("records without a streamed first token carry no sample", func(t *testing.T) {
		s, repo, cache := slowTTFTTestService(true)
		nonStream := slowTTFTUsageLog("a", 60000)
		nonStream.Stream = false
		missing := slowTTFTUsageLog("b", 0)
		missing.FirstTokenMs = nil
		image := slowTTFTUsageLog("c", 60000)
		image.ImageCount = 1
		cyber := slowTTFTUsageLog("d", 60000)
		cyber.RequestType = RequestTypeCyberBlocked
		for _, log := range []*UsageLog{nil, nonStream, missing, image, cyber} {
			s.ObserveUsageFirstToken(context.Background(), log, repo.account.SlowTTFTConfig())
		}
		attempts, _ := cache.snapshot()
		require.Empty(t, attempts)

		ws := slowTTFTUsageLog("e", 60000)
		ws.Stream = false
		ws.OpenAIWSMode = true
		s.ObserveUsageFirstToken(context.Background(), ws, repo.account.SlowTTFTConfig())
		attempts, slow := cache.snapshot()
		require.Equal(t, []string{"5:e"}, attempts)
		require.Equal(t, []bool{true}, slow)
	})
	t.Run("paused account is not sampled again", func(t *testing.T) {
		s, repo, cache := slowTTFTTestService(true)
		until := time.Now().Add(time.Hour)
		repo.account.SlowTTFTUntil = &until
		s.ObserveUsageFirstToken(context.Background(), slowTTFTUsageLog("r1", 60000), repo.account.SlowTTFTConfig())
		attempts, _ := cache.snapshot()
		require.Empty(t, attempts)
	})
	t.Run("tripped observation persists the pause", func(t *testing.T) {
		s, repo, cache := slowTTFTTestService(true)
		cache.result = SlowTTFTObservation{Tripped: true, Until: time.Now().Add(time.Hour), Reason: "consecutive"}
		s.ObserveUsageFirstToken(context.Background(), slowTTFTUsageLog("r1", 60000), repo.account.SlowTTFTConfig())
		_, pauses := repo.count()
		require.Equal(t, 1, pauses)
	})
	t.Run("group exemption suppresses sampling without suppressing other group", func(t *testing.T) {
		groupRepo := &slowTTFTGroupTestRepo{slowTTFTTestRepo: slowTTFTTestRepo{account: &Account{ID: 71, Extra: map[string]any{SlowTTFTConfigKey: func() SlowTTFTConfig { c := DefaultSlowTTFTConfig(); c.Enabled = true; return c }()}}}}
		cache := &slowTTFTTestCache{}
		s := NewRateLimitService(groupRepo, nil, nil, nil, cache)
		exempt, _ := withSlowTTFTGroup(context.Background(), groupRepo, ptrInt64(8))
		s.ObserveUsageFirstToken(exempt, slowTTFTUsageLog("r1", 60000), groupRepo.account.SlowTTFTConfig())
		other, _ := withSlowTTFTGroup(context.Background(), groupRepo, ptrInt64(9))
		s.ObserveUsageFirstToken(other, slowTTFTUsageLog("r2", 60000), groupRepo.account.SlowTTFTConfig())
		attempts, slow := cache.snapshot()
		require.Equal(t, []string{"5:r2"}, attempts)
		require.Equal(t, []bool{true}, slow)
	})
}

type slowTTFTGroupTestRepo struct{ slowTTFTTestRepo }

func (r *slowTTFTGroupTestRepo) RecoverSlowTTFTGroup(_ context.Context, id int64) (*time.Time, bool, error) {
	if id == 8 {
		until := time.Now().Add(time.Hour)
		return &until, false, nil
	}
	return nil, false, nil
}
func (r *slowTTFTGroupTestRepo) SlowTTFTGroupExemption(_ context.Context, id int64) (*time.Time, error) {
	if id == 8 {
		until := time.Now().Add(time.Hour)
		return &until, nil
	}
	return nil, nil
}

type slowTTFTGroupRecoveryTestRepo struct {
	AccountRepository
	recoverCalls []int64
	recover      func(int64) (*time.Time, bool, error)
}

func (r *slowTTFTGroupRecoveryTestRepo) RecoverSlowTTFTGroup(_ context.Context, id int64) (*time.Time, bool, error) {
	r.recoverCalls = append(r.recoverCalls, id)
	return r.recover(id)
}

func (r *slowTTFTGroupRecoveryTestRepo) SlowTTFTGroupExemption(context.Context, int64) (*time.Time, error) {
	return nil, nil
}

type slowTTFTFakeFrameConn struct{ writes int }

func (c *slowTTFTFakeFrameConn) ReadFrame(context.Context) (coderws.MessageType, []byte, error) {
	return 0, nil, errors.New("unused")
}
func (c *slowTTFTFakeFrameConn) WriteFrame(context.Context, coderws.MessageType, []byte) error {
	c.writes++
	return nil
}
func (c *slowTTFTFakeFrameConn) Close() error { return nil }
func TestSlowTTFTWSDecoratorRefusesNewRoundWhilePaused(t *testing.T) {
	s, repo, cache := slowTTFTTestService(true)
	base := &slowTTFTFakeFrameConn{}
	conn := &slowTTFTFrameConn{FrameConn: base, ctx: context.Background(), protection: s, accountID: repo.account.ID}
	create := []byte(`{"type":"response.create"}`)
	require.NoError(t, conn.WriteFrame(context.Background(), coderws.MessageText, create))
	until := time.Now().Add(time.Hour)
	repo.account.SlowTTFTUntil = &until
	require.ErrorIs(t, conn.WriteFrame(context.Background(), coderws.MessageText, create), ErrNoAvailableAccounts)
	require.NoError(t, conn.WriteFrame(context.Background(), coderws.MessageText, []byte(`{"type":"response.cancel"}`)))
	require.Equal(t, 2, base.writes)
	attempts, _ := cache.snapshot()
	require.Empty(t, attempts)
}
func TestAccountSelectionWithSlowTTFTContextScopesGroupWithoutReplacingBillingGroup(t *testing.T) {
	const billingGroupID int64 = 10
	const selectedGroupID int64 = 20
	billingGroup := &Group{ID: billingGroupID}
	ctx := context.WithValue(context.Background(), ctxkey.Group, billingGroup)

	selected := (&AccountSelectionResult{
		slowTTFTGroup: &slowTTFTGroupState{ID: selectedGroupID},
	}).WithSlowTTFTContext(ctx)
	require.Same(t, billingGroup, selected.Value(ctxkey.Group))
	require.Equal(t, selectedGroupID, slowTTFTGroupID(selected))

	previous := context.WithValue(ctx, slowTTFTGroupKey{}, slowTTFTGroupState{ID: selectedGroupID})
	ungrouped := (&AccountSelectionResult{
		slowTTFTGroup: &slowTTFTGroupState{},
	}).WithSlowTTFTContext(previous)
	require.Same(t, billingGroup, ungrouped.Value(ctxkey.Group))
	require.Zero(t, slowTTFTGroupID(ungrouped), "ungrouped selection must clear the prior scheduling group")
}
func TestWithSlowTTFTGroupClearsPriorGroupExemptionAfterRecoveryLookupFailure(t *testing.T) {
	groupA := int64(8)
	groupB := int64(9)
	until := time.Now().Add(time.Hour)
	repo := &slowTTFTGroupRecoveryTestRepo{
		recover: func(id int64) (*time.Time, bool, error) {
			if id == groupA {
				return &until, false, nil
			}
			require.Equal(t, groupB, id)
			return nil, false, errors.New("group recovery lookup failed")
		},
	}

	ctxA, recoveredA := withSlowTTFTGroup(context.Background(), repo, &groupA)
	require.False(t, recoveredA)
	require.True(t, slowTTFTContextExempt(ctxA), "group A should carry its recovered exemption")

	ctxB, recoveredB := withSlowTTFTGroup(ctxA, repo, &groupB)
	require.False(t, recoveredB)
	require.Equal(t, groupB, slowTTFTGroupID(ctxB))
	require.False(t, slowTTFTContextExempt(ctxB), "group B must not inherit group A's exemption")
	require.Equal(t, []int64{groupA, groupB}, repo.recoverCalls)
}

func TestWithSlowTTFTGroupRetriesRecoveryWhenPriorLookupFailed(t *testing.T) {
	groupID := int64(12)
	until := time.Now().Add(time.Hour)
	repo := &slowTTFTGroupRecoveryTestRepo{}
	repo.recover = func(id int64) (*time.Time, bool, error) {
		require.Equal(t, groupID, id)
		if len(repo.recoverCalls) == 1 {
			return nil, false, errors.New("temporary group recovery lookup failure")
		}
		return &until, true, nil
	}

	firstContext, firstRecovered := withSlowTTFTGroup(context.Background(), repo, &groupID)
	require.False(t, firstRecovered)
	require.False(t, slowTTFTContextExempt(firstContext))
	require.Equal(t, []int64{groupID}, repo.recoverCalls)

	nextSelectionContext, nextRecovered := withSlowTTFTGroup(firstContext, repo, &groupID)
	require.True(t, nextRecovered, "the next selection must retry group recovery")
	require.Equal(t, groupID, slowTTFTGroupID(nextSelectionContext))
	require.True(t, slowTTFTContextExempt(nextSelectionContext))
	require.True(t, slowTTFTGroupRecovered(nextSelectionContext))
	require.Equal(t, []int64{groupID, groupID}, repo.recoverCalls)
}

func TestSlowTTFTUsageRejectsOldPolicyAfterReset(t *testing.T) {
	s, repo, cache := slowTTFTTestService(true)
	admitted := repo.account.SlowTTFTConfig()
	admitted.Generation = "before-clear"
	current := admitted
	current.Generation = "after-clear"
	repo.account.Extra = map[string]any{SlowTTFTConfigKey: current}

	s.ObserveUsageFirstToken(context.Background(), slowTTFTUsageLog("late-old-request", 60000), admitted)
	attempts, _ := cache.snapshot()
	require.Empty(t, attempts, "old requests cannot enter a new epoch after a clear or configuration edit")
	s.ObserveUsageFirstToken(context.Background(), slowTTFTUsageLog("new-request", 60000), current)
	attempts, _ = cache.snapshot()
	require.Equal(t, []string{"5:new-request"}, attempts)
}

func TestCopySlowTTFTUsageContextPreservesSelectedGroup(t *testing.T) {
	s, repo, cache := slowTTFTTestService(true)
	groupRepo := &slowTTFTGroupTestRepo{slowTTFTTestRepo: slowTTFTTestRepo{account: repo.account}}
	s.accountRepo = groupRepo
	billing := context.WithValue(context.Background(), ctxkey.Group, &Group{ID: 9})
	until := time.Now().Add(time.Hour)
	parent, cancel := context.WithCancel((&AccountSelectionResult{
		slowTTFTGroup: &slowTTFTGroupState{ID: 8, Until: &until},
	}).WithSlowTTFTContext(billing))
	cancel()
	worker := CopySlowTTFTUsageContext(context.Background(), parent)
	require.NoError(t, worker.Err(), "copy values, not the canceled request lifetime")
	require.Equal(t, int64(8), slowTTFTGroupID(worker))
	s.ObserveUsageFirstToken(worker, slowTTFTUsageLog("exempt", 60000), repo.account.SlowTTFTConfig())
	attempts, _ := cache.snapshot()
	require.Empty(t, attempts)

	ungrouped := (&AccountSelectionResult{slowTTFTGroup: &slowTTFTGroupState{}}).WithSlowTTFTContext(billing)
	require.Zero(t, slowTTFTGroupID(CopySlowTTFTUsageContext(worker, ungrouped)))
}

func TestSlowTTFTPostSlotAdmissionRechecksPauseAndFreezesPolicy(t *testing.T) {
	protection, repo, _ := slowTTFTTestService(true)
	selected := *repo.account
	until := time.Now().Add(time.Hour)
	repo.account.SlowTTFTUntil = &until
	openai := &OpenAIGatewayService{rateLimitService: protection}
	gateway := &GatewayService{rateLimitService: protection}
	for _, admission := range []func(context.Context, *Account) (*Account, bool, string){openai.PostSlotAdmission, gateway.PostSlotAdmission} {
		_, vetoed, reason := admission(context.Background(), &selected)
		require.True(t, vetoed, "a pause set while waiting must reject the acquired slot")
		require.Equal(t, "slow_ttft_paused", reason)
		exempt := context.WithValue(context.Background(), slowTTFTGroupKey{}, slowTTFTGroupState{ID: 8, Until: &until})
		_, vetoed, _ = admission(exempt, &selected)
		require.False(t, vetoed, "an exempt group may still use this account")
	}
	repo.account.SlowTTFTUntil = nil
	policy := repo.account.SlowTTFTConfig()
	policy.Generation = "at-admission"
	repo.account.Extra = map[string]any{SlowTTFTConfigKey: policy}
	admitted, vetoed, _ := openai.PostSlotAdmission(context.Background(), &selected)
	require.False(t, vetoed)
	require.Equal(t, "at-admission", admitted.SlowTTFTConfig().Generation)
	require.Empty(t, selected.SlowTTFTConfig().Generation, "do not mutate the selected/shared snapshot")
	policy.Generation = "next-turn"
	repo.account.Extra[SlowTTFTConfigKey] = policy
	require.Equal(t, "at-admission", admitted.SlowTTFTConfig().Generation, "queued usage must retain its original policy")
}
