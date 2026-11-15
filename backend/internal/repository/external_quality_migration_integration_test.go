//go:build integration

package repository

import (
	"context"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
)

func TestExternalQualityEnterpriseSourceMigration(t *testing.T) {
	tx, err := integrationDB.BeginTx(context.Background(), nil)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()
	original, err := migrations.FS.ReadFile("337_external_quality_history.sql")
	require.NoError(t, err)
	// A temporary table shadows production-shaped fixtures for this connection.
	_, err = tx.ExecContext(context.Background(), strings.Replace(string(original),
		"CREATE TABLE IF NOT EXISTS external_quality_history", "CREATE TEMP TABLE external_quality_history", 1))
	require.NoError(t, err)
	insert := `INSERT INTO external_quality_history
		(source,source_event_id,remote_group_id,group_id,model_id,status,started_at,finished_at,duration_ms,content_sha256,imported_by)
		VALUES ('mdkj',$1,$2,$3,'gpt-6-astra','completed','2026-11-14T00:00:00Z','2026-11-14T00:01:00Z',60000,'original',42)`
	_, err = tx.ExecContext(context.Background(), insert, "00000000-0000-0000-0000-000000000001", 170, 237)
	require.NoError(t, err)
	migration, err := migrations.FS.ReadFile("348_external_quality_enterprise_source.sql")
	require.NoError(t, err)
	for range 2 {
		_, err = tx.ExecContext(context.Background(), string(migration))
		require.NoError(t, err)
	}
	var source int64
	var digest string
	err = tx.QueryRowContext(context.Background(), `SELECT remote_group_id,content_sha256 FROM external_quality_history WHERE group_id=237`).Scan(&source, &digest)
	require.NoError(t, err)
	require.EqualValues(t, 170, source)
	require.Equal(t, "original", digest)
	for index, pair := range [][2]int64{{237, 142}, {72, 62}} {
		id := []string{"00000000-0000-0000-0000-000000000002", "00000000-0000-0000-0000-000000000003"}[index]
		_, err = tx.ExecContext(context.Background(), insert, id, pair[1], pair[0])
		require.NoError(t, err)
	}
	for _, pair := range [][2]int64{{237, 62}, {72, 142}, {179, 142}} {
		_, err = tx.ExecContext(context.Background(), "SAVEPOINT invalid_binding")
		require.NoError(t, err)
		_, err = tx.ExecContext(context.Background(), insert, "00000000-0000-0000-0000-000000000004", pair[1], pair[0])
		require.ErrorContains(t, err, "external_quality_history_binding")
		_, err = tx.ExecContext(context.Background(), "ROLLBACK TO SAVEPOINT invalid_binding")
		require.NoError(t, err)
	}
}
