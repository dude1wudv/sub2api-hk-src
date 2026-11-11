package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

// GeminiCachedContentChargeIntent freezes the credential-free ledger command and
// usage snapshot in the same transaction that records the cache resource.
type GeminiCachedContentChargeIntent struct {
	Command                    UsageBillingCommand `json:"command"`
	Usage                      UsageLog            `json:"usage"`
	QuotaPlatform              string              `json:"quota_platform,omitempty"`
	SimpleModeKeyRateLimitOnly bool                `json:"simple_mode_key_rate_limit_only,omitempty"`
}

var ErrGeminiCachedContentChargeNotPersisted = errors.New("cached content charge was not persisted")

type preparedCachedContentChargeKey struct{}
type preparedCachedContentCharge struct {
	intent *GeminiCachedContentChargeIntent
}

func capturePreparedCachedContentCharge(ctx context.Context, usage *UsageLog, input *recordUsageCoreInput, cost *CostBreakdown, accountRate float64, subscriptionBill bool, cfg *config.Config) bool {
	capture, ok := ctx.Value(preparedCachedContentChargeKey{}).(*preparedCachedContentCharge)
	if !ok {
		return false
	}
	cmd := buildUsageBillingCommand(usage.RequestID, usage, &postUsageBillingParams{
		Cost: cost, User: input.User, APIKey: input.APIKey, Account: input.Account, Subscription: input.Subscription,
		IsSubscriptionBill: subscriptionBill, AccountRateMultiplier: accountRate, APIKeyService: input.APIKeyService,
		RequestPayloadHash:         resolveUsageBillingPayloadFingerprint(ctx, input.RequestPayloadHash),
		SimpleModeKeyRateLimitOnly: simpleModeKeyRateLimitBillingEnabled(cfg, input.APIKey),
	})
	if cmd == nil {
		return true
	}
	if input.Result == nil || !strings.HasPrefix(cmd.RequestID, GeminiCachedContentUsageRequestIDPrefix) {
		return true
	}
	if input.APIKey == nil || usage.UserID != cmd.UserID {
		return true
	}
	if capture != nil {
		capture.intent = &GeminiCachedContentChargeIntent{Command: *cmd, Usage: *usage, QuotaPlatform: QuotaPlatform(ctx, input.APIKey), SimpleModeKeyRateLimitOnly: simpleModeKeyRateLimitBillingEnabled(cfg, input.APIKey)}
		capture.intent.Usage.User, capture.intent.Usage.APIKey, capture.intent.Usage.Account, capture.intent.Usage.Group, capture.intent.Usage.Subscription = nil, nil, nil, nil, nil
	}
	return true
}

// PrepareCachedContentCharge freezes model prices and the existing multiplier
// rules. It executes no ledger effects and persists no usage row.
func (s *GatewayService) PrepareCachedContentCharge(ctx context.Context, input *RecordUsageInput) (*GeminiCachedContentChargeIntent, error) {
	if input == nil || input.APIKey == nil || input.Account == nil {
		return nil, errors.New("cached content billing identity is missing")
	}
	copyInput := *input
	if copyInput.User == nil {
		copyInput.User = copyInput.APIKey.User
	}
	if copyInput.User == nil {
		return nil, errors.New("cached content billing user is missing")
	}
	capture := &preparedCachedContentCharge{}
	if err := s.RecordUsage(context.WithValue(ctx, preparedCachedContentChargeKey{}, capture), &copyInput); err != nil {
		return nil, err
	}
	if capture.intent == nil {
		return nil, errors.New("cached content charge could not be prepared")
	}
	if s.cfg != nil && s.cfg.RunMode == config.RunModeSimple && !capture.intent.SimpleModeKeyRateLimitOnly {
		capture.intent.Command.BalanceCost, capture.intent.Command.SubscriptionCost, capture.intent.Command.APIKeyQuotaCost, capture.intent.Command.APIKeyRateLimitCost, capture.intent.Command.AccountQuotaCost = 0, 0, 0, 0, 0
		capture.intent.Command.RequestFingerprint = ""
		capture.intent.Command.Normalize()
	}
	return capture.intent, nil
}

