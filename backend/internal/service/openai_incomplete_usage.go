package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptrace"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

// Keep provenance separate from numeric fields: an explicit zero is usage, not
// permission to estimate it. Initial created/in_progress zeros are placeholders.
type openAIIncompleteUsage struct {
	output                                         incompleteOutputUsage
	usage                                          OpenAIUsage
	inputReported, cacheReported, creationReported bool
	outputReported                                 bool
	terminal                                       bool
	failed                                         bool
	responseID                                     string
}

func (u *openAIIncompleteUsage) Observe(payload []byte) {
	if !gjson.ValidBytes(payload) {
		return
	}
	v := gjson.ParseBytes(payload)
	typ := v.Get("type").String()
	if id := v.Get("response.id").String(); id != "" {
		u.responseID = id
	}
	// A top-level SSE id identifies an event, not a continuation anchor.
	// Only a standalone Responses object owns its top-level response id.
	if u.responseID == "" && v.Get("object").String() == "response" {
		u.responseID = v.Get("id").String()
	}
	terminal := openAIStreamEventTypeIsTerminal(typ)
	if terminal {
		u.terminal = true
	}
	if typ == "response.completed" || typ == "response.done" {
		u.failed = false
	}
	if typ == "error" || typ == "response.failed" || v.Get("error").Exists() {
		u.failed = true
	}
	// Observe raw provider deltas before any protocol conversion. Snapshot/done
	// events repeat the deltas, so only use their output when no deltas arrived.
	switch typ {
	case "response.output_text.delta", "response.refusal.delta", "response.reasoning_text.delta", "response.reasoning_summary_text.delta", "response.function_call_arguments.delta", "response.custom_tool_call_input.delta":
		u.output.AddText(v.Get("delta").String())
	case "response.output_item.added":
		item := v.Get("item")
		if item.Get("type").String() == "function_call" || item.Get("type").String() == "custom_tool_call" {
			u.output.AddText(item.Get("name").String())
		}
	case "response.completed", "response.done", "response.incomplete", "response.failed":
		if !u.output.HasOutput() {
			u.observeSnapshot(v.Get("response.output"))
		}
	case "":
		for _, choice := range v.Get("choices").Array() {
			delta := choice.Get("delta")
			if !delta.Exists() {
				delta = choice.Get("message")
			}
			u.output.AddText(delta.Get("content").String())
			u.output.AddText(delta.Get("reasoning_content").String())
			u.output.AddText(delta.Get("reasoning").String())
			u.output.AddText(delta.Get("refusal").String())
			for _, tool := range delta.Get("tool_calls").Array() {
				u.output.AddText(tool.Get("function.name").String())
				u.output.AddText(tool.Get("function.arguments").String())
			}
			if choice.Get("finish_reason").Type == gjson.String {
				u.terminal = true
			}
		}
		if !u.output.HasOutput() {
			u.observeSnapshot(v.Get("output"))
		}
	}
	usage := v.Get("response.usage")
	if !usage.IsObject() {
		usage = v.Get("usage")
	}
	if !usage.IsObject() {
		return
	}
	parsed, _ := openAIUsageFromGJSON(usage)
	finalChatUsage := typ == "" && (u.terminal || v.Get("choices.0.message").Exists() || (v.Get("choices").IsArray() && len(v.Get("choices").Array()) == 0))
	provisional := typ == "response.created" || typ == "response.in_progress" || typ == "response.queued" || (typ == "" && v.Get("choices").IsArray() && !finalChatUsage)
	has := func(paths ...string) bool {
		for _, path := range paths {
			if n, ok := streamOutputUsageTokenCount(usage.Get(path)); ok && (!provisional || n > 0) {
				return true
			}
		}
		return false
	}
	if has("input_tokens", "prompt_tokens") {
		u.usage.InputTokens = parsed.InputTokens
		u.inputReported = true
	}
	if has("input_tokens_details.cached_tokens", "prompt_tokens_details.cached_tokens", "cache_read_input_tokens", "cached_tokens", "prompt_cache_hit_tokens") {
		u.usage.CacheReadInputTokens = parsed.CacheReadInputTokens
		u.cacheReported = true
	}
	if has("cache_creation_input_tokens") {
		u.usage.CacheCreationInputTokens = parsed.CacheCreationInputTokens
		u.usage.CacheCreation30mInputTokens = parsed.CacheCreation30mInputTokens
		u.usage.CacheCreation12hInputTokens = parsed.CacheCreation12hInputTokens
		u.creationReported = true
	}
	if has("output_tokens", "completion_tokens") {
		if terminal || finalChatUsage {
			u.output.ReportFinal(parsed.OutputTokens, false)
		} else {
			u.output.Report(parsed.OutputTokens, provisional && parsed.OutputTokens == 0, false)
		}
		if terminal || finalChatUsage || parsed.OutputTokens >= u.usage.OutputTokens {
			u.usage.OutputTokens = parsed.OutputTokens
		}
		u.outputReported = true
	}
	if parsed.ImageInputTokens > 0 {
		u.usage.ImageInputTokens = parsed.ImageInputTokens
	}
	if parsed.ImageCacheReadTokens > 0 {
		u.usage.ImageCacheReadTokens = parsed.ImageCacheReadTokens
	}
	if parsed.ImageOutputTokens > 0 {
		u.usage.ImageOutputTokens = parsed.ImageOutputTokens
	}
}

