//go:build integration

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSchedulingProtectionIntegration(t *testing.T) {
	ctx := context.Background()
	name := fmt.Sprintf("slow-ttft-%d", time.Now().UnixNano())
	client := testEntClient(t)
	accounts := newAccountRepositoryWithSQL(client, integrationDB, nil)
	groups := newGroupRepositoryWithSQL(client, integrationDB)
	cfg := service.DefaultSlowTTFTConfig()
	cfg.Enabled = true

	t.Run("migration trigger changes generation and disabling clears pause", func(t *testing.T) {
		a := mustCreateAccount(t, client, &service.Account{Name: name + "-trigger", Extra: map[string]any{service.SlowTTFTConfigKey: cfg}})
		var generation string
		require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT extra->'slow_ttft_protection'->>'generation' FROM accounts WHERE id=$1`, a.ID).Scan(&generation))
		require.NotEmpty(t, generation)
		_, err := integrationDB.ExecContext(ctx, `UPDATE accounts SET slow_ttft_until=NOW()+INTERVAL '1 hour',slow_ttft_reason='slow' WHERE id=$1`, a.ID)
		require.NoError(t, err)
		_, err = integrationDB.ExecContext(ctx, `UPDATE accounts SET extra=jsonb_set(extra,'{slow_ttft_protection,threshold_seconds}','16'::jsonb) WHERE id=$1`, a.ID)
		require.NoError(t, err)
		var updatedGeneration string
		require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT extra->'slow_ttft_protection'->>'generation' FROM accounts WHERE id=$1`, a.ID).Scan(&updatedGeneration))
		require.NotEqual(t, generation, updatedGeneration)
		_, err = integrationDB.ExecContext(ctx, `UPDATE accounts SET extra=jsonb_set(extra,'{slow_ttft_protection,enabled}','false'::jsonb) WHERE id=$1`, a.ID)
		require.NoError(t, err)
		var until sql.NullTime
		var reason string
		require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT slow_ttft_until,slow_ttft_reason FROM accounts WHERE id=$1`, a.ID).Scan(&until, &reason))
		require.False(t, until.Valid)
		require.Empty(t, reason)
	})

	t.Run("manual clear preserves other limits and group scheduling conflicts", func(t *testing.T) {
		g := mustCreateGroup(t, client, &service.Group{Name: name + "-scheduling"})
		a := mustCreateAccount(t, client, &service.Account{Name: name + "-member", Extra: map[string]any{service.SlowTTFTConfigKey: cfg}})
		mustBindAccountToGroup(t, client, a.ID, g.ID, 7)
		var generation string
		require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT extra->'slow_ttft_protection'->>'generation' FROM accounts WHERE id=$1`, a.ID).Scan(&generation))
		var rateLimitUntil time.Time
		require.NoError(t, integrationDB.QueryRowContext(ctx, `UPDATE accounts SET slow_ttft_until=NULL,slow_ttft_reason='',rate_limit_reset_at=NOW()+INTERVAL '2 hours' WHERE id=$1 RETURNING rate_limit_reset_at`, a.ID).Scan(&rateLimitUntil))
		applied, err := accounts.SetSlowTTFTPause(ctx, a.ID, cfgWithGeneration(cfg, generation), time.Now().Add(time.Hour), "consecutive")
		require.NoError(t, err)
		require.True(t, applied, "the first conditional pause must win")
		applied, err = accounts.SetSlowTTFTPause(ctx, a.ID, cfgWithGeneration(cfg, generation), time.Now().Add(2*time.Hour), "consecutive")
		require.NoError(t, err)
		require.False(t, applied, "a concurrent pause must not extend the winner")
		require.NoError(t, accounts.ClearSlowTTFTPause(ctx, a.ID))
		var gotLimit sql.NullTime
		var gotReason string
		require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT rate_limit_reset_at,slow_ttft_reason FROM accounts WHERE id=$1`, a.ID).Scan(&gotLimit, &gotReason))
		require.True(t, gotLimit.Valid)
		require.WithinDuration(t, rateLimitUntil, gotLimit.Time, time.Second)
		require.Empty(t, gotReason)

		_, err = integrationDB.ExecContext(ctx, `UPDATE groups SET independent_scheduling=true WHERE id=$1`, g.ID)
		require.NoError(t, err)
		saved, err := groups.GetGroupScheduling(ctx, g.ID)
		require.NoError(t, err)
		require.Equal(t, 7, saved.Accounts[0].Priority, "the editor must expose the group binding priority")
		saved.Accounts[0].Priority = 3
		result, err := groups.SaveGroupScheduling(ctx, g.ID, saved)
		require.NoError(t, err)
		require.Equal(t, 3, result.Accounts[0].Priority)
		var independent bool
		require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT independent_scheduling FROM groups WHERE id=$1`, g.ID).Scan(&independent))
		require.False(t, independent, "saving priorities retires persisted strict scheduling")
		_, err = groups.SaveGroupScheduling(ctx, g.ID, saved)
		require.ErrorIs(t, err, service.ErrSchedulingConflict)
		missing := *result
		missing.Accounts = nil
		_, err = groups.SaveGroupScheduling(ctx, g.ID, &missing)
		require.ErrorIs(t, err, service.ErrSchedulingConflict, "complete group membership is required")
	})

	t.Run("BindGroups preserves existing group priority and nil clears all", func(t *testing.T) {
		g1 := mustCreateGroup(t, client, &service.Group{Name: name + "-bind-a"})
		g2 := mustCreateGroup(t, client, &service.Group{Name: name + "-bind-b"})
		a := mustCreateAccount(t, client, &service.Account{Name: name + "-bind-account", Priority: 23})
		require.NoError(t, accounts.AddToGroup(ctx, a.ID, g1.ID, 111))
		require.NoError(t, accounts.BindGroups(ctx, a.ID, []int64{g2.ID, g1.ID}))
		bound, err := accounts.GetByID(ctx, a.ID)
		require.NoError(t, err)
		priorities := make(map[int64]int, len(bound.AccountGroups))
		for _, membership := range bound.AccountGroups {
			priorities[membership.GroupID] = membership.Priority
		}
		require.Equal(t, map[int64]int{g1.ID: 111, g2.ID: 23}, priorities)
		require.NoError(t, accounts.BindGroups(ctx, a.ID, nil))
		bound, err = accounts.GetByID(ctx, a.ID)
		require.NoError(t, err)
		require.Empty(t, bound.GroupIDs)
		require.Empty(t, bound.AccountGroups)
	})

	t.Run("full-group recovery requires every eligible member paused and exemption is fixed", func(t *testing.T) {
		g := mustCreateGroup(t, client, &service.Group{Name: name + "-recovery"})
		a1 := mustCreateAccount(t, client, &service.Account{Name: name + "-recovery-a", Extra: map[string]any{service.SlowTTFTConfigKey: cfg}})
		a2 := mustCreateAccount(t, client, &service.Account{Name: name + "-recovery-b", Extra: map[string]any{service.SlowTTFTConfigKey: cfg}})
		mustBindAccountToGroup(t, client, a1.ID, g.ID, 1)
		mustBindAccountToGroup(t, client, a2.ID, g.ID, 1)
		_, err := integrationDB.ExecContext(ctx, `UPDATE accounts SET slow_ttft_until=NOW()+INTERVAL '1 hour',slow_ttft_reason='consecutive' WHERE id=$1`, a1.ID)
		require.NoError(t, err)
		_, recovered, err := accounts.RecoverSlowTTFTGroup(ctx, g.ID)
		require.NoError(t, err)
		require.False(t, recovered, "a group with a partially NULL pause must not recover")
		_, err = integrationDB.ExecContext(ctx, `UPDATE accounts SET slow_ttft_until=NOW()+INTERVAL '1 hour',slow_ttft_reason='window' WHERE id=$1`, a2.ID)
		require.NoError(t, err)
		until, recovered, err := accounts.RecoverSlowTTFTGroup(ctx, g.ID)
		require.NoError(t, err)
		require.True(t, recovered)
		require.NotNil(t, until)
		require.WithinDuration(t, time.Now().Add(30*time.Minute), *until, time.Second)
		_, recoveredAgain, err := accounts.RecoverSlowTTFTGroup(ctx, g.ID)
		require.NoError(t, err)
		require.False(t, recoveredAgain, "recovery exemption must not extend on repeated checks")
		var stored sql.NullTime
		require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT slow_ttft_exempt_until FROM groups WHERE id=$1`, g.ID).Scan(&stored))
		require.True(t, stored.Valid)
		require.WithinDuration(t, *until, stored.Time, time.Second)
	})
	t.Run("Redis scheduled acquisition respects capacity under concurrency and releases all slots", func(t *testing.T) {
		rdb := testRedis(t)
		concurrency := NewConcurrencyCache(rdb, 1, 1).(*concurrencyCache)
		candidates := []service.SchedulingCandidate{
			{ID: 10, Priority: 10, LoadFactor: 1, Concurrency: 4},
			{ID: 9, Priority: 9, LoadFactor: 1, Concurrency: 4},
			{ID: 8, Priority: 9, LoadFactor: 1, Concurrency: 4},
		}

		const requests = 24
		type acquisition struct {
			id        int64
			requestID string
			err       error
		}
		results := make(chan acquisition, requests)
		start := make(chan struct{})
		var workers sync.WaitGroup
		for i := range requests {
			workers.Add(1)
			go func(i int) {
				defer workers.Done()
				requestID := fmt.Sprintf("parallel-%d", i)
				<-start
				id, err := concurrency.AcquireScheduledAccount(ctx, candidates, requestID)
				results <- acquisition{id: id, requestID: requestID, err: err}
			}(i)
		}
		close(start)
		workers.Wait()
		close(results)

		successes := make([]acquisition, 0, 12)
		counts := map[int64]int{}
		for got := range results {
			require.NoError(t, got.err)
			if got.id == 0 {
				continue
			}
			require.Contains(t, []int64{10, 9, 8}, got.id)
			successes = append(successes, got)
			counts[got.id]++
		}
		require.Len(t, successes, 12, "requests exceed the combined capacity of the three candidates")
		for _, id := range []int64{10, 9, 8} {
			require.LessOrEqual(t, counts[id], 4, "account %d exceeded its concurrency capacity", id)
		}
		for _, got := range successes {
			require.NoError(t, concurrency.ReleaseAccountSlot(ctx, got.id, got.requestID))
		}
		for _, id := range []int64{10, 9, 8} {
			count, err := concurrency.GetAccountConcurrency(ctx, id)
			require.NoError(t, err)
			require.Zero(t, count, "all slots for account %d should be released", id)
		}
	})

	t.Run("Redis scheduled selection uses load factor within one priority", func(t *testing.T) {
		rdb := testRedis(t)
		concurrency := NewConcurrencyCache(rdb, 1, 1).(*concurrencyCache)
		candidates := []service.SchedulingCandidate{
			{ID: 10, Priority: 1, LoadFactor: 10, Concurrency: 4},
			{ID: 9, Priority: 1, LoadFactor: 9, Concurrency: 4},
			{ID: 8, Priority: 1, LoadFactor: 9, Concurrency: 4},
		}

		first, err := concurrency.AcquireScheduledAccount(ctx, candidates, "priority-first")
		require.NoError(t, err)
		require.Equal(t, int64(10), first)
		second, err := concurrency.AcquireScheduledAccount(ctx, candidates, "priority-second")
		require.NoError(t, err)
		require.Contains(t, []int64{9, 8}, second)
		require.NoError(t, concurrency.ReleaseAccountSlot(ctx, first, "priority-first"))
		require.NoError(t, concurrency.ReleaseAccountSlot(ctx, second, "priority-second"))
	})

	t.Run("Redis SlowTTFT concurrent same-epoch triggers do not extend pause", func(t *testing.T) {
		rdb := testRedis(t)
		firstCache := &tempUnschedCache{rdb: rdb}
		secondCache := &tempUnschedCache{rdb: rdb}
		observeCaches := []*tempUnschedCache{firstCache, secondCache}
		slowCfg := service.DefaultSlowTTFTConfig()
		slowCfg.Generation = name + "-redis-epoch"
		slowCfg.ConsecutiveCount = 1
		slowCfg.WindowSeconds = 60
		slowCfg.WindowCount = 20
		slowCfg.PauseSeconds = 120

		const observations = 12
		type observationResult struct {
			observation service.SlowTTFTObservation
			err         error
		}
		results := make(chan observationResult, observations)
		start := make(chan struct{})
		var workers sync.WaitGroup
		for i := range observations {
			workers.Add(1)
			go func(i int) {
				defer workers.Done()
				<-start
				observation, err := observeCaches[i%len(observeCaches)].ObserveSlowTTFT(
					ctx, 9001, slowCfg, fmt.Sprintf("attempt-%d", i), true,
				)
				results <- observationResult{observation: observation, err: err}
			}(i)
		}
		close(start)
		workers.Wait()
		close(results)

		var until time.Time
		for got := range results {
			require.NoError(t, got.err)
			require.True(t, got.observation.Tripped)
			if until.IsZero() {
				until = got.observation.Until
				require.False(t, until.IsZero())
			}
			require.Equal(t, until, got.observation.Until, "concurrent same-epoch observations must retain the first pause deadline")
		}
		require.NoError(t, firstCache.rdb.Ping(ctx).Err())
	})

	t.Run("concurrent group recovery commits one fixed exemption and leaves other groups untouched", func(t *testing.T) {
		group := mustCreateGroup(t, client, &service.Group{Name: name + "-concurrent-recovery"})
		otherGroup := mustCreateGroup(t, client, &service.Group{Name: name + "-no-recovery"})
		account := mustCreateAccount(t, client, &service.Account{Name: name + "-concurrent-recovery-member", Extra: map[string]any{service.SlowTTFTConfigKey: cfg}})
		mustBindAccountToGroup(t, client, account.ID, group.ID, 1)
		_, err := integrationDB.ExecContext(ctx, `UPDATE accounts SET slow_ttft_until=NOW()+INTERVAL '1 hour',slow_ttft_reason='window' WHERE id=$1`, account.ID)
		require.NoError(t, err)

		const workersCount = 12
		type recoveryResult struct {
			until     *time.Time
			recovered bool
			err       error
		}
		results := make(chan recoveryResult, workersCount)
		start := make(chan struct{})
		var workers sync.WaitGroup
		for range workersCount {
			workers.Add(1)
			go func() {
				defer workers.Done()
				<-start
				until, recovered, err := accounts.RecoverSlowTTFTGroup(ctx, group.ID)
				results <- recoveryResult{until: until, recovered: recovered, err: err}
			}()
		}
		close(start)
		workers.Wait()
		close(results)

		recoveredCount := 0
		var fixedUntil *time.Time
		for got := range results {
			require.NoError(t, got.err)
			if got.recovered {
				recoveredCount++
			}
			if got.until == nil {
				t.Errorf("concurrent recovery caller did not observe the committed exemption deadline")
				continue
			}
			if fixedUntil == nil {
				fixedUntil = got.until
			} else {
				require.Equal(t, *fixedUntil, *got.until, "all concurrent recovery callers must observe the same exemption deadline")
			}
		}
		require.Equal(t, 1, recoveredCount, "only one transaction may report the recovery as newly applied")
		stored, err := accounts.SlowTTFTGroupExemption(ctx, group.ID)
		require.NoError(t, err)
		require.NotNil(t, stored)
		require.Equal(t, *fixedUntil, *stored)

		otherUntil, otherRecovered, err := accounts.RecoverSlowTTFTGroup(ctx, otherGroup.ID)
		require.NoError(t, err)
		require.False(t, otherRecovered)
		require.Nil(t, otherUntil)
		storedOther, err := accounts.SlowTTFTGroupExemption(ctx, otherGroup.ID)
		require.NoError(t, err)
		require.Nil(t, storedOther, "a group without a full paused outage must not receive an exemption")
	})
}

func cfgWithGeneration(cfg service.SlowTTFTConfig, generation string) service.SlowTTFTConfig {
	cfg.Generation = generation
	return cfg
}
