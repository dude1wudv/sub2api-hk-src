package migrations

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMigration175CreatesIdempotentOnePurchasePolicy(t *testing.T) {
	content, err := FS.ReadFile("175_subscription_plan_one_purchase.sql")
	require.NoError(t, err)

	sql := string(content)
	require.Contains(t, sql, "ADD COLUMN IF NOT EXISTS one_purchase_per_user BOOLEAN NOT NULL DEFAULT FALSE")
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS subscription_purchase_claims")
	require.Contains(t, sql, "UNIQUE (user_id, subscription_group_id)")
	require.Contains(t, sql, "CREATE INDEX IF NOT EXISTS subscription_purchase_claims_status")
	require.Contains(t, sql, "CREATE INDEX IF NOT EXISTS payment_orders_subscription_purchase_history")
	require.True(t, strings.Contains(sql, "WHERE order_type = 'subscription' AND paid_at IS NOT NULL"))
}