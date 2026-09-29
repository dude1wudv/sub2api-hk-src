package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

func (r *accountRepository) SlowTTFTGroupExemption(ctx context.Context, id int64) (*time.Time, error) {
	var until sql.NullTime
	err := scanSingleRow(ctx, r.sql, `SELECT slow_ttft_exempt_until FROM groups WHERE id=$1 AND deleted_at IS NULL`, []any{id}, &until)
	if err != nil || !until.Valid {
		return nil, err
	}
	return &until.Time, nil
}
func (r *accountRepository) RecoverSlowTTFTGroup(ctx context.Context, id int64) (*time.Time, bool, error) {
	until, err := r.SlowTTFTGroupExemption(ctx, id)
	if err != nil {
		return nil, false, err
	}
	if until != nil && time.Now().Before(*until) {
		return until, false, nil
	}
	// Cheap read before the serialized recovery transaction; no mutation for
	// ordinary groups. Other blocks never qualify as slow-first-output outages.
	var exhausted bool
	err = scanSingleRow(ctx, r.sql, slowTTFTExhaustedSQL, []any{id}, &exhausted)
	if err != nil {
		return nil, false, err
	}
	if !exhausted {
		// Another request may have recovered the group between the two reads.
		// Return its exemption rather than treating its cleared pauses as healthy
		// ordinary state for this request.
		until, err = r.SlowTTFTGroupExemption(ctx, id)
		return until, false, err
	}
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return nil, false, err
	}
	defer func() { _ = tx.Rollback() }()
	var lockedID int64
	if err = scanSingleRow(ctx, tx.Client(), `SELECT id FROM groups WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, []any{id}, &lockedID); err != nil {
		return nil, false, err
	}
	var lockedUntil sql.NullTime
	if err = scanSingleRow(ctx, tx.Client(), `SELECT slow_ttft_exempt_until FROM groups WHERE id=$1`, []any{id}, &lockedUntil); err != nil {
		return nil, false, err
	}
	if lockedUntil.Valid && time.Now().Before(lockedUntil.Time) {
		return &lockedUntil.Time, false, nil
	}
	rows, err := tx.Client().QueryContext(ctx, `SELECT a.id FROM accounts a JOIN account_groups ag ON a.id=ag.account_id WHERE ag.group_id=$1 AND a.deleted_at IS NULL ORDER BY a.id FOR UPDATE OF a`, id)
	if err != nil {
		return nil, false, err
	}
	ids := []int64{}
	for rows.Next() {
		var accountID int64
		if err = rows.Scan(&accountID); err != nil {
			_ = rows.Close()
			return nil, false, err
		}
		ids = append(ids, accountID)
	}
	err = rows.Err()
	_ = rows.Close()
	if err != nil {
		return nil, false, err
	}
	if err = scanSingleRow(ctx, tx.Client(), slowTTFTExhaustedSQL, []any{id}, &exhausted); err != nil || !exhausted {
		return nil, false, err
	}
	var recoveredUntil time.Time
	if err = scanSingleRow(ctx, tx.Client(), `UPDATE groups SET slow_ttft_exempt_until=NOW()+INTERVAL '30 minutes',updated_at=NOW() WHERE id=$1 RETURNING slow_ttft_exempt_until`, []any{id}, &recoveredUntil); err != nil {
		return nil, false, err
	}
	_, err = tx.Client().ExecContext(ctx, `UPDATE accounts SET slow_ttft_until=NULL,slow_ttft_reason='',extra=jsonb_set(extra,'{slow_ttft_protection,generation}',to_jsonb(md5(random()::text || clock_timestamp()::text))),updated_at=NOW() WHERE id=ANY($1::bigint[]) AND slow_ttft_until>NOW()`, pq.Array(ids))
	if err != nil {
		return nil, false, err
	}
	for _, accountID := range ids {
		if err = enqueueSchedulerOutbox(ctx, tx.Client(), service.SchedulerOutboxEventAccountChanged, &accountID, nil, nil); err != nil {
			return nil, false, err
		}
	}
	if err = enqueueSchedulerOutbox(ctx, tx.Client(), service.SchedulerOutboxEventGroupChanged, nil, &id, nil); err != nil {
		return nil, false, err
	}
	if err = tx.Commit(); err != nil {
		return nil, false, err
	}
	for _, accountID := range ids {
		r.syncSchedulerAccountSnapshot(ctx, accountID)
	}
	return &recoveredUntil, true, nil
}

const slowTTFTExhaustedSQL = `SELECT COALESCE(bool_and(
 COALESCE(a.slow_ttft_until>NOW(),false) AND COALESCE((a.extra->'slow_ttft_protection'->>'enabled')::boolean,false)
 AND (a.temp_unschedulable_until IS NULL OR a.temp_unschedulable_until<=NOW())
 AND (a.overload_until IS NULL OR a.overload_until<=NOW())
 AND (a.rate_limit_reset_at IS NULL OR a.rate_limit_reset_at<=NOW())
),false) FROM accounts a JOIN account_groups ag ON ag.account_id=a.id
WHERE ag.group_id=$1 AND a.deleted_at IS NULL AND a.status='active' AND a.schedulable=true
AND (a.auto_pause_on_expired=false OR a.expires_at IS NULL OR a.expires_at>NOW())`
