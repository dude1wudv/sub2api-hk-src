package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
)

func TestTokenIncentivePeriodForUsesFiveDayWindows(t *testing.T) {
	launch := time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC)

	start, end := tokenIncentivePeriodFor(launch, time.Date(2026, 7, 10, 23, 59, 0, 0, time.UTC))
	if !start.Equal(launch) || !end.Equal(time.Date(2026, 7, 11, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("first window = %s - %s", start, end)
	}

	start, end = tokenIncentivePeriodFor(launch, time.Date(2026, 7, 11, 0, 0, 0, 0, time.UTC))
	if !start.Equal(time.Date(2026, 7, 11, 0, 0, 0, 0, time.UTC)) || !end.Equal(time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("second window = %s - %s", start, end)
	}
}

func TestBuildTokenIncentiveStatusMatches497MProgress(t *testing.T) {
	launch := time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC)
	status := buildTokenIncentiveStatus(launch, launch.Add(time.Hour), 497_300_000, map[int64]bool{
		100_000_000: true,
		200_000_000: true,
		300_000_000: true,
		400_000_000: true,
	})

	if status.ClaimedBalance != 4 || status.ClaimableBalance != 0.5 {
		t.Fatalf("balances claimed=%v claimable=%v", status.ClaimedBalance, status.ClaimableBalance)
	}
	if got := status.Tiers[0]; got.ThresholdTokens != 50_000_000 || got.RewardBalance != 0.5 {
		t.Fatalf("50M tier = %+v", got)
	}
	if status.NextThresholdTokens != 500_000_000 || status.RemainingTokens != 2_700_000 {
		t.Fatalf("next=%d remaining=%d", status.NextThresholdTokens, status.RemainingTokens)
	}
	if got := status.Tiers[5]; got.Status != "locked" || got.RemainingTokens != 2_700_000 {
		t.Fatalf("500M tier = %+v", got)
	}
}

func TestBuildTokenIncentiveStatusTotalsMaxRewardAt1B(t *testing.T) {
	launch := time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC)
	status := buildTokenIncentiveStatus(launch, launch.Add(time.Hour), 1_000_000_000, map[int64]bool{})
	if status.ClaimableBalance != 10.5 || status.MaxBalance != 10.5 {
		t.Fatalf("claimable=%v max=%v", status.ClaimableBalance, status.MaxBalance)
	}
	for _, tier := range status.Tiers {
		if tier.Status != "claimable" {
			t.Fatalf("tier %d status=%s", tier.ThresholdTokens, tier.Status)
		}
	}
}

func TestTokenIncentiveTierByThresholdRejectsUnknownTier(t *testing.T) {
	if _, ok := tokenIncentiveTierByThreshold(123); ok {
		t.Fatal("unexpected tier")
	}
	if tier, ok := tokenIncentiveTierByThreshold(50_000_000); !ok || tier.RewardBalance != 0.5 {
		t.Fatalf("50M tier = %+v ok=%v", tier, ok)
	}
	if tier, ok := tokenIncentiveTierByThreshold(1_000_000_000); !ok || tier.RewardBalance != 5 {
		t.Fatalf("1B tier = %+v ok=%v", tier, ok)
	}
}

func TestParseTokenIncentiveEligibleGroupIDs(t *testing.T) {
	ids, err := parseTokenIncentiveEligibleGroupIDs(`[1, 2, 3]`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(ids) != 3 || ids[0] != 1 || ids[2] != 3 {
		t.Fatalf("ids = %#v", ids)
	}

	ids, err = parseTokenIncentiveEligibleGroupIDs(`[]`)
	if err != nil {
		t.Fatalf("empty parse: %v", err)
	}
	if ids == nil || len(ids) != 0 {
		t.Fatalf("empty ids = %#v", ids)
	}

	ids, err = parseTokenIncentiveEligibleGroupIDs(`null`)
	if err != nil {
		t.Fatalf("null parse: %v", err)
	}
	if ids == nil || len(ids) != 0 {
		t.Fatalf("null ids = %#v", ids)
	}

	if _, err := parseTokenIncentiveEligibleGroupIDs(`{bad}`); err == nil {
		t.Fatal("expected parse error")
	}

	ids, err = parseTokenIncentiveEligibleGroupIDs("  [10]  ")
	if err != nil {
		t.Fatalf("trim parse: %v", err)
	}
	if len(ids) != 1 || ids[0] != 10 {
		t.Fatalf("trimmed ids = %#v", ids)
	}
	if !strings.Contains(TokenIncentiveSettingEligibleGroupIDs, "eligible_group") {
		t.Fatalf("unexpected setting key %q", TokenIncentiveSettingEligibleGroupIDs)
	}
}

func TestSumTokenIncentiveTokensAppliesGroupWhitelist(t *testing.T) {
	svc := &UsageService{}
	start := time.Date(2026, 7, 11, 0, 0, 0, 0, time.UTC)
	end := start.Add(5 * 24 * time.Hour)

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer func() { _ = db.Close() }()

	mock.ExpectQuery(`(?s)SELECT COALESCE\(SUM.*FROM usage_logs.*user_id = \$1 AND created_at >= \$2 AND created_at < \$3\s*$`).
		WithArgs(int64(7), start, end).
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(int64(42)))

	total, err := svc.sumTokenIncentiveTokens(context.Background(), db, 7, start, end, nil)
	if err != nil {
		t.Fatalf("no filter: %v", err)
	}
	if total != 42 {
		t.Fatalf("total = %d", total)
	}

	eligible := []int64{11, 22}
	mock.ExpectQuery(`(?s)group_id = ANY\(\$4\)`).
		WithArgs(int64(7), start, end, pq.Array(eligible)).
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(int64(9)))

	total, err = svc.sumTokenIncentiveTokens(context.Background(), db, 7, start, end, eligible)
	if err != nil {
		t.Fatalf("with filter: %v", err)
	}
	if total != 9 {
		t.Fatalf("filtered total = %d", total)
	}

	empty := []int64{}
	mock.ExpectQuery(`(?s)group_id = ANY\(\$4\)`).
		WithArgs(int64(7), start, end, pq.Array(empty)).
		WillReturnRows(sqlmock.NewRows([]string{"sum"}).AddRow(int64(0)))

	total, err = svc.sumTokenIncentiveTokens(context.Background(), db, 7, start, end, empty)
	if err != nil {
		t.Fatalf("empty whitelist: %v", err)
	}
	if total != 0 {
		t.Fatalf("empty whitelist total = %d", total)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}
