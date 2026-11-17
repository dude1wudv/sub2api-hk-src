//go:build unit

package service

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

// Fallback coalescing and the value-slice reader are deferred. Keep the pool,
// eligibility, ownership and cancellation regressions on the existing API.
type fallbackPoolCache struct {
	SchedulerCache
	accounts []*Account
	snapshot []Account
	publish  int
	hit      bool
}

func (c *fallbackPoolCache) GetSnapshot(context.Context, SchedulerBucket) ([]*Account, bool, error) {
	return c.accounts, c.hit, nil
}

func (c *fallbackPoolCache) CaptureBucketWriteToken(_ context.Context, bucket SchedulerBucket) (SchedulerBucketWriteToken, error) {
	return SchedulerBucketWriteToken{Bucket: bucket, Epoch: 1}, nil
}

func (c *fallbackPoolCache) SetSnapshot(_ context.Context, _ SchedulerBucket, _ SchedulerBucketWriteToken, accounts []Account) error {
	c.publish++
	// Retain the supplied slice to catch response filtering that mutates it.
	c.snapshot = accounts
	return nil
}

type fallbackPoolRepo struct {
	AccountRepository
	accounts []Account
	load     func(context.Context) ([]Account, error)
}

func (r *fallbackPoolRepo) ListModelAvailabilityCandidates(ctx context.Context, _ *int64, _ []string, _ bool) ([]Account, error) {
	if r.load != nil {
		return r.load(ctx)
	}
	return r.accounts, nil
}

func newFallbackPoolService(cache SchedulerCache, repo AccountRepository) *SchedulerSnapshotService {
	return NewSchedulerSnapshotService(cache, nil, repo, nil, &config.Config{
		RunMode: config.RunModeStandard,
		Gateway: config.GatewayConfig{Scheduling: config.GatewaySchedulingConfig{
			DbFallbackEnabled: true,
		}},
	})
}

func TestSchedulerSnapshotFiltersTimeBlockedAccountsWithoutMutatingCache(t *testing.T) {
	future := time.Now().Add(time.Hour)
	past := time.Now().Add(-time.Minute)
	cache := &fallbackPoolCache{hit: true, accounts: []*Account{
		nil,
		{ID: 7311, Platform: PlatformOpenAI, Status: StatusActive, Schedulable: true, RateLimitResetAt: &future},
		{ID: 7312, Name: "ready", Platform: PlatformOpenAI, Status: StatusActive, Schedulable: true, RateLimitResetAt: &past},
		{ID: 7313, Platform: PlatformOpenAI, Status: StatusActive, Schedulable: true, TempUnschedulableUntil: &future},
		{ID: 7314, Platform: PlatformOpenAI, Status: StatusActive, Schedulable: true, OverloadUntil: &past},
		{ID: 7315, Platform: PlatformOpenAI, Status: StatusActive, Schedulable: true, OverloadUntil: &future},
		{ID: 7316, Platform: PlatformOpenAI, Status: StatusActive, Schedulable: true, SlowTTFTUntil: &future},
	}}
	svc := newFallbackPoolService(cache, nil)
	groupID := int64(74)

	accounts, _, err := svc.ListSchedulableAccounts(context.Background(), &groupID, PlatformOpenAI, false)
	require.NoError(t, err)
	require.Len(t, accounts, 2)
	require.Equal(t, []int64{7312, 7314}, []int64{accounts[0].ID, accounts[1].ID})
	accounts[0].Name = "mutated"
	require.Equal(t, "ready", cache.accounts[2].Name)
	require.Len(t, cache.accounts, 7)
}

func TestSchedulerFallbackPublishesPoolAndReturnsSchedulable(t *testing.T) {
	future := time.Now().Add(time.Hour)
	cache := &fallbackPoolCache{}
	repo := &fallbackPoolRepo{accounts: []Account{
		{ID: 7402, Name: "cooling", Platform: PlatformOpenAI, Status: StatusActive, Schedulable: true, RateLimitResetAt: &future},
		{ID: 7401, Name: "ready", Platform: PlatformOpenAI, Status: StatusActive, Schedulable: true},
	}}
	svc := newFallbackPoolService(cache, repo)
	groupID := int64(75)

	accounts, _, err := svc.ListSchedulableAccounts(context.Background(), &groupID, PlatformOpenAI, false)
	require.NoError(t, err)
	require.Len(t, accounts, 1)
	require.Equal(t, int64(7401), accounts[0].ID)
	require.Equal(t, 1, cache.publish)
	require.Len(t, cache.snapshot, 2, "the published snapshot keeps the configured pool")
	require.Equal(t, int64(7402), cache.snapshot[0].ID, "filtering must not compact the retained pool in place")
	accounts[0].Name = "mutated"
	require.Equal(t, "ready", cache.snapshot[1].Name)
}

func TestSchedulerFallbackCancellationPreventsPublishAfterDBReturns(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cache := &fallbackPoolCache{}
	repo := &fallbackPoolRepo{load: func(loadCtx context.Context) ([]Account, error) {
		cancel()
		require.ErrorIs(t, loadCtx.Err(), context.Canceled, "DB work retains the caller's cancellation")
		// Even a repository returning a late success must not publish.
		return []Account{{ID: 7501, Platform: PlatformOpenAI, Status: StatusActive, Schedulable: true}}, nil
	}}
	svc := newFallbackPoolService(cache, repo)

	accounts, _, err := svc.ListSchedulableAccounts(ctx, nil, PlatformOpenAI, false)
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, accounts)
	require.Zero(t, cache.publish)
}

func TestSchedulerFallbackHonorsConfiguredTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cache := &fallbackPoolCache{}
		repo := &fallbackPoolRepo{load: func(ctx context.Context) ([]Account, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		}}
		svc := newFallbackPoolService(cache, repo)
		svc.cfg.Gateway.Scheduling.DbFallbackTimeoutSeconds = 1
		started := time.Now()

		accounts, _, err := svc.ListSchedulableAccounts(context.Background(), nil, PlatformOpenAI, false)
		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.Equal(t, time.Second, time.Since(started))
		require.Nil(t, accounts)
		require.Zero(t, cache.publish)
	})
}
