//go:build integration

package repository

import (
	"context"
	"database/sql"
	"testing"

	dbmigrations "github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
)

func TestChannelUltrafastPricingMigrationPreservesExistingCards(t *testing.T) {
	ctx := context.Background()
	tx := testTx(t)
	var constraint string
	require.NoError(t, tx.QueryRowContext(ctx, `
        SELECT pg_get_constraintdef(oid)
        FROM pg_constraint
        WHERE conrelid = 'channel_model_pricing'::regclass
          AND conname = 'channel_model_pricing_ultrafast_multiplier_positive'`).Scan(&constraint))
	require.Contains(t, constraint, "'NaN'::numeric", "252 must preserve the stricter HK constraint")
	_, err := tx.ExecContext(ctx, `ALTER TABLE channel_model_pricing DROP COLUMN ultrafast_multiplier`)
	require.NoError(t, err)
	var channelID, pricingID int64
	require.NoError(t, tx.QueryRowContext(ctx, `INSERT INTO channels (name) VALUES ('ultrafast-migration') RETURNING id`).Scan(&channelID))
	require.NoError(t, tx.QueryRowContext(ctx, `INSERT INTO channel_model_pricing (channel_id, models, fast_multiplier) VALUES ($1, '["gpt-6-astra"]', 2.5) RETURNING id`, channelID).Scan(&pricingID))
	migration, err := dbmigrations.FS.ReadFile("252_channel_pricing_ultrafast_multiplier.sql")
	require.NoError(t, err)
	for range 2 {
		_, err = tx.ExecContext(ctx, string(migration))
		require.NoError(t, err)
		var ultrafast sql.NullFloat64
		var fast float64
		require.NoError(t, tx.QueryRowContext(ctx, `SELECT ultrafast_multiplier, fast_multiplier FROM channel_model_pricing WHERE id = $1`, pricingID).Scan(&ultrafast, &fast))
		require.False(t, ultrafast.Valid)
		require.Equal(t, 2.5, fast)
	}
	_, err = tx.ExecContext(ctx, `UPDATE channel_model_pricing SET ultrafast_multiplier = 1.25 WHERE id = $1`, pricingID)
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, string(migration))
	require.NoError(t, err)
	var configured float64
	require.NoError(t, tx.QueryRowContext(ctx, `SELECT ultrafast_multiplier FROM channel_model_pricing WHERE id = $1`, pricingID).Scan(&configured))
	require.Equal(t, 1.25, configured)
}
