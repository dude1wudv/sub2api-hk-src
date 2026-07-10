package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/gin-gonic/gin"
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

func TestOpenAIReasoningSummarySanitizer(t *testing.T) {
	t.Run("removes empty comment split across deltas", func(t *testing.T) {
		var sanitizer openAIReasoningSummarySanitizer
		first, changed := sanitizer.sanitizeEvent([]byte(`{"type":"response.reasoning_summary_text.delta","delta":"thinking\n\n<!--"}`))
		require.True(t, changed)
		require.Equal(t, "thinking\n\n", gjson.GetBytes(first, "delta").String())
		second, changed := sanitizer.sanitizeEvent([]byte(`{"type":"response.reasoning_summary_text.delta","delta":" \t\n-->done"}`))
		require.True(t, changed)
		require.Equal(t, "done", gjson.GetBytes(second, "delta").String())
	})

	t.Run("removes complete empty comment", func(t *testing.T) {
		var sanitizer openAIReasoningSummarySanitizer
		got, changed := sanitizer.sanitizeEvent([]byte(`{"type":"response.reasoning_summary_text.delta","delta":"before<!-- -->after"}`))
		require.True(t, changed)
		require.Equal(t, "beforeafter", gjson.GetBytes(got, "delta").String())
	})

	t.Run("preserves non-empty and incomplete non-empty comments", func(t *testing.T) {
		for _, delta := range []string{`before<!--keep-->after`, `before<!--keep`} {
			var sanitizer openAIReasoningSummarySanitizer
			input := []byte(`{"type":"response.reasoning_summary_text.delta","delta":` + strconv.Quote(delta) + `}`)
			got, changed := sanitizer.sanitizeEvent(input)
			require.False(t, changed)
			require.Equal(t, delta, gjson.GetBytes(got, "delta").String())
		}
	})

	t.Run("cleans reasoning snapshots only", func(t *testing.T) {
		tests := []struct{ name, input, path string }{
			{"text done", `{"type":"response.reasoning_summary_text.done","text":"a<!-- -->b"}`, "text"},
			{"part done", `{"type":"response.reasoning_summary_part.done","part":{"type":"summary_text","text":"a<!--\n-->b"}}`, "part.text"},
			{"item done", `{"type":"response.output_item.done","item":{"type":"reasoning","summary":[{"type":"summary_text","text":"a<!-- -->b"}]}}`, "item.summary.0.text"},
			{"completed", `{"type":"response.completed","response":{"output":[{"type":"reasoning","summary":[{"type":"summary_text","text":"a<!-- -->b"}]}]}}`, "response.output.0.summary.0.text"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var sanitizer openAIReasoningSummarySanitizer
				got, changed := sanitizer.sanitizeEvent([]byte(tt.input))
				require.True(t, changed)
				require.Equal(t, "ab", gjson.GetBytes(got, tt.path).String())
			})
		}
	})

	t.Run("does not clean final answer", func(t *testing.T) {
		var sanitizer openAIReasoningSummarySanitizer
		input := []byte(`{"type":"response.output_text.delta","delta":"answer<!-- -->text"}`)
		got, changed := sanitizer.sanitizeEvent(input)
		require.False(t, changed)
		require.Equal(t, input, got)
	})
}

func TestHandleStreamingResponseRemovesEmptyReasoningComment(t *testing.T) {
	testStreamingReasoningCommentSanitizer(t, false)
}

func TestHandleStreamingResponsePassthroughRemovesEmptyReasoningComment(t *testing.T) {
	testStreamingReasoningCommentSanitizer(t, true)
}

func testStreamingReasoningCommentSanitizer(t *testing.T, passthrough bool) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	upstreamBody := strings.Join([]string{
		`data: {"type":"response.created","response":{"id":"resp_reasoning"}}`, "",
		`data: {"type":"response.reasoning_summary_text.delta","output_index":0,"item_id":"rs_1","summary_index":0,"delta":"thinking\n\n<!--"}`, "",
		`data: {"type":"response.reasoning_summary_text.delta","output_index":0,"item_id":"rs_1","summary_index":0,"delta":" -->"}`, "",
		`data: {"type":"response.completed","response":{"id":"resp_reasoning","output":[],"usage":{"input_tokens":1,"output_tokens":1,"total_tokens":2}}}`, "",
		"data: [DONE]", "",
	}, "\n")
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)
	resp := &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(upstreamBody))}
	svc := &OpenAIGatewayService{}
	var err error
	if passthrough {
		_, err = svc.handleStreamingResponsePassthrough(context.Background(), resp, c, &Account{ID: 1}, time.Now(), "gpt-5.5", "gpt-5.5")
	} else {
		_, err = svc.handleStreamingResponse(context.Background(), resp, c, &Account{ID: 1}, time.Now(), "gpt-5.5", "gpt-5.5")
	}
	require.NoError(t, err)
	require.NotContains(t, rec.Body.String(), "<!--")
	require.NotContains(t, rec.Body.String(), "-->")
	require.Contains(t, rec.Body.String(), "thinking")
}
