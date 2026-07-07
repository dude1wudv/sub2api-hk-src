package service

import (
	"testing"
	"time"
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
