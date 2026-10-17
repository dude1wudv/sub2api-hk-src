//go:build unit

package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func nativeAnthropicUsageSSE(events ...string) string {
	var s strings.Builder
	for _, event := range events {
		s.WriteString("data: ")
		s.WriteString(event)
		s.WriteString("\n\n")
	}
	return s.String()
}

func runNativeAnthropicUsageStream(t *testing.T, body io.ReadCloser, disconnect bool, intervalSeconds int) (*OpenAIForwardResult, error) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	requestBody := []byte(`{"model":"k3","max_tokens":128,"stream":true,"messages":[{"role":"user","content":"hi"}]}`)
	c := adaptiveProtocolTestContext("/v1/messages", requestBody)
	if disconnect {
		c.Writer = &failWriteResponseWriter{ResponseWriter: c.Writer}
	}
	cfg := rawChatCompletionsTestConfig()
	cfg.Gateway.StreamDataIntervalTimeout = intervalSeconds
	cfg.Gateway.StreamKeepaliveInterval = 0
	svc := &OpenAIGatewayService{
		cfg: cfg,
		httpUpstream: &httpUpstreamRecorder{resp: &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": {"text/event-stream"},
				"X-Request-Id": {"native-stream-usage-test"},
			},
			Body: body,
		}},
	}
	return svc.ForwardAsAnthropic(context.Background(), c, nativeAnthropicTestAccount(), requestBody, "", "")
}

func TestNativeAnthropicStreamUsageRecoversReceivedOutput(t *testing.T) {
	prefix := nativeAnthropicUsageSSE(
		`{"type":"message_start","message":{"usage":{"input_tokens":0,"output_tokens":0}}}`,
		`{"type":"content_block_delta","delta":{"type":"text_delta","text":"abcdefghij"}}`,
	)
	tests := []struct {
		name       string
		tail       string
		readErr    error
		disconnect bool
		wantErr    bool
	}{
		{name: "complete without final usage", tail: nativeAnthropicUsageSSE(`{"type":"message_stop"}`)},
		{name: "missing terminal", wantErr: true},
		{name: "upstream unexpected EOF", readErr: io.ErrUnexpectedEOF, wantErr: true},
		{name: "client disconnect then upstream unexpected EOF", readErr: io.ErrUnexpectedEOF, disconnect: true, wantErr: true},
		{name: "truncated final usage", tail: `data: {"type":"message_delta","usage":{"output_tokens":9999`, wantErr: true},
		{name: "zero terminal usage with output", tail: nativeAnthropicUsageSSE(
			`{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":0}}`,
			`{"type":"message_stop"}`,
		)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := runNativeAnthropicUsageStream(t, &streamReadCloser{payload: []byte(prefix + tt.tail), err: tt.readErr}, tt.disconnect, 0)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.NotNil(t, result)
			require.Equal(t, "/v1/messages", result.UpstreamEndpoint)
			require.Zero(t, result.Usage.InputTokens)
			require.Equal(t, 3, result.Usage.OutputTokens, "ten received bytes must survive every termination path")
			require.True(t, result.Usage.OutputTokensEstimated)
			require.Equal(t, tt.disconnect, result.ClientDisconnect)
		})
	}
}

func TestNativeAnthropicStreamUsagePreservesFinalMeasuredCountAndCache(t *testing.T) {
	payload := nativeAnthropicUsageSSE(
		`{"type":"message_start","message":{"usage":{"input_tokens":23,"cache_read_input_tokens":7,"cache_creation_input_tokens":4}}}`,
		`{"type":"content_block_delta","delta":{"type":"text_delta","text":"abcdefghij"}}`,
		`{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":2}}`,
		`{"type":"message_stop"}`,
	)
	result, err := runNativeAnthropicUsageStream(t, io.NopCloser(strings.NewReader(payload)), true, 0)
	require.NoError(t, err)
	require.True(t, result.ClientDisconnect)
	require.Equal(t, 34, result.Usage.InputTokens, "OpenAI usage stores total input, including both cache buckets")
	require.Equal(t, 7, result.Usage.CacheReadInputTokens)
	require.Equal(t, 4, result.Usage.CacheCreationInputTokens)
	require.Equal(t, 2, result.Usage.OutputTokens, "measured final usage replaces the local estimate")
	require.False(t, result.Usage.OutputTokensEstimated)
}

