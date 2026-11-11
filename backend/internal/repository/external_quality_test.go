package repository

import (
	"context"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestExternalQualityImportIsAtomicAndDeduplicates(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "deduplicate", true: "rollback"}[fail], func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer func() { _ = db.Close() }()
			repo := &channelMonitorV2Repository{db: db}
			records := make([]service.ExternalQualityRecord, 2)
			mock.ExpectBegin()
			mock.ExpectExec("LOCK TABLE external_quality_history").WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectExec("INSERT INTO external_quality_history").WillReturnResult(sqlmock.NewResult(1, 1))
			if fail {
				mock.ExpectExec("INSERT INTO external_quality_history").WillReturnError(errors.New("insert failed"))
				mock.ExpectRollback()
			} else {
				mock.ExpectExec("INSERT INTO external_quality_history").WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectExec(regexp.QuoteMeta(externalQualityPruneSQL)).WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectCommit()
			}
			count, err := repo.ImportExternalQuality(context.Background(), records)
			if fail {
				require.Error(t, err)
				require.Zero(t, count)
			} else {
				require.NoError(t, err)
				require.Equal(t, 1, count)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestExternalQualityPeriodicPruneIsBoundedAndAtomic(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	mock.ExpectBegin()
	mock.ExpectExec("LOCK TABLE external_quality_history").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`(?s)DELETE FROM external_quality_history.*PARTITION BY group_id,model_id.*WHERE rank > 10`).WillReturnResult(sqlmock.NewResult(0, 12))
	mock.ExpectCommit()
	require.NoError(t, (&channelMonitorV2Repository{db: db}).PruneExternalQuality(context.Background()))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestExternalQualityArtworkEnforcesBothGroupAndEventID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	repo := &channelMonitorV2Repository{db: db}
	query := regexp.QuoteMeta("SELECT artwork FROM external_quality_history WHERE group_id=$1 AND id=$2 AND status='completed'")
	mock.ExpectQuery(query).WithArgs(int64(237), int64(12)).WillReturnRows(sqlmock.NewRows([]string{"artwork"}))
	_, err = repo.GetExternalQualityArtwork(context.Background(), 237, 12)
	require.ErrorIs(t, err, service.ErrExternalQualityNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestExternalQualityListReturnsOnlyPublicMetadataWithPerModelBound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = db.Close() }()
	repo := &channelMonitorV2Repository{db: db}
	now := time.Now().UTC()
	mock.ExpectQuery(`(?s)SELECT id,model_id,response_model,status,started_at,finished_at,duration_ms,has_artwork,imported_at.*PARTITION BY model_id.*WHERE group_id=\$1.*WHERE rank <= 10.*LIMIT 20`).WithArgs(int64(72)).WillReturnRows(sqlmock.NewRows([]string{"id", "model_id", "response_model", "status", "started_at", "finished_at", "duration_ms", "has_artwork", "imported_at"}).AddRow(12, "gpt-6-astra", "gpt-6-astra", "completed", now, now, 100, true, now))
	events, err := repo.ListExternalQuality(context.Background(), 72)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Nil(t, events[0].Score)
	require.NoError(t, mock.ExpectationsWereMet())
}
