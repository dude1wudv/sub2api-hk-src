//go:build unit

package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestChatReasoningProfilesReachProtocolRequests(t *testing.T) {
	s, repo, _ := chatGenerationFixture(t)
	snapshot, err := DecodeChatSnapshot(repo.record.ID, repo.record.Snapshot)
	require.NoError(t, err)
	for _, tc := range []struct {
		name, model, transport, normal, deep, recorded string
		thinking, native                               bool
		want                                           map[string]string
		absent                                         []string
	}{
		{name: "responses normal default", model: "gpt-5", transport: "responses", want: map[string]string{"reasoning.effort": "low"}, absent: []string{"reasoning.summary"}},
		{name: "responses deep default", model: "gpt-5", transport: "responses", thinking: true, want: map[string]string{"reasoning.effort": "high", "reasoning.summary": "auto"}},
		{name: "responses custom normal", model: "gpt-5", transport: "responses", normal: "medium", want: map[string]string{"reasoning.effort": "medium"}},
		{name: "completions normal default", model: "gpt-5", transport: "chat_completions", want: map[string]string{"reasoning_effort": "low"}},
		{name: "completions custom deep", model: "gpt-5", transport: "chat_completions", thinking: true, deep: "low", want: map[string]string{"reasoning_effort": "low"}},
		{name: "recorded continuation beats edited profile", model: "gpt-5", transport: "responses", normal: "high", recorded: "medium", want: map[string]string{"reasoning.effort": "medium"}},
		{name: "native normal remains enabled", model: "deepseek-v4.1", transport: "messages", native: true, want: map[string]string{"thinking.type": "enabled", "output_config.effort": "low"}},
		{name: "native deep profile", model: "deepseek-v4.1", transport: "messages", native: true, thinking: true, want: map[string]string{"thinking.type": "enabled", "output_config.effort": "high"}},
		{name: "adaptive Claude normal", model: "claude-sonnet-5-5", transport: "messages", want: map[string]string{"thinking.type": "adaptive", "output_config.effort": "low"}, absent: []string{"thinking.budget_tokens"}},
		{name: "adaptive Claude deep", model: "claude-opus-4-6", transport: "messages", thinking: true, want: map[string]string{"thinking.type": "adaptive", "output_config.effort": "high"}},
		{name: "legacy Claude deep budget", model: "claude-sonnet-4-5", transport: "messages", thinking: true, want: map[string]string{"thinking.type": "enabled", "thinking.budget_tokens": "10240", "max_tokens": "18432"}, absent: []string{"output_config"}},
		{name: "Opus 4.5 budget and effort", model: "anthropic/claude-opus-4-5", transport: "messages", want: map[string]string{"thinking.type": "enabled", "thinking.budget_tokens": "1024", "output_config.effort": "low"}},
		{name: "Gemini 2.5 normal is not disabled", model: "gemini-2.5-flash", transport: "gemini", want: map[string]string{"generationConfig.thinkingConfig.thinkingBudget": "1024", "generationConfig.thinkingConfig.includeThoughts": "false", "generationConfig.maxOutputTokens": "9216"}},
		{name: "Gemini 2.5 deep reserves answer budget", model: "gemini-2.5-flash", transport: "gemini", thinking: true, want: map[string]string{"generationConfig.thinkingConfig.thinkingBudget": "24576", "generationConfig.thinkingConfig.includeThoughts": "true", "generationConfig.maxOutputTokens": "32768"}},
		{name: "Gemini 3 default high", model: "gemini-3-pro", transport: "gemini", thinking: true, want: map[string]string{"generationConfig.thinkingConfig.thinkingLevel": "high"}},
		{name: "Gemini 3.1 configurable medium", model: "gemini-3.1-pro", transport: "gemini", normal: "medium", want: map[string]string{"generationConfig.thinkingConfig.thinkingLevel": "medium"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			preset := &ChatModelPreset{Model: tc.model, NormalReasoningEffort: tc.normal, ThinkingReasoningEffort: tc.deep}
			model := &ChatCatalogModel{Capabilities: ChatCapabilities{Transport: tc.transport, Reasoning: true, NativeDeepSeekExtensions: tc.native}}
			prepared, err := s.buildChatRequest(context.Background(), 7, snapshot, preset, model, ChatRunOptions{Thinking: tc.thinking, ReasoningEffort: tc.recorded}, "send")
			require.NoError(t, err)
			raw, err := json.Marshal(prepared.body)
			require.NoError(t, err)
			for path, want := range tc.want {
				require.Equal(t, want, gjson.GetBytes(raw, path).String(), path)
			}
			for _, path := range tc.absent {
				require.False(t, gjson.GetBytes(raw, path).Exists(), path)
			}
		})
	}
	native := ChatCapabilities{Transport: "messages", Reasoning: true, NativeDeepSeekExtensions: true}
	require.Equal(t, []string{"low", "high"}, chatReasoningEfforts("native-alias", native))
	_, err = s.buildChatRequest(context.Background(), 7, snapshot, &ChatModelPreset{Model: "native-alias", NormalReasoningEffort: "medium"}, &ChatCatalogModel{Capabilities: native}, ChatRunOptions{}, "send")
	require.ErrorIs(t, err, ErrChatCapability)
	for _, transport := range []string{"responses", "chat_completions", "messages", "gemini"} {
		preset := &ChatModelPreset{Model: "gemini-2.5-flash"}
		model := &ChatCatalogModel{Capabilities: ChatCapabilities{Transport: transport}}
		prepared, err := s.buildChatRequest(context.Background(), 7, snapshot, preset, model, ChatRunOptions{}, "send")
		require.NoError(t, err)
		raw, err := json.Marshal(prepared.body)
		require.NoError(t, err)
		for _, path := range []string{"reasoning", "reasoning_effort", "thinking", "output_config", "generationConfig.thinkingConfig"} {
			require.False(t, gjson.GetBytes(raw, path).Exists(), transport+":"+path)
		}
	}
}

func TestChatPresetRejectsInvalidEffortBeforePersistence(t *testing.T) {
	s := &ChatModelService{}
	for _, effort := range []string{"", "none", "disabled", "max", " high", "LOW"} {
		for _, patch := range []ChatPresetPatch{{NormalReasoningEffort: &effort}, {ThinkingReasoningEffort: &effort}} {
			_, err := s.UpdatePreset(context.Background(), 7, uuid.New(), patch)
			require.ErrorIs(t, err, ErrChatInvalid)
		}
	}
}
