//go:build integration

package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

const (
	productionHistoryMigrationCount = 279
	candidateMigrationCount         = 292
)

var candidateOnlyMigrations = []string{
	"222_group_usage_daily_rollups.sql",
	"223_group_usage_rollup_timezone.sql",
	"224_user_platform_quotas_add_cn_providers.sql",
	"225_backfill_codex_fingerprint_seed.sql",
	"225_channel_model_time_pricing.sql",
	"226_add_usage_log_effective_model_indexes_notx.sql",
	"226_channel_monitor_quota_mode.sql",
	"227_composite_routes_add_cn_providers.sql",
	"228_channel_pricing_multipliers.sql",
	"229_plugins.sql",
	"230_plugin_artifacts.sql",
	"231_add_usage_log_requested_reasoning_effort.sql",
	"231_user_restrict_public_groups.sql",
}

func TestUpstreamMergeValidation_ProductionHistoryAndFreshSchemaConverge(t *testing.T) {
	ctx := context.Background()
	fullFiles := migrationFiles(t, migrations.FS)
	require.Len(t, fullFiles, candidateMigrationCount)
	for _, name := range candidateOnlyMigrations {
		require.Contains(t, fullFiles, name)
	}

	productionHistory := productionHistoryFS(t, migrations.FS)
	require.Len(t, migrationFiles(t, productionHistory), productionHistoryMigrationCount)

	freshFingerprint := schemaFingerprint(t, integrationDB, "public")
	freshChecksums := migrationChecksums(t, integrationDB, "public")
	require.Len(t, freshChecksums, candidateMigrationCount)
	requireMigrationChecksumsMatchFiles(t, migrations.FS, freshChecksums)

	legacyContainer, err := tcpostgres.Run(
		ctx,
		selectDockerImage(ctx, postgresImageTag),
		tcpostgres.WithDatabase("upstream_merge_history"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		tcpostgres.BasicWaitStrategies(),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = legacyContainer.Terminate(ctx) })

	legacyDSN, err := legacyContainer.ConnectionString(ctx, "sslmode=disable", "TimeZone=UTC")
	require.NoError(t, err)
	legacyDB, err := openSQLWithRetry(ctx, legacyDSN, 30*time.Second)
	require.NoError(t, err)
	t.Cleanup(func() { _ = legacyDB.Close() })

	require.NoError(t, applyMigrationsFS(ctx, legacyDB, productionHistory))
	productionChecksums := migrationChecksums(t, legacyDB, "public")
	require.Len(t, productionChecksums, productionHistoryMigrationCount)
	requireMigrationChecksumsMatchFiles(t, productionHistory, productionChecksums)

	require.NoError(t, applyMigrationsFS(ctx, legacyDB, migrations.FS))
	legacyChecksums := migrationChecksums(t, legacyDB, "public")
	require.Len(t, legacyChecksums, candidateMigrationCount)
	requireMigrationChecksumsMatchFiles(t, migrations.FS, legacyChecksums)
	require.Equal(t, freshFingerprint, schemaFingerprint(t, legacyDB, "public"))

	// A second pass must only close checksums; it must not attempt DDL again.
	require.NoError(t, applyMigrationsFS(ctx, legacyDB, migrations.FS))
	require.Zero(t, invalidIndexCount(t, legacyDB, "public"))
}

func TestUpstreamMergeValidation_RequiredContractsAndLegacyExclusions(t *testing.T) {
	tx := testTx(t)

	for _, table := range []string{
		"composite_model_routes",
		"batch_image_jobs",
		"batch_image_items",
		"batch_image_events",
		"audit_logs",
		"prompt_audit_jobs",
		"prompt_audit_events",
		"payment_orders",
		"subscription_plans",
		"subscription_purchase_claims",
	} {
		requireTable(t, tx, table)
	}

	requireColumn(t, tx, "groups", "allow_live", "boolean", 0, false)
	requireColumn(t, tx, "payment_orders", "subscription_expires_at", "timestamp with time zone", 0, true)
	requireColumn(t, tx, "usage_logs", "session_id", "character varying", 255, true)
	requireColumn(t, tx, "subscription_plans", "purchase_mode", "character varying", 20, false)
	requireColumn(t, tx, "subscription_plans", "fixed_expires_at", "timestamp with time zone", 0, true)
	requireColumn(t, tx, "subscription_plans", "one_purchase_per_user", "boolean", 0, false)
	requireColumn(t, tx, "subscription_plans", "sale_ends_at", "timestamp with time zone", 0, true)
	requireIndex(t, tx, "subscription_plans", "idx_subscription_plans_active_sale_window")
	requireIndex(t, tx, "subscription_purchase_claims", "subscription_purchase_claims_user_group_key")

	// Historical data remains readable, but deleted features have no Ent runtime entity.
	for _, table := range []string{
		"daily_balance_grants",
		"usage_sessions",
		"token_incentive_claims",
	} {
		requireTable(t, tx, table)
	}
	for _, forbidden := range []string{
		"DailyBalanceGrant",
		"TokenIncentiveClaim",
		"UsageSession",
	} {
		require.NotContains(t, handwrittenEntSchemaNames(t), forbidden)
	}
	for _, required := range []string{
		"CompositeModelRoute",
		"BatchImageJob",
		"BatchImageItem",
		"BatchImageEvent",
		"PaymentOrder",
		"SubscriptionPlan",
		"SubscriptionPurchaseClaim",
	} {
		require.Contains(t, handwrittenEntSchemaNames(t), required)
	}
}

