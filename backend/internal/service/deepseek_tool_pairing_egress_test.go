//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestSendCCUpstreamRequestDeepSeekToolNormalization(t *testing.T) {
	body := []byte(`{"model":"deepseek-chat","messages":[{"role":"tool","tool_call_id":"orphan","content":"stale"},{"role":"assistant","content":"","provider_state":{"count":9007199254740993},"tool_calls":[{"id":"a"}]},{"role":"system","content":"approval"},{"role":"tool","tool_call_id":"a","content":"done"}]}`)
	for _, platform := range []string{PlatformDeepseek, PlatformOpenAI} {
		t.Run(platform, func(t *testing.T) {
			upstream := newOKChatCompletionsUpstream("deepseek-tool-normalization", deepSeekChatFallbackOKBody)
			svc := &OpenAIGatewayService{cfg: deepSeekChatFallbackTestConfig(), httpUpstream: upstream}
			account := &Account{ID: 1, Platform: platform, Type: AccountTypeAPIKey, Credentials: map[string]any{"base_url": "https://upstream.example"}}
			c := newDeepSeekChatFallbackContext(t, body)
			response, err := svc.sendCCUpstreamRequest(context.Background(), c, account, "https://upstream.example/v1/chat/completions", body, false, "synthetic-key", "", "")
			require.NoError(t, err)
			require.NoError(t, response.Body.Close())
			if platform != PlatformDeepseek {
				require.Equal(t, string(body), string(upstream.lastBody), "other platforms pass through unchanged")
				return
			}
			messages := gjson.GetBytes(upstream.lastBody, "messages").Array()
			require.Len(t, messages, 3)
			require.Equal(t, "assistant", messages[0].Get("role").String())
			require.Equal(t, "tool", messages[1].Get("role").String())
			require.Equal(t, "system", messages[2].Get("role").String())
			require.Equal(t, "9007199254740993", messages[0].Get("provider_state.count").Raw)
			require.Equal(t, deepSeekChatReasoningPlaceholderText, messages[0].Get("reasoning_content").String(), "normalization precedes existing reasoning placeholders")
		})
	}
}
