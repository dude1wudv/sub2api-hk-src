package service

import (
	"context"
	"sort"
)

func (s *OpenAIGatewayService) independentGroupScheduling(ctx context.Context, _ *int64) bool {
	return slowTTFTGroupRecovered(ctx)
}

// Refresh the complete candidate pool before comparing priorities/load factors.
// A changed low-priority account must not beat a newly promoted account in a
// stale snapshot. Ordinary scheduling keeps its existing snapshot fast path.
func freshIndependentCandidates(ctx context.Context, repo AccountRepository, pool []*Account) ([]*Account, error) {
	ids := make([]int64, 0, len(pool))
	for _, a := range pool {
		ids = append(ids, a.ID)
	}
	accounts, err := repo.GetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range accounts {
		accounts[i] = accountForSlowTTFTContext(ctx, accounts[i])
	}
	return accounts, nil
}

func (s *defaultOpenAIAccountScheduler) tryAcquireIndependent(ctx context.Context, req OpenAIAccountScheduleRequest, order []openAIAccountCandidateScore) (*AccountSelectionResult, bool, error) {
	pool := make([]*Account, 0, len(order))
	for _, c := range order {
		if c.account != nil {
			pool = append(pool, c.account)
		}
	}
	pool, err := freshIndependentCandidates(ctx, s.service.accountRepo, pool)
	if err != nil {
		return nil, false, err
	}
	compactBlocked := false
	eligible := func(a *Account) *Account {
		a = s.service.recheckSelectedOpenAIAccountFromDB(ctx, a, req.GroupID, req.Platform, req.RequestedModel, false, req.RequiredCapability)
		if a == nil || !a.IsSchedulable() || !s.isAccountTransportCompatible(a, req.RequiredTransport) || !s.isAccountRequestCompatible(ctx, a, req) {
			return nil
		}
		if req.RequireCompact && openAICompactSupportTier(a) == 0 {
			compactBlocked = true
			return nil
		}
		if s.service.rateLimitService != nil && s.service.rateLimitService.SlowTTFTPaused(ctx, a.ID) {
			return nil
		}
		return a
	}
	filtered := pool[:0]
	for _, a := range pool {
		if a = eligible(a); a != nil {
			filtered = append(filtered, a)
		}
	}
	pool = filtered
	for len(pool) > 0 {
		a, slot, err := s.service.concurrencyService.AcquireScheduledAccount(ctx, pool, derefGroupID(req.GroupID))
		if err != nil || a == nil {
			return nil, compactBlocked, err
		}
		fresh := eligible(a)
		if fresh != nil && sameIndependentScheduling(a, fresh, derefGroupID(req.GroupID)) {
			if req.SessionHash != "" && !req.PreserveStickyBinding {
				_ = s.service.bindOpenAIStickySessionDuringSelection(ctx, req.GroupID, req.SessionHash, fresh.ID)
			}
			return attachSelectionProfitGate(ctx, &AccountSelectionResult{Account: fresh, Acquired: true, ReleaseFunc: slot.ReleaseFunc}), compactBlocked, nil
		}
		slot.ReleaseFunc()
		next := pool[:0]
		for _, c := range pool {
			if c.ID != a.ID {
				next = append(next, c)
			}
		}
		pool = next
	}
	return nil, compactBlocked, nil
}

func sameIndependentScheduling(a, b *Account, groupID int64) bool {
	return a.Concurrency == b.Concurrency && a.EffectiveLoadFactor() == b.EffectiveLoadFactor() && a.PriorityInGroup(groupID) == b.PriorityInGroup(groupID)
}

func (s *GatewayService) selectIndependentGateway(ctx context.Context, groupID *int64, session string, pool []*Account, model, platform string, mixed bool) (*AccountSelectionResult, error) {
	remaining, err := freshIndependentCandidates(ctx, s.accountRepo, pool)
	if err != nil {
		return nil, err
	}
	eligible := func(a *Account) bool {
		return a != nil && a.IsSchedulable() && s.isAccountInGroup(a, groupID) &&
			s.isAccountAllowedForPlatform(a, platform, mixed) && s.isGatewayAccountProfitEligible(ctx, a) &&
			(model == "" || s.isModelSupportedByAccountWithContext(ctx, a, model)) &&
			(!s.needsUpstreamChannelRestrictionCheck(ctx, groupID) || !s.isUpstreamModelRestrictedByChannel(ctx, *groupID, a, model)) &&
			s.isAccountSchedulableForModelSelection(ctx, a, model) && s.isAccountSchedulableForQuota(a) &&
			s.isAccountSchedulableForWindowCost(ctx, a, false) && s.isAccountSchedulableForRPM(ctx, a, false) &&
			(s.rateLimitService == nil || !s.rateLimitService.SlowTTFTPaused(ctx, a.ID))
	}
	filtered := remaining[:0]
	for _, a := range remaining {
		if eligible(a) {
			filtered = append(filtered, a)
		}
	}
	remaining = filtered
	for len(remaining) > 0 {
		a, slot, err := s.concurrencyService.AcquireScheduledAccount(ctx, remaining, derefGroupID(groupID))
		if err != nil {
			return nil, err
		}
		if a == nil {
			break
		}
		fresh, err := s.accountRepo.GetByID(ctx, a.ID)
		fresh = accountForSlowTTFTContext(ctx, fresh)
		if err == nil && eligible(fresh) && sameIndependentScheduling(a, fresh, derefGroupID(groupID)) && s.checkAndRegisterSession(ctx, fresh, session) {
			if session != "" && s.cache != nil {
				_ = s.bindGatewayStickySessionDuringSelection(ctx, groupID, session, fresh.ID)
			}
			return s.newSelectionResult(ctx, fresh, true, slot.ReleaseFunc, nil)
		}
		slot.ReleaseFunc()
		next := remaining[:0]
		for _, c := range remaining {
			if c.ID != a.ID {
				next = append(next, c)
			}
		}
		remaining = next
	}
	// Capacity exhaustion keeps the existing bounded waiting behavior.
	sort.SliceStable(remaining, func(i, j int) bool {
		return remaining[i].PriorityInGroup(derefGroupID(groupID)) < remaining[j].PriorityInGroup(derefGroupID(groupID))
	})
	cfg := s.schedulingConfig()
	for _, a := range remaining {
		if !eligible(a) || !s.checkAndRegisterSession(ctx, a, session) {
			continue
		}
		return s.newSelectionResult(ctx, a, false, nil, &AccountWaitPlan{AccountID: a.ID, MaxConcurrency: a.Concurrency, Timeout: cfg.FallbackWaitTimeout, MaxWaiting: cfg.FallbackMaxWaiting})
	}
	return nil, ErrNoAvailableAccounts
}