func productionHistoryFS(t *testing.T, full fs.FS) fstest.MapFS {
	t.Helper()

	files := fstest.MapFS{}
	for _, name := range migrationFiles(t, full) {
		if slices.Contains(candidateOnlyMigrations, name) {
			continue
		}
		content, err := fs.ReadFile(full, name)
		require.NoError(t, err)
		files[name] = &fstest.MapFile{Data: content}
	}
	return files
}

func migrationFiles(t *testing.T, fsys fs.FS) []string {
	t.Helper()

	files, err := fs.Glob(fsys, "*.sql")
	require.NoError(t, err)
	return files
}

func migrationChecksums(t *testing.T, db *sql.DB, schema string) map[string]string {
	t.Helper()

	rows, err := db.QueryContext(context.Background(), fmt.Sprintf("SELECT filename, checksum FROM %s.schema_migrations ORDER BY filename", schema))
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()

	checksums := map[string]string{}
	for rows.Next() {
		var name, checksum string
		require.NoError(t, rows.Scan(&name, &checksum))
		checksums[name] = checksum
	}
	require.NoError(t, rows.Err())
	return checksums
}

func requireMigrationChecksumsMatchFiles(t *testing.T, fsys fs.FS, checksums map[string]string) {
	t.Helper()

	for _, name := range migrationFiles(t, fsys) {
		content, err := fs.ReadFile(fsys, name)
		require.NoError(t, err)
		sum := sha256.Sum256([]byte(strings.TrimSpace(string(content))))
		require.Equal(t, hex.EncodeToString(sum[:]), checksums[name], "checksum mismatch for %s", name)
	}
}

func schemaFingerprint(t *testing.T, db *sql.DB, schema string) []string {
	t.Helper()

	rows, err := db.QueryContext(context.Background(), `
SELECT fingerprint
FROM (
    SELECT 'table:' || tablename AS fingerprint
    FROM pg_tables
    WHERE schemaname = $1
      AND tablename NOT IN ('schema_migrations', 'atlas_schema_revisions')
    UNION ALL
    SELECT 'column:' || table_name || ':' || column_name || ':' || data_type || ':' ||
           COALESCE(character_maximum_length::text, '') || ':' || is_nullable || ':' ||
           COALESCE(column_default, '')
    FROM information_schema.columns
    WHERE table_schema = $1
      AND table_name NOT IN ('schema_migrations', 'atlas_schema_revisions')
    UNION ALL
    SELECT 'index:' || indexname || ':' || indexdef
    FROM pg_indexes
    WHERE schemaname = $1
      AND tablename NOT IN ('schema_migrations', 'atlas_schema_revisions')
    UNION ALL
    SELECT 'constraint:' || tbl.relname || ':' || con.conname || ':' || pg_get_constraintdef(con.oid)
    FROM pg_constraint con
    JOIN pg_class tbl ON tbl.oid = con.conrelid
    JOIN pg_namespace ns ON ns.oid = tbl.relnamespace
    WHERE ns.nspname = $1
      AND tbl.relname NOT IN ('schema_migrations', 'atlas_schema_revisions')
) fingerprints
ORDER BY fingerprint
`, schema)
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()

	var fingerprint []string
	for rows.Next() {
		var item string
		require.NoError(t, rows.Scan(&item))
		fingerprint = append(fingerprint, item)
	}
	require.NoError(t, rows.Err())
	return fingerprint
}

func invalidIndexCount(t *testing.T, db *sql.DB, schema string) int {
	t.Helper()

	var count int
	err := db.QueryRowContext(context.Background(), `
SELECT COUNT(*)
FROM pg_index i
JOIN pg_class idx ON idx.oid = i.indexrelid
JOIN pg_namespace ns ON ns.oid = idx.relnamespace
WHERE ns.nspname = $1
  AND NOT i.indisvalid
`, schema).Scan(&count)
	require.NoError(t, err)
	return count
}

func requireTable(t *testing.T, tx *sql.Tx, table string) {
	t.Helper()

	var exists bool
	err := tx.QueryRowContext(context.Background(), `
SELECT EXISTS (
    SELECT 1
    FROM information_schema.tables
    WHERE table_schema = 'public' AND table_name = $1
)
`, table).Scan(&exists)
	require.NoError(t, err)
	require.True(t, exists, "expected table %s", table)
}

func handwrittenEntSchemaNames(t *testing.T) []string {
	t.Helper()

	entries, err := fs.ReadDir(os.DirFS("../../ent/schema"), ".")
	require.NoError(t, err)

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		content, err := fs.ReadFile(os.DirFS("../../ent/schema"), name)
		require.NoError(t, err)
		for _, line := range strings.Split(string(content), "\n") {
			if strings.HasPrefix(line, "type ") && strings.HasSuffix(line, " struct {") {
				names = append(names, strings.TrimSuffix(strings.TrimPrefix(line, "type "), " struct {"))
			}
		}
	}
	return names
}
