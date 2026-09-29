package repository

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func readGroupScheduling(ctx context.Context, db sqlExecutor, id int64, lock bool) (*service.GroupScheduling, error) {
	q := `SELECT independent_scheduling, scheduling_initialized, updated_at::text, slow_ttft_exempt_until FROM groups WHERE id=$1 AND deleted_at IS NULL`
	if lock {
		q += ` FOR UPDATE`
	}
	rows, err := db.QueryContext(ctx, q, id)
	if err != nil {
		return nil, err
	}
	var enabled, initialized bool
	var updated string
	var exempt sql.NullTime
	if !rows.Next() {
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
		return nil, service.ErrGroupNotFound
	}
	err = rows.Scan(&enabled, &initialized, &updated, &exempt)
	rows.Close()
	if err != nil {
		return nil, err
	}
	out := &service.GroupScheduling{Enabled: enabled, Accounts: []service.GroupSchedulingAccount{}}
	if exempt.Valid {
		out.SlowTTFTExemptUntil = &exempt.Time
	}
	q = `SELECT a.id,a.name,CASE WHEN $2 THEN ag.priority ELSE a.priority END,COALESCE(NULLIF(a.load_factor,0),NULLIF(a.concurrency,0),1),a.concurrency FROM account_groups ag JOIN accounts a ON a.id=ag.account_id WHERE ag.group_id=$1 AND a.deleted_at IS NULL ORDER BY a.id`
	if lock {
		q += ` FOR UPDATE OF ag, a`
	}
	rows, err = db.QueryContext(ctx, q, id, initialized)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var a service.GroupSchedulingAccount
		if err = rows.Scan(&a.AccountID, &a.Name, &a.Priority, &a.LoadFactor, &a.Concurrency); err != nil {
			return nil, err
		}
		out.Accounts = append(out.Accounts, a)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	b, _ := json.Marshal([]any{id, enabled, initialized, updated, out.Accounts})
	sum := sha256.Sum256(b)
	out.Version = hex.EncodeToString(sum[:])
	return out, nil
}
func (r *groupRepository) GetGroupScheduling(ctx context.Context, id int64) (*service.GroupScheduling, error) {
	return readGroupScheduling(ctx, r.sql, id, false)
}
func (r *groupRepository) SaveGroupScheduling(ctx context.Context, id int64, in *service.GroupScheduling) (*service.GroupScheduling, error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	current, err := readGroupScheduling(ctx, tx.Client(), id, true)
	if err != nil {
		return nil, err
	}
	if current.Version != in.Version || len(current.Accounts) != len(in.Accounts) {
		return nil, service.ErrSchedulingConflict
	}
	members := make(map[int64]bool, len(current.Accounts))
	for _, a := range current.Accounts {
		members[a.AccountID] = true
	}
	for _, a := range in.Accounts {
		if !members[a.AccountID] || a.Priority < 0 || a.Priority > 1000000 {
			return nil, service.ErrSchedulingConflict
		}
		delete(members, a.AccountID)
		res, execErr := tx.Client().ExecContext(ctx, `UPDATE account_groups SET priority=$1 WHERE group_id=$2 AND account_id=$3`, a.Priority, id, a.AccountID)
		if execErr != nil {
			return nil, execErr
		}
		n, execErr := res.RowsAffected()
		if execErr != nil {
			return nil, execErr
		}
		if n != 1 {
			return nil, service.ErrSchedulingConflict
		}
	}
	if len(members) != 0 {
		return nil, service.ErrSchedulingConflict
	}
	if _, err = tx.Client().ExecContext(ctx, `UPDATE groups SET independent_scheduling=$1,scheduling_initialized=true,updated_at=NOW() WHERE id=$2`, in.Enabled, id); err != nil {
		return nil, err
	}
	if err = enqueueSchedulerOutbox(ctx, tx.Client(), service.SchedulerOutboxEventGroupChanged, nil, &id, nil); err != nil {
		return nil, fmt.Errorf("scheduling outbox: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetGroupScheduling(ctx, id)
}
