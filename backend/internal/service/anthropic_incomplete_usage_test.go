package service

import (
	"context"
	"errors"
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

func TestIncompleteCaptureProtocolPresenceAndTail(t *testing.T) {
	for _, tc := range []struct {
		name                             string
		frames                           []string
		wantInput, wantCache, wantOutput int
		estimated                        bool
		reason                           string
	}{
		{"anthropic prefix plus unicode tail", []string{`{"type":"message_start","message":{"usage":{"input_tokens":7,"cache_read_input_tokens":0,"output_tokens":1}}}`, `{"type":"content_block_delta","delta":{"type":"text_delta","text":"hello"}}`, `{"type":"message_delta","usage":{"output_tokens":10}}`, `{"type":"content_block_delta","delta":{"type":"text_delta","text":"中文尾部"}}`, `{"type":"message_stop"}`}, 7, 0, 14, true, "missing_usage"},
		{"anthropic real zero", []string{`{"type":"message_start","message":{"usage":{"input_tokens":0,"cache_read_input_tokens":0,"output_tokens":0}}}`, `{"type":"content_block_delta","delta":{"type":"text_delta","text":"中文"}}`, `{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":0}}`}, 0, 0, 0, false, ""},
		{"gemini prefix plus unicode tail", []string{`{"candidates":[{"content":{"parts":[{"text":"hello"}]}}],"usageMetadata":{"promptTokenCount":9,"cachedContentTokenCount":2,"candidatesTokenCount":10}}`, `{"candidates":[{"content":{"parts":[{"text":"中文尾部"}]},"finishReason":"STOP"}]}`}, 7, 2, 14, true, "missing_usage"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newIncompleteForwardCapture(context.Background(), nil, nil)
			p := &incompleteProtocolProgress{}
			for _, frame := range tc.frames {
				p.observe([]byte(frame))
			}
			f.current = &incompleteForwardAttempt{streaming: true, resp: &http.Response{StatusCode: 200}, progress: p, body: []byte(`{"messages":[{"role":"user","content":"hello world"}]}`)}
			// Force incomplete finalization even for terminal zero, simulating cancellation.
			existing := &ClaudeUsage{InputTokens: 99, CacheReadInputTokens: 99, InputTokensEstimated: true, CacheReadTokensEstimated: true}
			if !tc.estimated {
				f.writeFailed.Store(true)
			}
			got, _ := f.usage(existing, nil, false)
			require.Equal(t, tc.wantInput, got.InputTokens)
			require.Equal(t, tc.wantCache, got.CacheReadInputTokens)
			require.Equal(t, tc.wantOutput, got.OutputTokens)
			require.Equal(t, tc.estimated, got.OutputTokensEstimated)
			require.False(t, got.InputTokensEstimated)
			require.False(t, got.CacheReadTokensEstimated)
			require.Equal(t, tc.reason, got.UsageEstimationReason)
		})
	}
}

func TestIncompleteCaptureDoesNotEstimateUnsentRejectedOrEmptyFault(t *testing.T) {
	for _, tc := range []struct {
		name    string
		status  int
		written bool
		gone    bool
		err     error
	}{
		{"unsent cancellation", 0, false, true, context.Canceled},
		{"HTTP rejection", 429, true, true, context.Canceled},
		{"empty upstream failure", 200, true, false, io.ErrUnexpectedEOF},
		{"failover", 200, true, true, &UpstreamFailoverError{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newIncompleteForwardCapture(context.Background(), nil, nil)
			a := &incompleteForwardAttempt{streaming: true, progress: &incompleteProtocolProgress{}, body: []byte(`{"messages":[{"content":"hello"}]}`), err: tc.err}
			if tc.status != 0 {
				a.resp = &http.Response{StatusCode: tc.status}
			}
			a.written.Store(tc.written)
			f.current = a
			f.writeFailed.Store(tc.gone)
			got, billable := f.usage(nil, tc.err, false)
			require.Nil(t, got)
			require.False(t, billable)
		})
	}
}

