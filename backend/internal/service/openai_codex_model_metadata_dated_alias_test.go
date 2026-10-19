package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCodexManifestDeepSeekDatedAliasesPreserveDefaultReasoning(t *testing.T) {
	const publicModel = "deepseek-v4.1-flash-1017"
	newAccount := func(id int64, upstreamModel string) Account {
		return Account{
			ID: id, Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
			Credentials: map[string]any{"model_mapping": map[string]any{publicModel: upstreamModel}},
		}
	}

	// These are the two live account mapping shapes: account 1109 maps the
	// dated public name to the base model, while account 1112 serves that name
	// directly. With no metadata or explicit model_routing, the group manifest
	// must retain the DeepSeek defaults for both accounts.
	accounts := []Account{
		newAccount(1109, "deepseek-v4.1-flash"),
		newAccount(1112, publicModel),
	}
	body, err := buildCodexModelsManifestForAccounts(PlatformOpenAI, []string{publicModel}, accounts, nil, nil, true)
	require.NoError(t, err)

	model := decodeCodexManifestModels(t, body)[0]
	require.Equal(t, publicModel, model["slug"])
	require.Equal(t, "high", model["default_reasoning_level"])
	require.Equal(t, []string{"low", "high", "max"}, effortsFromManifestModel(t, model))
	require.Equal(t, "deepseek-v4.1-flash", accounts[0].GetMappedModel(publicModel))
	require.Equal(t, publicModel, accounts[1].GetMappedModel(publicModel))
}

func TestCodexManifestDeepSeekDatedAliasBaseAndDateDirections(t *testing.T) {
	for _, tt := range []struct {
		name        string
		publicModel string
		upstream    string
	}{
		{name: "MMDD suffix to base", publicModel: "deepseek-v4.1-flash-1017", upstream: "deepseek-v4.1-flash"},
		{name: "base to MMDD suffix", publicModel: "deepseek-v4.1-flash", upstream: "deepseek-v4.1-flash-1017"},
		{name: "YYYYMMDD suffix to base", publicModel: "deepseek-v4.1-flash-20261017", upstream: "deepseek-v4.1-flash"},
		{name: "base to YYYYMMDD suffix", publicModel: "deepseek-v4.1-flash", upstream: "deepseek-v4.1-flash-20261017"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			accounts := []Account{
				{ID: 1109, Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
					Credentials: map[string]any{"model_mapping": map[string]any{tt.publicModel: tt.upstream}}},
				{ID: 1112, Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
					Credentials: map[string]any{"model_mapping": map[string]any{tt.publicModel: tt.publicModel}}},
			}
			body, err := buildCodexModelsManifestForAccounts(PlatformOpenAI, []string{tt.publicModel}, accounts, nil, nil, true)
			require.NoError(t, err)

			model := decodeCodexManifestModels(t, body)[0]
			require.Equal(t, tt.publicModel, model["slug"])
			require.Equal(t, "high", model["default_reasoning_level"])
			require.Equal(t, []string{"low", "high", "max"}, effortsFromManifestModel(t, model))
			require.Equal(t, tt.upstream, accounts[0].GetMappedModel(tt.publicModel))
			require.Equal(t, tt.publicModel, accounts[1].GetMappedModel(tt.publicModel))
		})
	}
}

func TestCodexManifestDeepSeekDatedAliasDoesNotMergeDifferentFamilies(t *testing.T) {
	const publicModel = "deepseek-v4.1-flash-1017"
	for _, tt := range []struct {
		name     string
		upstream string
	}{
		{name: "pro and flash", upstream: "deepseek-v4.1-pro"},
		{name: "different version", upstream: "deepseek-v4.2-flash"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			accounts := []Account{
				{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
					Credentials: map[string]any{"model_mapping": map[string]any{publicModel: tt.upstream}}},
				{ID: 2, Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
					Credentials: map[string]any{"model_mapping": map[string]any{publicModel: "deepseek-v4.1-flash"}}},
			}
			body, err := buildCodexModelsManifestForAccounts(PlatformOpenAI, []string{publicModel}, accounts, nil, nil, true)
			require.NoError(t, err)

			model := decodeCodexManifestModels(t, body)[0]
			require.Empty(t, effortsFromManifestModel(t, model), "a conflicting model mapping must remain fail-closed")
		})
	}
}

func TestCodexManifestDeepSeekDatedAliasRejectsInvalidDateSuffix(t *testing.T) {
	const publicModel = "deepseek-v4.1-flash-1332"
	accounts := []Account{
		{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
			Credentials: map[string]any{"model_mapping": map[string]any{publicModel: "deepseek-v4.1-flash"}}},
		{ID: 2, Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
			Credentials: map[string]any{"model_mapping": map[string]any{publicModel: "deepseek-v4.1-flash-1332"}}},
	}
	body, err := buildCodexModelsManifestForAccounts(PlatformOpenAI, []string{publicModel}, accounts, nil, nil, true)
	require.NoError(t, err)

	model := decodeCodexManifestModels(t, body)[0]
	require.Empty(t, effortsFromManifestModel(t, model), "an invalid date-like suffix must not be normalized to the base model")
}