func TestNativeAnthropicStreamUsageReadsErrorUsageSnapshot(t *testing.T) {
	payload := nativeAnthropicUsageSSE(
		`{"type":"message_start","message":{"usage":{"input_tokens":0,"output_tokens":0}}}`,
		`{"type":"content_block_delta","delta":{"type":"text_delta","text":"abcdefghij"}}`,
		`{"type":"error","error":{"type":"upstream_error","message":"interrupted"},"usage":{"input_tokens":23,"cache_read_input_tokens":7,"cache_creation_input_tokens":4,"output_tokens":9,"output_tokens_estimated":true}}`,
	)
	result, err := runNativeAnthropicUsageStream(t, io.NopCloser(strings.NewReader(payload)), false, 0)
	require.Error(t, err)
	require.NotNil(t, result)
	require.Equal(t, 34, result.Usage.InputTokens)
	require.Equal(t, 7, result.Usage.CacheReadInputTokens)
	require.Equal(t, 4, result.Usage.CacheCreationInputTokens)
	require.Equal(t, 9, result.Usage.OutputTokens)
	require.True(t, result.Usage.OutputTokensEstimated)
}

func TestNativeAnthropicStreamUsageEstimatesAllSemanticOutput(t *testing.T) {
	payload := nativeAnthropicUsageSSE(
		`{"type":"content_block_start","content_block":{"type":"text","text":"a"}}`,
		`{"type":"content_block_delta","delta":{"type":"text_delta","text":"b"}}`,
		`{"type":"content_block_delta","delta":{"type":"thinking_delta","thinking":"c"}}`,
		`{"type":"content_block_delta","delta":{"type":"input_json_delta","partial_json":"{}"}}`,
		`{"type":"ping"}`,
		`{"type":"message_stop"}`,
	)
	result, err := runNativeAnthropicUsageStream(t, io.NopCloser(strings.NewReader(payload)), false, 0)
	require.NoError(t, err)
	require.Equal(t, 2, result.Usage.OutputTokens, "round once across five semantic bytes; ping contributes none")
	require.True(t, result.Usage.OutputTokensEstimated)
}

func TestNativeAnthropicStreamUsageEstimatedOutputIsBilled(t *testing.T) {
	payload := nativeAnthropicUsageSSE(
		`{"type":"message_start","message":{"usage":{"input_tokens":0,"output_tokens":0}}}`,
		`{"type":"content_block_delta","delta":{"type":"text_delta","text":"abcdefghij"}}`,
	)
	result, forwardErr := runNativeAnthropicUsageStream(t, &streamReadCloser{payload: []byte(payload), err: io.ErrUnexpectedEOF}, true, 0)
	require.Error(t, forwardErr)
	require.NotNil(t, result)

	usageRepo := &openAIRecordUsageLogRepoStub{inserted: true}
	billingRepo := &openAIRecordUsageBillingRepoStub{result: &UsageBillingApplyResult{Applied: true}}
	svc := newOpenAIRecordUsageServiceWithBillingRepoForTest(usageRepo, billingRepo, &openAIRecordUsageUserRepoStub{}, &openAIRecordUsageSubRepoStub{}, nil)
	err := svc.RecordUsage(context.Background(), &OpenAIRecordUsageInput{
		Result:  result,
		APIKey:  &APIKey{ID: 41, Group: &Group{Platform: PlatformKimi, RateMultiplier: 1}},
		User:    &User{ID: 42},
		Account: nativeAnthropicTestAccount(),
	})
	require.NoError(t, err)
	require.Equal(t, 1, billingRepo.calls)
	require.NotNil(t, billingRepo.lastCmd)
	require.Equal(t, 3, billingRepo.lastCmd.OutputTokens)
	require.Positive(t, billingRepo.lastCmd.BalanceCost)
	require.Equal(t, 1, usageRepo.calls)
	require.NotNil(t, usageRepo.lastLog)
	require.Equal(t, 3, usageRepo.lastLog.OutputTokens)
	require.True(t, usageRepo.lastLog.OutputTokensEstimated)
	require.Positive(t, usageRepo.lastLog.OutputCost)
	require.Positive(t, usageRepo.lastLog.ActualCost)
	require.InDelta(t, usageRepo.lastLog.ActualCost, billingRepo.lastCmd.BalanceCost, 1e-12)
}

