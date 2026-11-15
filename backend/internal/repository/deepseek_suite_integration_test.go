//go:build integration

package repository

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestDeepSeekSuiteDurableOwnerAndDeletion(t *testing.T) {
	ctx := context.Background()
	user := mustCreateUser(t, integrationEntClient, &service.User{})
	group := mustCreateGroup(t, integrationEntClient, &service.Group{Name: "suite-" + uuid.NewString()})
	// Minimal records use the real FK schema and actual repository, not mocks.
	var keyID, accountID int64
	t.Cleanup(func() {
		// Receipts deliberately retain foreign keys in production. Remove this
		// test's owned rows in dependency order so later suites remain isolated.
		for _, statement := range []struct {
			sql string
			id  int64
		}{
			{"DELETE FROM deepseek_suite_operations WHERE user_id=$1", user.ID},
			{"DELETE FROM api_keys WHERE id=$1", keyID},
			{"DELETE FROM accounts WHERE id=$1", accountID},
			{"DELETE FROM groups WHERE id=$1", group.ID},
			{"DELETE FROM users WHERE id=$1", user.ID},
		} {
			_, err := integrationDB.ExecContext(ctx, statement.sql, statement.id)
			require.NoError(t, err)
		}
	})
	require.NoError(t, integrationDB.QueryRowContext(ctx, `INSERT INTO api_keys(user_id,key,name,group_id) VALUES($1,$2,'suite',$3) RETURNING id`, user.ID, uuid.NewString(), group.ID).Scan(&keyID))
	require.NoError(t, integrationDB.QueryRowContext(ctx, `INSERT INTO accounts(name,platform,type,credentials) VALUES('suite','openai','apikey','{}') RETURNING id`).Scan(&accountID))
	repo := &usageLogRepository{sql: integrationDB, db: integrationDB}
	op := &service.DeepSeekSuiteOperation{ID: uuid.NewString(), UserID: user.ID, GroupID: group.ID, KeyID: keyID, AccountID: accountID, Kind: "files", Status: "started", CreatedAt: time.Now().UTC()}
	require.NoError(t, repo.CreateDeepSeekOperation(ctx, op))
	op.RemoteID, op.Status = "file-"+uuid.NewString(), "awaiting_meter"
	op.Metadata = json.RawMessage(`{"filename":"report.pdf"}`)
	require.NoError(t, repo.SaveDeepSeekOperation(ctx, op))
	found, err := repo.GetDeepSeekFile(ctx, user.ID, group.ID, op.ID)
	require.NoError(t, err)
	require.Equal(t, accountID, found.AccountID)
	_, err = repo.GetDeepSeekFile(ctx, user.ID+1000000, group.ID, op.ID)
	require.Error(t, err)
	_, err = repo.GetDeepSeekFile(ctx, user.ID, group.ID+1000000, op.ID)
	require.Error(t, err)
	require.NoError(t, repo.MarkDeepSeekFileDeleted(ctx, user.ID, group.ID, op.ID))
	// Settlement's stale metadata must never resurrect a concurrently deleted file.
	op.Status = "settling"
	require.NoError(t, repo.SaveDeepSeekOperation(ctx, op))
	op.Status = "settled"
	require.NoError(t, repo.SaveDeepSeekOperation(ctx, op))
	_, err = repo.GetDeepSeekFile(ctx, user.ID, group.ID, op.ID)
	require.Error(t, err)
	op.Status = "awaiting_meter"
	require.Error(t, repo.SaveDeepSeekOperation(ctx, op), "a settled receipt cannot be changed back into a new charge")
	ids := []string{}
	for i := range 3 {
		item := *op
		item.ID, item.RemoteID, item.Status = uuid.NewString(), "file-"+uuid.NewString(), "awaiting_meter"
		item.CreatedAt = time.Now().Add(time.Duration(i) * time.Second)
		require.NoError(t, repo.CreateDeepSeekOperation(ctx, &item))
		require.NoError(t, repo.SaveDeepSeekOperation(ctx, &item))
		ids = append(ids, item.ID)
	}
	page, err := repo.ListDeepSeekFiles(ctx, user.ID, group.ID, service.DeepSeekFileListQuery{Limit: 1})
	require.NoError(t, err)
	require.Len(t, page, 2) // One extra record indicates has_more.
	require.Equal(t, ids[2], page[0].ID)
	page, err = repo.ListDeepSeekFiles(ctx, user.ID, group.ID, service.DeepSeekFileListQuery{Limit: 1, After: ids[2]})
	require.NoError(t, err)
	require.Equal(t, ids[1], page[0].ID)
	page, err = repo.ListDeepSeekFiles(ctx, user.ID, group.ID, service.DeepSeekFileListQuery{Limit: 1, Before: ids[0]})
	require.NoError(t, err)
	require.Equal(t, ids[1], page[0].ID)
	// Old uploads with missing provider meters cannot starve ready receipts.
	_, err = integrationDB.ExecContext(ctx, `INSERT INTO deepseek_suite_operations
		(id,user_id,api_key_id,group_id,account_id,kind,status,upstream_name,created_at,updated_at)
		SELECT gen_random_uuid(),$1,$2,$3,$4,'files','unknown','synthetic-upload-'||n,NOW()-INTERVAL '1 day',NOW()-INTERVAL '1 day'
		FROM generate_series(1,101) n`, user.ID, keyID, group.ID, accountID)
	require.NoError(t, err)
	ready := *op
	ready.ID, ready.RemoteID, ready.Status = uuid.NewString(), "", "settling"
	ready.Kind, ready.Receipt = "tts", json.RawMessage(`{"synthetic":"ready"}`)
	require.NoError(t, repo.CreateDeepSeekOperation(ctx, &ready))
	require.NoError(t, repo.SaveDeepSeekOperation(ctx, &ready))
	pending, err := repo.ListDeepSeekPending(ctx)
	require.NoError(t, err)
	require.Len(t, pending, 100)
	require.Equal(t, ready.ID, pending[0].ID)
}

