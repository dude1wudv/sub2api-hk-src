//go:build integration

package repository

import (
	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
	"testing"
)

// Run the exact additive migration over pre-existing bills, not just new rows
// inserted after migration. The temporary table shadows usage_logs only in this
// isolated transaction and is removed by rollback.
func TestInputEstimationMigrationPreservesHistoricalBills(t *testing.T) {
	tx := testTx(t)
	_, err := tx.Exec(`CREATE TEMP TABLE usage_logs (
 id bigint PRIMARY KEY, input_tokens integer NOT NULL, output_tokens integer NOT NULL,
 cache_read_tokens integer NOT NULL, total_cost numeric(20,10) NOT NULL,
 actual_cost numeric(20,10) NOT NULL, created_at timestamptz NOT NULL);
 INSERT INTO usage_logs VALUES (7,123,456,789,1.2345678901,0.1234567890,'2026-01-01T00:00:00Z')`)
	require.NoError(t, err)
	var before, after string
	require.NoError(t, tx.QueryRow(`SELECT row_to_json(u)::text FROM usage_logs u`).Scan(&before))
	migration, err := migrations.FS.ReadFile("341_add_usage_log_input_estimation.sql")
	require.NoError(t, err)
	for i := 0; i < 2; i++ {
		_, err = tx.Exec(string(migration))
		require.NoError(t, err, "migration must be idempotent")
		require.NoError(t, tx.QueryRow(`SELECT row_to_json(u)::text FROM
   (SELECT id,input_tokens,output_tokens,cache_read_tokens,total_cost,actual_cost,created_at FROM usage_logs) u`).Scan(&after))
		require.JSONEq(t, before, after, "historical counters, amounts and identity must remain unchanged")
		var inputEstimated, cacheEstimated bool
		var reason string
		require.NoError(t, tx.QueryRow(`SELECT input_tokens_estimated,cache_read_tokens_estimated,usage_estimation_reason FROM usage_logs WHERE id=7`).Scan(&inputEstimated, &cacheEstimated, &reason))
		require.False(t, inputEstimated)
		require.False(t, cacheEstimated)
		require.Empty(t, reason)
	}
}
