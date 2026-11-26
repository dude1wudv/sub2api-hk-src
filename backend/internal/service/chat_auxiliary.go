package service

import (
	"context"
	"errors"
	"net/url"
	"slices"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var ErrChatAuxiliaryUnavailable = errors.New("chat auxiliary service unavailable")

type chatAuxiliaryContextKey struct{}

// Only the authenticated chat BFF can mint this scope. It never crosses the wire.
type chatAuxiliaryScope struct {
	userID              int64
	presetID            uuid.UUID
	sourceGroupID       int64
	configuredAccountID int64
	operationID         uuid.UUID
	kind                string
	parse               *ChatParseRecord
	chat                *ChatService
	capabilities        ChatCapabilities
}

type ChatAuxiliaryService struct {
	cfg      *config.Config
	accounts AccountRepository
}

func NewChatAuxiliaryService(cfg *config.Config, accounts AccountRepository) *ChatAuxiliaryService {
	return &ChatAuxiliaryService{cfg: cfg, accounts: accounts}
}

var chatAuxiliaryKinds = []string{"search", "fetch", "asr", "tts", "ocr"}

func chatAuxiliaryKind(kind string) bool { return slices.Contains(chatAuxiliaryKinds, kind) }

func chatAuxiliaryAccountValid(account *Account) bool {
	return account != nil && chatDeepSeekAccountAtBaseURL(account, account.GetOpenAIBaseURL()) && account.IsSchedulable()
}

func chatNativeDeepSeekAccount(account *Account) bool {
	return account != nil && chatDeepSeekAccountAtBaseURL(account, account.GetAnthropicProtocolBaseURL())
}

func chatDeepSeekAccountAtBaseURL(account *Account, raw string) bool {
	if account == nil || account.ID <= 0 || account.Type != AccountTypeAPIKey ||
		(account.Platform != PlatformOpenAI && account.Platform != PlatformDeepseek) ||
		strings.TrimSpace(account.GetCredential("api_key")) == "" {
		return false
	}
	u, err := url.Parse(raw)
	if err != nil || u == nil || u.Scheme != "https" || !strings.EqualFold(u.Hostname(), "api.deepseek.com") ||
		(u.Port() != "" && u.Port() != "443") || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || u.RawPath != "" {
		return false
	}
	switch u.Path {
	case "", "/", "/v1", "/v1/", "/anthropic", "/anthropic/":
		return true
	}
	return false
}

func (s *ChatAuxiliaryService) account(ctx context.Context) (*Account, error) {
	if s == nil || s.cfg == nil || s.cfg.Chat.AuxiliaryAccountID <= 0 || s.accounts == nil {
		return nil, ErrChatAuxiliaryUnavailable
	}
	a, err := s.accounts.GetByID(ctx, s.cfg.Chat.AuxiliaryAccountID)
	if err != nil || !chatAuxiliaryAccountValid(a) || a.ID != s.cfg.Chat.AuxiliaryAccountID {
		return nil, ErrChatAuxiliaryUnavailable
	}
	return a, nil
}

func (s *ChatAuxiliaryService) accountFor(ctx context.Context, groupID int64, kind string) (*Account, error) {
	if !chatAuxiliaryKind(kind) || groupID <= 0 {
		return nil, ErrChatAuxiliaryUnavailable
	}
	accounts, err := s.suiteAccounts(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if account := chatSuiteAccountFor(accounts, groupID, kind); account != nil {
		return account, nil
	}
	return s.account(ctx)
}

func (s *ChatAuxiliaryService) availableKinds(ctx context.Context, groupID int64) (map[string]bool, error) {
	available := make(map[string]bool, len(chatAuxiliaryKinds))
	accounts, err := s.suiteAccounts(ctx, groupID)
	if err != nil {
		return nil, err
	}
	for _, kind := range chatAuxiliaryKinds {
		if chatSuiteAccountFor(accounts, groupID, kind) != nil {
			available[kind] = true
		}
	}
	if len(available) < len(chatAuxiliaryKinds) {
		if _, err := s.account(ctx); err == nil {
			for _, kind := range chatAuxiliaryKinds {
				available[kind] = true
			}
		}
	}
	return available, nil
}

func chatSuiteAccountFor(accounts []Account, groupID int64, kind string) *Account {
	for index := range accounts {
		if chatSuiteAccountAllowed(&accounts[index], groupID, kind) {
			return &accounts[index]
		}
	}
	return nil
}

func (s *ChatAuxiliaryService) suiteAccounts(ctx context.Context, groupID int64) ([]Account, error) {
	if s == nil || s.accounts == nil {
		return nil, nil
	}
	return s.accounts.ListSchedulableByGroupIDAndPlatforms(ctx, groupID, []string{PlatformOpenAI, PlatformDeepseek})
}

func chatSuiteKind(kind string) string {
	if kind == "tts" {
		return "tts_ws"
	}
	return kind
}

func chatSuiteAccountAllowed(account *Account, groupID int64, kind string) bool {
	return account != nil && account.ID > 0 && account.IsSchedulable() &&
		strings.TrimSpace(account.GetCredential("api_key")) != "" &&
		openAIStickyAccountMatchesGroup(account, &groupID) &&
		account.DeepSeekSuiteAllowed(&groupID, chatSuiteKind(kind))
}

func (s *OpenAIGatewayService) SelectChatAuxiliaryAccount(ctx context.Context, accountID int64) (*Account, error) {
	scope, _ := ctx.Value(chatAuxiliaryContextKey{}).(*chatAuxiliaryScope)
	if s == nil || s.cfg == nil || s.accountRepo == nil || scope == nil || accountID <= 0 ||
		scope.configuredAccountID != accountID || !chatAuxiliaryKind(scope.kind) {
		return nil, ErrChatAuxiliaryUnavailable
	}
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil || account == nil || account.ID != accountID ||
		(!chatSuiteAccountAllowed(account, scope.sourceGroupID, scope.kind) &&
			(accountID != s.cfg.Chat.AuxiliaryAccountID || !chatAuxiliaryAccountValid(account))) {
		return nil, ErrChatAuxiliaryUnavailable
	}
	return account, nil
}

// HasChatAuxiliaryScope is for the pre-auth route guard, not an authorization check.
func HasChatAuxiliaryScope(ctx context.Context) bool {
	scope, ok := ctx.Value(chatAuxiliaryContextKey{}).(*chatAuxiliaryScope)
	return ok && scope != nil && scope.operationID != uuid.Nil && chatAuxiliaryKind(scope.kind)
}

func (s *OpenAIGatewayService) ValidateChatAuxiliaryScope(ctx context.Context, key *APIKey, kind string) (int64, error) {
	scope, ok := ctx.Value(chatAuxiliaryContextKey{}).(*chatAuxiliaryScope)
	if !ok || scope == nil || s == nil || s.cfg == nil || !chatAuxiliaryKind(kind) || scope.kind != kind ||
		scope.presetID == uuid.Nil || scope.operationID == uuid.Nil || scope.configuredAccountID <= 0 ||
		!chatCredentialValid(key, scope.userID, scope.sourceGroupID) {
		return 0, ErrChatAuxiliaryUnavailable
	}
	return scope.configuredAccountID, nil
}

func (s *OpenAIGatewayService) chatAuxiliaryTarget(ctx context.Context, account *Account, kind string) (string, error) {
	scope, _ := ctx.Value(chatAuxiliaryContextKey{}).(*chatAuxiliaryScope)
	if scope != nil && chatSuiteAccountAllowed(account, scope.sourceGroupID, kind) {
		return s.deepSeekSuiteURL(account, chatSuiteKind(kind), "")
	}
	if account == nil || account.ID != s.cfg.Chat.AuxiliaryAccountID || !chatAuxiliaryAccountValid(account) {
		return "", ErrChatAuxiliaryUnavailable
	}
	const base = "https://api.deepseek.com"
	if _, err := s.validateUpstreamBaseURL(base); err != nil {
		return "", err
	}
	return base + "/v1/tools/" + kind, nil
}

func (s *ChatService) resolveChatAuxiliary(ctx context.Context, userID int64, presetID, sessionID uuid.UUID, kind string) (*APIKey, *chatAuxiliaryScope, error) {
	if s == nil || s.models == nil || s.credentials == nil || !chatAuxiliaryKind(kind) {
		return nil, nil, ErrChatUnavailable
	}
	preset, model, _, err := s.models.ResolvePreset(ctx, userID, presetID)
	if err != nil {
		return nil, nil, err
	}
	if sessionID != uuid.Nil {
		if _, err := s.repo.GetSession(ctx, userID, sessionID); err != nil {
			return nil, nil, err
		}
	}
	key, err := s.credentials.Ensure(ctx, userID, preset.GroupID)
	if err != nil {
		return nil, nil, err
	}
	account, err := s.auxiliary.accountFor(ctx, preset.GroupID, kind)
	if err != nil {
		return nil, nil, err
	}
	return key, &chatAuxiliaryScope{userID: userID, presetID: presetID, sourceGroupID: preset.GroupID, configuredAccountID: account.ID, operationID: uuid.New(), kind: kind, capabilities: model.Capabilities}, nil
}

func (s *OpenAIGatewayService) ForwardChatAuxiliaryJSON(ctx context.Context, c *gin.Context, key *APIKey, account *Account, kind string, body []byte) (*OpenAIForwardResult, error) {
	id, err := s.ValidateChatAuxiliaryScope(ctx, key, kind)
	if err != nil || account == nil || id != account.ID {
		return nil, ErrChatAuxiliaryUnavailable
	}
	if kind != "search" && kind != "fetch" {
		return nil, ErrChatInvalid
	}
	current, err := s.SelectChatAuxiliaryAccount(ctx, id)
	if err != nil {
		return nil, err
	}
	if _, err := ValidateDeepSeekToolRequest(DeepSeekToolKind(kind), body); err != nil {
		return nil, err
	}
	scope, ok := ctx.Value(chatAuxiliaryContextKey{}).(*chatAuxiliaryScope)
	if !ok || scope == nil {
		return nil, ErrChatAuxiliaryUnavailable
	}
	target, err := s.chatAuxiliaryTarget(ctx, current, kind)
	if err != nil {
		return nil, ErrChatAuxiliaryUnavailable
	}
	result, err := s.forwardDeepSeekToolJSON(ctx, c, current, DeepSeekToolKind(kind), body, target, "chat_auxiliary:"+scope.operationID.String())
	if err == nil && result != nil {
		quantity := int64(result.BillableToolCalls)
		result.ChatAuxiliaryUsage, err = chatAuxiliaryUsage(kind, &quantity, "confirmed")
		result.chatAuxiliaryOnly = true
		result.DeepSeekToolKind = ""
	}
	return result, err
}
