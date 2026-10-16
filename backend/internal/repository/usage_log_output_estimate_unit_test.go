//go:build unit

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUsageLogInsertPersistsOutputTokensEstimated(t *testing.T) {
	for _, estimated := range []bool{false, true} {
		t.Run(map[bool]string{false: "real", true: "estimated"}[estimated], func(t *testing.T) {
			log := &service.UsageLog{
				UserID: 1, APIKeyID: 2, AccountID: 3,
				RequestID: "output-estimated-" + map[bool]string{false: "real", true: "estimated"}[estimated],
				Model:     "claude-sonnet-4", OutputTokens: 17, OutputTokensEstimated: estimated,
				CreatedAt: time.Now().UTC(),
			}
			prepared := prepareUsageLogInsert(log)
			idx := -1
			for i, typ := range usageLogInsertArgTypes {
				if typ == "boolean" && i == len(usageLogInsertArgTypes)-1 {
					idx = i
				}
			}
			require.NotEqual(t, -1, idx, "estimated provenance column must be part of insert arguments")
			require.Equal(t, estimated, prepared.args[idx])

			var captured []string
			db, mock := newSQLCapturingMock(t, &captured)
			repo := &usageLogRepository{sql: db}
			args := anySliceToDriverValues(prepared.args)
			mock.ExpectQuery("INSERT INTO usage_logs").WithArgs(args...).WillReturnRows(
				sqlmock.NewRows([]string{"id", "created_at"}).AddRow(int64(1), log.CreatedAt),
			)
			inserted, err := repo.createSingle(context.Background(), db, log)
			require.NoError(t, err)
			require.True(t, inserted)
			require.NoError(t, mock.ExpectationsWereMet())
			require.Len(t, captured, 1)
			requireStaticInsertMatchesArgTypes(t, captured[0])
		})
	}
}