func (u *openAIIncompleteUsage) observeSnapshot(output gjson.Result) {
	for _, item := range output.Array() {
		for _, part := range item.Get("content").Array() {
			u.output.AddText(part.Get("text").String())
			u.output.AddText(part.Get("refusal").String())
		}
		for _, part := range item.Get("summary").Array() {
			u.output.AddText(part.Get("text").String())
		}
		u.output.AddText(item.Get("name").String())
		u.output.AddText(item.Get("arguments").String())
		if item.Get("type").String() == "custom_tool_call" {
			u.output.AddText(item.Get("input").String())
		}
	}
}

func (u *openAIIncompleteUsage) Apply(usage *OpenAIUsage, inputEstimate int, estimate bool, reason string) {
	if u.inputReported {
		usage.InputTokens = u.usage.InputTokens
		usage.InputTokensEstimated = false
		if usage.CacheReadTokensEstimated && !u.cacheReported {
			usage.CacheReadInputTokens = 0
			usage.CacheReadTokensEstimated = false
		}
	}
	if u.cacheReported {
		usage.CacheReadInputTokens = u.usage.CacheReadInputTokens
		usage.CacheReadTokensEstimated = false
	}
	if u.creationReported {
		usage.CacheCreationInputTokens = u.usage.CacheCreationInputTokens
		usage.CacheCreation30mInputTokens = u.usage.CacheCreation30mInputTokens
		usage.CacheCreation12hInputTokens = u.usage.CacheCreation12hInputTokens
	}
	usage.ImageInputTokens = max(usage.ImageInputTokens, u.usage.ImageInputTokens)
	usage.ImageOutputTokens = max(usage.ImageOutputTokens, u.usage.ImageOutputTokens)
	if !estimate {
		if u.outputReported {
			usage.OutputTokens = u.usage.OutputTokens
			usage.OutputTokensEstimated = false
		}
		if !usage.InputTokensEstimated && !usage.CacheReadTokensEstimated && !usage.OutputTokensEstimated {
			usage.UsageEstimationReason = ""
		}
		return
	}
	usage.OutputTokens, usage.OutputTokensEstimated = u.output.Apply(usage.OutputTokens)
	// Fill only the missing total when cache fields were reported. Never turn
	// actual cache usage (including free/zero cache) into an estimated discount.
	if !u.inputReported && usage.InputTokens == 0 && usage.ImageInputTokens == 0 && (inputEstimate > 0 || usage.CacheReadInputTokens+usage.CacheCreationInputTokens > 0) {
		if u.cacheReported || u.creationReported || usage.CacheReadInputTokens > 0 || usage.CacheCreationInputTokens > 0 {
			usage.InputTokens = max(inputEstimate, usage.CacheReadInputTokens+usage.CacheCreationInputTokens)
		} else {
			split := splitIncompleteInput(inputEstimate)
			usage.InputTokens = split.Total
			usage.CacheReadInputTokens = split.CacheRead
			usage.CacheReadTokensEstimated = split.CacheRead > 0
		}
		usage.InputTokensEstimated = true
	}
	if usage.InputTokensEstimated || usage.CacheReadTokensEstimated || usage.OutputTokensEstimated {
		usage.UsageEstimationReason = reason
	} else {
		usage.UsageEstimationReason = ""
	}
}

