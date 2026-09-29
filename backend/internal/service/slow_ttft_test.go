//go:build unit

package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	openaiwsv2 "github.com/Wei-Shaw/sub2api/internal/service/openai_ws_v2"
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
func TestSlowTTFTObserverTimingAndFencing(t *testing.T) {
	t.Run("disabled by default", func(t *testing.T) {
		s, repo, cache := slowTTFTTestService(false)
		require.Nil(t, s.BeginSlowTTFT(context.Background(), repo.account))
		gets, _ := repo.count()
		attempts, _ := cache.snapshot()
		require.Zero(t, gets)
		require.Empty(t, attempts)
	})
	t.Run("strict threshold, expiry once, quick output, cancellation", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			s, repo, cache := slowTTFTTestService(true)
			ctx := context.Background()
			o := s.BeginSlowTTFT(ctx, repo.account)
			require.NotNil(t, o)
			time.Sleep(15 * time.Second)
			synctest.Wait()
			o.FirstOutput()
			attempts, slow := cache.snapshot()
			require.Len(t, attempts, 1)
			require.Equal(t, []bool{false}, slow)
			o = s.BeginSlowTTFT(ctx, repo.account)
			time.Sleep(15*time.Second + 2*time.Millisecond)
			synctest.Wait()
			attempts, slow = cache.snapshot()
			require.Len(t, attempts, 2)
			require.Equal(t, []bool{false, true}, slow)
			o.FirstOutput()
			attempts, _ = cache.snapshot()
			require.Len(t, attempts, 2)
			o = s.BeginSlowTTFT(ctx, repo.account)
			o.FirstOutput()
			attempts, slow = cache.snapshot()
			require.Len(t, attempts, 3)
			require.Equal(t, []bool{false, true, false}, slow)
			cancelCtx, cancel := context.WithCancel(ctx)
			o = s.BeginSlowTTFT(cancelCtx, repo.account)
			cancel()
			synctest.Wait()
			attempts, _ = cache.snapshot()
			require.Len(t, attempts, 3)
			o.Close()
			attempts, _ = cache.snapshot()
			require.Len(t, attempts, 3)
		})
	})
	t.Run("group exemption suppresses observer without suppressing other group", func(t *testing.T) {
		// An exempt context is obtained via the account group repository contract.
		groupRepo := &slowTTFTGroupTestRepo{slowTTFTTestRepo: slowTTFTTestRepo{account: &Account{ID: 71, Extra: map[string]any{SlowTTFTConfigKey: func() SlowTTFTConfig { c := DefaultSlowTTFTConfig(); c.Enabled = true; return c }()}}}}
		cache := &slowTTFTTestCache{}
		s := NewRateLimitService(groupRepo, nil, nil, nil, cache)
		gid := int64(8)
		ctx, _ := withSlowTTFTGroup(context.Background(), groupRepo, &gid)
		require.Nil(t, s.BeginSlowTTFT(ctx, groupRepo.account))
		other, _ := withSlowTTFTGroup(context.Background(), groupRepo, ptrInt64(9))
		observer := s.BeginSlowTTFT(other, groupRepo.account)
		require.NotNil(t, observer)
		observer.FirstOutput()
		observer.Close()
		attempts, slow := cache.snapshot()
		require.Len(t, attempts, 1)
		require.Equal(t, []bool{false}, slow)
	})
	t.Run("stale-generation output is discarded", func(t *testing.T) {
		s, repo, cache := slowTTFTTestService(true)
		o := s.BeginSlowTTFT(context.Background(), repo.account)
		cfg := repo.account.SlowTTFTConfig()
		cfg.Generation = "new-generation"
		repo.account.Extra[SlowTTFTConfigKey] = cfg
		o.FirstOutput()
		attempts, _ := cache.snapshot()
		require.Empty(t, attempts)
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

type slowTTFTTestUpstream struct {
	HTTPUpstream
	calls int
	body  string
}

func (u *slowTTFTTestUpstream) Do(_ *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	u.calls++
	return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(u.body))}, nil
}
func (u *slowTTFTTestUpstream) DoWithTLS(r *http.Request, p string, id int64, c int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return u.Do(r, p, id, c)
}
func TestSlowTTFTHTTPDecoratorScopesEachAttemptAndMeaningfulOutput(t *testing.T) {
	s, repo, cache := slowTTFTTestService(true)
	upstream := &slowTTFTTestUpstream{body: "data: {\"choices\":[{\"delta\":{\"content\":\"hello\"}}]}\n\n"}
	wrapped := WithSlowTTFTUpstream(upstream, s)
	for i := 0; i < 2; i++ {
		req, err := http.NewRequest(http.MethodPost, "http://example.com/v1/chat/completions", bytes.NewReader([]byte(`{"stream":true}`)))
		require.NoError(t, err)
		resp, err := wrapped.Do(req, "", repo.account.ID, 1)
		require.NoError(t, err)
		_, err = io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
	}
	attempts, slow := cache.snapshot()
	require.Equal(t, 2, upstream.calls)
	require.Len(t, attempts, 2)
	require.Equal(t, []bool{false, false}, slow)
}

