//go:build integration

package repository

import (
	"context"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestUsagePanelsTimezoneOwnershipActiveUsersAndFinished(t *testing.T) {
	ctx := context.Background()
	tx := testEntTx(t)
	client := tx.Client()
	repo := newUsageLogRepositoryWithSQL(client, tx)
	u1 := mustCreateUser(t, client, &service.User{Email: uuid.NewString() + "@example.com"})
	u2 := mustCreateUser(t, client, &service.User{Email: uuid.NewString() + "@example.com"})
	k1 := mustCreateApiKey(t, client, &service.APIKey{UserID: u1.ID, Key: uuid.NewString(), Name: "panel1"})
	k2 := mustCreateApiKey(t, client, &service.APIKey{UserID: u2.ID, Key: uuid.NewString(), Name: "panel2"})
	a := mustCreateAccount(t, client, &service.Account{Name: uuid.NewString()})
	g := mustCreateGroup(t, client, &service.Group{Name: uuid.NewString()})
	zone, err := time.LoadLocation("America/Denver")
	require.NoError(t, err)
	start := time.Date(2026, 3, 8, 0, 0, 0, 0, zone)
	end := start.AddDate(0, 0, 1)
	require.Equal(t, 23*time.Hour, end.Sub(start))
	requestID := uuid.NewString()
	insert := func(user, key int64, at time.Time, id string) {
		row := &service.UsageLog{UserID: user, APIKeyID: key, AccountID: a.ID, GroupID: &g.ID, RequestID: id, Model: "synthetic-panel", InputTokens: 3, OutputTokens: 5, CacheReadTokens: 7, CacheCreationTokens: 11, TotalCost: 2, ActualCost: 1, CreatedAt: at}
		ok, err := repo.createSingle(ctx, tx, row)
		require.NoError(t, err)
		require.True(t, ok)
	}
	insert(u1.ID, k1.ID, start.Add(time.Hour), requestID)
	insert(u1.ID, k1.ID, end.Add(-time.Second), uuid.NewString())
	insert(u1.ID, k1.ID, end, uuid.NewString()) // exclusive end
	insert(u2.ID, k2.ID, start.Add(2*time.Hour), uuid.NewString())
	rows, err := repo.GetAPIKeyDailyUsageInTimezone(ctx, u1.ID, k1.ID, start, end, zone.String())
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "2026-03-08", rows[0].Date)
	require.Equal(t, int64(2), rows[0].Requests)
	require.Equal(t, int64(52), rows[0].TotalTokens)
	require.Equal(t, 4.0, rows[0].Cost)
	require.Equal(t, 2.0, rows[0].ActualCost)
	rows, err = repo.GetAPIKeyDailyUsageInTimezone(ctx, u2.ID, k1.ID, start, end, zone.String())
	require.NoError(t, err)
	require.Empty(t, rows)
	now := time.Now().UTC()
	insert(u1.ID, k1.ID, now.Add(-time.Minute), uuid.NewString())
	insert(u1.ID, k1.ID, now.Add(-2*time.Minute), uuid.NewString())
	insert(u2.ID, k2.ID, now.Add(-3*time.Minute), uuid.NewString())
	insert(u2.ID, k2.ID, now.Add(-6*time.Minute), uuid.NewString())
	counts, err := repo.GetGroupActiveUserCounts(ctx, []int64{g.ID}, now.Add(-5*time.Minute))
	require.NoError(t, err)
	require.Equal(t, 2, counts[g.ID])
	done, err := repo.FinishedInflightRequests(ctx, []service.UsageInflightSnapshot{{UserID: u1.ID, RequestID: requestID, StartedAt: start}, {UserID: u2.ID, RequestID: requestID, StartedAt: start}})
	require.NoError(t, err)
	require.Contains(t, done, service.InflightFinishedKey(u1.ID, requestID))
	require.NotContains(t, done, service.InflightFinishedKey(u2.ID, requestID))
}
