package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

var _ service.ExternalQualityRepository = (*channelMonitorV2Repository)(nil)

// Bound both retained artwork storage and list work. Keeping the newest sample
// even when stale lets users see its actual timestamp during source outages.
const externalQualityPruneSQL = `DELETE FROM external_quality_history WHERE id IN (
	SELECT id FROM (SELECT id,row_number() OVER (
		PARTITION BY group_id,model_id ORDER BY started_at DESC,id DESC) AS rank
		FROM external_quality_history) history WHERE rank > 10)`

func (r *channelMonitorV2Repository) ImportExternalQuality(ctx context.Context, records []service.ExternalQualityRecord) (int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	// Serialize imports/pruning without blocking normal SELECT readers.
	if _, err := tx.ExecContext(ctx, "LOCK TABLE external_quality_history IN SHARE ROW EXCLUSIVE MODE"); err != nil {
		return 0, err
	}
	inserted := 0
	for _, record := range records {
		e := record.Event
		result, err := tx.ExecContext(ctx, `INSERT INTO external_quality_history
			(source,source_event_id,remote_group_id,group_id,model_id,response_model,status,started_at,finished_at,duration_ms,artwork,content_sha256,imported_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
			ON CONFLICT (source,group_id,source_event_id) DO NOTHING`,
			record.Source, record.SourceEventID, record.RemoteGroupID, record.GroupID, e.Model, e.ResponseModel, e.Status, e.StartedAt, e.FinishedAt, e.DurationMs, e.HTML, record.ContentSHA256, record.ImportedBy)
		if err != nil {
			return 0, err
		}
		count, err := result.RowsAffected()
		if err != nil {
			return 0, err
		}
		inserted += int(count)
	}
	if _, err := tx.ExecContext(ctx, externalQualityPruneSQL); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return inserted, nil
}

func (r *channelMonitorV2Repository) PruneExternalQuality(ctx context.Context) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, "LOCK TABLE external_quality_history IN SHARE ROW EXCLUSIVE MODE"); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, externalQualityPruneSQL); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *channelMonitorV2Repository) ListExternalQuality(ctx context.Context, groupID int64) ([]service.ExternalQualityEvent, error) {
	// The window is per requested model; one busy model cannot crowd out the
	// other. Raw artwork and foreign provenance never enter the public list.
	rows, err := r.db.QueryContext(ctx, `SELECT id,model_id,response_model,status,started_at,finished_at,duration_ms,has_artwork,imported_at
		FROM (SELECT id,model_id,response_model,status,started_at,finished_at,duration_ms,(artwork <> '') AS has_artwork,imported_at,
			row_number() OVER (PARTITION BY model_id ORDER BY started_at DESC,id DESC) AS rank
			FROM external_quality_history WHERE group_id=$1) history
		WHERE rank <= 10 ORDER BY started_at DESC,id DESC LIMIT 20`, groupID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	events := make([]service.ExternalQualityEvent, 0, 20)
	for rows.Next() {
		var event service.ExternalQualityEvent
		if err := rows.Scan(&event.ID, &event.ModelID, &event.ResponseModel, &event.Status, &event.CreatedAt, &event.FinishedAt, &event.DurationMs, &event.HasArtwork, &event.ImportedAt); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (r *channelMonitorV2Repository) GetExternalQualityArtwork(ctx context.Context, groupID, id int64) (string, error) {
	var artwork string
	err := r.db.QueryRowContext(ctx, `SELECT artwork FROM external_quality_history WHERE group_id=$1 AND id=$2 AND status='completed'`, groupID, id).Scan(&artwork)
	if errors.Is(err, sql.ErrNoRows) {
		return "", service.ErrExternalQualityNotFound
	}
	return artwork, err
}
