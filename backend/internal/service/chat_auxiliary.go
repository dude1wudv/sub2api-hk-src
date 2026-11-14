package service

import (
	"context"
	"errors"
	"net/url"
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

func chatAuxiliaryKind(kind string) bool {
	switch kind {
	case "search", "fetch", "asr", "tts", "ocr":
		return true
	}
	return false
}

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

func (s *OpenAIGatewayService) SelectChatAuxiliaryAccount(ctx context.Context, accountID int64) (*Account, error) {
	if s == nil || s.cfg == nil || accountID <= 0 || accountID != s.cfg.Chat.AuxiliaryAccountID {
		return nil, ErrChatAuxiliaryUnavailable
	}
	return NewChatAuxiliaryService(s.cfg, s.accountRepo).account(ctx)
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
		scope.configuredAccountID != s.cfg.Chat.AuxiliaryAccountID || !chatCredentialValid(key, scope.userID, scope.sourceGroupID) {
		return 0, ErrChatAuxiliaryUnavailable
	}
	return scope.configuredAccountID, nil
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
	account, err := s.auxiliary.account(ctx)
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
	if _, err := s.validateUpstreamBaseURL("https://api.deepseek.com"); err != nil {
		return nil, ErrChatAuxiliaryUnavailable
	}
	scope := ctx.Value(chatAuxiliaryContextKey{}).(*chatAuxiliaryScope)
	result, err := s.forwardDeepSeekToolJSON(ctx, c, current, DeepSeekToolKind(kind), body, "https://api.deepseek.com/v1/tools/"+kind, "chat_auxiliary:"+scope.operationID.String())
	if result != nil {
		quantity := int64(result.BillableToolCalls)
		result.ChatAuxiliaryUsage, err = chatAuxiliaryUsage(kind, &quantity, "confirmed")
		result.chatAuxiliaryOnly = true
		result.DeepSeekToolKind = ""
	}
	return result, err
}
