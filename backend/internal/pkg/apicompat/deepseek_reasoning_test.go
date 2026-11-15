package apicompat

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRepairDeepSeekReasoningPreservesToolSignaturesAndIDs(t *testing.T) {
	var input any
	decoder := json.NewDecoder(strings.NewReader(`[
		{"type":"function_call","id":"rs_sub2api_1","call_id":"call_1","name":"f","arguments":"{}","signature":"opaque","encrypted_content":"tool-value"},
		{"type":"function_call_output","call_id":"call_1","output":"done"},
		{"role":"user","content":"next"},
		{"type":"reasoning","encrypted_content":"remove","summary":[{"text":"keep"}],"future":9007199254740993},
		{"type":"custom_tool_call","call_id":"call_2","input":"raw"}
	]`))
	decoder.UseNumber()
	require.NoError(t, decoder.Decode(&input))
	result, changed := RepairDeepSeekReasoning(input)
	require.True(t, changed)
	items, ok := result.([]any)
	require.True(t, ok)
	require.Len(t, items, 6)
	item := func(index int) map[string]any {
		t.Helper()
		value, ok := items[index].(map[string]any)
		require.True(t, ok)
		return value
	}
	require.Equal(t, "rs_sub2api_2", item(0)["id"])
	require.Equal(t, "opaque", item(1)["signature"])
	require.Equal(t, "tool-value", item(1)["encrypted_content"])
	require.Equal(t, "call_1", item(2)["call_id"])
	require.Equal(t, json.Number("9007199254740993"), item(4)["future"])
	require.NotContains(t, items[4], "encrypted_content")
	_, changed = RepairDeepSeekReasoning(result)
	require.False(t, changed, "repeated normalization must not add items or IDs")
}