// ApplyCachedContentCharge replays only frozen, credential-free ledger data.
// A ledger commit followed by a process crash is safe: Apply's durable dedup key
// and UsageLogRepository's unique request key make recovery idempotent.
func (s *GatewayService) ApplyCachedContentCharge(ctx context.Context, intent *GeminiCachedContentChargeIntent) error {
	if s == nil || intent == nil || s.usageBillingRepo == nil || s.usageLogRepo == nil {
		return errors.New("cached content settlement is unavailable")
	}
	cmd := &intent.Command
	if !strings.HasPrefix(cmd.RequestID, GeminiCachedContentUsageRequestIDPrefix) || cmd.RequestID != intent.Usage.RequestID || cmd.UserID != intent.Usage.UserID || cmd.APIKeyID != intent.Usage.APIKeyID || cmd.AccountID != intent.Usage.AccountID {
		return errors.New("cached content charge identity conflict")
	}
	result, err := s.usageBillingRepo.Apply(ctx, cmd)
	if err != nil {
		return err
	}
	if s.billingCacheService != nil {
		_ = s.billingCacheService.InvalidateUserBalance(ctx, cmd.UserID)
		if cmd.SubscriptionID != nil && intent.Usage.GroupID != nil {
			_ = s.billingCacheService.InvalidateSubscription(ctx, cmd.UserID, *intent.Usage.GroupID)
		}
		_ = s.billingCacheService.InvalidateAPIKeyRateLimit(ctx, cmd.APIKeyID)
	}
	if result != nil && result.Applied {
		// Recovery evicts financial snapshots instead of applying deltas: a
		// concurrent cache refill may already include this committed charge.
		if s.deferredService != nil {
			s.deferredService.ScheduleLastUsedUpdate(cmd.AccountID)
		}
		if !intent.SimpleModeKeyRateLimitOnly && cmd.BalanceCost > 0 && intent.QuotaPlatform != "" && s.userPlatformQuotaRepo != nil && s.billingCacheService != nil {
			if s.billingCacheService.HasUserPlatformQuotaLimit(ctx, cmd.UserID, intent.QuotaPlatform) {
				s.billingCacheService.IncrementUserPlatformQuotaUsage(cmd.UserID, intent.QuotaPlatform, cmd.BalanceCost)
				if s.cfg == nil || !s.cfg.Database.UserPlatformQuotaFlusherEnabled {
					persistPostUsagePlatformQuota(ctx, s.userPlatformQuotaRepo, cmd.UserID, intent.QuotaPlatform, cmd.BalanceCost)
				}
			}
		}
		if s.balanceNotifyService != nil && !intent.SimpleModeKeyRateLimitOnly {
			params := &postUsageBillingParams{
				Cost: &CostBreakdown{ActualCost: cmd.BalanceCost, TotalCost: intent.Usage.TotalCost},
				User: &User{ID: cmd.UserID}, APIKey: &APIKey{ID: cmd.APIKeyID, UserID: cmd.UserID},
				Account:            &Account{ID: cmd.AccountID, Type: cmd.AccountType},
				IsSubscriptionBill: cmd.SubscriptionID != nil,
			}
			s.balanceNotifyService.submitUsageNotifications(params, s.billingDeps(), result)
		}
	}
	usage := intent.Usage
	if _, err := s.usageLogRepo.Create(ctx, &usage); err != nil {
		return err
	}
	return nil
}

// Additional repository contract is separate from legacy metadata fixtures.
// Production implementations must support this contract; no charged operation
// falls back to a non-durable save.
type GeminiCachedContentChargeRepository interface {
	CommitChargedMutation(context.Context, int64, string, time.Time, *GeminiCachedContentChargeIntent) error
	CompleteCharge(context.Context, int64, string) error
	ListPendingCharges(context.Context, int) ([]*GeminiCachedContent, error)
	RecordChargeAttempt(context.Context, int64) error
}

func (s *GeminiCachedContentService) SetBillingGateway(gateway *GatewayService) {
	s.billingGateway = gateway
}

