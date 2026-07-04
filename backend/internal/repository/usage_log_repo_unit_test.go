//go:build unit

package repository

import (
	"context"
	"database/sql/driver"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestSafeDateFormat(t *testing.T) {
	tests := []struct {
		name        string
		granularity string
		expected    string
	}{
		// 合法值
		{"hour", "hour", "YYYY-MM-DD HH24:00"},
		{"day", "day", "YYYY-MM-DD"},
		{"week", "week", "IYYY-IW"},
		{"month", "month", "YYYY-MM"},

		// 非法值回退到默认
		{"空字符串", "", "YYYY-MM-DD"},
		{"未知粒度 year", "year", "YYYY-MM-DD"},
		{"未知粒度 minute", "minute", "YYYY-MM-DD"},

		// 恶意字符串
		{"SQL 注入尝试", "'; DROP TABLE users; --", "YYYY-MM-DD"},
		{"带引号", "day'", "YYYY-MM-DD"},
		{"带括号", "day)", "YYYY-MM-DD"},
		{"Unicode", "日", "YYYY-MM-DD"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := safeDateFormat(tc.granularity)
			require.Equal(t, tc.expected, got, "safeDateFormat(%q)", tc.granularity)
		})
	}
}

func TestBuildUsageLogBatchInsertQuery_UsesConflictDoNothing(t *testing.T) {
	log := &service.UsageLog{
		UserID:       1,
		APIKeyID:     2,
		AccountID:    3,
		RequestID:    "req-batch-no-update",
		Model:        "gpt-5",
		InputTokens:  10,
		OutputTokens: 5,
		TotalCost:    1.2,
		ActualCost:   1.2,
		CreatedAt:    time.Now().UTC(),
	}
	prepared := prepareUsageLogInsert(log)

	query, _ := buildUsageLogBatchInsertQuery([]string{usageLogBatchKey(log.RequestID, log.APIKeyID)}, map[string]usageLogInsertPrepared{
		usageLogBatchKey(log.RequestID, log.APIKeyID): prepared,
	})

	require.Contains(t, query, "ON CONFLICT (request_id, api_key_id) DO NOTHING")
	require.NotContains(t, strings.ToUpper(query), "DO UPDATE")
}

func TestUsageLogRepositoryCreateAllocatesUsageSession(t *testing.T) {
	db, mock := newSQLMock(t)
	repo := &usageLogRepository{sql: db}
	createdAt := time.Date(2025, 1, 7, 12, 0, 0, 0, time.UTC)
	log := &service.UsageLog{
		UserID:         7,
		APIKeyID:       8,
		AccountID:      9,
		RequestID:      "req-session",
		Model:          "gpt-5.5",
		SessionKeyHash: "hash-a",
		CreatedAt:      createdAt,
	}

	mock.ExpectQuery("SELECT s\\.id, s\\.session_index").
		WithArgs(log.UserID, log.SessionKeyHash).
		WillReturnRows(sqlmock.NewRows([]string{"id", "session_index"}))
	mock.ExpectQuery("WITH next_index AS").
		WithArgs(log.UserID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "session_index"}).AddRow(int64(55), 1))
	mock.ExpectExec("INSERT INTO usage_session_keys").
		WithArgs(log.UserID, log.SessionKeyHash, int64(55)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("INSERT INTO usage_logs").
		WithArgs(anySQLArgs(len(usageLogInsertArgTypes))...).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(int64(101), createdAt))

	inserted, err := repo.Create(context.Background(), log)
	require.NoError(t, err)
	require.True(t, inserted)
	require.NotNil(t, log.SessionID)
	require.Equal(t, int64(55), *log.SessionID)
	require.NotNil(t, log.SessionIndex)
	require.Equal(t, 1, *log.SessionIndex)
	require.NoError(t, mock.ExpectationsWereMet())
}

func anySQLArgs(n int) []driver.Value {
	args := make([]driver.Value, n)
	for i := range args {
		args[i] = sqlmock.AnyArg()
	}
	return args
}
