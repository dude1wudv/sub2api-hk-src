package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

var ErrDeepSeekSuiteUnavailable = errors.New("DeepSeek capability is not enabled for this account and group")

const DeepSeekSuiteCapabilitiesKey = "deepseek_suite_capabilities"

func appendDeepSeekSuiteAudioModels(accounts []Account, groupID *int64, models []string) []string {
	for _, account := range accounts {
		for _, kind := range []string{"asr", "tts"} {
			model := "deepseek-" + kind
			if account.DeepSeekSuiteAllowed(groupID, kind) && !slices.Contains(models, model) {
				models = append(models, model)
			}
		}
	}
	return models
}

// Explicit capabilities are independent of an account's display name and of
// other models sold by the same relay.
func (a *Account) DeepSeekSuiteAllowed(groupID *int64, kind string) bool {
	if !a.DeepSeekToolsAllowedForGroup(groupID) {
		return false
	}
	if kind == "search" || kind == "fetch" {
		return true
	}
	switch values := a.Extra[DeepSeekSuiteCapabilitiesKey].(type) {
	case []string:
		for _, v := range values {
			if v == kind {
				return true
			}
		}
	case []any:
		for _, v := range values {
			if v == kind {
				return true
			}
		}
	}
	return false
}

type deepSeekSuiteScopeKey struct{}
type deepSeekSuiteScope struct {
	Kind      string
	GroupID   int64
	AccountID int64
}

func WithDeepSeekSuiteScope(ctx context.Context, kind string, groupID, accountID int64) context.Context {
	return context.WithValue(ctx, deepSeekSuiteScopeKey{}, deepSeekSuiteScope{kind, groupID, accountID})
}

func deepSeekSuiteAccountAllowed(ctx context.Context, account *Account) bool {
	scope, ok := ctx.Value(deepSeekSuiteScopeKey{}).(deepSeekSuiteScope)
	return !ok || (account != nil && (scope.AccountID == 0 || account.ID == scope.AccountID) &&
		account.DeepSeekSuiteAllowed(&scope.GroupID, scope.Kind))
}

// DeepSeekSuiteConversation reports explicit DeepSeek extensions. Plain model
// requests stay on the established gateway path.
func DeepSeekSuiteConversation(body []byte) bool {
	for _, name := range []string{"safety_check", "title"} {
		if gjson.GetBytes(body, name).Exists() {
			return true
		}
	}
	found := false
	var visit func(gjson.Result)
	visit = func(v gjson.Result) {
		if v.IsObject() {
			if strings.HasPrefix(v.Get("type").String(), "deepseek_") ||
				v.Get("role").String() == "latest_reminder" ||
				v.Get("server_tool_calls").Exists() ||
				v.Get("file_id").Exists() {
				found = true
			}
		}
		if v.IsArray() || v.IsObject() {
			v.ForEach(func(_, child gjson.Result) bool { visit(child); return !found })
		}
	}
	visit(gjson.ParseBytes(body))
	return found
}

