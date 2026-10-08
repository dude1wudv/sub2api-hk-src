package service

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
)

// Anthropic groups may include mixed-scheduling Antigravity accounts. Only
// mappings that actually forward to Claude belong in their smart catalog.
func gatewaySmartRoutingAccountModels(account *Account) []string {
	models := smartRoutingAccountModels(account)
	if account.Platform != PlatformAntigravity {
		return models
	}
	filtered := make([]string, 0, len(models))
	for _, model := range models {
		if strings.HasPrefix(mapAntigravityModel(account, model), "claude-") {
			filtered = append(filtered, model)
		}
	}
	return filtered
}

func gatewaySmartRoutingAccountClaims(account *Account, model string) bool {
	if account == nil {
		return false
	}
	if account.Platform != PlatformAntigravity {
		return smartRoutingAccountClaims(account, model)
	}
	for _, id := range gatewaySmartRoutingAccountModels(account) {
		if id == model {
			return true
		}
	}
	return false
}

func (s *GatewayService) SmartRoutingModels(ctx context.Context, key *APIKey) ([]string, error) {
	ctx = context.WithValue(ctx, ctxkey.Group, key.Group)
	accounts, _, err := s.listSchedulableAccounts(ctx, key.GroupID, PlatformAnthropic, false)
	if err != nil {
		return nil, err
	}
	models := make([]string, 0)
	for i := range accounts {
		for _, model := range gatewaySmartRoutingAccountModels(&accounts[i]) {
			if key.Group.ModelAllowlist.Allows(model) {
				models = append(models, model)
			}
		}
	}
	return dedupeAndSortModelIDs(models), nil
}

// PrepareSmartRouting uses the real Claude scheduler, including sticky and
// session limits. Count-tokens selects an account without acquiring a generation
// slot. The handler consumes this selection once; retries stay in this group.
func (s *GatewayService) PrepareSmartRouting(ctx context.Context, key *APIKey, parsed *ParsedRequest, countTokens bool) (context.Context, bool, error) {
	if !key.Group.ModelAllowlist.Allows(parsed.Model) ||
		(key.Group.ClaudeCodeOnly && !IsClaudeCodeClient(ctx)) {
		return ctx, false, nil
	}
	ctx = context.WithValue(ctx, ctxkey.UserID, key.UserID)
	ctx = context.WithValue(ctx, ctxkey.Group, key.Group)
	ctx = WithSmartRouting(ctx)
	sessionHash := s.GenerateSessionHash(parsed)
	ctx = context.WithValue(ctx, smartRoutingSessionsContextKey{}, &smartRoutingSessions{
		gateway: s, sessionHash: sessionHash, accounts: make(map[int64]*Account),
	})
	ctx, _ = WithGatewayTokenRequestPricing(ctx)
	excluded := make(map[int64]struct{})
	for {
		var selection *AccountSelectionResult
		var err error
		if countTokens {
			var account *Account
			account, err = s.SelectAccountForModelWithExclusions(ctx, key.GroupID, sessionHash, parsed.Model, excluded)
			if account != nil {
				selection = &AccountSelectionResult{Account: account}
			}
		} else {
			selection, err = s.SelectAccountWithLoadAwareness(ctx, key.GroupID, sessionHash, parsed.Model, excluded, parsed.MetadataUserID, key.UserID)
		}
		if err != nil {
			FinishSmartRouting(ctx)
			if errors.Is(err, ErrNoAvailableAccounts) || errors.Is(err, ErrClaudeCodeOnly) {
				return ctx, false, nil
			}
			return ctx, false, err
		}
		if selection == nil || selection.Account == nil {
			FinishSmartRouting(ctx)
			return ctx, false, nil
		}
		account := selection.Account
		// Revalidate the hydrated account: a snapshot must not resurrect a
		// removed mapping or a paused account after the scheduler selected it.
		if !s.isAccountInGroup(account, key.GroupID) ||
			!s.isAccountSchedulableForModelSelection(ctx, account, parsed.Model) ||
			!s.isModelSupportedByAccountWithContext(ctx, account, parsed.Model) {
			if selection.ReleaseFunc != nil {
				selection.ReleaseFunc()
			}
			s.ReleaseAccountSession(context.Background(), account, sessionHash)
			excluded[account.ID] = struct{}{}
			continue
		}
		if release := selection.ReleaseFunc; release != nil {
			var once sync.Once
			selection.ReleaseFunc = func() { once.Do(release) }
		}
		return context.WithValue(ctx, smartRoutingSelectionContextKey{}, &smartRoutingSelection{
			selection: selection, groupID: *key.GroupID, model: parsed.Model,
		}), true, nil
	}
}

type smartRoutingSessionsContextKey struct{}

type smartRoutingSessions struct {
	mu          sync.Mutex
	once        sync.Once
	gateway     *GatewayService
	sessionHash string
	accounts    map[int64]*Account
	served      bool
}

func trackSmartRoutingSession(ctx context.Context, account *Account) {
	if state, ok := ctx.Value(smartRoutingSessionsContextKey{}).(*smartRoutingSessions); ok {
		state.mu.Lock()
		state.accounts[account.ID] = account
		state.mu.Unlock()
	}
}

func MarkSmartRoutingSessionServed(ctx context.Context) {
	if state, ok := ctx.Value(smartRoutingSessionsContextKey{}).(*smartRoutingSessions); ok {
		state.mu.Lock()
		state.served = true
		state.mu.Unlock()
	}
}

// FinishSmartRouting also covers auth rejection and handler early returns,
// before the handler had a chance to install its own session cleanup.
func FinishSmartRouting(ctx context.Context) {
	ReleaseSmartRoutingSelection(ctx)
	if state, ok := ctx.Value(smartRoutingSessionsContextKey{}).(*smartRoutingSessions); ok {
		state.once.Do(func() {
			state.mu.Lock()
			defer state.mu.Unlock()
			if !state.served {
				for _, account := range state.accounts {
					state.gateway.ReleaseAccountSession(context.Background(), account, state.sessionHash)
				}
			}
		})
	}
}