func TestDeepSeekSuiteEnrollmentAndRollback(t *testing.T) {
	ctx := context.Background()
	conn, err := integrationDB.Conn(ctx)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()
	defer func() { _, _ = conn.ExecContext(ctx, "ROLLBACK") }()
	t.Cleanup(func() {
		// The release SQL commits intentionally; a deferred ROLLBACK alone
		// cannot clean these fixtures or the release-only backup table.
		_, cleanupErr := integrationDB.ExecContext(ctx, `
			DELETE FROM scheduler_outbox WHERE account_id IN (882,1043,1181,1112,1182) OR group_id IN (81,179,223);
			DELETE FROM account_groups WHERE account_id IN (882,1043,1181,1112,1182);
			DELETE FROM accounts WHERE id IN (882,1043,1181,1112,1182);
			DELETE FROM groups WHERE id IN (81,179,223);
			DROP TABLE IF EXISTS hk_deepseek_suite_20261115_backup;`)
		require.NoError(t, cleanupErr)
	})
	_, err = conn.ExecContext(ctx, `
		INSERT INTO groups(id,name,platform,model_allowlist) VALUES
		(81,'suite81','openai','{"enabled":true,"models":["deepseek-flash"]}'),
		(179,'suite179','deepseek','{"enabled":true,"models":["deepseek-flash"]}'),
		(223,'suite223','openai','{"enabled":false}');
		INSERT INTO accounts(id,name,platform,type,credentials,extra) VALUES
		(882,'synthetic official auxiliary','openai','apikey','{"base_url":"https://api.deepseek.com"}','{"keep":true}'),
		(1043,'synthetic全家桶','deepseek','apikey','{"base_url":"https://kedaya.ai/v1"}','{"keep":true}'),
		(1181,'synthetic全家桶','deepseek','apikey','{"base_url":"https://vip.mdkj.lol/v1"}','{"keep":true}'),
		(1112,'synthetic全家桶','openai','apikey','{"base_url":"https://kedaya.ai/v1"}','{"keep":true}'),
		(1182,'synthetic全家桶','openai','apikey','{"base_url":"https://vip.mdkj.lol"}','{"keep":true}');
		INSERT INTO account_groups(account_id,group_id) VALUES(1043,179),(1181,179),(1112,81),(1112,223),(1182,81),(1182,223);`)
	require.NoError(t, err)
	enroll, err := os.ReadFile("../../../deploy/deepseek-suite-enroll-20261115.sql")
	require.NoError(t, err)
	_, err = conn.ExecContext(ctx, string(enroll))
	require.NoError(t, err)
	_, err = conn.ExecContext(ctx, string(enroll))
	require.NoError(t, err, "enrollment is idempotent")
	var extra string
	require.NoError(t, conn.QueryRowContext(ctx, `SELECT extra::text FROM accounts WHERE id=1043`).Scan(&extra))
	require.True(t, strings.Contains(extra, `"deepseek_suite_capabilities"`))
	require.True(t, strings.Contains(extra, `"keep": true`))
	require.Contains(t, extra, `"deepseek_suite_fallback_account_id": 1181`)
	var count int
	require.NoError(t, conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM account_groups WHERE account_id=882 AND group_id IN (81,223)`).Scan(&count))
	require.Equal(t, 2, count)
	var name string
	require.NoError(t, conn.QueryRowContext(ctx, `SELECT name FROM accounts WHERE id=882`).Scan(&name))
	require.Equal(t, "synthetic official auxiliary 全家桶", name)
	require.NoError(t, conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM scheduler_outbox WHERE event_type='account_changed' AND account_id IN (1043,1181,1112,1182)`).Scan(&count))
	require.GreaterOrEqual(t, count, 4)
	rollback, err := os.ReadFile("../../../deploy/deepseek-suite-rollback-20261115.sql")
	require.NoError(t, err)
	_, err = conn.ExecContext(ctx, string(rollback))
	require.NoError(t, err)
	require.NoError(t, conn.QueryRowContext(ctx, `SELECT extra::text FROM accounts WHERE id=1043`).Scan(&extra))
	require.JSONEq(t, `{"keep":true}`, extra)
	require.NoError(t, conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM account_groups WHERE account_id=882`).Scan(&count))
	require.Zero(t, count)
	require.NoError(t, conn.QueryRowContext(ctx, `SELECT name FROM accounts WHERE id=882`).Scan(&name))
	require.Equal(t, "synthetic official auxiliary", name)
	var allowlist string
	require.NoError(t, conn.QueryRowContext(ctx, `SELECT model_allowlist::text FROM groups WHERE id=81`).Scan(&allowlist))
	require.JSONEq(t, `{"enabled":true,"models":["deepseek-flash"]}`, allowlist)
}

func TestSuitePositiveBalancesIncludeGiftAndExcludeDeleted(t *testing.T) {
	ctx := context.Background()
	repo := &usageLogRepository{sql: integrationDB, db: integrationDB}
	now := time.Now().UTC()
	before, after := &DashboardStats{}, &DashboardStats{}
	require.NoError(t, repo.fillDashboardEntityStats(ctx, before, now.Truncate(24*time.Hour), now))
	gift := mustCreateUser(t, integrationEntClient, &service.User{Balance: 1.25})
	paid := mustCreateUser(t, integrationEntClient, &service.User{Balance: 2.25})
	mustCreateUser(t, integrationEntClient, &service.User{Balance: 0})
	mustCreateUser(t, integrationEntClient, &service.User{Balance: -1})
	deleted := mustCreateUser(t, integrationEntClient, &service.User{Balance: 99})
	_, err := integrationDB.ExecContext(ctx, `UPDATE users SET total_recharged=10 WHERE id=$1;`, paid.ID)
	require.NoError(t, err)
	_, err = integrationDB.ExecContext(ctx, `UPDATE users SET deleted_at=NOW() WHERE id=$1`, deleted.ID)
	require.NoError(t, err)
	var recharged float64
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT total_recharged FROM users WHERE id=$1`, gift.ID).Scan(&recharged))
	require.Zero(t, recharged)
	require.NoError(t, repo.fillDashboardEntityStats(ctx, after, now.Truncate(24*time.Hour), now))
	require.Equal(t, before.UsersWithBalance+2, after.UsersWithBalance)
	require.InDelta(t, before.UsersWithBalanceTotal+3.5, after.UsersWithBalanceTotal, 1e-10)
	var width int
	require.NoError(t, integrationDB.QueryRowContext(ctx, `SELECT character_maximum_length FROM information_schema.columns WHERE table_schema='public' AND table_name='proxies' AND column_name='password'`).Scan(&width))
	require.Equal(t, 255, width)
}

