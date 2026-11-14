//go:build unit

package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestChatDefaultTransportsAndExplicitPolicy(t *testing.T) {
	for model, want := range map[string]string{"claude-sonnet-4-5": "messages", "vendor/Claude-Sonnet": "messages", "anthropic.claude-sonnet": "messages", "gemini-2.5-flash": "gemini", "gpt-5": "responses", "deepseek-v4.1-flash": "responses", "custom-alias": "responses"} {
		require.Equal(t, want, defaultChatTransport(model), model)
	}
	s, groups, catalog, accounts := chatModelFixture()
	catalog.models = []GatewayModel{{ID: "gemini-2.5-flash"}}
	out, err := s.Catalog(context.Background(), 7, 11, "")
	require.NoError(t, err)
	require.Empty(t, out.Models, "native Gemini cannot silently fall back on an OpenAI group")
	groups.groups[0].Platform = PlatformGemini
	accounts.accounts[0].Platform = PlatformGemini
	out, err = s.Catalog(context.Background(), 7, 11, "")
	require.NoError(t, err)
	require.Len(t, out.Models, 1)
	require.Equal(t, "gemini", out.Models[0].Capabilities.Transport)
	require.False(t, out.Models[0].Capabilities.Search)
	require.False(t, out.Models[0].Capabilities.DocumentInput)
	groups.groups[0].ChatModelPolicies = map[string]ChatModelPolicy{"gemini-2.5-flash": {Transport: "responses"}}
	out, err = s.Catalog(context.Background(), 7, 11, "")
	require.NoError(t, err)
	require.Equal(t, "responses", out.Models[0].Capabilities.Transport)
	require.NoError(t, domain.ValidateChatModelPolicies(map[string]ChatModelPolicy{"gemini-2.5-flash": {Transport: "gemini"}}))
	require.Error(t, domain.ValidateChatModelPolicies(map[string]ChatModelPolicy{"gemini-2.5-flash": {Transport: "gemini", NativeSearch: true}}))
}

func TestChatGenerateDefaultRoutesPersist(t *testing.T) {
	for _, test := range []struct{ model, platform, path, frames string }{
		{"gpt-5", PlatformOpenAI, "/v1/responses", chatTestFrames(`{"type":"response.output_text.delta","item_id":"a","delta":"回答"}`, `{"type":"response.completed","response":{"usage":{"input_tokens":3,"output_tokens":2}}}`)},
		{"claude-sonnet-4-5", PlatformAnthropic, "/v1/messages", chatTestFrames(`{"type":"content_block_start","index":0,"content_block":{"type":"text","text":"回答"}}`, `{"type":"content_block_stop","index":0}`, `{"type":"message_delta","delta":{"stop_reason":"end_turn"}}`, `{"type":"message_stop"}`)},
		{"gemini-2.5-flash", PlatformGemini, "/v1beta/models/gemini-2.5-flash:streamGenerateContent", chatTestFrames(`{"candidates":[{"content":{"role":"model","parts":[{"text":"回答","thoughtSignature":"opaque"}]},"finishReason":"STOP"}]}`, `{"usageMetadata":{"promptTokenCount":3,"candidatesTokenCount":2}}`)},
	} {
		t.Run(test.model, func(t *testing.T) {
			s, repo, request := chatGenerationFixture(t)
			groups := s.models.groups.(*chatGroupsStub)
			groups.groups[0].Platform, groups.groups[0].ChatModelPolicies = test.platform, nil
			s.models.catalog.(*chatCatalogStub).models = []GatewayModel{{ID: test.model}}
			s.models.accounts.(*chatAccountsStub).accounts[0].Platform = test.platform
			s.models.repo.(*chatGenerationPresets).preset.Model = test.model
			calls := 0
			s.dispatch = func(w http.ResponseWriter, r *http.Request) {
				calls++
				require.Equal(t, test.path, r.URL.Path)
				require.Len(t, repo.requestIDs, 1)
				var body map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				if test.platform == PlatformGemini {
					require.Equal(t, "alt=sse", r.URL.RawQuery)
					require.NotContains(t, body, "stream")
					require.NotContains(t, body, "model")
					require.Equal(t, "user", chatObject(body["contents"].([]any)[0])["role"])
				}
				_, _ = w.Write([]byte(test.frames))
			}
			recorder := httptest.NewRecorder()
			require.NoError(t, s.Generate(context.Background(), 7, repo.record.ID, request, "192.0.2.1", recorder))
			require.Equal(t, 1, calls)
			require.Equal(t, "done", repo.run.Status)
			require.Contains(t, string(repo.record.Snapshot), "回答")
			require.Contains(t, recorder.Body.String(), "event: chat_saved")
			if test.platform == PlatformGemini {
				require.Contains(t, string(repo.record.Snapshot), `"signature":"opaque"`)
			}
		})
	}
}

func TestChatGeminiPathsAreClosed(t *testing.T) {
	require.True(t, isChatGeminiPath("/v1beta/models/gemini-2.5-flash:streamGenerateContent?alt=sse"))
	for _, path := range []string{"/v1beta/models/../x:streamGenerateContent?alt=sse", "/v1beta/models/gemini%2fsecret:streamGenerateContent?alt=sse", "/v1beta/models/gemini:generateContent?alt=sse", "/v1beta/models/gemini:streamGenerateContent?alt=sse&key=x", "/v1beta/models/gemini:streamGenerateContent", "/v1beta/cachedContents", "https://evil.test/v1beta/models/gemini:streamGenerateContent?alt=sse"} {
		require.False(t, isChatGeminiPath(path), path)
	}
}

