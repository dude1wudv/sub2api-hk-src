//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/ent/group"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestGroupRateHistoryAllWritersAndLaunchSnapshot(t *testing.T) {
	ctx := context.Background()
	tx := testEntTx(t)
	repo := newGroupRepositoryWithSQL(tx.Client(), tx)
	at := time.Now().UTC().Truncate(time.Microsecond)
	g := &service.Group{Name: "rate-history-" + uuid.NewString(), Platform: service.PlatformOpenAI, RateMultiplier: 2, Status: service.StatusActive, SubscriptionType: service.SubscriptionTypeStandard, LaunchedAt: &at}
	require.NoError(t, repo.Create(ctx, g))
	loaded, err := repo.GetByID(ctx, g.ID)
	require.NoError(t, err)
	require.Equal(t, 2.0, *loaded.LaunchRate)
	require.WithinDuration(t, at, *loaded.LaunchedAt, time.Microsecond)
	changes, err := repo.GetRecentRateChanges(ctx, []int64{g.ID}, time.Now().Add(-48*time.Hour))
	require.NoError(t, err)
	require.Empty(t, changes, "initial insert is not a rate change")

	// An arbitrary SQL writer gets the same audit trail as the Ent repository.
	_, err = tx.ExecContext(ctx, "UPDATE groups SET rate_multiplier=1.5 WHERE id=$1", g.ID)
	require.NoError(t, err)
	changes, err = repo.GetRecentRateChanges(ctx, []int64{g.ID}, time.Now().Add(-48*time.Hour))
	require.NoError(t, err)
	require.Equal(t, 2.0, changes[g.ID].OldRate)
	require.Equal(t, 1.5, changes[g.ID].NewRate)
	loaded, err = repo.GetByID(ctx, g.ID)
	require.NoError(t, err)
	require.Equal(t, 2.0, *loaded.LaunchRate, "rate edits do not change launch snapshot")
	loaded.RateMultiplier = 1.25
	require.NoError(t, repo.Update(ctx, loaded))
	// Same-value and unrelated edits must not create extra history rows.
	_, err = tx.ExecContext(ctx, "UPDATE groups SET rate_multiplier=1.25, name=name WHERE id=$1", g.ID)
	require.NoError(t, err)
	rows, err := tx.QueryContext(ctx, "SELECT COUNT(*) FROM group_rate_history WHERE group_id=$1", g.ID)
	require.NoError(t, err)
	require.True(t, rows.Next())
	var count int
	require.NoError(t, rows.Scan(&count))
	require.NoError(t, rows.Close())
	require.Equal(t, 2, count)

	// Exactly 48h is included; the latest row wins deterministically.
	since := time.Now().UTC().Truncate(time.Microsecond).Add(-48 * time.Hour)
	_, err = tx.ExecContext(ctx, "UPDATE group_rate_history SET changed_at=$2 WHERE group_id=$1", g.ID, since)
	require.NoError(t, err)
	changes, err = repo.GetRecentRateChanges(ctx, []int64{g.ID}, since)
	require.NoError(t, err)
	require.Equal(t, 1.25, changes[g.ID].NewRate)
	changes, err = repo.GetRecentRateChanges(ctx, []int64{g.ID}, since.Add(time.Microsecond))
	require.NoError(t, err)
	require.Empty(t, changes)

	_, err = tx.ExecContext(ctx, "UPDATE groups SET launched_at=$2, rate_multiplier=1.1, launch_rate=999 WHERE id=$1", g.ID, at.Add(time.Hour))
	require.NoError(t, err)
	loaded, err = repo.GetByID(ctx, g.ID)
	require.NoError(t, err)
	require.Equal(t, 1.1, *loaded.LaunchRate, "trigger snapshots actual rate even for raw SQL writers")
	_, err = tx.ExecContext(ctx, "UPDATE groups SET launch_rate=999 WHERE id=$1", g.ID)
	require.NoError(t, err)
	loaded, err = repo.GetByID(ctx, g.ID)
	require.NoError(t, err)
	require.Equal(t, 1.1, *loaded.LaunchRate, "ordinary writes cannot rewrite historical snapshot")
	loaded.LaunchedAt = nil
	require.NoError(t, repo.Update(ctx, loaded))
	loaded, err = repo.GetByID(ctx, g.ID)
	require.NoError(t, err)
	require.Nil(t, loaded.LaunchedAt)
	require.Nil(t, loaded.LaunchRate)
}

func TestGroupRateHistoryRollbackLeavesNoRows(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	g, err := client.Group.Create().SetName("rate-rollback-" + uuid.NewString()).SetRateMultiplier(2).Save(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Group.DeleteOneID(g.ID).Exec(ctx)) })
	tx, err := integrationDB.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, "UPDATE groups SET rate_multiplier=1 WHERE id=$1", g.ID)
	require.NoError(t, err)
	var count int
	require.NoError(t, tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM group_rate_history WHERE group_id=$1", g.ID).Scan(&count))
	require.Equal(t, 1, count, "trigger writes in the updating transaction")
	require.NoError(t, tx.Rollback())
	require.NoError(t, integrationDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM group_rate_history WHERE group_id=$1", g.ID).Scan(&count))
	require.Zero(t, count)
	stored, err := client.Group.Query().Where(group.IDEQ(g.ID)).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, 2.0, stored.RateMultiplier)
}
