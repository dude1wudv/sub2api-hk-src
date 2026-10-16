//go:build unit

package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type billingPacedBody struct {
	data      []byte
	chunkSize int
	delay     time.Duration
	position  int
	reads     int
	delayEOF  time.Duration
}

func (r *billingPacedBody) Read(p []byte) (int, error) {
	if r.position >= len(r.data) {
		if r.delayEOF > 0 {
			delay := r.delayEOF
			r.delayEOF = 0
			time.Sleep(delay)
		}
		return 0, io.EOF
	}
	if r.delay > 0 {
		time.Sleep(r.delay)
	}
	limit := r.chunkSize
	if limit <= 0 || limit > len(p) {
		limit = len(p)
	}
	if limit > len(r.data)-r.position {
		limit = len(r.data) - r.position
	}
	n := copy(p[:limit], r.data[r.position:r.position+limit])
	r.position += n
	r.reads++
	return n, nil
}

func (r *billingPacedBody) Close() error { return nil }

func runAnthropicBillingStream(t *testing.T, passthrough bool, body io.ReadCloser, failClientWrite bool) (*streamingResult, error) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	svc := newMinimalGatewayService()
	svc.cfg = &config.Config{Gateway: config.GatewayConfig{
		StreamDataIntervalTimeout: 1,
		MaxLineSize:               defaultMaxLineSize,
	}}
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	if failClientWrite {
		c.Writer = &failWriteResponseWriter{ResponseWriter: c.Writer}
	}
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       body,
	}
	var result *streamingResult
	var err error
	if passthrough {
		result, err = svc.handleStreamingResponseAnthropicAPIKeyPassthrough(context.Background(), resp, c, &Account{ID: 1}, time.Now(), "claude-sonnet")
	} else {
		result, err = svc.handleStreamingResponse(context.Background(), resp, c, &Account{ID: 1}, time.Now(), "claude-sonnet", "claude-sonnet", false)
	}
	return result, err
}

func newBillingPartialSSE(tail string) []byte {
	return []byte(strings.Join([]string{
		`data: {"type":"message_start","message":{"usage":{"input_tokens":23,"cache_read_input_tokens":7,"cache_creation_input_tokens":4}}}`,
		"",
		`data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"abcdefghij"}}` + tail,
	}, "\n"))
}

func TestAnthropicStreamBillingRecoversPartialOutputOnUnexpectedEOF(t *testing.T) {
	for _, passthrough := range []bool{false, true} {
		name := "native"
		if passthrough {
			name = "api-key passthrough"
		}
		t.Run(name, func(t *testing.T) {
			body := &streamReadCloser{payload: newBillingPartialSSE("\n\n"), err: io.ErrUnexpectedEOF}
			result, err := runAnthropicBillingStream(t, passthrough, body, false)
			require.Error(t, err)
			require.NotNil(t, result)
			require.Equal(t, 23, result.usage.InputTokens)
			require.Equal(t, 7, result.usage.CacheReadInputTokens)
			require.Equal(t, 4, result.usage.CacheCreationInputTokens)
			require.Equal(t, 3, result.usage.OutputTokens, "10 received UTF-8 bytes should estimate ceil(10/4)")
			require.True(t, result.usage.OutputTokensEstimated)
		})
	}
}

func TestAnthropicStreamBillingFlushesPartialEventWithoutFinalBlankLine(t *testing.T) {
	for _, passthrough := range []bool{false, true} {
		name := "native"
		if passthrough {
			name = "api-key passthrough"
		}
		t.Run(name, func(t *testing.T) {
			body := &streamReadCloser{payload: newBillingPartialSSE("")}
			result, err := runAnthropicBillingStream(t, passthrough, body, false)
			require.Error(t, err)
			require.Contains(t, err.Error(), "missing terminal event")
			require.NotNil(t, result)
			require.Equal(t, 23, result.usage.InputTokens)
			require.Equal(t, 7, result.usage.CacheReadInputTokens)
			require.Equal(t, 4, result.usage.CacheCreationInputTokens)
			require.Equal(t, 3, result.usage.OutputTokens)
			require.True(t, result.usage.OutputTokensEstimated)
		})
	}
}

func TestAnthropicStreamBillingKeepsAuthoritativeUsageBeforeUnexpectedEOF(t *testing.T) {
	for _, passthrough := range []bool{false, true} {
		name := "native"
		if passthrough {
			name = "api-key passthrough"
		}
		t.Run(name, func(t *testing.T) {
			payload := append(newBillingPartialSSE("\n\n"), []byte("data: {\"type\":\"message_delta\",\"usage\":{\"output_tokens\":15355}}\n\n")...)
			body := &streamReadCloser{payload: payload, err: io.ErrUnexpectedEOF}
			result, err := runAnthropicBillingStream(t, passthrough, body, false)
			require.Error(t, err)
			require.NotNil(t, result)
			require.Equal(t, 15355, result.usage.OutputTokens)
			require.False(t, result.usage.OutputTokensEstimated)
		})
	}
}

