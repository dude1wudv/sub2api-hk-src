package service

import (
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	TokenIncentiveSettingLaunchAt = "token_incentive_launch_at"
	tokenIncentivePeriod          = 5 * 24 * time.Hour
)

var (
	ErrTokenIncentiveDisabled    = infraerrors.NotFound("TOKEN_INCENTIVE_DISABLED", "token incentive is disabled")
	ErrTokenIncentiveBadLaunchAt = infraerrors.InternalServer("TOKEN_INCENTIVE_BAD_LAUNCH_AT", "token incentive launch time is invalid")
	ErrTokenIncentiveBadTier     = infraerrors.BadRequest("TOKEN_INCENTIVE_BAD_TIER", "invalid threshold")
	ErrTokenIncentiveLocked      = infraerrors.BadRequest("TOKEN_INCENTIVE_LOCKED", "threshold not reached")
	ErrTokenIncentiveClaimed     = infraerrors.Conflict("TOKEN_INCENTIVE_CLAIMED", "reward already claimed")
)

type TokenIncentiveTier struct {
	ThresholdTokens int64   `json:"threshold_tokens"`
	RewardBalance   float64 `json:"reward_balance"`
	Status          string  `json:"status"`
	RemainingTokens int64   `json:"remaining_tokens,omitempty"`
}

type TokenIncentiveStatus struct {
	Enabled             bool                 `json:"enabled"`
	PeriodStart         time.Time            `json:"period_start"`
	PeriodEnd           time.Time            `json:"period_end"`
	TotalTokens         int64                `json:"total_tokens"`
	NextThresholdTokens int64                `json:"next_threshold_tokens"`
	RemainingTokens     int64                `json:"remaining_tokens"`
	ClaimableBalance    float64              `json:"claimable_balance"`
	ClaimedBalance      float64              `json:"claimed_balance"`
	MaxBalance          float64              `json:"max_balance"`
	Tiers               []TokenIncentiveTier `json:"tiers"`
}

type TokenIncentiveClaimResult struct {
	Claimed       bool               `json:"claimed"`
	RewardBalance float64            `json:"reward_balance"`
	Balance       float64            `json:"balance"`
	Tier          TokenIncentiveTier `json:"tier"`
}

func defaultTokenIncentiveTiers() []TokenIncentiveTier {
	return []TokenIncentiveTier{
		{ThresholdTokens: 50_000_000, RewardBalance: 0.5},
		{ThresholdTokens: 100_000_000, RewardBalance: 1},
		{ThresholdTokens: 200_000_000, RewardBalance: 1},
		{ThresholdTokens: 300_000_000, RewardBalance: 1},
		{ThresholdTokens: 400_000_000, RewardBalance: 1},
		{ThresholdTokens: 500_000_000, RewardBalance: 1},
		{ThresholdTokens: 1_000_000_000, RewardBalance: 5},
	}
}

func tokenIncentivePeriodFor(launchAt, now time.Time) (time.Time, time.Time) {
	launchAt = launchAt.UTC()
	now = now.UTC()
	if now.Before(launchAt) {
		return launchAt, launchAt.Add(tokenIncentivePeriod)
	}
	idx := int64(now.Sub(launchAt) / tokenIncentivePeriod)
	start := launchAt.Add(time.Duration(idx) * tokenIncentivePeriod)
	return start, start.Add(tokenIncentivePeriod)
}

func buildTokenIncentiveStatus(launchAt, now time.Time, totalTokens int64, claimed map[int64]bool) *TokenIncentiveStatus {
	periodStart, periodEnd := tokenIncentivePeriodFor(launchAt, now)
	status := &TokenIncentiveStatus{
		Enabled:     true,
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		TotalTokens: totalTokens,
		Tiers:       defaultTokenIncentiveTiers(),
	}
	for i := range status.Tiers {
		tier := &status.Tiers[i]
		status.MaxBalance += tier.RewardBalance
		if claimed[tier.ThresholdTokens] {
			tier.Status = "claimed"
			status.ClaimedBalance += tier.RewardBalance
			continue
		}
		if totalTokens >= tier.ThresholdTokens {
			tier.Status = "claimable"
			status.ClaimableBalance += tier.RewardBalance
			continue
		}
		tier.Status = "locked"
		tier.RemainingTokens = tier.ThresholdTokens - totalTokens
		if status.NextThresholdTokens == 0 {
			status.NextThresholdTokens = tier.ThresholdTokens
			status.RemainingTokens = tier.RemainingTokens
		}
	}
	if status.NextThresholdTokens == 0 && len(status.Tiers) > 0 {
		status.NextThresholdTokens = status.Tiers[len(status.Tiers)-1].ThresholdTokens
	}
	return status
}

func tokenIncentiveTierByThreshold(threshold int64) (TokenIncentiveTier, bool) {
	for _, tier := range defaultTokenIncentiveTiers() {
		if tier.ThresholdTokens == threshold {
			return tier, true
		}
	}
	return TokenIncentiveTier{}, false
}