func TestGroupUsageRetentionBoundaryAndCrossingUpdate(t *testing.T) {
	for _, partitioned := range []bool{false, true} {
		ctx := context.Background()
		schema := createGroupUsageRollupTriggerTestSchema(t, ctx, partitioned)
		tx := beginGroupUsageRollupTriggerTestTx(t, ctx, schema)
		defer func() { _ = tx.Rollback() }()
		migration, err := migrations.FS.ReadFile("345_group_usage_rollup_retention_guard.sql")
		require.NoError(t, err)
		_, err = tx.ExecContext(ctx, string(migration))
		require.NoError(t, err)
		_, err = tx.ExecContext(ctx, `SET LOCAL TIME ZONE 'UTC';
			INSERT INTO users(id) VALUES(1); INSERT INTO groups(id) VALUES(1);
			UPDATE usage_group_rollup_state SET retained_from='2026-11-01',closed_before='2026-11-15' WHERE id=1;
			INSERT INTO usage_logs(id,user_id,group_id,actual_cost,created_at) VALUES
			(1,1,1,1,'2026-10-01'),(2,1,1,1,'2026-11-03');`)
		require.NoError(t, err)
		var date string
		read := func(want string) {
			require.NoError(t, tx.QueryRowContext(ctx, `SELECT closed_before::text FROM usage_group_rollup_state WHERE id=1`).Scan(&date))
			require.Equal(t, want, date)
		}
		read("2026-11-03")
		_, err = tx.ExecContext(ctx, `UPDATE usage_group_rollup_state SET closed_before='2026-11-15' WHERE id=1; DELETE FROM usage_logs WHERE id=1;`)
		require.NoError(t, err)
		read("2026-11-15")
		_, err = tx.ExecContext(ctx, `INSERT INTO usage_logs VALUES(3,1,1,1,'2026-10-01'); UPDATE usage_logs SET created_at='2026-11-01' WHERE id=3;`)
		require.NoError(t, err)
		read("2026-11-01")
		_, err = tx.ExecContext(ctx, `UPDATE usage_group_rollup_state SET closed_before='2026-11-15' WHERE id=1; DELETE FROM usage_logs WHERE id=3;`)
		require.NoError(t, err)
		read("2026-11-01")
	}
}
