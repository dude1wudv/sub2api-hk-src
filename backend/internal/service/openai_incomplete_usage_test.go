package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"strings"
	"testing"
	"time"

	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOpenAIIncompleteUsageConfirmedPrefixAndTail(t *testing.T) {
	var u openAIIncompleteUsage
	u.Observe([]byte(`{"type":"response.output_text.delta","delta":"abcdefghijklmnopqrstuvwxyz"}`))
	u.Observe([]byte(`{"type":"response.in_progress","response":{"usage":{"input_tokens":100,"output_tokens":2,"input_tokens_details":{"cached_tokens":7}}}}`))
	u.Observe([]byte(`{"type":"response.function_call_arguments.delta","delta":"你好世界"}`))
	var usage OpenAIUsage
	u.Apply(&usage, 1000, true, "upstream_eof")
	require.Equal(t, 6, usage.OutputTokens)
	require.True(t, usage.OutputTokensEstimated)
	require.Equal(t, 100, usage.InputTokens)
	require.Equal(t, 7, usage.CacheReadInputTokens)
	require.False(t, usage.InputTokensEstimated)
}

func TestOpenAIIncompleteUsagePresence(t *testing.T) {
	for _, tc := range []struct {
		name, event  string
		input, cache int
		estimated    bool
	}{
		{"missing", `{"type":"response.created","response":{}}`, 100, 95, true},
		{"placeholder", `{"type":"response.created","response":{"usage":{"input_tokens":0,"output_tokens":0}}}`, 100, 95, true},
		{"real_zero", `{"type":"response.completed","response":{"usage":{"input_tokens":0,"output_tokens":0}}}`, 0, 0, false},
		{"real_cache_zero", `{"type":"response.completed","response":{"usage":{"input_tokens_details":{"cached_tokens":0}}}}`, 100, 0, true},
		{"real_input", `{"type":"response.in_progress","response":{"usage":{"input_tokens":17}}}`, 17, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var u openAIIncompleteUsage
			u.Observe([]byte(tc.event))
			var usage OpenAIUsage
			u.Apply(&usage, 100, true, "client_cancelled")
			require.Equal(t, tc.input, usage.InputTokens)
			require.Equal(t, tc.cache, usage.CacheReadInputTokens)
			require.Equal(t, tc.estimated, usage.InputTokensEstimated)
		})
	}
}

func TestOpenAIIncompleteUsageTerminalZeroAndNormalUsage(t *testing.T) {
	var u openAIIncompleteUsage
	u.Observe([]byte(`{"type":"response.output_text.delta","delta":"hello world"}`))
	u.Observe([]byte(`{"type":"response.completed","response":{"usage":{"input_tokens":0,"output_tokens":0}}}`))
	usage := OpenAIUsage{}
	u.Apply(&usage, 100, true, "client_cancelled")
	require.Zero(t, usage.OutputTokens)
	require.False(t, usage.OutputTokensEstimated)
	u = openAIIncompleteUsage{}
	u.Observe([]byte(`{"type":"response.completed","response":{"usage":{"input_tokens":12,"output_tokens":3,"input_tokens_details":{"cached_tokens":4}}}}`))
	u.Apply(&usage, 100, false, "")
	require.Equal(t, 12, usage.InputTokens)
	require.Equal(t, 3, usage.OutputTokens)
	require.Equal(t, 4, usage.CacheReadInputTokens)
	require.Empty(t, usage.UsageEstimationReason)
}