func (s *GeminiCachedContentService) SaveCharged(ctx context.Context, record *GeminiCachedContent, intent *GeminiCachedContentChargeIntent) error {
	if _, ok := s.repo.(GeminiCachedContentChargeRepository); !ok {
		return errors.Join(ErrGeminiCachedContentChargeNotPersisted, errors.New("durable cached content charges are unsupported"))
	}
	if record == nil || intent == nil {
		return errors.Join(ErrGeminiCachedContentChargeNotPersisted, errors.New("cached content charge is missing"))
	}
	if intent.Command.UserID != record.UserID || intent.Command.APIKeyID != record.APIKeyID || intent.Command.AccountID != record.AccountID {
		return errors.Join(ErrGeminiCachedContentChargeNotPersisted, errors.New("cached content charge owner conflict"))
	}
	record.BillingPending, record.ChargeIntent = true, intent
	target := record.ExpireTime
	record.ChargeTargetExpire = &target
	if err := s.repo.Create(ctx, record); err != nil {
		// INSERT may have committed before RETURNING or the connection failed.
		// A fresh read is the only basis for deleting the upstream resource.
		verifyCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
		defer cancel()
		saved, lookupErr := s.repo.GetForOwner(verifyCtx, record.APIKeyID, record.PublicID)
		if errors.Is(lookupErr, ErrGeminiCachedContentNotFound) {
			return errors.Join(ErrGeminiCachedContentChargeNotPersisted, err)
		}
		if lookupErr != nil {
			return errors.Join(err, lookupErr)
		}
		if saved != nil && saved.ChargeIntent != nil && saved.ChargeIntent.Command.RequestID == intent.Command.RequestID && saved.ChargeIntent.Command.RequestFingerprint == intent.Command.RequestFingerprint {
			*record = *saved
			return nil
		}
		return errors.Join(err, errors.New("cached content charge persistence could not be confirmed"))
	}
	return nil
}

func (s *GeminiCachedContentService) SetAuthCacheInvalidator(invalidator APIKeyAuthCacheInvalidator) {
	s.authCacheInvalidator = invalidator
}

func (s *GeminiCachedContentService) CommitChargedMutation(ctx context.Context, m *GeminiCachedContentMutation, target time.Time, intent *GeminiCachedContentChargeIntent) error {
	repo, ok := s.repo.(GeminiCachedContentChargeRepository)
	if !ok || intent == nil {
		return errors.New("durable cached content charges are unsupported")
	}
	if intent.Command.UserID != m.record.UserID || intent.Command.APIKeyID != m.record.APIKeyID || intent.Command.AccountID != m.record.AccountID {
		return errors.New("cached content charge owner conflict")
	}
	if err := repo.CommitChargedMutation(ctx, m.record.ID, m.token, target, intent); err != nil {
		return err
	}
	m.record.BillingPending, m.record.ChargeIntent, m.record.ChargeTargetExpire = true, intent, &target
	return nil
}

func (s *GeminiCachedContentService) SettleCharge(ctx context.Context, record *GeminiCachedContent) error {
	if record.ChargeIntent == nil {
		return nil
	}
	cmd := &record.ChargeIntent.Command
	if cmd.UserID != record.UserID || cmd.APIKeyID != record.APIKeyID || cmd.AccountID != record.AccountID {
		return errors.New("cached content settlement owner conflict")
	}
	if s.billingGateway == nil {
		return errors.New("cached content billing service is unavailable")
	}
	repo, ok := s.repo.(GeminiCachedContentChargeRepository)
	if !ok {
		return errors.New("durable cached content charges are unsupported")
	}
	applyErr := s.billingGateway.ApplyCachedContentCharge(ctx, record.ChargeIntent)
	// The ledger may have committed even when the subsequent usage write fails.
	// Evict auth snapshots on that path as well so key quotas fail closed.
	if s.authCacheInvalidator != nil {
		s.authCacheInvalidator.InvalidateAuthCacheByUserID(ctx, record.UserID)
	}
	if applyErr != nil {
		return applyErr
	}
	if err := repo.CompleteCharge(ctx, record.ID, record.ChargeIntent.Command.RequestID); err != nil {
		return err
	}
	if record.ChargeTargetExpire != nil {
		record.ExpireTime = *record.ChargeTargetExpire
	}
	record.BillingPending, record.ChargeIntent, record.ChargeTargetExpire = false, nil, nil
	return nil
}

func (s *GeminiCachedContentService) RecoverPendingCharges(ctx context.Context) (int, error) {
	repo, ok := s.repo.(GeminiCachedContentChargeRepository)
	if !ok {
		return 0, nil
	}
	queryCtx, cancelQuery := context.WithTimeout(ctx, 15*time.Second)
	records, err := repo.ListPendingCharges(queryCtx, geminiCachedContentMutationRecoveryBatch)
	cancelQuery()
	if err != nil {
		return 0, err
	}
	count := 0
	var failures []error
	for _, record := range records {
		workCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		err := repo.RecordChargeAttempt(workCtx, record.ID)
		if err == nil {
			err = s.SettleCharge(workCtx, record)
		}
		cancel()
		if err != nil {
			failures = append(failures, err)
		} else {
			count++
		}
	}
	return count, errors.Join(failures...)
}
