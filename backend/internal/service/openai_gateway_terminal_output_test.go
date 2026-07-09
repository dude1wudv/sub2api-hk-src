package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestResponsesStreamEventMayContributeToOutput_ReasoningText(t *testing.T) {
	assert.True(t, responsesStreamEventMayContributeToOutput("response.reasoning_text.delta"))
	assert.True(t, responsesStreamEventMayContributeToOutput("response.reasoning_summary_text.delta"))
	assert.True(t, responsesStreamEventMayContributeToOutput("response.custom_tool_call_input.delta"))
	assert.False(t, responsesStreamEventMayContributeToOutput("response.reasoning_text.done"))
}

func TestReconstructResponseOutputFromSSE_ReasoningTextDelta(t *testing.T) {
	body := joinSSEData(
		`{"type":"response.reasoning_text.delta","output_index":0,"delta":"think step one"}`,
		`{"type":"response.reasoning_text.delta","output_index":0,"delta":" then two"}`,
		`{"type":"response.completed","response":{"id":"resp_1","status":"completed","output":[]}}`,
	)

	outputJSON, ok := reconstructResponseOutputFromSSE(body)
	require.True(t, ok)
	require.True(t, gjson.ValidBytes(outputJSON))

	require.Equal(t, 1, int(gjson.GetBytes(outputJSON, "#").Int()))
	assert.Equal(t, "reasoning", gjson.GetBytes(outputJSON, "0.type").String())
	assert.Equal(t, "think step one then two", gjson.GetBytes(outputJSON, "0.summary.0.text").String())
}

func TestNormalizeResponsesStreamingTerminalOutput_FillsFromReasoningText(t *testing.T) {
	body := joinSSEData(
		`{"type":"response.reasoning_text.delta","output_index":0,"delta":"visible thinking"}`,
	)
	acc := mustAccumulatorFromSSE(t, body)
	terminal := []byte(`{"type":"response.completed","response":{"id":"resp_empty","status":"completed","output":[]}}`)
	updated, normalized := normalizeResponsesStreamingTerminalOutput(terminal, acc, nil)
	require.True(t, normalized)
	assert.Equal(t, "reasoning", gjson.GetBytes(updated, "response.output.0.type").String())
	assert.Equal(t, "visible thinking", gjson.GetBytes(updated, "response.output.0.summary.0.text").String())
}

func joinSSEData(payloads ...string) string {
	var b strings.Builder
	for _, p := range payloads {
		b.WriteString("data: ")
		b.WriteString(p)
		b.WriteString("\n\n")
	}
	return b.String()
}

func mustAccumulatorFromSSE(t *testing.T, bodyText string) *apicompat.BufferedResponseAccumulator {
	t.Helper()
	acc := apicompat.NewBufferedResponseAccumulator()
	forEachOpenAISSEDataPayload(bodyText, func(data []byte) {
		eventType := strings.TrimSpace(gjson.GetBytes(data, "type").String())
		if !responsesStreamEventMayContributeToOutput(eventType) {
			return
		}
		var event apicompat.ResponsesStreamEvent
		require.NoError(t, json.Unmarshal(data, &event))
		acc.ProcessEvent(&event)
	})
	return acc
}
