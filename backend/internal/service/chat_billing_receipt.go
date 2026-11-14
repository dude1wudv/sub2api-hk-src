package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

// Only server-created, credential-free snapshots are persisted. Commands retain
// their original fingerprint; replay never resolves a model price or calls OCR.
type ChatBillingReceipt struct {
	Version               int                    `json:"version"`
	Command               *UsageBillingCommand   `json:"command"`
	Log                   *UsageLog              `json:"log"`
	Cost                  *CostBreakdown         `json:"cost"`
	User                  APIKeyAuthUserSnapshot `json:"user"`
	KeyQuota              float64                `json:"key_quota"`
	KeyLimits             [3]float64             `json:"key_limits"`
	GroupID               *int64                 `json:"group_id"`
	AccountName           string                 `json:"account_name"`
	AccountPlatform       string                 `json:"account_platform"`
	AccountExtra          map[string]any         `json:"account_quota"`
	AccountRateMultiplier float64                `json:"account_rate_multiplier"`
	SubscriptionBill      bool                   `json:"subscription_bill"`
	Platform              string                 `json:"platform"`
}

func (s *OpenAIGatewayService) PrepareChatAuxiliaryBilling(ctx context.Context, input *OpenAIRecordUsageInput) (json.RawMessage, error) {
	if input == nil || input.Result == nil || !input.Result.chatAuxiliaryOnly || input.Result.ChatAuxiliaryUsage == nil || s.usageBillingRepo == nil || s.usageLogRepo == nil || (s.cfg != nil && s.cfg.RunMode == config.RunModeSimple) {
		return nil, ErrChatAuxiliaryUnavailable
	}
	copyInput := *input
	var raw json.RawMessage
	copyInput.prepareBilling = func(log *UsageLog, p *postUsageBillingParams) error {
		cmd := buildUsageBillingCommand(log.RequestID, log, p)
		if cmd == nil || cmd.RequestFingerprint == "" {
			return ErrChatInvalid
		}
		user := p.User
		extra := map[string]any{}
		for _, name := range []string{"quota_limit", "quota_daily_limit", "quota_weekly_limit", "quota_used", "quota_daily_used", "quota_weekly_used", "quota_notify_total_enabled", "quota_notify_total_threshold", "quota_notify_total_threshold_type", "quota_notify_daily_enabled", "quota_notify_daily_threshold", "quota_notify_daily_threshold_type", "quota_notify_weekly_enabled", "quota_notify_weekly_threshold", "quota_notify_weekly_threshold_type", "quota_notify_extra_emails"} {
			if value, ok := p.Account.Extra[name]; ok {
				extra[name] = value
			}
		}
		receipt := ChatBillingReceipt{Version: 1, Command: cmd, Log: log, Cost: p.Cost, User: APIKeyAuthUserSnapshot{ID: user.ID, Email: user.Email, Username: user.Username, Balance: user.Balance, BalanceNotifyEnabled: user.BalanceNotifyEnabled, BalanceNotifyThresholdType: user.BalanceNotifyThresholdType, BalanceNotifyThreshold: user.BalanceNotifyThreshold, BalanceNotifyExtraEmails: user.BalanceNotifyExtraEmails, TotalRecharged: user.TotalRecharged}, KeyQuota: p.APIKey.Quota, KeyLimits: [3]float64{p.APIKey.RateLimit5h, p.APIKey.RateLimit1d, p.APIKey.RateLimit7d}, GroupID: p.APIKey.GroupID, AccountName: p.Account.Name, AccountPlatform: p.Account.Platform, AccountExtra: extra, AccountRateMultiplier: p.AccountRateMultiplier, SubscriptionBill: p.IsSubscriptionBill, Platform: p.Platform}
		encoded, err := json.Marshal(receipt)
		if err != nil {
			return err
		}
		if len(encoded) > 60<<10 {
			return ErrChatTooLarge
		}
		raw = encoded
		return nil
	}
	if err := s.RecordUsage(ctx, &copyInput); err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, ErrChatAuxiliaryUnavailable
	}
	return raw, nil
}

func decodeChatBillingReceipt(raw json.RawMessage) (*ChatBillingReceipt, error) {
	var r ChatBillingReceipt
	if len(raw) > 64<<10 || json.Unmarshal(raw, &r) != nil || r.Version != 1 || r.Command == nil || r.Log == nil || r.Cost == nil {
		return nil, ErrChatInvalid
	}
	cmd := r.Command
	if !strings.HasPrefix(cmd.RequestID, "chat_auxiliary:") || len(cmd.RequestFingerprint) != 64 || cmd.UserID != r.User.ID || cmd.UserID != r.Log.UserID || cmd.APIKeyID != r.Log.APIKeyID || cmd.AccountID != r.Log.AccountID || cmd.RequestID != r.Log.RequestID {
		return nil, ErrChatInvalid
	}
	return &r, nil
}

func (r *ChatBillingReceipt) post(updater APIKeyQuotaUpdater) *postUsageBillingParams {
	user := r.User
	p := &postUsageBillingParams{Cost: r.Cost, User: &User{ID: user.ID, Email: user.Email, Username: user.Username, Balance: user.Balance, BalanceNotifyEnabled: user.BalanceNotifyEnabled, BalanceNotifyThresholdType: user.BalanceNotifyThresholdType, BalanceNotifyThreshold: user.BalanceNotifyThreshold, BalanceNotifyExtraEmails: user.BalanceNotifyExtraEmails, TotalRecharged: user.TotalRecharged}, APIKey: &APIKey{ID: r.Command.APIKeyID, UserID: user.ID, GroupID: r.GroupID, Quota: r.KeyQuota, RateLimit5h: r.KeyLimits[0], RateLimit1d: r.KeyLimits[1], RateLimit7d: r.KeyLimits[2]}, Account: &Account{ID: r.Command.AccountID, Type: r.Command.AccountType, Name: r.AccountName, Platform: r.AccountPlatform, Extra: r.AccountExtra}, IsSubscriptionBill: r.SubscriptionBill, AccountRateMultiplier: r.AccountRateMultiplier, APIKeyService: updater, Platform: r.Platform}
	if r.Command.SubscriptionID != nil {
		p.Subscription = &UserSubscription{ID: *r.Command.SubscriptionID}
	}
	return p
}

// The callback CASes the persisted receipt only after both the original charge
// and its exact usage row are durable. Notification follows that CAS.
func (s *OpenAIGatewayService) ApplyChatBillingReceipt(ctx context.Context, raw json.RawMessage, updater APIKeyQuotaUpdater, mark func(context.Context) error) error {
	receipt, err := decodeChatBillingReceipt(raw)
	if err != nil {
		return err
	}
	if s.usageBillingRepo == nil || s.usageLogRepo == nil || mark == nil {
		return ErrChatAuxiliaryUnavailable
	}
	p := receipt.post(updater)
	deps := s.billingDeps()
	notifications := deps.balanceNotifyService
	deps.balanceNotifyService = nil
	result, err := applyPreparedUsageBilling(ctx, receipt.Command, p, deps, s.usageBillingRepo)
	if err != nil {
		return err
	}
	if result == nil {
		return errors.New("billing apply returned no result")
	}
	persist, cancel := detachedBillingContext(ctx)
	defer cancel()
	if _, err = s.usageLogRepo.Create(persist, receipt.Log); err != nil {
		return err
	}
	if err = mark(persist); err != nil {
		return err
	}
	if result.Applied && notifications != nil {
		deps.balanceNotifyService = notifications
		notifications.submitUsageNotifications(p, deps, result)
	}
	return nil
}