func TestCodexManifestNonDeepSeekDatedAliasConflictHasNoReasoningLevels(t *testing.T) {
	const publicModel = "custom-flash-1017"
	accounts := []Account{
		{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
			Credentials: map[string]any{"model_mapping": map[string]any{publicModel: "custom-flash"}}},
		{ID: 2, Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
			Credentials: map[string]any{"model_mapping": map[string]any{publicModel: publicModel}}},
	}
	body, err := buildCodexModelsManifestForAccounts(PlatformOpenAI, []string{publicModel}, accounts, nil, nil, true)
	require.NoError(t, err)

	model := decodeCodexManifestModels(t, body)[0]
	require.Empty(t, effortsFromManifestModel(t, model))
}

func TestCodexManifestDeepSeekDatedAliasHonorsExplicitReasoningMetadata(t *testing.T) {
	const publicModel = "deepseek-v4.1-flash-1017"
	for _, tt := range []struct {
		name        string
		reasoningA  bool
		levelsA     []string
		defaultA    string
		reasoningB  bool
		levelsB     []string
		defaultB    string
		want        []string
		wantDefault string
	}{
		{name: "reasoning disabled", reasoningA: false, reasoningB: false, want: []string{"none"}, wantDefault: "none"},
		{name: "narrow declared levels", reasoningA: true, levelsA: []string{"low", "high"}, defaultA: "low",
			reasoningB: true, levelsB: []string{"high"}, defaultB: "high", want: []string{"high"}, wantDefault: "high"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			accounts := []Account{
				{ID: 1, Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
					Credentials: map[string]any{"model_mapping": map[string]any{publicModel: "deepseek-v4.1-flash"}}},
				{ID: 2, Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
					Credentials: map[string]any{"model_mapping": map[string]any{publicModel: publicModel}}},
			}
			accounts[0].SetUpstreamModelMetadataSnapshot(UpstreamModelMetadataSnapshot{Models: map[string]UpstreamModelMetadata{
				"deepseek-v4.1-flash": {
					ID: "deepseek-v4.1-flash", Reasoning: &tt.reasoningA,
					SupportedReasoningLevels: tt.levelsA, DefaultReasoningLevel: tt.defaultA,
				},
			}})
			accounts[1].SetUpstreamModelMetadataSnapshot(UpstreamModelMetadataSnapshot{Models: map[string]UpstreamModelMetadata{
				publicModel: {
					ID: publicModel, Reasoning: &tt.reasoningB,
					SupportedReasoningLevels: tt.levelsB, DefaultReasoningLevel: tt.defaultB,
				},
			}})
			body, err := buildCodexModelsManifestForAccounts(PlatformOpenAI, []string{publicModel}, accounts, nil, nil, true)
			require.NoError(t, err)

			model := decodeCodexManifestModels(t, body)[0]
			require.Equal(t, tt.want, effortsFromManifestModel(t, model))
			require.Equal(t, tt.wantDefault, model["default_reasoning_level"])
		})
	}
}

func TestCodexManifestDeepSeekDatedAliasPartialMetadataRetainsKnownLimit(t *testing.T) {
	const publicModel = "deepseek-v4.1-flash-1017"
	known := Account{
		ID: 1109, Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
		Credentials: map[string]any{"model_mapping": map[string]any{publicModel: "deepseek-v4.1-flash"}},
	}
	reasoning := true
	known.SetUpstreamModelMetadataSnapshot(UpstreamModelMetadataSnapshot{Models: map[string]UpstreamModelMetadata{
		"deepseek-v4.1-flash": {
			ID: "deepseek-v4.1-flash", Reasoning: &reasoning,
			SupportedReasoningLevels: []string{"high"}, DefaultReasoningLevel: "high",
		},
	}})
	missing := Account{
		ID: 1112, Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
		Credentials: map[string]any{"model_mapping": map[string]any{publicModel: publicModel}},
	}

	body, err := buildCodexModelsManifestForAccounts(PlatformOpenAI, []string{publicModel}, []Account{known, missing}, nil, nil, true)
	require.NoError(t, err)

	model := decodeCodexManifestModels(t, body)[0]
	require.Empty(t, effortsFromManifestModel(t, model), "unknown peer metadata must keep the prior fail-closed empty profile")
	require.NotContains(t, model, "default_reasoning_level")

	known.SetUpstreamModelMetadataSnapshot(UpstreamModelMetadataSnapshot{Models: map[string]UpstreamModelMetadata{
		"deepseek-v4.1-flash": {ID: "deepseek-v4.1-flash", Reasoning: new(bool)},
	}})
	for _, accounts := range [][]Account{{known, missing}, {missing, known}} {
		body, err = buildCodexModelsManifestForAccounts(PlatformOpenAI, []string{publicModel}, accounts, nil, nil, true)
		require.NoError(t, err)
		model = decodeCodexManifestModels(t, body)[0]
		require.Empty(t, effortsFromManifestModel(t, model), "reasoning=false plus unknown metadata remains fail-closed regardless of account order")
		require.NotContains(t, model, "default_reasoning_level")
	}
}