func TestChatGeminiThinkingUsesModelRules(t *testing.T) {
	require.Nil(t, chatGeminiThinking("unknown-alias", true))
	require.Equal(t, 0, chatGeminiThinking("gemini-2.5-flash", false)["thinkingBudget"])
	require.Equal(t, 128, chatGeminiThinking("gemini-2.5-pro", false)["thinkingBudget"])
	require.Equal(t, "low", chatGeminiThinking("gemini-3-pro", false)["thinkingLevel"])
	require.NotContains(t, chatGeminiThinking("gemini-3-pro", true), "thinkingBudget")
	require.Equal(t, true, chatGeminiThinking("gemini-3-pro", true)["includeThoughts"])
	require.NotContains(t, chatGeminiThinking("gemini-3-pro-high", false), "thinkingLevel")
}

func TestChatGeminiRefusalsAndUnexpectedTools(t *testing.T) {
	for _, frame := range []string{`{"promptFeedback":{"blockReason":"SAFETY"}}`, `{"candidates":[{"finishReason":"LANGUAGE"}]}`, `{"candidates":[{"finishReason":"IMAGE_PROHIBITED_CONTENT"}]}`} {
		state := newChatReplyAccumulator(map[string]any{"blocks": []any{}})
		writer := newChatGatewayWriter(context.Background(), newChatStreamDecoder("gemini", 0, state.accept))
		_, err := writer.Write([]byte(chatTestFrames(frame)))
		require.NoError(t, err)
		require.NoError(t, writer.finish())
		require.Equal(t, "refusal", state.message["stopReason"])
	}
	for _, frame := range []string{`{"promptFeedback":{"blockReason":"BLOCKED_REASON_UNSPECIFIED"}}`, `{"candidates":[{"content":{"parts":[{"text":"partial"}]}}]}`} {
		writer := newChatGatewayWriter(context.Background(), newChatStreamDecoder("gemini", 0, func(map[string]any) error { return nil }))
		_, err := writer.Write([]byte(chatTestFrames(frame)))
		require.NoError(t, err)
		require.ErrorIs(t, writer.finish(), ErrChatStream)
	}
	decoder := newChatStreamDecoder("gemini", 0, func(map[string]any) error { return nil })
	require.ErrorIs(t, decoder.accept([]byte(`{"candidates":[{"content":{"parts":[{"functionCall":{"name":"chat_search","args":{"query":"x"}}}]}}]}`)), ErrChatCapability)
}

func TestChatGeminiSignatureRoundTripAndRouteBinding(t *testing.T) {
	state := newChatReplyAccumulator(map[string]any{"blocks": []any{}})
	decoder := newChatStreamDecoder("gemini", 0, state.accept)
	writer := newChatGatewayWriter(context.Background(), decoder)
	_, err := writer.Write([]byte(chatTestFrames(`{"candidates":[{"content":{"parts":[{"text":"answer"}]}}]}`, `{"candidates":[{"content":{"parts":[{"thoughtSignature":"opaque"}]},"finishReason":"STOP"}]}`, `{"usageMetadata":{"promptTokenCount":12,"cachedContentTokenCount":4,"candidatesTokenCount":3,"thoughtsTokenCount":2}}`)))
	require.NoError(t, err)
	require.NoError(t, writer.finish())
	require.Equal(t, 8, chatObject(state.message["usage"])["input_tokens"])
	require.Equal(t, 5, chatObject(state.message["usage"])["output_tokens"])
	raw, err := json.Marshal(state.blocks[0])
	require.NoError(t, err)
	message := ChatMessage{ID: "signed", Role: "assistant", Status: "done", Blocks: []json.RawMessage{raw}}
	parts := chatGeminiAssistantParts(message)
	require.Equal(t, "opaque", chatObject(parts[0])["thoughtSignature"])
	s, repo, _ := chatGenerationFixture(t)
	preset := &ChatModelPreset{GroupID: 11, Model: "gemini-2.5-flash"}
	snapshot := &ChatSnapshot{ID: repo.record.ID, Linear: true, Messages: []ChatMessage{{ID: "old", Role: "user", Content: "old"}, message, {ID: "q", Role: "user", Content: "next"}, {ID: "a", Role: "assistant", Status: "pending"}}}
	model := &ChatCatalogModel{Capabilities: ChatCapabilities{Transport: "gemini"}}
	repo.run = &ChatRunRecord{GroupID: 11, Model: preset.Model, Options: json.RawMessage(`{"transport":"gemini"}`)}
	prepared, err := s.buildChatRequest(context.Background(), 7, snapshot, preset, model, ChatRunOptions{}, "send")
	require.NoError(t, err)
	wire, err := json.Marshal(prepared.body)
	require.NoError(t, err)
	require.Contains(t, string(wire), `"thoughtSignature":"opaque"`)
	repo.run.GroupID = 12
	_, err = s.buildChatRequest(context.Background(), 7, snapshot, preset, model, ChatRunOptions{}, "send")
	require.ErrorIs(t, err, ErrChatProviderContext)
	repo.run.GroupID, repo.run.Options = 11, json.RawMessage(`{"transport":"responses"}`)
	_, err = s.buildChatRequest(context.Background(), 7, snapshot, preset, model, ChatRunOptions{}, "send")
	require.ErrorIs(t, err, ErrChatProviderContext)
}
