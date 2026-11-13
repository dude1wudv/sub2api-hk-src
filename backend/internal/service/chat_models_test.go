//go:build unit

package service

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"testing"
)

type chatGroupsStub struct {
	groups []Group
	err    error
}

func (s *chatGroupsStub) GetAvailableGroups(context.Context, int64) ([]Group, error) {
	return s.groups, s.err
}

type chatCatalogStub struct {
	models []GatewayModel
	calls  int
	err    error
}

func (s *chatCatalogStub) ListForGroup(context.Context, *Group) ([]GatewayModel, error) {
	s.calls++
	return s.models, s.err
}

type chatAccountsStub struct {
	accounts  []Account
	calls     int
	platforms []string
}

func (s *chatAccountsStub) ListModelAvailabilityCandidates(_ context.Context, group *int64, platforms []string, _ bool) ([]Account, error) {
	s.calls++
	s.platforms = platforms
	if group == nil || len(platforms) == 0 {
		return nil, errors.New("unscoped account query")
	}
	return s.accounts, nil
}

type chatPresetsStub struct {
	ChatPresetRepository
	presets []ChatModelPreset
	created *ChatModelPreset
}

func (s *chatPresetsStub) ListPresets(context.Context, int64) ([]ChatModelPreset, error) {
	return s.presets, nil
}
func (s *chatPresetsStub) CreatePreset(_ context.Context, user, group int64, model, name string) (*ChatModelPreset, error) {
	s.created = &ChatModelPreset{ID: uuid.New(), UserID: user, GroupID: group, Model: model, DisplayName: name}
	return s.created, nil
}

func chatModelFixture() (*ChatModelService, *chatGroupsStub, *chatCatalogStub, *chatAccountsStub) {
	groups := &chatGroupsStub{groups: []Group{{ID: 11, Name: "first", Platform: PlatformOpenAI, RateMultiplier: 1}}}
	catalog := &chatCatalogStub{models: []GatewayModel{{ID: "shared-model"}, {ID: "text-embedding-3-small"}}}
	accounts := &chatAccountsStub{accounts: []Account{{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true}}}
	return &ChatModelService{groups: groups, catalog: catalog, accounts: accounts}, groups, catalog, accounts
}

func TestChatCatalogPermissionPrecedesDiscovery(t *testing.T) {
	s, groups, catalog, accounts := chatModelFixture()
	for _, id := range []int64{0, -1, 12} {
		_, err := s.Catalog(context.Background(), 7, id, "")
		require.Error(t, err)
	}
	require.Zero(t, catalog.calls)
	require.Zero(t, accounts.calls)
	groups.err = errors.New("permission unavailable")
	_, err := s.Catalog(context.Background(), 7, 11, "")
	require.ErrorIs(t, err, groups.err)
	require.Zero(t, catalog.calls)
}

func TestChatCatalogFiltersAndScopesCandidates(t *testing.T) {
	s, groups, _, accounts := chatModelFixture()
	out, err := s.Catalog(context.Background(), 7, 11, "SHARED")
	require.NoError(t, err)
	require.Len(t, out.Models, 1)
	require.Equal(t, "shared-model", out.Models[0].ID)
	require.Equal(t, []string{PlatformOpenAI}, accounts.platforms)
	require.False(t, out.Models[0].Capabilities.Voice)
	require.False(t, out.Models[0].Capabilities.Search)
	require.NotEmpty(t, out.Models[0].Capabilities.Reasons["search"])
	groups.groups[0].ClaudeCodeOnly = true
	out, err = s.Catalog(context.Background(), 7, 11, "")
	require.NoError(t, err)
	require.Empty(t, out.Models)
}

func TestChatCatalogEmptyScopeAndNonChatAccounts(t *testing.T) {
	s, _, _, accounts := chatModelFixture()
	accounts.accounts[0].Credentials = map[string]any{"openai_capabilities": []any{"embeddings"}}
	out, err := s.Catalog(context.Background(), 7, 11, "")
	require.NoError(t, err)
	require.Empty(t, out.Models)
	accounts.accounts = nil
	out, err = s.Catalog(context.Background(), 7, 11, "")
	require.NoError(t, err)
	require.Empty(t, out.Models)
}

