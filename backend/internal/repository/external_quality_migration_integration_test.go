//go:build integration

package repository

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/google/uuid"
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

func TestExternalQualityGPT62MigrationAndRetention(t *testing.T) {
	ctx := t.Context()
	db, err := sql.Open("postgres", integrationDSN)
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	db.SetMaxOpenConns(1)
	repo := &channelMonitorV2Repository{db: db}

	for _, name := range []string{"337_external_quality_history.sql", "348_external_quality_enterprise_source.sql"} {
		content, err := migrations.FS.ReadFile(name)
		require.NoError(t, err)
		_, err = db.ExecContext(ctx, strings.Replace(string(content),
			"CREATE TABLE IF NOT EXISTS external_quality_history", "CREATE TEMP TABLE external_quality_history", 1))
		require.NoError(t, err)
	}
	originalTime := time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC)
	legacyID := uuid.NewString()
	_, err = db.ExecContext(ctx, `INSERT INTO external_quality_history
		(source,source_event_id,remote_group_id,group_id,model_id,status,started_at,finished_at,duration_ms,content_sha256,imported_by)
		VALUES ('mdkj',$1,170,237,'gpt-6-astra','completed',$2,$2,0,'original',42)`, legacyID, originalTime)
	require.NoError(t, err)
	migration, err := migrations.FS.ReadFile("350_external_quality_gpt62_models.sql")
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, string(migration))
	require.NoError(t, err)

	var legacyRemoteID int64
	var legacyTime time.Time
	err = db.QueryRowContext(ctx, `SELECT remote_group_id,started_at FROM external_quality_history
		WHERE source='mdkj' AND source_event_id=$1 AND group_id=237`, legacyID).Scan(&legacyRemoteID, &legacyTime)
	require.NoError(t, err)
	require.EqualValues(t, 170, legacyRemoteID)
	require.True(t, legacyTime.Equal(originalTime))
	_, err = db.ExecContext(ctx, `UPDATE external_quality_history SET model_id='gpt-6.2-sol-preview'`)
	require.ErrorContains(t, err, "external_quality_history_model")

	models := []string{"gpt-6-astra", "gpt-6.1-sol", "gpt-6.2-astra", "gpt-6.2-sol"}
	for _, model := range models {
		var records []service.ExternalQualityRecord
		for sample := range 11 {
			started := originalTime.Add(time.Duration(sample) * time.Minute)
			records = append(records, service.ExternalQualityRecord{
				Source: "mdkj", SourceEventID: uuid.NewString(), GroupID: 72, RemoteGroupID: 62, ImportedBy: 42,
				Event: service.ExternalQualityRemoteEvent{
					Model: model, ResponseModel: model, Status: "completed", StartedAt: started, FinishedAt: started,
				},
			})
		}
		count, err := repo.ImportExternalQuality(ctx, records)
		require.NoError(t, err)
		require.Equal(t, 11, count)
	}

	events, err := repo.ListExternalQuality(ctx, 72)
	require.NoError(t, err)
	require.Len(t, events, 40)
	counts := make(map[string]int)
	for _, event := range events {
		require.Equal(t, event.ModelID, event.ResponseModel)
		require.True(t, event.CreatedAt.Equal(originalTime.Add(time.Duration(10-counts[event.ModelID])*time.Minute)))
		counts[event.ModelID]++
	}
	for _, model := range models {
		require.Equal(t, 10, counts[model], model)
	}
	var retained int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT count(*) FROM external_quality_history WHERE group_id=72`).Scan(&retained))
	require.Equal(t, 40, retained)
}
