//go:build unit

package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func chatTestFrames(events ...string) string {
	return "data: " + strings.Join(events, "\r\n\r\ndata: ") + "\r\n\r\n"
}

func TestChatStreamProtocolsSplitUTF8AndTerminal(t *testing.T) {
	fixtures := map[string]string{
		"messages":         chatTestFrames(`{"type":"message_start","message":{"id":"r"}}`, `{"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":""}}`, `{"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"思考"}}`, `{"type":"content_block_stop","index":0}`, `{"type":"content_block_start","index":1,"content_block":{"type":"text","text":""}}`, `{"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"你好🌍"}}`, `{"type":"content_block_stop","index":1}`, `{"type":"message_delta","delta":{"stop_reason":"end_turn"}}`, `{"type":"message_stop"}`),
		"chat_completions": chatTestFrames(`{"choices":[{"delta":{"reasoning_content":"思考"}}]}`, `{"choices":[{"delta":{"content":"你好🌍"}}]}`, `{"choices":[{"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":12,"completion_tokens":3,"prompt_tokens_details":{"cached_tokens":4}}}`, `[DONE]`),
		"responses":        chatTestFrames(`{"type":"response.reasoning_summary_text.delta","item_id":"a","delta":"思考"}`, `{"type":"response.output_text.delta","item_id":"b","content_index":0,"delta":"你好🌍"}`, `{"type":"response.output_text.annotation.added","item_id":"b","content_index":0,"annotation":{"type":"url_citation","url":"https://example.com","title":"引用"}}`, `{"type":"response.completed","response":{"status":"completed","usage":{"input_tokens":12,"output_tokens":3}}}`),
		"gemini":           chatTestFrames(`{"candidates":[{"index":0,"content":{"role":"model","parts":[{"text":"思考","thought":true,"thoughtSignature":"opaque"}]}}]}`, `{"candidates":[{"index":0,"content":{"role":"model","parts":[{"text":"你好🌍"}]},"finishReason":"STOP"}]}`, `{"usageMetadata":{"promptTokenCount":12,"candidatesTokenCount":3,"cachedContentTokenCount":4,"thoughtsTokenCount":2}}`),
	}
	for transport, fixture := range fixtures {
		t.Run(transport, func(t *testing.T) {
			state := newChatReplyAccumulator(map[string]any{"blocks": []any{}})
			events := []string{}
			decoder := newChatStreamDecoder(transport, 0, func(event map[string]any) error {
				events = append(events, chatString(event["type"]))
				return state.accept(event)
			})
			writer := newChatGatewayWriter(context.Background(), decoder)
			for _, b := range []byte(fixture) {
				_, err := writer.Write([]byte{b})
				require.NoError(t, err)
			}
			require.NoError(t, writer.finish())
			require.Len(t, state.blocks, 2)
			require.Equal(t, "思考", chatObject(state.blocks[0])["thinking"])
			require.Equal(t, "你好🌍", chatObject(state.blocks[1])["text"])
			require.NotContains(t, events, "message_stop", "upstream completion must wait for canonical persistence")
			if transport == "responses" {
				require.Len(t, chatObject(state.blocks[1])["citations"], 1)
			}
			if transport == "chat_completions" {
				require.Equal(t, 8, chatObject(state.message["usage"])["input_tokens"])
			}
		})
	}
}

func TestChatStreamRejectsMissingTerminatorAndErrors(t *testing.T) {
	for _, transport := range []string{"messages", "chat_completions", "responses", "gemini"} {
		decoder := newChatStreamDecoder(transport, 0, func(map[string]any) error { return nil })
		writer := newChatGatewayWriter(context.Background(), decoder)
		require.ErrorIs(t, writer.finish(), ErrChatStream)
		writer = newChatGatewayWriter(context.Background(), decoder)
		_, err := writer.Write([]byte(chatTestFrames(`{"error":{"message":"private upstream detail"}}`)))
		require.ErrorIs(t, err, ErrChatStream)
		require.NotContains(t, err.Error(), "private")
		_ = writer.finish()
	}
}

func TestChatStreamNativeClientToolsDoNotCorruptDisplayIndexes(t *testing.T) {
	state := newChatReplyAccumulator(map[string]any{"blocks": []any{}})
	decoder := newChatStreamDecoder("messages", 0, state.accept)
	writer := newChatGatewayWriter(context.Background(), decoder)
	_, err := writer.Write([]byte(chatTestFrames(`{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"tool-a","name":"chat_search","input":{}}}`, `{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"query\":\"hello\"}"}}`, `{"type":"content_block_stop","index":0}`, `{"type":"content_block_start","index":1,"content_block":{"type":"text","text":"checking"}}`, `{"type":"content_block_stop","index":1}`, `{"type":"message_stop"}`)))
	require.NoError(t, err)
	require.NoError(t, writer.finish())
	require.Len(t, state.blocks, 1)
	require.Len(t, decoder.calls(), 1)
	require.JSONEq(t, `{"query":"hello"}`, decoder.calls()[0].Arguments)
}

func TestChatStreamRefusalAndUnknownBlockFieldsSurvive(t *testing.T) {
	state := newChatReplyAccumulator(map[string]any{"blocks": []any{}})
	decoder := newChatStreamDecoder("messages", 0, state.accept)
	require.NoError(t, decoder.accept([]byte(`{"type":"content_block_start","index":0,"content_block":{"type":"web_fetch_tool_result","kind":"read_link","signature":"opaque","future":{"keep":true}}}`)))
	require.NoError(t, decoder.accept([]byte(`{"type":"message_delta","delta":{"stop_reason":"refusal","stop_details":{"type":"refusal"},"title":"标题"}}`)))
	raw, err := json.Marshal(state.message)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"future":{"keep":true}`)
	require.Equal(t, "refusal", state.message["stopReason"])
	require.Equal(t, "标题", state.title)
}
