//go:build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestUsageLogRepository_ListWithFilters_SessionTransitions(t *testing.T) {
	ctx := context.Background()
	tx := testEntTx(t)
	client := tx.Client()
	repo := newUsageLogRepositoryWithSQL(client, tx)

	userA := mustCreateUser(t, client, &service.User{Email: "session-transition-a-" + uuid.NewString() + "@example.com"})
	userB := mustCreateUser(t, client, &service.User{Email: "session-transition-b-" + uuid.NewString() + "@example.com"})
	apiKeyA := mustCreateApiKey(t, client, &service.APIKey{UserID: userA.ID, Key: "sk-session-transition-a-" + uuid.NewString(), Name: "transition-a"})
	apiKeyB := mustCreateApiKey(t, client, &service.APIKey{UserID: userB.ID, Key: "sk-session-transition-b-" + uuid.NewString(), Name: "transition-b"})
	accountA := mustCreateAccount(t, client, &service.Account{Name: "session-transition-account-a-" + uuid.NewString()})
	accountB := mustCreateAccount(t, client, &service.Account{Name: "session-transition-account-b-" + uuid.NewString()})

	base := time.Now().UTC().Truncate(time.Microsecond).Add(-time.Hour)
	sessionID := "session-transition-" + uuid.NewString()
	createLog := func(user *service.User, apiKey *service.APIKey, account *service.Account, session *string, createdAt time.Time) *service.UsageLog {
		t.Helper()
		log := &service.UsageLog{
			UserID:       user.ID,
			APIKeyID:     apiKey.ID,
			AccountID:    account.ID,
			RequestID:    uuid.NewString(),
			Model:        "claude-3",
			InputTokens:  1,
			OutputTokens: 1,
			TotalCost:    1,
			ActualCost:   1,
			SessionID:    session,
			CreatedAt:    createdAt,
		}
		_, err := repo.Create(ctx, log)
		require.NoError(t, err, "create usage log")
		require.NotZero(t, log.ID)
		return log
	}

	// The session history is A -> A -> B -> B -> A. The empty-session row is
	// deliberately before it so the final paginated row is the A transition.
	emptySessionLog := createLog(userA, apiKeyA, accountA, nil, base.Add(-time.Minute))
	firstA := createLog(userA, apiKeyA, accountA, &sessionID, base)
	sameA := createLog(userA, apiKeyA, accountA, &sessionID, base.Add(time.Minute))
	firstB := createLog(userA, apiKeyA, accountB, &sessionID, base.Add(2*time.Minute))
	sameB := createLog(userA, apiKeyA, accountB, &sessionID, base.Add(3*time.Minute))
	lastA := createLog(userA, apiKeyA, accountA, &sessionID, base.Add(4*time.Minute))
	// A same-named session for a different user must not use userA's history.
	otherUser := createLog(userB, apiKeyB, accountA, &sessionID, base.Add(150*time.Second))

	assertTransition := func(logs []service.UsageLog, id int64, switched bool, previousID *int64) {
		t.Helper()
		var got *service.UsageLog
		for i := range logs {
			if logs[i].ID == id {
				got = &logs[i]
				break
			}
		}
		require.NotNil(t, got, "usage log %d should be returned", id)
		require.Equal(t, switched, got.SessionAccountSwitched, "usage log %d transition state", id)
		if previousID == nil {
			require.Nil(t, got.PreviousAccountID, "usage log %d previous account id", id)
			require.Nil(t, got.PreviousAccount, "usage log %d previous account", id)
			return
		}
		require.NotNil(t, got.PreviousAccountID, "usage log %d previous account id", id)
		require.Equal(t, *previousID, *got.PreviousAccountID, "usage log %d previous account id", id)
		require.NotNil(t, got.PreviousAccount, "usage log %d previous account", id)
		require.Equal(t, *previousID, got.PreviousAccount.ID, "usage log %d previous account", id)
	}

	previousA := accountA.ID
	previousB := accountB.ID

	filters := usagestats.UsageLogFilters{
		UserID:                    userA.ID,
		IncludeSessionTransitions: true,
		ExactTotal:                true,
	}
	page1, page, err := repo.ListWithFilters(ctx, pagination.PaginationParams{
		Page: 1, PageSize: 2, SortBy: "created_at", SortOrder: pagination.SortOrderAsc,
	}, filters)
	require.NoError(t, err)
	require.Equal(t, int64(6), page.Total)
	require.Len(t, page1, 2)
	assertTransition(page1, emptySessionLog.ID, false, nil)
	assertTransition(page1, firstA.ID, false, nil)

	page2, _, err := repo.ListWithFilters(ctx, pagination.PaginationParams{
		Page: 2, PageSize: 2, SortBy: "created_at", SortOrder: pagination.SortOrderAsc,
	}, filters)
	require.NoError(t, err)
	require.Len(t, page2, 2)
	assertTransition(page2, sameA.ID, false, nil)
	assertTransition(page2, firstB.ID, true, &previousA)

	page3, _, err := repo.ListWithFilters(ctx, pagination.PaginationParams{
		Page: 3, PageSize: 2, SortBy: "created_at", SortOrder: pagination.SortOrderAsc,
	}, filters)
	require.NoError(t, err)
	require.Len(t, page3, 2)
	assertTransition(page3, sameB.ID, false, nil)
	assertTransition(page3, lastA.ID, true, &previousB)

	// Both predecessor rows are excluded by the current date and account
	// filters. The complete retained history must still supply each predecessor.
	start := base.Add(2 * time.Minute)
	end := base.Add(2*time.Minute + time.Second)
	middleFiltered, _, err := repo.ListWithFilters(ctx, pagination.PaginationParams{
		Page: 1, PageSize: 10, SortBy: "created_at", SortOrder: pagination.SortOrderAsc,
	}, usagestats.UsageLogFilters{
		UserID:                    userA.ID,
		AccountID:                 accountB.ID,
		StartTime:                 &start,
		EndTime:                   &end,
		IncludeSessionTransitions: true,
		ExactTotal:                true,
	})
	require.NoError(t, err)
	require.Len(t, middleFiltered, 1)
	assertTransition(middleFiltered, firstB.ID, true, &previousA)

	start = base.Add(4 * time.Minute)
	end = base.Add(4*time.Minute + time.Second)
	lastFiltered, _, err := repo.ListWithFilters(ctx, pagination.PaginationParams{
		Page: 1, PageSize: 10, SortBy: "created_at", SortOrder: pagination.SortOrderAsc,
	}, usagestats.UsageLogFilters{
		UserID:                    userA.ID,
		AccountID:                 accountA.ID,
		StartTime:                 &start,
		EndTime:                   &end,
		IncludeSessionTransitions: true,
		ExactTotal:                true,
	})
	require.NoError(t, err)
	require.Len(t, lastFiltered, 1)
	assertTransition(lastFiltered, lastA.ID, true, &previousB)

	otherUserLogs, _, err := repo.ListWithFilters(ctx, pagination.PaginationParams{
		Page: 1, PageSize: 10, SortBy: "created_at", SortOrder: pagination.SortOrderAsc,
	}, usagestats.UsageLogFilters{
		UserID:                    userB.ID,
		IncludeSessionTransitions: true,
		ExactTotal:                true,
	})
	require.NoError(t, err)
	require.Len(t, otherUserLogs, 1)
	assertTransition(otherUserLogs, otherUser.ID, false, nil)

	// The opt-in flag is intentionally required: without it, even a real
	// transition must not expose derived fields to non-admin callers.
	withoutDerivation, _, err := repo.ListWithFilters(ctx, pagination.PaginationParams{
		Page: 1, PageSize: 10,
	}, usagestats.UsageLogFilters{
		UserID:     userA.ID,
		RequestID:  firstB.RequestID,
		ExactTotal: true,
	})
	require.NoError(t, err)
	require.Len(t, withoutDerivation, 1)
	require.False(t, withoutDerivation[0].SessionAccountSwitched)
	require.Nil(t, withoutDerivation[0].PreviousAccountID)
	require.Nil(t, withoutDerivation[0].PreviousAccount)

	// A deleted predecessor still contributes its durable ID to transition
	// detection, but cannot be hydrated as an Account association.
	softPrevious := mustCreateAccount(t, client, &service.Account{Name: "session-transition-soft-previous-" + uuid.NewString()})
	softCurrent := mustCreateAccount(t, client, &service.Account{Name: "session-transition-soft-current-" + uuid.NewString()})
	softSessionID := "session-transition-soft-" + uuid.NewString()
	softPreviousLog := createLog(userA, apiKeyA, softPrevious, &softSessionID, base.Add(20*time.Minute))
	softCurrentLog := createLog(userA, apiKeyA, softCurrent, &softSessionID, base.Add(21*time.Minute))
	err = client.Account.DeleteOneID(softPrevious.ID).Exec(ctx)
	require.NoError(t, err)

	softLogs, _, err := repo.ListWithFilters(ctx, pagination.PaginationParams{
		Page: 1, PageSize: 10, SortBy: "created_at", SortOrder: pagination.SortOrderAsc,
	}, usagestats.UsageLogFilters{
		UserID:                    userA.ID,
		RequestID:                 softCurrentLog.RequestID,
		IncludeSessionTransitions: true,
		ExactTotal:                true,
	})
	require.NoError(t, err)
	require.Len(t, softLogs, 1)
	require.Equal(t, softCurrentLog.ID, softLogs[0].ID)
	require.True(t, softLogs[0].SessionAccountSwitched)
	require.NotNil(t, softLogs[0].PreviousAccountID)
	require.Equal(t, softPrevious.ID, *softLogs[0].PreviousAccountID)
	require.Nil(t, softLogs[0].PreviousAccount)
	require.NotNil(t, softLogs[0].Account)
	require.Equal(t, softCurrent.ID, softLogs[0].Account.ID)
	require.Equal(t, softPreviousLog.AccountID, *softLogs[0].PreviousAccountID)

	// Equal timestamps must use id as the deterministic tie-breaker. Use a
	// fresh session and reverse the account from the earlier fixture so a
	// created_at-only predecessor lookup would miss the transition.
	tieTime := base.Add(10 * time.Minute)
	tieSessionID := "session-transition-tie-" + uuid.NewString()
	tieA := createLog(userA, apiKeyA, accountB, &tieSessionID, tieTime)
	tieB := createLog(userA, apiKeyA, accountA, &tieSessionID, tieTime)
	require.Equal(t, tieA.CreatedAt, tieB.CreatedAt)
	require.Greater(t, tieB.ID, tieA.ID)

	tieStart := tieTime.Add(-time.Second)
	tieEnd := tieTime.Add(time.Second)
	tieLogs, _, err := repo.ListWithFilters(ctx, pagination.PaginationParams{
		Page: 1, PageSize: 10, SortBy: "created_at", SortOrder: pagination.SortOrderAsc,
	}, usagestats.UsageLogFilters{
		UserID:                    userA.ID,
		StartTime:                 &tieStart,
		EndTime:                   &tieEnd,
		IncludeSessionTransitions: true,
		ExactTotal:                true,
	})
	require.NoError(t, err)
	require.Len(t, tieLogs, 2)
	assertTransition(tieLogs, tieA.ID, false, nil)
	assertTransition(tieLogs, tieB.ID, true, &previousB)
}