func TestAnthropicStreamBillingIgnoresTruncatedTerminalUsage(t *testing.T) {
	for _, passthrough := range []bool{false, true} {
		name := "native"
		if passthrough {
			name = "api-key passthrough"
		}
		t.Run(name, func(t *testing.T) {
			t.Run("valid text before truncation", func(t *testing.T) {
				payload := append(newBillingPartialSSE("\n\n"), []byte("data: {\"type\":\"message_delta\",\"usage\":{\"output_tokens\":9999")...)
				result, err := runAnthropicBillingStream(t, passthrough, &streamReadCloser{payload: payload}, false)
				require.Error(t, err)
				require.NotNil(t, result)
				require.Equal(t, 3, result.usage.OutputTokens)
				require.True(t, result.usage.OutputTokensEstimated)
			})

			t.Run("no prior semantic output", func(t *testing.T) {
				payload := []byte("data: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":23}}}\n\n" +
					"data: {\"type\":\"message_delta\",\"usage\":{\"output_tokens\":9999")
				result, err := runAnthropicBillingStream(t, passthrough, &streamReadCloser{payload: payload}, false)
				require.Error(t, err)
				require.NotNil(t, result)
				require.Equal(t, 23, result.usage.InputTokens)
				require.Zero(t, result.usage.OutputTokens)
				require.False(t, result.usage.OutputTokensEstimated)
			})
		})
	}
}

func TestAnthropicStreamBillingEmptyStreamDoesNotEstimate(t *testing.T) {
	for _, passthrough := range []bool{false, true} {
		name := "native"
		if passthrough {
			name = "api-key passthrough"
		}
		t.Run(name, func(t *testing.T) {
			result, err := runAnthropicBillingStream(t, passthrough, io.NopCloser(strings.NewReader("")), false)
			require.Error(t, err)
			if result != nil {
				require.Zero(t, result.usage.OutputTokens)
				require.False(t, result.usage.OutputTokensEstimated)
			}
		})
	}
}

func TestAnthropicStreamBillingIdleTimeoutPreservesPartialEstimateWithoutSuccess(t *testing.T) {
	partial := newBillingPartialSSE("\n\n")
	for _, passthrough := range []bool{false, true} {
		name := "native"
		if passthrough {
			name = "api-key passthrough"
		}
		t.Run(name, func(t *testing.T) {
			body := &billingPacedBody{data: partial, chunkSize: len(partial), delayEOF: 1300 * time.Millisecond}
			result, err := runAnthropicBillingStream(t, passthrough, body, false)
			require.Error(t, err)
			require.Contains(t, err.Error(), "timeout")
			require.NotNil(t, result)
			require.Equal(t, 3, result.usage.OutputTokens)
			require.True(t, result.usage.OutputTokensEstimated)
		})
	}
}

func TestAnthropicStreamBillingTracksLongLineAndDrainsAfterClientWriteFailure(t *testing.T) {
	longDelta := fmt.Sprintf("data: {\"type\":\"message_delta\",\"usage\":{\"output_tokens\":15355},\"padding\":%q}\n\n", strings.Repeat("x", 1400))
	payload := []byte(longDelta + "data: {\"type\":\"message_stop\"}\n\n")
	for _, passthrough := range []bool{false, true} {
		name := "native"
		if passthrough {
			name = "api-key passthrough"
		}
		t.Run(name, func(t *testing.T) {
			body := &billingPacedBody{data: payload, chunkSize: 100, delay: 200 * time.Millisecond}
			result, err := runAnthropicBillingStream(t, passthrough, body, true)
			require.NoError(t, err)
			require.NotNil(t, result)
			require.True(t, result.clientDisconnect)
			require.Equal(t, 15355, result.usage.OutputTokens)
			require.False(t, result.usage.OutputTokensEstimated)
			require.Equal(t, len(payload), body.position, "client write failure must not stop upstream draining")
			require.Greater(t, body.reads, 10, "the long event should arrive in many reads across more than two seconds")
		})
	}
}

func TestRecordUsageBillsEstimatedOutputAndPersistsProvenance(t *testing.T) {
	usageRepo := &openAIRecordUsageLogRepoStub{inserted: true}
	billingRepo := &openAIRecordUsageBillingRepoStub{result: &UsageBillingApplyResult{Applied: true}}
	userRepo := &openAIRecordUsageUserRepoStub{}
	svc := newGatewayRecordUsageServiceWithBillingRepoForTest(usageRepo, billingRepo, userRepo, &openAIRecordUsageSubRepoStub{})
	svc.billingService = NewBillingService(svc.cfg, &PricingService{pricingData: map[string]*LiteLLMModelPricing{
		"claude-sonnet-4": {OutputCostPerToken: 0.00001},
	}})
	svc.resolver = NewModelPricingResolver(nil, svc.billingService)

	err := svc.RecordUsage(context.Background(), &RecordUsageInput{
		Result: &ForwardResult{
			RequestID: "anthropic_partial_output_estimate",
			Model:     "claude-sonnet-4",
			Stream:    true,
			Usage: ClaudeUsage{
				OutputTokens:          3,
				OutputTokensEstimated: true,
			},
			Duration: time.Second,
		},
		APIKey:  &APIKey{ID: 501, Quota: 100},
		User:    &User{ID: 601},
		Account: &Account{ID: 701},
	})

	require.NoError(t, err)
	require.NotNil(t, usageRepo.lastLog)
	require.Equal(t, 3, usageRepo.lastLog.OutputTokens)
	require.True(t, usageRepo.lastLog.OutputTokensEstimated)
	require.InDelta(t, 0.00003, usageRepo.lastLog.OutputCost, 1e-12, "estimated output must be included in the billed output amount")
	require.NotNil(t, billingRepo.lastCmd)
	require.Equal(t, 3, billingRepo.lastCmd.OutputTokens)
	require.InDelta(t, 0.000033, billingRepo.lastCmd.BalanceCost, 1e-12, "the billing repository receives the amount including estimated output")
}
