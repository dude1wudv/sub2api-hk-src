//go:build unit

package service

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOpenAIFirstTokenElapsedMs_UsesFastestOfWallClockAndCreated(t *testing.T) {
	t.Parallel()

	observedAt := time.Now()
	firstTokenStart := observedAt.Add(-5 * time.Second)
	createdAt := observedAt.Add(-800 * time.Millisecond).Unix()

	ms := openAIFirstTokenElapsedMs(
		firstTokenStart,
		observedAt,
		[]byte(fmt.Sprintf(`{"id":"chatcmpl_1","object":"chat.completion.chunk","created":%d,"choices":[{"delta":{"content":"h"}}]}`, createdAt)),
	)
	require.GreaterOrEqual(t, ms, 500)
	require.Less(t, ms, 2000, "should prefer upstream created over 5s wall-clock")
}

func TestOpenAIFirstTokenElapsedMs_FallsBackToWallClockWithoutCreated(t *testing.T) {
	t.Parallel()

	observedAt := time.Now()
	firstTokenStart := observedAt.Add(-1200 * time.Millisecond)

	ms := openAIFirstTokenElapsedMs(
		firstTokenStart,
		observedAt,
		[]byte(`{"type":"response.output_text.delta","delta":"h"}`),
	)
	require.GreaterOrEqual(t, ms, 1000)
	require.Less(t, ms, 2000)
}

func TestOpenAIFirstTokenElapsedMsWithKnownCreated_UsesEarlierCreatedAt(t *testing.T) {
	t.Parallel()

	observedAt := time.Now()
	firstTokenStart := observedAt.Add(-5 * time.Second)
	known := observedAt.Add(-900 * time.Millisecond)

	ms := openAIFirstTokenElapsedMsWithKnownCreated(
		firstTokenStart,
		observedAt,
		[]byte(`{"type":"response.output_text.delta","delta":"h"}`),
		&known,
	)
	require.GreaterOrEqual(t, ms, 500)
	require.Less(t, ms, 2000)
}

func TestOpenAIResponseCreatedAt_AcceptsChatCompletionsCreated(t *testing.T) {
	t.Parallel()

	observedAt := time.Now()
	created := observedAt.Add(-2 * time.Second).Unix()
	got, ok := openAIResponseCreatedAt(
		[]byte(fmt.Sprintf(`{"id":"chatcmpl_1","created":%d,"choices":[{"delta":{"reasoning_content":"x"}}]}`, created)),
		observedAt,
	)
	require.True(t, ok)
	require.Equal(t, created, got.Unix())
}

func TestOpenAIResponsesStreamEventIsFirstToken_MatchesWSDefinition(t *testing.T) {
	t.Parallel()

	cases := []struct {
		eventType string
		want      bool
	}{
		{"response.created", false},
		{"response.in_progress", false},
		{"response.output_item.added", false},
		{"response.output_item.done", false},
		{"response.completed", false},
		{"response.done", false},
		{"response.failed", false},
		{"response.reasoning_text.delta", true},
		{"response.reasoning_summary_text.delta", true},
		{"response.output_text.delta", true},
		{"response.function_call_arguments.delta", true},
		{"response.output_text.done", true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.eventType, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, openAIResponsesStreamEventIsFirstToken(tc.eventType))
			require.Equal(t, tc.want, isOpenAIWSTokenEvent(tc.eventType))
		})
	}
}

