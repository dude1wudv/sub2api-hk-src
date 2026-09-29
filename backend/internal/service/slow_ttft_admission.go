package service

import (
	"context"
	"log/slog"
	"maps"
	"time"
)

// PostSlotAdmission rechecks protection after acquiring a slot, including slots
// obtained by waiting in a handler. Callers release and reselect on a veto.
// Keeping this hook separate leaves the upstream profit-control gate unchanged.
func (s *OpenAIGatewayService) PostSlotAdmission(ctx context.Context, selected *Account) (*Account, bool, string) {
	latest, vetoed, reason := s.ProfitControlVetoLatest(ctx, selected)
	if vetoed || s == nil {
		return latest, vetoed, reason
	}
	return s.rateLimitService.slowTTFTAdmission(ctx, latest)
}

func (s *GatewayService) PostSlotAdmission(ctx context.Context, selected *Account) (*Account, bool, string) {
	if s == nil {
		return selected, false, ""
	}
	latest, vetoed, reason := s.GatewayProfitControlVetoLatest(ctx, selected)
	if vetoed {
		return latest, true, reason
	}
	return s.rateLimitService.slowTTFTAdmission(ctx, latest)
}

func (s *RateLimitService) slowTTFTAdmission(ctx context.Context, selected *Account) (*Account, bool, string) {
	if s == nil || s.accountRepo == nil || selected == nil {
		return selected, false, ""
	}
	current, err := s.accountRepo.GetByID(ctx, selected.ID)
	if err != nil || current == nil {
		slog.Warn("slow_ttft_admission_refresh_failed", "account_id", selected.ID, "error", err)
		return selected, false, ""
	}
	cfg := current.SlowTTFTConfig()
	if cfg.Enabled && current.SlowTTFTUntil != nil && time.Now().Before(*current.SlowTTFTUntil) && !s.slowTTFTGroupExempt(ctx) {
		return selected, true, "slow_ttft_paused"
	}
	// Preserve the selected account's other admission/billing metadata. Copy the
	// protection policy rather than mutating an account shared with a snapshot or
	// an earlier asynchronous usage task.
	latest := *selected
	latest.Extra = maps.Clone(selected.Extra)
	if latest.Extra == nil {
		latest.Extra = make(map[string]any)
	}
	latest.Extra[SlowTTFTConfigKey] = cfg
	latest.SlowTTFTUntil = current.SlowTTFTUntil
	latest.SlowTTFTReason = current.SlowTTFTReason
	return &latest, false, ""
}