type openAIIncompleteContextKey struct{}
type openAIIncompleteAttempt struct {
	mu                           sync.Mutex
	client                       context.Context
	requestClient                context.Context
	tracker                      openAIIncompleteUsage
	additionalUsage              OpenAIUsage
	inputEstimate                int
	requestBody                  func() (io.ReadCloser, error)
	model, endpoint              string
	serviceTier, reasoningEffort *string
	headers                      http.Header
	dispatched                   atomic.Bool
	rejected                     bool
	active                       bool
	clientGone                   bool
	transportErr                 error
	upstreamEndedBeforeCancel    bool
	stop                         func()
}

// Keep independently billed auxiliary requests separate from the main
// response snapshot. The caller has already included these actual tokens in
// result.Usage; finalizing the main request must neither erase nor double them.
func recordOpenAIIncompleteAdditionalUsage(ctx context.Context, usage OpenAIUsage) {
	a, ok := ctx.Value(openAIIncompleteContextKey{}).(*openAIIncompleteAttempt)
	if !ok || !openAIUsageHasTokens(&usage) {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	addOpenAIUsage(&a.additionalUsage, usage)
}

func subtractOpenAIAdditionalUsage(usage *OpenAIUsage, additional OpenAIUsage) {
	usage.InputTokens = max(0, usage.InputTokens-additional.InputTokens)
	usage.ImageInputTokens = max(0, usage.ImageInputTokens-additional.ImageInputTokens)
	usage.OutputTokens = max(0, usage.OutputTokens-additional.OutputTokens)
	usage.CacheCreationInputTokens = max(0, usage.CacheCreationInputTokens-additional.CacheCreationInputTokens)
	usage.CacheReadInputTokens = max(0, usage.CacheReadInputTokens-additional.CacheReadInputTokens)
	usage.ImageOutputTokens = max(0, usage.ImageOutputTokens-additional.ImageOutputTokens)
}

func openAIOriginalClientGone(ctx context.Context, c *gin.Context) bool {
	if c != nil && c.Request != nil {
		return c.Request.Context().Err() != nil
	}
	return ctx != nil && ctx.Err() != nil
}

func (a *openAIIncompleteAttempt) clientErr() error {
	if a.requestClient != nil {
		return a.requestClient.Err()
	}
	if a.client != nil {
		return a.client.Err()
	}
	return nil
}
func (a *openAIIncompleteAttempt) onClientDone(fn func()) func() {
	var stops []func() bool
	if a.client != nil {
		stops = append(stops, context.AfterFunc(a.client, fn))
	}
	if a.requestClient != nil {
		stops = append(stops, context.AfterFunc(a.requestClient, fn))
	}
	return func() {
		for _, stop := range stops {
			stop()
		}
	}
}

// Decode/tokenize the outbound request only when an incomplete result needs
// fallback accounting or metadata. Fully metered normal requests do no extra
// request-body scan and retain only the transport's existing GetBody closure.
func (a *openAIIncompleteAttempt) inspectRequest(estimate bool) {
	if a.requestBody == nil {
		return
	}
	body, err := a.requestBody()
	if err != nil {
		return
	}
	defer func() { _ = body.Close() }()
	data, err := io.ReadAll(io.LimitReader(body, (32<<20)+1))
	if err != nil || len(data) > 32<<20 || !gjson.ValidBytes(data) {
		return
	}
	if estimate {
		a.inputEstimate = estimateRequestInputTokens(data)
	}
	a.model = gjson.GetBytes(data, "model").String()
	a.serviceTier = extractOpenAIServiceTierFromBody(data)
	a.reasoningEffort = extractOpenAIReasoningEffortFromBody(data, a.model)
}

func beginOpenAIIncompleteAttempt(ctx context.Context, c *gin.Context, body []byte) (context.Context, func(**OpenAIForwardResult, *error)) {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Value(openAIIncompleteContextKey{}).(*openAIIncompleteAttempt); ok {
		return ctx, func(**OpenAIForwardResult, *error) {}
	}
	a := &openAIIncompleteAttempt{client: ctx}
	if c != nil && c.Request != nil {
		a.requestClient = c.Request.Context()
	}
	started := time.Now()
	model := gjson.GetBytes(body, "model").String()
	stream := gjson.GetBytes(body, "stream").Bool()
	return context.WithValue(ctx, openAIIncompleteContextKey{}, a), func(result **OpenAIForwardResult, forwardErr *error) {
		a.mu.Lock()
		defer a.mu.Unlock()
		if a.stop != nil {
			a.stop()
		}
		if !a.active || !a.dispatched.Load() || a.rejected || GetOpsCyberPolicy(c) != nil {
			return
		}
		gone := a.clientGone || (!a.upstreamEndedBeforeCancel && a.clientErr() != nil)
		// Some legacy parsers classify cancellation of an execution/lease
		// context as a client disconnect. Only the original request or an
		// observed downstream write failure establishes billing eligibility.
		if *result != nil {
			(*result).ClientDisconnect = gone
		}
		var failover *UpstreamFailoverError
		if errors.As(*forwardErr, &failover) {
			// Only client cancellation supersedes a transport failover. Provider
			// refusals and replayable failures never acquire estimated charges.
			if !gone || a.tracker.failed || (!a.clientGone && !errors.Is(a.transportErr, context.Canceled) && !errors.Is(a.transportErr, context.DeadlineExceeded)) {
				return
			}
			*forwardErr = a.clientErr()
			if *forwardErr == nil {
				*forwardErr = context.Canceled
			}
		}
		_, unmeteredTail := a.tracker.output.Apply(a.tracker.usage.OutputTokens)
		missingUsage := !a.tracker.inputReported || !a.tracker.outputReported || unmeteredTail
		abnormal := *forwardErr != nil || gone || a.tracker.failed || missingUsage || (stream && !a.tracker.terminal)
		hasOutput := a.tracker.output.HasOutput() || a.tracker.usage.OutputTokens > 0
		estimate := abnormal && (gone || hasOutput) && (!a.tracker.failed || hasOutput)
		mainUsage := OpenAIUsage{}
		if *result != nil {
			mainUsage = (*result).Usage
			subtractOpenAIAdditionalUsage(&mainUsage, a.additionalUsage)
		}
		if estimate && !a.tracker.inputReported && mainUsage.InputTokens == 0 {
			a.inspectRequest(true)
		} else if *result == nil {
			a.inspectRequest(false)
		}
		if *result == nil {
			if !estimate && !openAIUsageHasTokens(&a.tracker.usage) {
				return
			}
			*result = &OpenAIForwardResult{Model: model, BillingModel: model, UpstreamModel: a.model, Stream: stream, Duration: time.Since(started), ResponseHeaders: a.headers, UpstreamHeaders: a.headers, UpstreamEndpoint: a.endpoint, ResponseID: a.tracker.responseID, ServiceTier: a.serviceTier, ReasoningEffort: a.reasoningEffort, UpstreamResponseModel: observedUpstreamResponseModel(c), UpstreamResponseModelConflict: observedUpstreamResponseModelConflict(c), UpstreamResponseServiceTier: observedUpstreamResponseServiceTier(c)}
			if a.headers != nil {
				(*result).RequestID = a.headers.Get("x-request-id")
			}
		}
		reason := openAIIncompleteReason(*forwardErr, gone, missingUsage && a.tracker.terminal)
		if gone {
			reason = "client_cancelled"
			(*result).ClientDisconnect = true
		}
		a.tracker.Apply(&mainUsage, a.inputEstimate, estimate, reason)
		addOpenAIUsage(&mainUsage, a.additionalUsage)
		(*result).Usage = mainUsage
	}
}

// Attach only to model inference requests. Image endpoints, search tools and
// account probes retain their existing completion and fixed-fee policies.
func prepareOpenAIIncompleteRequest(req *http.Request) (*http.Request, func(*http.Response, error)) {
	a, _ := req.Context().Value(openAIIncompleteContextKey{}).(*openAIIncompleteAttempt)
	path := strings.TrimRight(req.URL.Path, "/")
	if a == nil || (!strings.HasSuffix(path, "/responses") && !strings.HasSuffix(path, "/responses/compact") && !strings.HasSuffix(path, "/chat/completions")) {
		return req, func(*http.Response, error) {}
	}
	a.mu.Lock()
	if a.stop != nil {
		a.stop()
	}
	a.tracker = openAIIncompleteUsage{}
	a.additionalUsage = OpenAIUsage{}
	a.dispatched.Store(false)
	a.active = true
	a.rejected = false
	a.clientGone = false
	a.transportErr = nil
	a.upstreamEndedBeforeCancel = false
	a.inputEstimate = 0
	a.endpoint = path
	a.requestBody = req.GetBody
	requestCtx, cancel := context.WithCancel(req.Context())
	stopCancel := a.onClientDone(cancel)
	if a.clientErr() != nil {
		cancel()
	}
	cleanup := func() { stopCancel(); cancel() }
	a.stop = cleanup
	a.mu.Unlock()
	requestCtx = httptrace.WithClientTrace(requestCtx, &httptrace.ClientTrace{WroteRequest: func(info httptrace.WroteRequestInfo) {
		if info.Err == nil {
			a.dispatched.Store(true)
		}
	}})
	return req.WithContext(requestCtx), func(resp *http.Response, err error) {
		a.mu.Lock()
		defer a.mu.Unlock()
		if resp == nil {
			a.transportErr = err
			a.upstreamEndedBeforeCancel = a.clientErr() == nil
			if err != nil {
				cleanup()
			}
			return
		}
		a.dispatched.Store(true)
		a.headers = resp.Header.Clone()
		if resp.Request == nil {
			resp.Request = req
		}
		a.rejected = resp.StatusCode >= 400
		if resp.Body != nil {
			body := resp.Body
			stopClose := a.onClientDone(func() { _ = body.Close() })
			resp.Body = &openAIIncompleteBody{ReadCloser: body, attempt: a, sse: strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream"), cleanup: func() { stopClose(); cleanup() }}
		}
	}
}

// The observer retains at most one bounded event, never the full generated
// response. Oversized/incomplete payloads are discarded rather than estimated.
const openAIIncompleteEventLimit = 2 << 20

type openAIIncompleteBody struct {
	io.ReadCloser
	attempt *openAIIncompleteAttempt
	sse     bool
	pending []byte
	discard bool
	cleanup func()
	once    sync.Once
}

func (b *openAIIncompleteBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	b.attempt.mu.Lock()
	defer b.attempt.mu.Unlock()
	if err != nil && b.attempt.clientErr() == nil && !b.attempt.clientGone {
		b.attempt.upstreamEndedBeforeCancel = true
	}
	if !b.attempt.rejected {
		for _, ch := range p[:n] {
			if !b.sse && ch == '\n' && (bytes.HasPrefix(b.pending, []byte("data:")) || bytes.HasPrefix(b.pending, []byte("event:"))) {
				b.sse = true
			}
			if b.sse && ch == '\n' {
				if !b.discard {
					b.observeLine(b.pending)
				}
				b.pending = b.pending[:0]
				b.discard = false
			} else if !b.discard {
				if len(b.pending) >= openAIIncompleteEventLimit {
					b.pending = nil
					b.discard = true
				} else {
					b.pending = append(b.pending, ch)
				}
			}
		}
		if err != nil && len(b.pending) > 0 && !b.discard {
			if b.sse {
				b.observeLine(b.pending)
			} else {
				b.attempt.tracker.Observe(b.pending)
			}
			b.pending = nil
		}
	}
	return n, err
}
func (b *openAIIncompleteBody) observeLine(line []byte) {
	line = bytes.TrimSpace(line)
	if !bytes.HasPrefix(line, []byte("data:")) {
		return
	}
	data := bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
	if bytes.Equal(data, []byte("[DONE]")) {
		b.attempt.tracker.terminal = true
		return
	}
	b.attempt.tracker.Observe(data)
}
func (b *openAIIncompleteBody) Close() error { b.once.Do(b.cleanup); return b.ReadCloser.Close() }

// Compile-time assertion prevents accidentally wrapping a non-streaming-only
// reader when the scanner changes its buffering strategy.
var _ io.ReadCloser = (*openAIIncompleteBody)(nil)

func openAIIncompleteReason(err error, gone, missing bool) string {
	if gone {
		return "client_cancelled"
	}
	if missing && err == nil {
		return "missing_usage"
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "upstream_timeout"
	}
	var timed interface{ Timeout() bool }
	if errors.As(err, &timed) && timed.Timeout() {
		return "upstream_timeout"
	}
	if err == nil || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return "upstream_eof"
	}
	return "upstream_error"
}

func closeOpenAIIncompleteClientBody(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	if body, ok := resp.Body.(*openAIIncompleteBody); ok {
		body.attempt.mu.Lock()
		body.attempt.clientGone = true
		body.attempt.mu.Unlock()
	}
	if resp.Request != nil {
		if a, ok := resp.Request.Context().Value(openAIIncompleteContextKey{}).(*openAIIncompleteAttempt); ok {
			a.mu.Lock()
			a.clientGone = true
			a.mu.Unlock()
		}
	}
	_ = resp.Body.Close()
}