func TestOpenAIIncompleteAttemptDispatchAndFailurePolicy(t *testing.T) {
	for _, tc := range []struct {
		name                                   string
		sent, cancel, reject, output, failover bool
		want                                   bool
	}{
		{"never_sent_cancel", false, true, false, false, false, false},
		{"sent_cancel", true, true, false, false, false, true},
		{"rejected_cancel", true, true, true, false, false, false},
		{"upstream_fault_no_output", true, false, false, false, false, false},
		{"upstream_fault_output", true, false, false, true, false, true},
		{"retry_attempt", true, false, false, true, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client, cancel := context.WithCancel(context.Background())
			defer cancel()
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("POST", "/v1/responses", nil).WithContext(client)
			ctx, finish := beginOpenAIIncompleteAttempt(client, c, []byte(`{"model":"gpt-test","stream":true}`))
			a, ok := ctx.Value(openAIIncompleteContextKey{}).(*openAIIncompleteAttempt)
			require.True(t, ok)
			a.active = true
			a.dispatched.Store(tc.sent)
			a.rejected = tc.reject
			a.inputEstimate = 100
			if tc.output {
				a.tracker.Observe([]byte(`{"type":"response.output_text.delta","delta":"hello world"}`))
			}
			if tc.cancel {
				cancel()
			}
			var result *OpenAIForwardResult
			err := io.ErrUnexpectedEOF
			if tc.failover {
				err = &UpstreamFailoverError{StatusCode: 502}
			}
			finish(&result, &err)
			if !tc.want {
				require.Nil(t, result)
				return
			}
			require.NotNil(t, result)
			require.Equal(t, 100, result.Usage.InputTokens)
			require.Equal(t, 95, result.Usage.CacheReadInputTokens)
			require.True(t, result.Usage.InputTokensEstimated)
		})
	}
}

func TestOpenAIIncompleteTransportCancelClosesSilentBody(t *testing.T) {
	client, cancel := context.WithCancel(context.Background())
	defer cancel()
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/responses", nil).WithContext(client)
	ctx, finishAttempt := beginOpenAIIncompleteAttempt(context.Background(), c, []byte(`{"model":"gpt-test","stream":true}`))
	req, _ := http.NewRequestWithContext(ctx, "POST", "https://example.invalid/v1/responses", strings.NewReader(`{"model":"gpt-test","input":"hello world"}`))
	req, finishRequest := prepareOpenAIIncompleteRequest(req)
	httptrace.ContextClientTrace(req.Context()).WroteRequest(httptrace.WroteRequestInfo{})
	reader, writer := io.Pipe()
	defer func() { require.NoError(t, writer.Close()) }()
	resp := &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"text/event-stream"}}, Body: reader}
	finishRequest(resp, nil)
	done := make(chan error, 1)
	go func() { _, err := io.ReadAll(resp.Body); done <- err }()
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("client cancel did not release upstream reader")
	}
	var result *OpenAIForwardResult
	err := context.Canceled
	finishAttempt(&result, &err)
	require.NotNil(t, result)
	require.True(t, result.ClientDisconnect)
	require.True(t, result.Usage.InputTokensEstimated)
	_ = resp.Body.Close()
}

func TestOpenAIIncompleteBodyRejectsOversizedPartialJSON(t *testing.T) {
	a := &openAIIncompleteAttempt{}
	body := &openAIIncompleteBody{ReadCloser: io.NopCloser(strings.NewReader("data: {\"type\":\"response.output_text.delta\",\"delta\":\"" + strings.Repeat("x", openAIIncompleteEventLimit) + "\"}\n")), attempt: a, sse: true, cleanup: func() {}}
	_, err := io.Copy(io.Discard, body)
	require.NoError(t, err)
	require.False(t, a.tracker.output.HasOutput())
}

func TestOpenAIWSIncompleteTurnIsolation(t *testing.T) {
	var turn openAIWSIncompleteTurn
	turn.Sent([]byte(`{"type":"response.create","input":"first"}`))
	turn.Observe([]byte(`{"type":"response.completed","response":{"id":"resp_one","usage":{"input_tokens":5,"output_tokens":3}}}`))
	first := &OpenAIForwardResult{}
	require.True(t, turn.Apply(first, false, nil))
	turn.Complete()
	require.False(t, turn.Active())
	turn.Sent([]byte(`{"type":"response.create","input":"second"}`))
	second := &OpenAIForwardResult{}
	require.True(t, turn.Apply(second, true, context.Canceled))
	require.Zero(t, second.Usage.OutputTokens)
	require.True(t, second.Usage.InputTokensEstimated)
	require.NotEqual(t, first.RequestID, second.RequestID)
}

