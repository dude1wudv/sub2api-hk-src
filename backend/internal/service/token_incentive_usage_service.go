package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
)

type sqlQueryer interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

func (s *UsageService) GetTokenIncentiveStatus(ctx context.Context, userID int64, now time.Time) (*TokenIncentiveStatus, error) {
	launchAt, err := s.tokenIncentiveLaunchAt(ctx)
	if err != nil {
		return nil, err
	}
	periodStart, periodEnd := tokenIncentivePeriodFor(launchAt, now)
	total, err := s.sumTokenIncentiveTokens(ctx, s.entClient, userID, periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("sum token incentive usage: %w", err)
	}
	claimed, err := s.tokenIncentiveClaimedThresholds(ctx, s.entClient, userID, periodStart)
	if err != nil {
		return nil, fmt.Errorf("list token incentive claims: %w", err)
	}
	return buildTokenIncentiveStatus(launchAt, now, total, claimed), nil
}

func (s *UsageService) ClaimTokenIncentive(ctx context.Context, userID, thresholdTokens int64, now time.Time) (*TokenIncentiveClaimResult, error) {
	tier, ok := tokenIncentiveTierByThreshold(thresholdTokens)
	if !ok {
		return nil, ErrTokenIncentiveBadTier
	}
	launchAt, err := s.tokenIncentiveLaunchAt(ctx)
	if err != nil {
		return nil, err
	}
	periodStart, periodEnd := tokenIncentivePeriodFor(launchAt, now)

	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin token incentive claim: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	txCtx := dbent.NewTxContext(ctx, tx)
	client := tx.Client()

	total, err := s.sumTokenIncentiveTokens(txCtx, client, userID, periodStart, periodEnd)
	if err != nil {
		return nil, fmt.Errorf("sum token incentive usage: %w", err)
	}
	if total < thresholdTokens {
		return nil, ErrTokenIncentiveLocked
	}

	inserted, err := s.insertTokenIncentiveClaim(txCtx, client, userID, periodStart, periodEnd, tier, total)
	if err != nil {
		return nil, fmt.Errorf("insert token incentive claim: %w", err)
	}
	if !inserted {
		return nil, ErrTokenIncentiveClaimed
	}

	balance, err := s.addTokenIncentiveBalance(txCtx, client, userID, tier.RewardBalance)
	if err != nil {
		return nil, fmt.Errorf("add token incentive balance: %w", err)
	}
	if err := s.insertTokenIncentiveBalanceRecord(txCtx, client, userID, tier, periodStart, total); err != nil {
		return nil, fmt.Errorf("insert token incentive balance record: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit token incentive claim: %w", err)
	}
	if s.authCacheInvalidator != nil {
		s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, userID)
	}
	tier.Status = "claimed"
	return &TokenIncentiveClaimResult{Claimed: true, RewardBalance: tier.RewardBalance, Balance: balance, Tier: tier}, nil
}

func (s *UsageService) tokenIncentiveLaunchAt(ctx context.Context) (time.Time, error) {
	rows, err := s.entClient.QueryContext(ctx, `SELECT value FROM settings WHERE key = $1`, TokenIncentiveSettingLaunchAt)
	if err != nil {
		return time.Time{}, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return time.Time{}, ErrTokenIncentiveDisabled
	}
	var raw string
	if err := rows.Scan(&raw); err != nil {
		return time.Time{}, err
	}
	launchAt, err := parseTokenIncentiveLaunchAt(raw)
	if err != nil {
		return time.Time{}, ErrTokenIncentiveBadLaunchAt.WithCause(err)
	}
	return launchAt, nil
}

func parseTokenIncentiveLaunchAt(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05Z07", "2006-01-02 15:04:05.999999Z07"} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("parse launch_at %q", raw)
}

func (s *UsageService) sumTokenIncentiveTokens(ctx context.Context, q sqlQueryer, userID int64, start, end time.Time) (int64, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT COALESCE(SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens), 0)
		FROM usage_logs
		WHERE user_id = $1 AND created_at >= $2 AND created_at < $3
	`, userID, start, end)
	if err != nil {
		return 0, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return 0, nil
	}
	var total int64
	if err := rows.Scan(&total); err != nil {
		return 0, err
	}
	return total, rows.Err()
}

func (s *UsageService) tokenIncentiveClaimedThresholds(ctx context.Context, q sqlQueryer, userID int64, periodStart time.Time) (map[int64]bool, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT threshold_tokens
		FROM token_incentive_claims
		WHERE user_id = $1 AND period_start = $2
	`, userID, periodStart)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	claimed := map[int64]bool{}
	for rows.Next() {
		var threshold int64
		if err := rows.Scan(&threshold); err != nil {
			return nil, err
		}
		claimed[threshold] = true
	}
	return claimed, rows.Err()
}

func (s *UsageService) insertTokenIncentiveClaim(ctx context.Context, q sqlQueryer, userID int64, start, end time.Time, tier TokenIncentiveTier, totalTokens int64) (bool, error) {
	rows, err := q.QueryContext(ctx, `
		INSERT INTO token_incentive_claims (user_id, period_start, period_end, threshold_tokens, reward_balance, usage_tokens_at_claim, claimed_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (user_id, period_start, threshold_tokens) DO NOTHING
		RETURNING id
	`, userID, start, end, tier.ThresholdTokens, tier.RewardBalance, totalTokens)
	if err != nil {
		return false, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return false, rows.Err()
	}
	var id int64
	if err := rows.Scan(&id); err != nil {
		return false, err
	}
	return true, rows.Err()
}

func (s *UsageService) addTokenIncentiveBalance(ctx context.Context, q sqlQueryer, userID int64, reward float64) (float64, error) {
	rows, err := q.QueryContext(ctx, `UPDATE users SET balance = balance + $1 WHERE id = $2 RETURNING balance`, reward, userID)
	if err != nil {
		return 0, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return 0, ErrUserNotFound
	}
	var balance float64
	if err := rows.Scan(&balance); err != nil {
		return 0, err
	}
	return balance, rows.Err()
}

func (s *UsageService) insertTokenIncentiveBalanceRecord(ctx context.Context, q sqlQueryer, userID int64, tier TokenIncentiveTier, periodStart time.Time, totalTokens int64) error {
	code, err := GenerateRedeemCode()
	if err != nil {
		return err
	}
	notes := fmt.Sprintf("Token incentive reward: threshold=%d period_start=%s usage_tokens=%d", tier.ThresholdTokens, periodStart.UTC().Format(time.RFC3339), totalTokens)
	rows, err := q.QueryContext(ctx, `
		INSERT INTO redeem_codes (code, type, value, status, used_by, used_at, notes, created_at)
		VALUES ($1, 'admin_balance', $2, 'used', $3, NOW(), $4, NOW())
		RETURNING id
	`, code, tier.RewardBalance, userID, notes)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return fmt.Errorf("redeem_codes insert returned no id")
	}
	var id int64
	if err := rows.Scan(&id); err != nil {
		return err
	}
	return rows.Err()
}
