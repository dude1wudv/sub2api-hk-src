//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUsageLogServiceTierObservationRoundTrip(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := newUsageLogRepositoryWithSQL(client, integrationDB)
	user := mustCreateUser(t, client, &service.User{Email: "tier-observation-" + uuid.NewString() + "@example.com"})
	key := mustCreateApiKey(t, client, &service.APIKey{UserID: user.ID, Key: "sk-test-tier-" + uuid.NewString(), Name: "tier-test"})
	account := mustCreateAccount(t, client, &service.Account{Name: "tier-observation-" + uuid.NewString()})
	str := func(v string) *string { return &v }
	boolean := func(v bool) *bool { return &v }
	for _, tc := range []struct {
		name                        string
		requested, upstream, billed *string
		downgraded                  *bool
	}{
		{name: "forced downgrade", requested: str("ultrafast"), upstream: str("default"), billed: str("ultrafast"), downgraded: boolean(true)},
		{name: "honored", requested: str("priority"), upstream: str("priority"), billed: str("priority"), downgraded: boolean(false)},
		{name: "missing upstream", requested: str("priority"), billed: str("priority")},
		{name: "legacy null"},
	} {
		for _, path := range []string{"single", "batch", "best-effort"} {
			t.Run(tc.name+"/"+path, func(t *testing.T) {
				log := &service.UsageLog{UserID: user.ID, APIKeyID: key.ID, AccountID: account.ID, RequestID: uuid.NewString(), Model: "gpt-6.1-sol", CreatedAt: time.Now().UTC(), ServiceTier: tc.billed, RequestedServiceTier: tc.requested, UpstreamServiceTier: tc.upstream, ServiceTierDowngraded: tc.downgraded}
				switch path {
				case "single":
					log.RequestID = ""
					inserted, err := repo.Create(ctx, log)
					require.NoError(t, err)
					require.True(t, inserted)
				case "batch":
					inserted, err := repo.Create(ctx, log)
					require.NoError(t, err)
					require.True(t, inserted)
				case "best-effort":
					require.NoError(t, repo.CreateBestEffort(ctx, log))
					require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT id FROM usage_logs WHERE api_key_id=$1 AND request_id=$2`, key.ID, log.RequestID).Scan(&log.ID))
				}
				stored, err := repo.GetByID(ctx, log.ID)
				require.NoError(t, err)
				require.Equal(t, tc.billed, stored.ServiceTier)
				require.Equal(t, tc.requested, stored.RequestedServiceTier)
				require.Equal(t, tc.upstream, stored.UpstreamServiceTier)
				require.Equal(t, tc.downgraded, stored.ServiceTierDowngraded)
			})
		}
	}
	// A legacy INSERT that never names these columns must remain unknown, not false.
	var historicalID int64
	require.NoError(t, integrationDB.QueryRowContext(ctx, `INSERT INTO usage_logs (user_id, api_key_id, account_id, model) VALUES ($1,$2,$3,'gpt-6.1-sol') RETURNING id`, user.ID, key.ID, account.ID).Scan(&historicalID))
	historical, err := repo.GetByID(ctx, historicalID)
	require.NoError(t, err)
	require.Nil(t, historical.RequestedServiceTier)
	require.Nil(t, historical.UpstreamServiceTier)
	require.Nil(t, historical.ServiceTierDowngraded)
}