func TestChatPresetsKeepUnavailableAndDuplicatePublicNames(t *testing.T) {
	s, groups, _, _ := chatModelFixture()
	groups.groups = append(groups.groups, Group{ID: 12, Name: "second", Platform: PlatformOpenAI})
	repo := &chatPresetsStub{presets: []ChatModelPreset{{ID: uuid.New(), GroupID: 11, Model: "shared-model"}, {ID: uuid.New(), GroupID: 12, Model: "shared-model"}, {ID: uuid.New(), GroupID: 13, Model: "shared-model"}, {ID: uuid.New(), GroupID: 11, Model: "removed-model"}}}
	s.repo = repo
	out, err := s.ListPresets(context.Background(), 7)
	require.NoError(t, err)
	require.Len(t, out, 4)
	require.True(t, out[0].Available)
	require.True(t, out[1].Available)
	require.NotEqual(t, out[0].Group.Name, out[1].Group.Name)
	require.False(t, out[2].Available)
	require.Empty(t, out[2].Group.Name)
	require.NotEmpty(t, out[2].UnavailableReason)
	require.False(t, out[3].Available)
	require.NotEmpty(t, out[3].UnavailableReason)
	_, err = s.CreatePreset(context.Background(), 7, 13, "shared-model", "")
	require.ErrorIs(t, err, ErrChatNotFound)
	require.Nil(t, repo.created)
	_, err = s.CreatePreset(context.Background(), 7, 12, "shared-model", "mine")
	require.NoError(t, err)
	require.Equal(t, int64(12), repo.created.GroupID)
}

func TestChatCapabilitiesMixedNativeFailClosed(t *testing.T) {
	s, groups, _, accounts := chatModelFixture()
	groups.groups[0].Platform = PlatformDeepseek
	accounts.accounts[0].Platform = PlatformDeepseek
	model := "deepseek-v4-flash"
	native := Account{ID: 1, Platform: PlatformDeepseek, Type: AccountTypeAPIKey, Status: StatusActive, Schedulable: true, Credentials: map[string]any{"api_protocol": "anthropic"}}
	policy := ChatModelPolicy{Transport: "messages", DocumentInput: true, NativeDeepSeekExtensions: true, NativeFiles: true, NativeSearch: true, NativeReadURL: true}
	caps, _, ok, err := s.modelCapabilities(context.Background(), &groups.groups[0], model, ModelMetadata{}, policy, []Account{native})
	require.NoError(t, err)
	require.True(t, ok)
	require.True(t, caps.DocumentInput)
	require.True(t, caps.Search)
	require.False(t, caps.Voice)
	caps, _, ok, err = s.modelCapabilities(context.Background(), &groups.groups[0], model, ModelMetadata{}, policy, append(accounts.accounts, native))
	require.NoError(t, err)
	require.True(t, ok)
	require.False(t, caps.NativeFiles)
	require.False(t, caps.DocumentInput)
	require.False(t, caps.Search)
	require.NotEmpty(t, caps.Reasons["document_input"])
	policy.Transport = "responses"
	_, _, _, err = s.modelCapabilities(context.Background(), &groups.groups[0], model, ModelMetadata{}, policy, []Account{native})
	require.ErrorIs(t, err, ErrChatInvalid)
}

func TestChatSearchRequiresExactToolsAndSameSupportedGroup(t *testing.T) {
	s, groups, _, accounts := chatModelFixture()
	yes := true
	metadata := ModelMetadata{MetadataMatch: "exact", ToolCall: &yes}
	policy := ChatModelPolicy{Transport: "chat_completions"}
	account := accounts.accounts[0]
	check := func(meta ModelMetadata, want bool) {
		t.Helper()
		caps, _, ok, err := s.modelCapabilities(context.Background(), &groups.groups[0], "shared-model", meta, policy, []Account{account})
		require.NoError(t, err)
		require.True(t, ok)
		require.Equal(t, want, caps.Search)
		require.Equal(t, want, caps.ReadURL)
	}
	check(metadata, false)
	account.Extra = map[string]any{deepSeekToolsGroupIDsExtraKey: []int64{12}}
	check(metadata, false)
	account.Extra[deepSeekToolsGroupIDsExtraKey] = []int64{11}
	check(metadata, true)
	metadata.MetadataMatch = "family"
	check(metadata, false)
}

func TestChatPresetValidation(t *testing.T) {
	for _, model := range []string{"", "*", "a?b", " trailing"} {
		require.ErrorIs(t, validateChatPreset(11, model, ""), ErrChatInvalid)
	}
	require.ErrorIs(t, validateChatPreset(0, "m", ""), ErrChatInvalid)
	require.NoError(t, validateChatPreset(11, "provider/model", "我的模型"))
}
