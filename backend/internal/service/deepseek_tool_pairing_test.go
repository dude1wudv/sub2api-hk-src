//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestDropOrphanDeepSeekToolMessages(t *testing.T) {
	body := []byte(`{"messages":[{"role":"tool","tool_call_id":"c1","content":"too early"},{"role":"assistant","tool_calls":[{"id":"c1"}]},{"role":"tool","tool_call_id":"c1","content":"ok"},{"role":"tool","tool_call_id":"orphan"},{"role":"tool","content":"missing id"}]}`)
	out := dropOrphanDeepSeekToolMessages(body)
	messages := gjson.GetBytes(out, "messages").Array()
	require.Len(t, messages, 2)
	require.Equal(t, "assistant", messages[0].Get("role").String())
	require.Equal(t, "ok", messages[1].Get("content").String())
}

func TestNormalizeDeepSeekChatMessages(t *testing.T) {
	for _, tc := range []struct {
		name, input, want string
	}{
		{
			name:  "parallel replies move before intervening messages in call order",
			input: `[{"role":"assistant","reasoning_content":"thinking","tool_calls":[{"id":"a"},{"id":"b"}]},{"role":"system","content":"approved"},{"role":"tool","tool_call_id":"b","content":"B"},{"role":"user","content":"continue"},{"role":"tool","tool_call_id":"a","content":"A"}]`,
			want:  `[{"role":"assistant","reasoning_content":"thinking","tool_calls":[{"id":"a"},{"id":"b"}]},{"role":"tool","tool_call_id":"a","content":"A"},{"role":"tool","tool_call_id":"b","content":"B"},{"role":"system","content":"approved"},{"role":"user","content":"continue"}]`,
		},
		{
			name:  "unanswered calls removed without deleting assistant text",
			input: `[{"role":"assistant","content":"working","tool_calls":[{"id":"a"},{"id":"missing"},{"id":""}]},{"role":"tool","tool_call_id":"a","content":"A"},{"role":"assistant","content":"done","tool_calls":[{"id":"missing2"}]}]`,
			want:  `[{"role":"assistant","content":"working","tool_calls":[{"id":"a"}]},{"role":"tool","tool_call_id":"a","content":"A"},{"role":"assistant","content":"done"}]`,
		},
		{
			name:  "empty unanswered assistant and orphan replies removed",
			input: `[{"role":"tool","tool_call_id":"a"},{"role":"assistant","content":null,"tool_calls":[{"id":"a"}]},{"role":"tool","content":"missing id"},{"role":"user","content":"next"}]`,
			want:  `[{"role":"user","content":"next"}]`,
		},
		{
			name:  "duplicate call ids and replies pair once with last result",
			input: `[{"role":"assistant","tool_calls":[{"id":"a","extra":"first"},{"id":"a","extra":"duplicate"}]},{"role":"tool","tool_call_id":"a","content":"old"},{"role":"tool","tool_call_id":"a","content":"new"}]`,
			want:  `[{"role":"assistant","tool_calls":[{"id":"a","extra":"first"}]},{"role":"tool","tool_call_id":"a","content":"new"}]`,
		},
		{
			name:  "reused id in later assistant turn keeps each result",
			input: `[{"role":"assistant","tool_calls":[{"id":"a"}]},{"role":"tool","tool_call_id":"a","content":"first"},{"role":"assistant","tool_calls":[{"id":"a"}]},{"role":"system","content":"notice"},{"role":"tool","tool_call_id":"a","content":"second"}]`,
			want:  `[{"role":"assistant","tool_calls":[{"id":"a"}]},{"role":"tool","tool_call_id":"a","content":"first"},{"role":"assistant","tool_calls":[{"id":"a"}]},{"role":"tool","tool_call_id":"a","content":"second"},{"role":"system","content":"notice"}]`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := []byte(`{"messages":` + tc.input + `}`)
			out := normalizeDeepSeekChatMessages(dropOrphanDeepSeekToolMessages(body))
			require.JSONEq(t, `{"messages":`+tc.want+`}`, string(out))
			require.Equal(t, string(out), string(normalizeDeepSeekChatMessages(out)), "normalization is idempotent")
		})
	}
}

func TestNormalizeDeepSeekChatMessagesPreservesRawExtensions(t *testing.T) {
	body := []byte(`{"provider_request":{"count":9007199254740993},"messages":[{"role":"assistant", "content":"working", "reasoning_content":"real reasoning", "provider_state":{"opaque":"x","count":9007199254740993},"tool_calls":[{"id":"a","vendor_call":{"exact":9007199254740993}},{"id":"unanswered"}]},{"role":"system","content":"approval","vendor_notice":[1,2]},{"role":"tool","tool_call_id":"a","content":"done","provider_result":{"exact":9007199254740993}}]}`)
	out := normalizeDeepSeekChatMessages(body)
	require.Equal(t, gjson.GetBytes(body, "provider_request").Raw, gjson.GetBytes(out, "provider_request").Raw)
	require.Equal(t, gjson.GetBytes(body, "messages.0.provider_state").Raw, gjson.GetBytes(out, "messages.0.provider_state").Raw)
	require.Equal(t, "real reasoning", gjson.GetBytes(out, "messages.0.reasoning_content").String())
	require.Equal(t, gjson.GetBytes(body, "messages.0.tool_calls.0").Raw, gjson.GetBytes(out, "messages.0.tool_calls.0").Raw)
	require.Equal(t, gjson.GetBytes(body, "messages.2").Raw, gjson.GetBytes(out, "messages.1").Raw)
	require.Equal(t, gjson.GetBytes(body, "messages.1").Raw, gjson.GetBytes(out, "messages.2").Raw)
}

func TestNormalizeDeepSeekChatMessagesNoChangePreservesBytes(t *testing.T) {
	for _, raw := range []string{
		`{"model":"deepseek-v4-pro"}`,
		`{"messages":null}`,
		`{"messages":"not an array"}`,
		`{ "messages" : [ {"role":"user", "content":"hi"}, {"role":"assistant", "tool_calls":[{"id":"a"}],"reasoning_content":"real"}, {"role":"tool", "tool_call_id":"a","content":"done"} ], "vendor":{"exact":9007199254740993} }`,
	} {
		t.Run(raw, func(t *testing.T) {
			body := []byte(raw)
			require.Equal(t, raw, string(dropOrphanDeepSeekToolMessages(body)))
			require.Equal(t, raw, string(normalizeDeepSeekChatMessages(body)))
		})
	}
}
