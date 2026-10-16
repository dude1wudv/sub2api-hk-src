//go:build integration

package repository

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUsageLogOutputTokensEstimatedMigrationAndRoundTrip(t *testing.T) {
	ctx := context.Background()
	client := testEntClient(t)
	repo := newUsageLogRepositoryWithSQL(client, integrationDB)
	user := mustCreateUser(t, client, &service.User{Email: "output-estimate-" + uuid.NewString() + "@example.com"})
	apiKey := mustCreateApiKey(t, client, &service.APIKey{UserID: user.ID, Key: "sk-output-estimate-" + uuid.NewString(), Name: "output-estimate"})
	account := mustCreateAccount(t, client, &service.Account{Name: "output-estimate-" + uuid.NewString()})

	var nullable string
	var defaultValue sql.NullString
	err := integrationDB.QueryRowContext(ctx, `
		SELECT is_nullable, column_default
		FROM information_schema.columns
		WHERE table_schema = current_schema()
		  AND table_name = 'usage_logs'
		  AND column_name = 'output_tokens_estimated'
	`).Scan(&nullable, &defaultValue)
	require.NoError(t, err, "migration must add the output_tokens_estimated column")
	require.Equal(t, "NO", nullable)
	require.True(t, defaultValue.Valid)
	require.Contains(t, strings.ToLower(defaultValue.String), "false", "existing rows must default to non-estimated")

	// A historical insert that omits the new column reads as false through the
	// repository's full SELECT/scan path.
	var historicalID int64
	require.NoError(t, integrationDB.QueryRowContext(ctx, `
		INSERT INTO usage_logs (user_id, api_key_id, account_id, request_id, model, created_at)
		VALUES ($1, $2, $3, $4, 'claude-sonnet-4', $5)
		RETURNING id
	`, user.ID, apiKey.ID, account.ID, uuid.NewString(), time.Now().UTC()).Scan(&historicalID))
	historical, err := repo.GetByID(ctx, historicalID)
	require.NoError(t, err)
	require.False(t, historical.OutputTokensEstimated)

	requestID := uuid.NewString()
	estimated := &service.UsageLog{
		UserID: user.ID, APIKeyID: apiKey.ID, AccountID: account.ID,
		RequestID: requestID, Model: "claude-sonnet-4", OutputTokens: 17,
		OutputTokensEstimated: true, CreatedAt: time.Now().UTC(),
	}
	inserted, err := repo.Create(ctx, estimated)
	require.NoError(t, err)
	require.True(t, inserted)
	stored, err := repo.GetByID(ctx, estimated.ID)
	require.NoError(t, err)
	require.Equal(t, 17, stored.OutputTokens)
	require.True(t, stored.OutputTokensEstimated)

	duplicate := &service.UsageLog{
		UserID: user.ID, APIKeyID: apiKey.ID, AccountID: account.ID,
		RequestID: requestID, Model: "claude-sonnet-4", OutputTokens: 99,
		OutputTokensEstimated: false, CreatedAt: time.Now().UTC(),
	}
	inserted, err = repo.Create(ctx, duplicate)
	require.NoError(t, err)
	require.False(t, inserted, "same request id and API key must remain idempotent")
	require.Equal(t, estimated.ID, duplicate.ID)
	stored, err = repo.GetByID(ctx, estimated.ID)
	require.NoError(t, err)
	require.Equal(t, 17, stored.OutputTokens, "duplicate insert must not replace the original usage")
	require.True(t, stored.OutputTokensEstimated, "duplicate insert must preserve the original provenance marker")

	real := &service.UsageLog{
		UserID: user.ID, APIKeyID: apiKey.ID, AccountID: account.ID,
		RequestID: uuid.NewString(), Model: "claude-sonnet-4", OutputTokens: 21,
		OutputTokensEstimated: false, CreatedAt: time.Now().UTC(),
	}
	inserted, err = repo.Create(ctx, real)
	require.NoError(t, err)
	require.True(t, inserted)
	stored, err = repo.GetByID(ctx, real.ID)
	require.NoError(t, err)
	require.Equal(t, 21, stored.OutputTokens)
	require.False(t, stored.OutputTokensEstimated)
}