func TestHandleStreamingResponsePassthrough_FirstTokenUsesReasoningDeltaNotCompleted(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	bodyReader, bodyWriter := io.Pipe()
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer bodyWriter.Close()
		_, _ = fmt.Fprintln(bodyWriter, `data: {"type":"response.created","response":{"id":"resp_think"}}`)
		_, _ = fmt.Fprintln(bodyWriter)
		_, _ = fmt.Fprintln(bodyWriter, `data: {"type":"response.output_item.added","item":{"type":"reasoning","id":"rs_1"}}`)
		_, _ = fmt.Fprintln(bodyWriter)
		time.Sleep(80 * time.Millisecond)
		_, _ = fmt.Fprintln(bodyWriter, `data: {"type":"response.reasoning_text.delta","delta":"step1"}`)
		_, _ = fmt.Fprintln(bodyWriter)
		time.Sleep(250 * time.Millisecond)
		_, _ = fmt.Fprintln(bodyWriter, `data: {"type":"response.output_text.delta","delta":"answer"}`)
		_, _ = fmt.Fprintln(bodyWriter)
		_, _ = fmt.Fprintln(bodyWriter, `data: {"type":"response.completed","response":{"id":"resp_think","usage":{"input_tokens":1,"output_tokens":2}}}`)
		_, _ = fmt.Fprintln(bodyWriter)
		_, _ = fmt.Fprintln(bodyWriter, "data: [DONE]")
		_, _ = fmt.Fprintln(bodyWriter)
	}()

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}, "x-request-id": []string{"rid-think"}},
		Body:       bodyReader,
	}
	svc := &OpenAIGatewayService{}
	start := time.Now()
	result, err := svc.handleStreamingResponsePassthrough(
		t.Context(),
		resp,
		c,
		&Account{ID: 1, Name: "acc", Platform: PlatformOpenAI},
		start,
		"gpt-5.2",
		"gpt-5.2",
		start,
	)
	<-done
	require.NoError(t, err)
	require.NotNil(t, result.firstTokenMs)
	require.Less(t, *result.firstTokenMs, 200, "first token must be reasoning delta, not later text/completed")
	require.GreaterOrEqual(t, *result.firstTokenMs, 50)
}

func TestHandleStreamingResponsePassthrough_CompletedOnlyDoesNotSetFirstToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	upstreamSSE := strings.Join([]string{
		`data: {"type":"response.created","response":{"id":"resp_done"}}`,
		"",
		`data: {"type":"response.output_item.added","item":{"type":"message","id":"msg_1"}}`,
		"",
		`data: {"type":"response.completed","response":{"id":"resp_done","usage":{"input_tokens":1,"output_tokens":1}}}`,
		"",
		"data: [DONE]",
		"",
	}, "\n")
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader(upstreamSSE)),
	}
	svc := &OpenAIGatewayService{}
	result, err := svc.handleStreamingResponsePassthrough(
		t.Context(),
		resp,
		c,
		&Account{ID: 1, Name: "acc", Platform: PlatformOpenAI},
		time.Now(),
		"gpt-5.2",
		"gpt-5.2",
	)
	require.NoError(t, err)
	require.Nil(t, result.firstTokenMs, "terminals/item lifecycle must not inflate first_token_ms")
}

func TestHandleAnthropicStreamingResponse_FirstTokenUsesReasoningDelta(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	bodyReader, bodyWriter := io.Pipe()
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer bodyWriter.Close()
		_, _ = fmt.Fprintln(bodyWriter, `data: {"type":"response.created","response":{"id":"resp_msg"}}`)
		_, _ = fmt.Fprintln(bodyWriter)
		_, _ = fmt.Fprintln(bodyWriter, `data: {"type":"response.output_item.added","item":{"type":"reasoning","id":"rs_1"}}`)
		_, _ = fmt.Fprintln(bodyWriter)
		time.Sleep(70 * time.Millisecond)
		_, _ = fmt.Fprintln(bodyWriter, `data: {"type":"response.reasoning_text.delta","delta":"think"}`)
		_, _ = fmt.Fprintln(bodyWriter)
		time.Sleep(220 * time.Millisecond)
		_, _ = fmt.Fprintln(bodyWriter, `data: {"type":"response.output_text.delta","delta":"hi"}`)
		_, _ = fmt.Fprintln(bodyWriter)
		_, _ = fmt.Fprintln(bodyWriter, `data: {"type":"response.completed","response":{"id":"resp_msg","usage":{"input_tokens":1,"output_tokens":2}}}`)
		_, _ = fmt.Fprintln(bodyWriter)
	}()

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}, "x-request-id": []string{"msg-ft"}},
		Body:       bodyReader,
	}
	svc := &OpenAIGatewayService{}
	start := time.Now()
	result, err := svc.handleAnthropicStreamingResponse(
		resp,
		c,
		&Account{ID: 2, Name: "grok", Platform: PlatformGrok},
		"grok-4",
		"grok-4",
		"grok-4",
		start,
		start,
	)
	<-done
	require.NoError(t, err)
	require.NotNil(t, result.FirstTokenMs)
	require.Less(t, *result.FirstTokenMs, 180, "messages path must record reasoning delta, not text after thinking")
	require.GreaterOrEqual(t, *result.FirstTokenMs, 40)
}
