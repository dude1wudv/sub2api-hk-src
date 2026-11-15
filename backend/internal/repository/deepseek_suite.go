package repository

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

var _ service.DeepSeekSuiteRepository = (*usageLogRepository)(nil)

const deepSeekOperationColumns = `id::text,user_id,api_key_id,group_id,account_id,kind,status,
remote_id,upstream_name,filename,purpose,bytes,metadata,receipt,created_at,updated_at`

func (r *usageLogRepository) CreateDeepSeekOperation(ctx context.Context, o *service.DeepSeekSuiteOperation) error {
	_, err := r.sql.ExecContext(ctx, `INSERT INTO deepseek_suite_operations
	(id,user_id,api_key_id,group_id,account_id,kind,status,created_at,updated_at)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$8)`, o.ID, o.UserID, o.KeyID, o.GroupID, o.AccountID, o.Kind, o.Status, o.CreatedAt)
	return err
}

func suiteJSON(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	return string(raw)
}

func (r *usageLogRepository) SaveDeepSeekOperation(ctx context.Context, o *service.DeepSeekSuiteOperation) error {
	result, err := r.sql.ExecContext(ctx, `UPDATE deepseek_suite_operations SET
	status=$2,remote_id=$3,upstream_name=$4,filename=$5,purpose=$6,bytes=$7,
metadata=CASE WHEN metadata->>'deleted'='true' THEN COALESCE($8::jsonb,'{}'::jsonb)||'{"deleted":true}'::jsonb ELSE $8::jsonb END,
receipt=$9::jsonb,updated_at=NOW()
	WHERE id=$1 AND user_id=$10 AND group_id=$11 AND account_id=$12
	AND (status NOT IN ('settled','deleted') OR status=$2 OR $2='deleted')`,
		o.ID, o.Status, o.RemoteID, o.UpstreamName, o.Filename, o.Purpose, o.Bytes,
		suiteJSON(o.Metadata), suiteJSON(o.Receipt), o.UserID, o.GroupID, o.AccountID)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err == nil && count != 1 {
		return service.ErrChatInvalid
	}
	return err
}

func (r *usageLogRepository) MarkDeepSeekFileDeleted(ctx context.Context, userID, groupID int64, id string) error {
	_, err := r.sql.ExecContext(ctx, `UPDATE deepseek_suite_operations
	SET metadata=COALESCE(metadata,'{}'::jsonb)||'{"deleted":true}'::jsonb,updated_at=NOW()
	WHERE id::text=$1 AND user_id=$2 AND group_id=$3 AND kind='files'`, id, userID, groupID)
	return err
}

func scanDeepSeekOperation(row interface{ Scan(...any) error }) (*service.DeepSeekSuiteOperation, error) {
	o := &service.DeepSeekSuiteOperation{}
	var metadata, receipt []byte
	err := row.Scan(&o.ID, &o.UserID, &o.KeyID, &o.GroupID, &o.AccountID, &o.Kind, &o.Status,
		&o.RemoteID, &o.UpstreamName, &o.Filename, &o.Purpose, &o.Bytes, &metadata, &receipt, &o.CreatedAt, &o.UpdatedAt)
	o.Metadata, o.Receipt = json.RawMessage(metadata), json.RawMessage(receipt)
	if err == sql.ErrNoRows {
		return nil, service.ErrChatNotFound
	}
	return o, err
}

func (r *usageLogRepository) GetDeepSeekFile(ctx context.Context, userID, groupID int64, id string) (*service.DeepSeekSuiteOperation, error) {
	rows, err := r.sql.QueryContext(ctx, `SELECT `+deepSeekOperationColumns+` FROM deepseek_suite_operations
	WHERE id::text=$1 AND user_id=$2 AND group_id=$3 AND kind='files'
	AND remote_id<>'' AND status NOT IN ('deleted','failed') AND COALESCE(metadata->>'deleted','false')<>'true'`, id, userID, groupID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, service.ErrChatNotFound
	}
	return scanDeepSeekOperation(rows)
}

func (r *usageLogRepository) ListDeepSeekFiles(ctx context.Context, userID, groupID int64, query service.DeepSeekFileListQuery) ([]*service.DeepSeekSuiteOperation, error) {
	if query.Limit < 1 || query.Limit > 10000 || (query.After != "" && query.Before != "") {
		return nil, service.ErrChatInvalid
	}
	after, before, order := "<", ">", "DESC"
	if query.Ascending {
		after, before, order = ">", "<", "ASC"
	}
	if query.Before != "" {
		if order == "ASC" {
			order = "DESC"
		} else {
			order = "ASC"
		}
	}
	return r.deepSeekOperations(ctx, `WHERE user_id=$1 AND group_id=$2 AND kind='files'
	AND remote_id<>'' AND status NOT IN ('deleted','failed') AND COALESCE(metadata->>'deleted','false')<>'true'
	AND ($3='' OR purpose=$3)
	AND ($4='' OR (created_at,id) `+after+` (SELECT created_at,id FROM deepseek_suite_operations WHERE id::text=$4 AND user_id=$1 AND group_id=$2))
	AND ($5='' OR (created_at,id) `+before+` (SELECT created_at,id FROM deepseek_suite_operations WHERE id::text=$5 AND user_id=$1 AND group_id=$2))
	ORDER BY created_at `+order+`,id `+order+` LIMIT $6`,
		userID, groupID, query.Purpose, query.After, query.Before, query.Limit+1)
}

func (r *usageLogRepository) ListDeepSeekPending(ctx context.Context) ([]*service.DeepSeekSuiteOperation, error) {
	return r.deepSeekOperations(ctx, `WHERE (status='settling' AND receipt IS NOT NULL)
	OR (kind='files' AND upstream_name<>'' AND status IN ('started','unknown','awaiting_meter') AND updated_at<NOW()-INTERVAL '12 minutes')
	ORDER BY CASE WHEN status='settling' THEN 0 ELSE 1 END,updated_at,id LIMIT 100`)
}

func (r *usageLogRepository) deepSeekOperations(ctx context.Context, where string, args ...any) ([]*service.DeepSeekSuiteOperation, error) {
	rows, err := r.sql.QueryContext(ctx, `SELECT `+deepSeekOperationColumns+` FROM deepseek_suite_operations `+where, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []*service.DeepSeekSuiteOperation{}
	for rows.Next() {
		record, err := scanDeepSeekOperation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	return out, rows.Err()
}