func TestIncompleteCaptureDispatchedCancelBeforeHeadersClosesUpstream(t *testing.T) {
	arrived := make(chan struct{})
	closed := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		close(arrived)
		<-r.Context().Done()
		close(closed)
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/", nil).WithContext(ctx)
	f := newIncompleteForwardCapture(context.Background(), c, nil)
	defer f.restoreWriter()
	req, err := http.NewRequest("POST", server.URL, strings.NewReader(`{"model":"claude-test","messages":[{"role":"user","content":"hello world"}]}`))
	require.NoError(t, err)
	done := make(chan error, 1)
	go func() { _, err := f.Do(req, server.Client().Do); done <- err }()
	select {
	case <-arrived:
	case <-time.After(2 * time.Second):
		t.Fatal("request not dispatched")
	}
	cancel()
	var sendErr error
	select {
	case sendErr = <-done:
	case <-time.After(time.Second):
		t.Fatal("cancel did not stop request")
	}
	require.True(t, errors.Is(sendErr, context.Canceled))
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("upstream connection remained open")
	}
	got, billable := f.usage(nil, sendErr, false)
	require.True(t, billable)
	require.True(t, got.InputTokensEstimated)
	require.True(t, got.CacheReadTokensEstimated)
	require.Positive(t, got.InputTokens+got.CacheReadInputTokens)
	require.Zero(t, got.OutputTokens)
	require.Equal(t, "client_cancelled", got.UsageEstimationReason)
}

func TestIncompleteCaptureExecutionDeadlineIsNotClientCancellation(t *testing.T) {
	for _, hasOutput := range []bool{false, true} {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("POST", "/", nil)
		f := newIncompleteForwardCapture(ctx, c, nil)
		p := &incompleteProtocolProgress{}
		if hasOutput {
			p.observe([]byte(`{"candidates":[{"content":{"parts":[{"text":"hello"}]}}]}`))
		}
		f.current = &incompleteForwardAttempt{
			resp: &http.Response{StatusCode: 200}, progress: p,
			body: []byte(`{"contents":[{"parts":[{"text":"hello world"}]}]}`),
		}
		got, billable := f.usage(nil, context.DeadlineExceeded, true)
		require.False(t, f.clientGone())
		require.False(t, f.clientDisconnected(true), "a legacy disconnect flag cannot reclassify a server deadline")
		if hasOutput {
			require.True(t, billable)
			require.Equal(t, "upstream_timeout", got.UsageEstimationReason)
		} else {
			require.False(t, billable, "silent upstream timeout cannot create an input estimate")
			require.Nil(t, got)
		}
		f.restoreWriter()
		cancel()
	}
}

func TestIncompleteCaptureOversizedRequestIsNeverPartiallyEstimated(t *testing.T) {
	f := newIncompleteForwardCapture(context.Background(), nil, nil)
	req, err := http.NewRequest("POST", "https://example.invalid", strings.NewReader(`{"messages":[{"content":"`+strings.Repeat("a", 32<<20)+`"}]}`))
	require.NoError(t, err)
	_, _ = f.Do(req, func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(""))}, nil
	})
	require.Nil(t, f.current.body)
	f.writeFailed.Store(true)
	got, billable := f.usage(nil, context.Canceled, false)
	require.False(t, billable)
	require.Zero(t, got.InputTokens)
}