// DeepSeekSuiteOperation holds only ownership, file metadata and a credential-
// free billing receipt. Prompts, audio and OCR output are never stored here.
type DeepSeekSuiteOperation struct {
	ID           string          `json:"id"`
	UserID       int64           `json:"user_id"`
	KeyID        int64           `json:"api_key_id"`
	GroupID      int64           `json:"group_id"`
	AccountID    int64           `json:"account_id"`
	Kind         string          `json:"kind"`
	Status       string          `json:"status"`
	RemoteID     string          `json:"-"`
	UpstreamName string          `json:"-"`
	Filename     string          `json:"filename,omitempty"`
	Purpose      string          `json:"purpose,omitempty"`
	Bytes        int64           `json:"bytes,omitempty"`
	Metadata     json.RawMessage `json:"-"`
	Receipt      json.RawMessage `json:"-"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

func (o *DeepSeekSuiteOperation) FileID() string { return "file-ds-" + o.ID }

type DeepSeekSuiteRepository interface {
	CreateDeepSeekOperation(context.Context, *DeepSeekSuiteOperation) error
	SaveDeepSeekOperation(context.Context, *DeepSeekSuiteOperation) error
	GetDeepSeekFile(context.Context, int64, int64, string) (*DeepSeekSuiteOperation, error)
	ListDeepSeekFiles(context.Context, int64, int64, DeepSeekFileListQuery) ([]*DeepSeekSuiteOperation, error)
	ListDeepSeekPending(context.Context) ([]*DeepSeekSuiteOperation, error)
	MarkDeepSeekFileDeleted(context.Context, int64, int64, string) error
}

type DeepSeekFileListQuery struct {
	After, Before, Purpose string
	Limit                  int
	Ascending              bool
}

func (s *OpenAIGatewayService) DeepSeekSuiteRepository() (DeepSeekSuiteRepository, error) {
	repo, ok := s.usageLogRepo.(DeepSeekSuiteRepository)
	if !ok {
		return nil, ErrDeepSeekSuiteUnavailable
	}
	return repo, nil
}

func (s *OpenAIGatewayService) NewDeepSeekOperation(ctx context.Context, key *APIKey, account *Account, kind string) (*DeepSeekSuiteOperation, error) {
	repo, err := s.DeepSeekSuiteRepository()
	if err != nil || key == nil || key.GroupID == nil || account == nil {
		return nil, ErrDeepSeekSuiteUnavailable
	}
	op := &DeepSeekSuiteOperation{ID: uuid.NewString(), UserID: key.UserID, KeyID: key.ID,
		GroupID: *key.GroupID, AccountID: account.ID, Kind: kind, Status: "started",
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	return op, repo.CreateDeepSeekOperation(ctx, op)
}

func (s *OpenAIGatewayService) DeepSeekSuiteAccount(ctx context.Context, key *APIKey, account *Account, kind string) (*Account, error) {
	if key == nil || key.GroupID == nil || account == nil {
		return nil, ErrDeepSeekSuiteUnavailable
	}
	current, err := s.accountRepo.GetByID(ctx, account.ID)
	if err != nil || current == nil || !current.IsSchedulable() ||
		!current.DeepSeekSuiteAllowed(key.GroupID, kind) ||
		!s.openAIAccountMatchesSchedulingGroup(current, key.GroupID) {
		return nil, ErrDeepSeekSuiteUnavailable
	}
	return current, nil
}

func (s *OpenAIGatewayService) deepSeekSuiteURL(account *Account, kind, remoteID string) (string, error) {
	paths := map[string]string{"search": "/v1/tools/search", "fetch": "/v1/tools/fetch",
		"ocr": "/v1/tools/ocr", "asr": "/v1/tools/asr", "tts": "/v1/tools/tts", "tts_ws": "/v1/tools/tts",
		"files": "/v1/files", "chat": "/v1/chat/completions", "responses": "/v1/responses", "messages": "/v1/messages"}
	path, ok := paths[kind]
	if !ok {
		return "", ErrDeepSeekSuiteUnavailable
	}
	base := account.GetOpenAIBaseURL()
	// Optional per-protocol bases use the account's existing credential schema.
	if values, ok := account.Credentials["api_base_urls"].(map[string]any); ok {
		protocol := map[string]string{"chat": APIProtocolChatCompletions, "responses": APIProtocolResponses, "messages": APIProtocolAnthropic, "files": "files"}[kind]
		if v, ok := values[protocol].(string); ok && strings.TrimSpace(v) != "" {
			base = v
		}
	}
	validated, err := s.validateUpstreamBaseURL(base)
	if err != nil {
		return "", err
	}
	target := buildOpenAIEndpointURL(validated, path)
	u, err := url.Parse(validated)
	if err != nil {
		return "", err
	}
	if strings.EqualFold(u.Hostname(), "api.deepseek.com") {
		switch kind {
		case "messages":
			u.Path = "/anthropic/v1/messages"
		case "chat", "responses", "files":
			u.Path = strings.TrimPrefix(path, "/v1")
		default:
			u.Path = path
		}
		u.RawPath, u.Fragment = "", ""
		target = u.String()
	}
	if remoteID != "" {
		if !chatValidRemoteID(remoteID) {
			return "", ErrChatInvalid
		}
		u, err = url.Parse(target)
		if err != nil {
			return "", err
		}
		u.Path += "/" + remoteID
		target = u.String()
	}
	return target, nil
}

// ResolveDeepSeekFiles only rewrites tenant-owned gateway IDs. Never accept a
// provider file ID from a caller or fail over a bound file to another account.
func (s *OpenAIGatewayService) ResolveDeepSeekFiles(ctx context.Context, key *APIKey, body []byte) ([]byte, int64, error) {
	if key == nil || key.GroupID == nil {
		return nil, 0, ErrChatInvalid
	}
	var bound int64
	var walk func(gjson.Result, string) error
	var patches = map[string]string{}
	repo, repoErr := s.DeepSeekSuiteRepository()
	walk = func(value gjson.Result, path string) error {
		var walkErr error
		if !value.IsObject() && !value.IsArray() {
			return nil
		}
		value.ForEach(func(k, v gjson.Result) bool {
			next := strings.NewReplacer(`\`, `\\`, `.`, `\.`).Replace(k.String())
			if path != "" {
				next = path + "." + next
			}
			if value.IsObject() && k.String() == "file_id" {
				id := strings.TrimPrefix(v.String(), "file-ds-")
				if repoErr != nil || id == v.String() {
					walkErr = ErrChatNotFound
					return false
				}
				record, err := repo.GetDeepSeekFile(ctx, key.UserID, *key.GroupID, id)
				if err != nil || !DeepSeekSuiteFileVisible(record) {
					walkErr = ErrChatNotFound
					return false
				}
				if bound != 0 && bound != record.AccountID {
					walkErr = errors.New("files belong to different upstream accounts")
					return false
				}
				bound = record.AccountID
				patches[next] = record.RemoteID
			} else {
				walkErr = walk(v, next)
			}
			return walkErr == nil
		})
		return walkErr
	}
	if err := walk(gjson.ParseBytes(body), ""); err != nil {
		return nil, 0, err
	}
	var err error
	for path, remote := range patches {
		body, err = sjson.SetBytes(body, path, remote)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid file reference: %w", err)
		}
	}
	return body, bound, nil
}