type slowTTFTFakeFrameConn struct {
	messages [][]byte
	writes   int
}

func (c *slowTTFTFakeFrameConn) ReadFrame(context.Context) (coderws.MessageType, []byte, error) {
	if len(c.messages) == 0 {
		return 0, nil, io.EOF
	}
	p := c.messages[0]
	c.messages = c.messages[1:]
	return coderws.MessageText, p, nil
}
func (c *slowTTFTFakeFrameConn) WriteFrame(context.Context, coderws.MessageType, []byte) error {
	c.writes++
	return nil
}
func (c *slowTTFTFakeFrameConn) Close() error { return nil }
func TestSlowTTFTWSDecoratorObservesEachResponseRound(t *testing.T) {
	s, repo, cache := slowTTFTTestService(true)
	base := &slowTTFTFakeFrameConn{messages: [][]byte{[]byte(`{"type":"response.output_text.delta","delta":"a"}`), []byte(`{"type":"response.completed"}`), []byte(`{"type":"response.output_text.delta","delta":"b"}`), []byte(`{"type":"response.completed"}`)}}
	conn := &slowTTFTFrameConn{FrameConn: base, ctx: context.Background(), protection: s, accountID: repo.account.ID}
	for range 2 {
		require.NoError(t, conn.WriteFrame(context.Background(), coderws.MessageText, []byte(`{"type":"response.create"}`)))
		_, _, err := conn.ReadFrame(context.Background())
		require.NoError(t, err)
		_, _, err = conn.ReadFrame(context.Background())
		require.NoError(t, err)
	}
	var frame openaiwsv2.FrameConn = base
	require.NotNil(t, frame)
	attempts, slow := cache.snapshot()
	require.Len(t, attempts, 2)
	require.Equal(t, []bool{false, false}, slow)
}
func TestSlowTTFTMeaningfulOutputExcludesMetadataAndAcceptsTools(t *testing.T) {
	cases := []struct {
		data       string
		meaningful bool
	}{
		{`{"type":"response.created"}`, false},
		{`{"type":"heartbeat"}`, false},
		{`{"type":"response.output_text.delta","delta":""}`, false},
		{`{"type":"response.function_call_arguments.delta","delta":"{\"x\":"}`, true},
		{`{"type":"response.output_text.delta","delta":"hello"}`, true},
		{`{"type":"content_block_delta","delta":{"partial_json":"{}"}}`, true},
	}
	for _, tc := range cases {
		require.Equal(t, tc.meaningful, SlowTTFTMeaningfulOutput([]byte(tc.data)), tc.data)
	}
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

func TestIndependentGroupSchedulingOnlyActivatesForSlowTTFTRecovery(t *testing.T) {
	service := &OpenAIGatewayService{}
	groupID := int64(7)

	require.False(t, service.independentGroupScheduling(context.Background(), &groupID))

	ctx := context.WithValue(context.Background(), slowTTFTGroupKey{}, slowTTFTGroupState{
		ID:        groupID,
		Recovered: true,
	})
	require.True(t, service.independentGroupScheduling(ctx, &groupID))
}