func TestIncompleteCaptureJSONMissingUsageEstimatesReceivedOutput(t *testing.T) {
	f := newIncompleteForwardCapture(context.Background(), nil, nil)
	req, err := http.NewRequest("POST", "https://example.invalid", strings.NewReader(`{"contents":[{"parts":[{"text":"hello"}]}]}`))
	require.NoError(t, err)
	resp, err := f.Do(req, func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{
 "candidates": [{"content":{"parts":[{"text":"hello"}]},"finishReason":"STOP"}]
}`))}, nil
	})
	require.NoError(t, err)
	_, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	got, billable := f.usage(nil, nil, false)
	require.NotNil(t, got)
	require.True(t, billable)
	require.True(t, got.InputTokensEstimated)
	require.True(t, got.OutputTokensEstimated)
	require.Equal(t, "missing_usage", got.UsageEstimationReason)
}

func TestIncompleteCaptureNonStreamAcceptedCancelAndLateCancel(t *testing.T) {
	for _, endedFirst := range []bool{false, true} {
		t.Run(fmt.Sprintf("upstream_ended_first=%t", endedFirst), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("POST", "/", nil).WithContext(ctx)
			f := newIncompleteForwardCapture(context.Background(), c, nil)
			defer f.restoreWriter()
			req, err := http.NewRequest("POST", "https://example.invalid", strings.NewReader(`{"messages":[{"content":"hello world"}]}`))
			require.NoError(t, err)
			pr, pw := io.Pipe()
			defer func() { _ = pw.Close() }()
			resp, err := f.Do(req, func(*http.Request) (*http.Response, error) { return &http.Response{StatusCode: 200, Body: pr}, nil })
			require.NoError(t, err)
			if endedFirst {
				_ = pw.Close()
				_, _ = io.ReadAll(resp.Body)
			}
			cancel()
			if !endedFirst {
				readDone := make(chan struct{})
				go func() { _, _ = io.ReadAll(resp.Body); close(readDone) }()
				select {
				case <-readDone:
				case <-time.After(time.Second):
					t.Fatal("body not closed immediately")
				}
			}
			got, billable := f.usage(nil, io.ErrUnexpectedEOF, false)
			if endedFirst {
				require.Nil(t, got)
				require.False(t, billable)
			} else {
				require.True(t, billable)
				require.True(t, got.InputTokensEstimated)
				require.Equal(t, "client_cancelled", got.UsageEstimationReason)
			}
		})
	}
}

func TestIncompleteCaptureGeminiPromptAndCacheAcrossFrames(t *testing.T) {
	for _, cacheFirst := range []bool{false, true} {
		var p incompleteProtocolProgress
		prompt := []byte(`{"usageMetadata":{"promptTokenCount":100}}`)
		cache := []byte(`{"usageMetadata":{"cachedContentTokenCount":95}}`)
		if cacheFirst {
			p.observe(cache)
			p.observe(prompt)
		} else {
			p.observe(prompt)
			p.observe(cache)
		}
		var got ClaudeUsage
		p.mergeActualInput(&got)
		require.Equal(t, 5, got.InputTokens)
		require.Equal(t, 95, got.CacheReadInputTokens)
		require.False(t, got.InputTokensEstimated)
		require.False(t, got.CacheReadTokensEstimated)
		p.observe([]byte(`{"candidates":[{"finishReason":"STOP"}],"usageMetadata":{"cachedContentTokenCount":0}}`))
		p.mergeActualInput(&got)
		require.Equal(t, 100, got.InputTokens, "an actual final cache correction reuses the retained total")
		require.Zero(t, got.CacheReadInputTokens)
	}
}

func TestIncompleteCaptureGeminiThinkingPlaceholderDoesNotDoubleCount(t *testing.T) {
	var p incompleteProtocolProgress
	p.observe([]byte(`{"candidates":[{"content":{"parts":[{"thought":true,"text":"think"}]}}],"usageMetadata":{"thoughtsTokenCount":10}}`))
	p.observe([]byte(`{"candidates":[{"content":{"parts":[{"text":"hi"}]}}],"usageMetadata":{"thoughtsTokenCount":0}}`))
	got, estimated := p.applyGeminiOutput(10)
	require.Equal(t, 11, got, "retain ten measured thinking tokens and estimate only ordinary hi")
	require.True(t, estimated)
	require.Equal(t, 10, p.thoughtTokens, "an intermediate placeholder must not erase the real thinking bucket")

	p.observe([]byte(`{"candidates":[{"finishReason":"STOP"}],"usageMetadata":{"thoughtsTokenCount":0}}`))
	got, estimated = p.applyGeminiOutput(10)
	require.Equal(t, 1, got, "a real final thinking correction cannot leak the stale aggregate into ordinary output")
	require.True(t, estimated)
}

func TestIncompleteCaptureRecognizesMeasuredHiddenOutput(t *testing.T) {
	var gemini incompleteProtocolProgress
	gemini.observe([]byte(`{"usageMetadata":{"thoughtsTokenCount":10}}`))
	require.True(t, gemini.hasOutput(), "real hidden thinking usage proves work occurred without exposing text")
	got, estimated := gemini.applyGeminiOutput(10)
	require.Equal(t, 10, got)
	require.False(t, estimated)

	var anthropic incompleteProtocolProgress
	anthropic.observe([]byte(`{"type":"message_start","message":{"usage":{"output_tokens":1}}}`))
	require.False(t, anthropic.hasOutput(), "initial Anthropic output is provisional, not proof of generated output")
	anthropic.observe([]byte(`{"type":"message_delta","usage":{"output_tokens":10}}`))
	require.True(t, anthropic.hasOutput())
}

func TestIncompleteCaptureGeminiThinkingPresenceAndCumulative(t *testing.T) {
	p := &incompleteProtocolProgress{geminiCumulative: true}
	p.observe([]byte(`{"candidates":[{"content":{"parts":[{"text":"hello"},{"thought":true,"text":"中文思考"}]}}],"usageMetadata":{"promptTokenCount":5,"candidatesTokenCount":2}}`))
	p.observe([]byte(`{"candidates":[{"content":{"parts":[{"text":"hello world"},{"thought":true,"text":"中文思考"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":5,"candidatesTokenCount":3}}`))
	got, estimated := p.applyGeminiOutput(3)
	require.Equal(t, 7, got)
	require.True(t, estimated)
	require.False(t, p.outputReported)
	p.observe([]byte(`{"usageMetadata":{"candidatesTokenCount":3,"thoughtsTokenCount":10}}`))
	got, estimated = p.applyGeminiOutput(13)
	require.Equal(t, 13, got)
	require.False(t, estimated)
	require.True(t, p.outputReported)
}

func TestIncompleteCaptureAnthropicInputZeroAndCompatibleCachePlaceholder(t *testing.T) {
	for _, tc := range []struct {
		name, delta          string
		wantInput, wantCache int
	}{
		{"independent zero correction", `{"type":"message_delta","usage":{"input_tokens":0}}`, 0, 10},
		{"compatible cache placeholder", `{"type":"message_delta","usage":{"input_tokens":0,"cache_read_input_tokens":12}}`, 20, 12},
		{"authoritative total confirms full cache", `{"type":"message_delta","usage":{"input_tokens":0,"prompt_tokens":20,"cache_read_input_tokens":20}}`, 0, 20},
		{"nonstream real zero", `{"type":"message","usage":{"input_tokens":0,"cache_read_input_tokens":0,"output_tokens":0}}`, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := &incompleteProtocolProgress{}
			p.observe([]byte(`{"type":"message_start","message":{"usage":{"input_tokens":20,"cache_read_input_tokens":10}}}`))
			p.observe([]byte(tc.delta))
			got := ClaudeUsage{InputTokensEstimated: true, CacheReadTokensEstimated: true}
			p.mergeActualInput(&got)
			require.Equal(t, tc.wantInput, got.InputTokens)
			require.Equal(t, tc.wantCache, got.CacheReadInputTokens)
			require.False(t, got.InputTokensEstimated)
			require.False(t, got.CacheReadTokensEstimated)
		})
	}
}