func TestNativeAnthropicStreamUsageEmptyResponseStaysZero(t *testing.T) {
	payload := nativeAnthropicUsageSSE(
		`{"type":"message_start","message":{"usage":{"input_tokens":0,"output_tokens":0}}}`,
		`{"type":"ping"}`,
		`{"type":"message_delta","usage":{"output_tokens":0}}`,
		`{"type":"message_stop"}`,
	)
	result, err := runNativeAnthropicUsageStream(t, io.NopCloser(strings.NewReader(payload)), false, 0)
	require.NoError(t, err)
	require.Zero(t, result.Usage.OutputTokens)
	require.False(t, result.Usage.OutputTokensEstimated)
}

func TestNativeAnthropicBufferedUsagePreservesEstimateFlag(t *testing.T) {
	for _, estimated := range []bool{false, true} {
		t.Run(fmt.Sprint(estimated), func(t *testing.T) {
			body := []byte(fmt.Sprintf(`{"type":"message","usage":{"input_tokens":23,"cache_read_input_tokens":7,"cache_creation_input_tokens":4,"output_tokens":9,"output_tokens_estimated":%t}}`, estimated))
			usage := claudeUsageToOpenAIUsage(parseClaudeUsageFromResponseBody(body))
			require.Equal(t, 34, usage.InputTokens)
			require.Equal(t, 9, usage.OutputTokens)
			require.Equal(t, 7, usage.CacheReadInputTokens)
			require.Equal(t, 4, usage.CacheCreationInputTokens)
			require.Equal(t, estimated, usage.OutputTokensEstimated)
		})
	}
}

func TestNativeAnthropicStreamUsagePreservesSnapshotOnIdleTimeout(t *testing.T) {
	reader, writer := io.Pipe()
	defer writer.Close()
	go func() {
		_, _ = io.WriteString(writer, nativeAnthropicUsageSSE(
			`{"type":"content_block_delta","delta":{"type":"text_delta","text":"abcdefghij"}}`,
		))
	}()
	result, err := runNativeAnthropicUsageStream(t, reader, false, 1)
	require.ErrorContains(t, err, "stream data interval timeout")
	require.NotNil(t, result)
	require.Equal(t, 3, result.Usage.OutputTokens)
	require.True(t, result.Usage.OutputTokensEstimated)
}

func TestNativeAnthropicStreamUsageContinuousSubLineReadsDoNotTimeout(t *testing.T) {
	payload := nativeAnthropicUsageSSE(
		`{"type":"message_start","message":{"usage":{"input_tokens":1}}}`,
		fmt.Sprintf(`{"type":"content_block_delta","delta":{"type":"text_delta","text":%q}}`, strings.Repeat("x", 1200)),
		`{"type":"message_stop"}`,
	)
	body := &billingPacedBody{data: []byte(payload), chunkSize: 64, delay: 100 * time.Millisecond}
	result, err := runNativeAnthropicUsageStream(t, body, false, 1)
	require.NoError(t, err, "a long SSE line keeps arriving and must not trigger the one-second idle timer")
	require.NotNil(t, result)
	require.Equal(t, 300, result.Usage.OutputTokens)
	require.True(t, result.Usage.OutputTokensEstimated)
}