func TestOpenAIWSIncompleteReaderPipelinedClose(t *testing.T) {
	for _, queued := range []int{1, 8, 33} {
		t.Run(fmt.Sprint(queued), func(t *testing.T) {
			ready := make(chan *openAIWSClientReadPump, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				conn, err := coderws.Accept(w, r, nil)
				if err != nil {
					return
				}
				defer func() {
					if closeErr := conn.CloseNow(); closeErr != nil && !errors.Is(closeErr, net.ErrClosed) {
						t.Errorf("close websocket: %v", closeErr)
					}
				}()
				c, _ := gin.CreateTestContext(w)
				c.Request = r
				pump := openAIWSIncompleteClientReader(c, conn)
				ready <- pump
				<-pump.ctx.Done()
			}))
			defer server.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			conn, _, err := coderws.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http"), nil)
			require.NoError(t, err)
			defer func() {
				if closeErr := conn.CloseNow(); closeErr != nil && !errors.Is(closeErr, net.ErrClosed) {
					t.Errorf("close websocket: %v", closeErr)
				}
			}()
			pump := <-ready
			for i := 0; i < queued; i++ {
				require.NoError(t, conn.Write(ctx, coderws.MessageText, []byte(`{"type":"response.create","input":"next"}`)))
			}
			if queued < 32 {
				require.Eventually(t, func() bool { return len(pump.messages) == queued }, time.Second, time.Millisecond)
				require.NoError(t, pump.ctx.Err(), "ordinary pipelined turns must remain connected")
				_ = conn.CloseNow()
			}
			select {
			case <-pump.ctx.Done():
			case <-time.After(time.Second):
				t.Fatal("queued requests prevented disconnect detection")
			}
		})
	}
}

func TestOpenAIIncompleteUsageToolNamesAndActualFlags(t *testing.T) {
	var u openAIIncompleteUsage
	u.Observe([]byte(`{"type":"response.output_item.added","item":{"type":"function_call","name":"tool_name"}}`))
	u.Observe([]byte(`{"type":"response.function_call_arguments.delta","delta":"你好"}`))
	u.Observe([]byte(`{"type":"response.completed","response":{"output":[{"type":"function_call","name":"tool_name","arguments":"你好"}],"usage":{"input_tokens":0,"input_tokens_details":{"cached_tokens":0}}}}`))
	usage := OpenAIUsage{InputTokens: 100, CacheReadInputTokens: 95, InputTokensEstimated: true, CacheReadTokensEstimated: true}
	u.Apply(&usage, 100, true, "missing_usage")
	require.Zero(t, usage.InputTokens)
	require.Zero(t, usage.CacheReadInputTokens)
	require.False(t, usage.InputTokensEstimated)
	require.False(t, usage.CacheReadTokensEstimated)
	require.Equal(t, 3, usage.OutputTokens, "tool name and arguments counted once")
	require.True(t, usage.OutputTokensEstimated)
	u.Observe([]byte(`{"type":"response.completed","response":{"usage":{"input_tokens":0,"output_tokens":1,"input_tokens_details":{"cached_tokens":0}}}}`))
	u.Apply(&usage, 100, false, "")
	require.False(t, usage.OutputTokensEstimated)
	require.Empty(t, usage.UsageEstimationReason)
}
func TestOpenAIIncompleteUsageCacheOnlyPreservesActual(t *testing.T) {
	var u openAIIncompleteUsage
	u.Observe([]byte(`{"type":"response.completed","response":{"usage":{"input_tokens_details":{"cached_tokens":150}}}}`))
	var usage OpenAIUsage
	u.Apply(&usage, 100, true, "missing_usage")
	require.Equal(t, 150, usage.InputTokens)
	require.Equal(t, 150, usage.CacheReadInputTokens)
	require.True(t, usage.InputTokensEstimated)
	require.False(t, usage.CacheReadTokensEstimated)
}
func TestOpenAIIncompleteAttemptIgnoresLateCancelAfterTransportFailure(t *testing.T) {
	client, cancel := context.WithCancel(context.Background())
	defer cancel()
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/responses", nil).WithContext(client)
	ctx, finish := beginOpenAIIncompleteAttempt(client, c, []byte(`{"model":"test"}`))
	a, ok := ctx.Value(openAIIncompleteContextKey{}).(*openAIIncompleteAttempt)
	require.True(t, ok)
	a.active = true
	a.dispatched.Store(true)
	a.inputEstimate = 100
	a.transportErr = io.ErrUnexpectedEOF
	a.upstreamEndedBeforeCancel = true
	cancel()
	var result *OpenAIForwardResult
	var err error = &UpstreamFailoverError{StatusCode: 502}
	finish(&result, &err)
	require.Nil(t, result)
	var failover *UpstreamFailoverError
	require.ErrorAs(t, err, &failover)
}

func TestOpenAIIncompleteInputEstimationIsLazy(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("POST", "/v1/responses", nil)
	ctx, finish := beginOpenAIIncompleteAttempt(context.Background(), c, []byte(`{"model":"test"}`))
	req, _ := http.NewRequestWithContext(ctx, "POST", "https://example.invalid/v1/responses", strings.NewReader(`{"model":"test","input":"hello world"}`))
	reads := 0
	original := req.GetBody
	req.GetBody = func() (io.ReadCloser, error) { reads++; return original() }
	_, finishRequest := prepareOpenAIIncompleteRequest(req)
	response := &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"usage":{"input_tokens":3,"output_tokens":1}}`))}
	finishRequest(response, nil)
	_, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	result := &OpenAIForwardResult{}
	finish(&result, &err)
	require.Zero(t, reads, "fully metered requests must not reopen/tokenize outbound body")
	require.Equal(t, 3, result.Usage.InputTokens)
	require.False(t, result.Usage.InputTokensEstimated)
	var turn openAIWSIncompleteTurn
	turn.Sent([]byte(`{"type":"response.create","input":"hello world"}`))
	require.Zero(t, turn.input, "starting a websocket turn must not tokenize input")
	turn.Observe([]byte(`{"type":"response.completed","response":{"usage":{"input_tokens":3,"output_tokens":1}}}`))
	require.True(t, turn.Apply(&OpenAIForwardResult{}, false, nil))
	require.Zero(t, turn.input, "fully metered websocket turns must not tokenize input")
	turn.Complete()
	require.Nil(t, turn.request)
}

func TestOpenAIIncompleteAttemptPreservesAdditionalUsage(t *testing.T) {
	requestBody := `{"model":"test","input":"Describe the supplied image and explain its main features."}`
	inputEstimate := estimateRequestInputTokens([]byte(requestBody))
	require.Positive(t, inputEstimate)
	for _, tc := range []struct {
		name          string
		events        []string
		mainUsage     OpenAIUsage
		err           error
		want          OpenAIUsage
		wantBodyReads int
	}{
		{
			name:      "actual_main",
			events:    []string{`{"type":"response.completed","response":{"usage":{"input_tokens":3,"output_tokens":5}}}`},
			mainUsage: OpenAIUsage{InputTokens: 3, OutputTokens: 5},
			want:      OpenAIUsage{InputTokens: 14, OutputTokens: 12},
		},
		{
			name: "final_main_zero",
			events: []string{
				`{"type":"response.in_progress","response":{"usage":{"input_tokens":3,"output_tokens":5}}}`,
				`{"type":"response.completed","response":{"usage":{"input_tokens":0,"output_tokens":0}}}`,
			},
			mainUsage: OpenAIUsage{InputTokens: 3, OutputTokens: 5},
			want:      OpenAIUsage{InputTokens: 11, OutputTokens: 7},
		},
		{
			name: "confirmed_prefix_and_tail",
			events: []string{
				`{"type":"response.created","response":{"usage":{"input_tokens":3,"output_tokens":3}}}`,
				`{"type":"response.output_text.delta","delta":"partial"}`,
			},
			mainUsage: OpenAIUsage{InputTokens: 3, OutputTokens: 3},
			err:       io.ErrUnexpectedEOF,
			want:      OpenAIUsage{InputTokens: 14, OutputTokens: 12, OutputTokensEstimated: true, UsageEstimationReason: "upstream_eof"},
		},
		{
			name:   "missing_main_input",
			events: []string{`{"type":"response.output_text.delta","delta":"partial"}`},
			err:    io.ErrUnexpectedEOF,
			want: OpenAIUsage{
				InputTokens:              11 + inputEstimate,
				CacheReadInputTokens:     splitIncompleteInput(inputEstimate).CacheRead,
				OutputTokens:             9,
				InputTokensEstimated:     true,
				CacheReadTokensEstimated: splitIncompleteInput(inputEstimate).CacheRead > 0,
				OutputTokensEstimated:    true,
				UsageEstimationReason:    "upstream_eof",
			},
			wantBodyReads: 1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("POST", "/v1/responses", nil)
			ctx, finish := beginOpenAIIncompleteAttempt(context.Background(), c, []byte(`{"model":"test","stream":true}`))
			a, ok := ctx.Value(openAIIncompleteContextKey{}).(*openAIIncompleteAttempt)
			require.True(t, ok)
			a.active = true
			a.dispatched.Store(true)
			bodyReads := 0
			a.requestBody = func() (io.ReadCloser, error) {
				bodyReads++
				return io.NopCloser(strings.NewReader(requestBody)), nil
			}
			for _, event := range tc.events {
				a.tracker.Observe([]byte(event))
			}
			result := &OpenAIForwardResult{Usage: tc.mainUsage}
			additional := OpenAIUsage{InputTokens: 11, OutputTokens: 7}
			addOpenAIUsage(&result.Usage, additional)
			recordOpenAIIncompleteAdditionalUsage(ctx, additional)

			err := tc.err
			finish(&result, &err)

			require.Equal(t, tc.want, result.Usage)
			require.Equal(t, tc.wantBodyReads, bodyReads, "auxiliary input must not suppress missing main input estimation")
		})
	}
}

func TestOpenAIIncompleteUsageIntermediateZeroKeepsConfirmedPrefix(t *testing.T) {
	var u openAIIncompleteUsage
	u.Observe([]byte(`{"type":"response.in_progress","response":{"usage":{"input_tokens":4,"output_tokens":8}}}`))
	u.Observe([]byte(`{"type":"response.in_progress","response":{"usage":{"output_tokens":0}}}`))
	var usage OpenAIUsage
	u.Apply(&usage, 100, false, "")
	require.Equal(t, 8, usage.OutputTokens)
	u.Observe([]byte(`{"type":"response.completed","response":{"usage":{"input_tokens":0,"output_tokens":0}}}`))
	u.Apply(&usage, 100, true, "client_cancelled")
	require.Zero(t, usage.OutputTokens)
	require.False(t, usage.OutputTokensEstimated)
}

func TestOpenAIIncompleteChatPlaceholderThenRealZero(t *testing.T) {
	var u openAIIncompleteUsage
	u.Observe([]byte(`{"choices":[{"delta":{"content":"hello"}}],"usage":{"prompt_tokens":0,"completion_tokens":0}}`))
	var usage OpenAIUsage
	u.Apply(&usage, 100, true, "client_cancelled")
	require.Equal(t, 100, usage.InputTokens)
	require.True(t, usage.InputTokensEstimated)
	u.Observe([]byte(`{"choices":[],"usage":{"prompt_tokens":0,"completion_tokens":0}}`))
	u.Apply(&usage, 100, true, "client_cancelled")
	require.Zero(t, usage.InputTokens)
	require.Zero(t, usage.OutputTokens)
	require.False(t, usage.InputTokensEstimated)
	require.Zero(t, usage.CacheReadInputTokens)
	require.False(t, usage.CacheReadTokensEstimated)
	require.False(t, usage.OutputTokensEstimated)
}

func TestOpenAIIncompleteExecutionCancelDoesNotEstimateBeforeOutput(t *testing.T) {
	for _, deadline := range []bool{false, true} {
		t.Run(fmt.Sprint(deadline), func(t *testing.T) {
			var execution context.Context
			var cancel context.CancelFunc
			if deadline {
				execution, cancel = context.WithTimeout(context.Background(), 25*time.Millisecond)
			} else {
				execution, cancel = context.WithCancel(context.Background())
			}
			defer cancel()
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest("POST", "/v1/responses", nil)
			ctx, finish := beginOpenAIIncompleteAttempt(execution, c, []byte(`{"model":"test","stream":true}`))
			req, _ := http.NewRequestWithContext(ctx, "POST", "https://example.invalid/v1/responses", strings.NewReader(`{"model":"test","input":"hello world"}`))
			req, finishRequest := prepareOpenAIIncompleteRequest(req)
			httptrace.ContextClientTrace(req.Context()).WroteRequest(httptrace.WroteRequestInfo{})
			reader, writer := io.Pipe()
			defer func() { require.NoError(t, writer.Close()) }()
			resp := &http.Response{StatusCode: 200, Header: http.Header{}, Body: reader}
			finishRequest(resp, nil)
			done := make(chan error, 1)
			go func() { _, err := io.ReadAll(resp.Body); done <- err }()
			if !deadline {
				cancel()
			}
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("execution cancellation did not close upstream")
			}
			result := &OpenAIForwardResult{ClientDisconnect: true} // legacy parser classification
			err := execution.Err()
			finish(&result, &err)
			require.NotNil(t, result)
			require.False(t, result.ClientDisconnect, "execution cancellation is not a physical client disconnect")
			require.Zero(t, result.Usage.InputTokens)
			require.False(t, result.Usage.InputTokensEstimated)
			require.False(t, result.Usage.CacheReadTokensEstimated)
			require.Empty(t, result.Usage.UsageEstimationReason)
		})
	}
}

func TestOpenAIWSIncompleteBillingIDIsNotContinuationID(t *testing.T) {
	var turn openAIWSIncompleteTurn
	turn.Sent([]byte(`{"type":"response.create","model":"test","input":"hello"}`))
	turn.Observe([]byte(`{"type":"response.output_text.delta","id":"evt_delta","delta":"ok"}`))
	turn.Observe([]byte(`{"type":"response.completed","id":"evt_completed","response":{"usage":{"input_tokens":1,"output_tokens":1}}}`))
	result := &OpenAIForwardResult{}
	require.True(t, turn.Apply(result, false, nil))
	require.NotEmpty(t, result.RequestID, "billing still requires a unique deduplication id")
	require.NotEqual(t, "evt_delta", result.RequestID)
	require.NotEqual(t, "evt_completed", result.RequestID)
	require.Empty(t, result.ResponseID, "neither event ids nor billing ids are continuation anchors")
	turn.Complete()
	turn.Sent([]byte(`{"type":"response.create","model":"test","input":"next"}`))
	turn.Observe([]byte(`{"type":"response.completed","id":"evt_second","response":{"id":"resp_actual","usage":{"input_tokens":1,"output_tokens":1}}}`))
	result = &OpenAIForwardResult{}
	require.True(t, turn.Apply(result, false, nil))
	require.Equal(t, "resp_actual", result.ResponseID)
	require.Equal(t, "resp_actual", result.RequestID)
	var standalone openAIIncompleteUsage
	standalone.Observe([]byte(`{"object":"response","id":"resp_standalone","usage":{"input_tokens":1,"output_tokens":1}}`))
	require.Equal(t, "resp_standalone", standalone.responseID)
}
