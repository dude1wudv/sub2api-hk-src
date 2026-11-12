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

// incompleteForwardCapture belongs to one forwarding call, never a session. Each
// retry replaces its attempt, so a rejected attempt cannot authorize billing a
// later request that was never sent. Only the already-received body is inspected.
type incompleteForwardCapture struct {
	ctx                         context.Context
	c                           *gin.Context
	model                       string
	stream                      bool
	start                       time.Time
	current                     *incompleteForwardAttempt
	writer                      gin.ResponseWriter
	writeFailed                 atomic.Bool
	upstreamModel, billingModel string
	geminiCumulative            bool
	reasoningEffort             *string
	imageInputSize              string
}

type incompleteForwardAttempt struct {
	upstreamEndedBeforeCancel atomic.Bool
	streaming                 bool
	written                   atomic.Bool
	answered                  atomic.Bool
	body                      []byte
	resp                      *http.Response
	err                       error
	progress                  *incompleteProtocolProgress
	cancel                    context.CancelFunc
	stops                     []func() bool
}

func newIncompleteForwardCapture(ctx context.Context, c *gin.Context, body []byte) *incompleteForwardCapture {
	f := &incompleteForwardCapture{ctx: ctx, c: c, model: gjson.GetBytes(body, "model").String(), stream: gjson.GetBytes(body, "stream").Bool(), start: time.Now()}
	if c != nil && c.Writer != nil {
		f.writer = c.Writer
		c.Writer = &incompleteResponseWriter{ResponseWriter: c.Writer, owner: f}
	}
	return f
}

type incompleteResponseWriter struct {
	gin.ResponseWriter
	owner *incompleteForwardCapture
}

func (w *incompleteResponseWriter) Write(data []byte) (int, error) {
	n, err := w.ResponseWriter.Write(data)
	if err != nil {
		w.owner.abortWrite()
	}
	return n, err
}
func (w *incompleteResponseWriter) WriteString(data string) (int, error) {
	n, err := w.ResponseWriter.WriteString(data)
	if err != nil {
		w.owner.abortWrite()
	}
	return n, err
}
func (f *incompleteForwardCapture) abortWrite() {
	f.writeFailed.Store(true)
	if a := f.current; a != nil {
		if a.cancel != nil {
			a.cancel()
		}
		if a.resp != nil && a.resp.Body != nil {
			_ = a.resp.Body.Close()
		}
	}
}
func (f *incompleteForwardCapture) restoreWriter() {
	if f.c != nil && f.writer != nil {
		f.c.Writer = f.writer
	}
}

func (f *incompleteForwardCapture) clientGone() bool {
	if f.writeFailed.Load() {
		return true
	}
	if f.c != nil && f.c.Request != nil {
		return f.c.Request.Context().Err() != nil
	}
	return f.ctx != nil && f.ctx.Err() != nil
}

func (f *incompleteForwardCapture) clientDisconnected(reported bool) bool {
	if f.clientGone() {
		return true
	}
	// Some legacy parsers classify execution-context expiration as a client
	// disconnect. A live original request distinguishes server-side timeout
	// from an actual downstream write failure or client cancellation.
	if f.c != nil && f.c.Request != nil && f.ctx != nil && f.ctx.Err() != nil {
		return false
	}
	return reported
}

func (f *incompleteForwardCapture) Do(req *http.Request, send func(*http.Request) (*http.Response, error)) (*http.Response, error) {
	if f == nil {
		return send(req)
	}
	if old := f.current; old != nil {
		old.stop()
	}
	a := &incompleteForwardAttempt{progress: &incompleteProtocolProgress{geminiCumulative: f.geminiCumulative}}
	f.current = a
	if req.GetBody != nil {
		if body, err := req.GetBody(); err == nil {
			a.body, _ = io.ReadAll(io.LimitReader(body, (32<<20)+1))
			if len(a.body) > 32<<20 {
				a.body = nil
			}
			_ = body.Close()
		}
	}
	if model := gjson.GetBytes(a.body, "model").String(); model != "" {
		f.upstreamModel = model
	}
	if model := gjson.GetBytes(a.body, "request.model").String(); model != "" {
		f.upstreamModel = model
	}
	if _, tail, ok := strings.Cut(req.URL.Path, "/models/"); ok {
		f.upstreamModel = strings.SplitN(tail, ":", 2)[0]
	}
	callCtx, cancel := context.WithCancel(req.Context())
	a.cancel = cancel
	if f.ctx != nil {
		a.stops = append(a.stops, context.AfterFunc(f.ctx, cancel))
	}
	if f.c != nil && f.c.Request != nil {
		a.stops = append(a.stops, context.AfterFunc(f.c.Request.Context(), cancel))
	}
	if f.clientGone() {
		cancel()
	}
	callCtx = httptrace.WithClientTrace(callCtx, &httptrace.ClientTrace{
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			if info.Err == nil {
				a.written.Store(true)
			}
		},
		GotFirstResponseByte: func() { a.answered.Store(true) },
	})
	resp, err := send(req.WithContext(callCtx))
	a.resp, a.err = resp, err
	a.streaming = gjson.GetBytes(a.body, "stream").Bool() || strings.Contains(req.URL.Path, ":streamGenerateContent") || req.URL.Query().Get("alt") == "sse"
	if resp != nil {
		a.streaming = a.streaming || strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream")
	}
	if resp != nil {
		a.answered.Store(true)
		if resp.StatusCode >= 200 && resp.StatusCode < 300 && resp.Body != nil {
			body := &incompleteObservedBody{ReadCloser: resp.Body, progress: a.progress, streaming: a.streaming, owner: f, attempt: a}
			resp.Body = body
			a.stops = append(a.stops, context.AfterFunc(callCtx, func() { _ = body.Close() }))
		}
	}
	return resp, err
}

func (a *incompleteForwardAttempt) stop() {
	for _, stop := range a.stops {
		stop()
	}
	if a.cancel != nil {
		a.cancel()
	}
}

func (f *incompleteForwardCapture) usage(existing *ClaudeUsage, err error, disconnected bool) (*ClaudeUsage, bool) {
	a := f.current
	if a == nil {
		return existing, false
	}
	defer a.stop()
	var failover *UpstreamFailoverError
	var streamReject *sseStreamErrorEventError
	if errors.As(err, &failover) || errors.As(err, &streamReject) {
		return existing, false
	}
	if a.resp != nil && (a.resp.StatusCode < 200 || a.resp.StatusCode >= 300) {
		return existing, false
	}
	clientGone := f.clientDisconnected(disconnected) && !a.upstreamEndedBeforeCancel.Load()
	accepted := a.resp != nil && a.resp.StatusCode >= 200 && a.resp.StatusCode < 300
	// A transport fault plus an unrelated late cancellation is not evidence of
	// client-abandoned work. Before response headers require a written request
	// and the transport's own cancellation error.
	cancelledAfterDispatch := clientGone && a.written.Load() && !a.answered.Load() &&
		(errors.Is(a.err, context.Canceled) || errors.Is(a.err, context.DeadlineExceeded))
	if !accepted && !cancelledAfterDispatch {
		return existing, false
	}
	p := a.progress
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.rejected {
		return existing, false
	}
	if existing != nil {
		p.mergeActualInput(existing)
	}
	if !clientGone && !p.hasOutput() {
		// Confirmed input remains billable, but an empty upstream fault never
		// creates an estimated input charge.
		if existing == nil && p.usage.hasObservedTokens() {
			copy := p.usage
			return &copy, true
		}
		return existing, false
	}
	if existing == nil {
		copy := p.usage
		existing = &copy
	}
	p.mergeActualInput(existing)
	// Preserve the accounting parser's normal buckets. The captured original
	// protocol supplies absent counters and provenance only on incomplete work.
	if p.anthropic {
		p.anthropicOutput.ApplyFallback(existing)
	} else {
		existing.OutputTokens, existing.OutputTokensEstimated = p.applyGeminiOutput(existing.OutputTokens)
	}
	if !p.inputReported && existing.InputTokens == 0 {
		total := estimateRequestInputTokens(a.body)
		if total > 0 {
			if !p.cacheReported && existing.CacheReadInputTokens == 0 && existing.CacheCreationInputTokens == 0 {
				split := splitIncompleteInput(total)
				existing.InputTokens = split.Fresh
				existing.CacheReadInputTokens = split.CacheRead
				existing.CacheReadTokensEstimated = split.CacheRead > 0
			} else {
				existing.InputTokens = max(0, total-existing.CacheReadInputTokens-existing.CacheCreationInputTokens)
			}
			existing.InputTokensEstimated = true
		}
	}
	if existing.InputTokensEstimated || existing.CacheReadTokensEstimated || existing.OutputTokensEstimated {
		existing.UsageEstimationReason = "upstream_eof"
		if p.terminal {
			existing.UsageEstimationReason = "missing_usage"
		}
		if err != nil {
			existing.UsageEstimationReason = "upstream_error"
		}
		if errors.Is(err, context.DeadlineExceeded) || strings.Contains(strings.ToLower(errorText(err)), "timeout") {
			existing.UsageEstimationReason = "upstream_timeout"
		}
		if clientGone {
			existing.UsageEstimationReason = "client_cancelled"
		}
	}
	if !existing.InputTokensEstimated && !existing.CacheReadTokensEstimated && !existing.OutputTokensEstimated {
		existing.UsageEstimationReason = ""
	}
	return existing, existing.hasObservedTokens()
}

func (f *incompleteForwardCapture) finishForward(result **ForwardResult, err *error) {
	defer f.restoreWriter()
	var usage *ClaudeUsage
	disconnected := false
	if *result != nil {
		usage = &(*result).Usage
		disconnected = (*result).ClientDisconnect
		(*result).ClientDisconnect = f.clientDisconnected(disconnected)
	}
	u, billable := f.usage(usage, *err, disconnected)
	if *result == nil && billable {
		*result = &ForwardResult{Model: f.model, UpstreamModel: f.upstreamModel, Stream: f.stream, Duration: time.Since(f.start), ClientDisconnect: f.clientGone(), Usage: *u, UpstreamResponseModel: observedUpstreamResponseModel(f.c), UpstreamResponseModelConflict: observedUpstreamResponseModelConflict(f.c), UpstreamResponseServiceTier: observedUpstreamResponseServiceTier(f.c)}
		(*result).ReasoningEffort = f.reasoningEffort
		(*result).ImageCount = observedGeminiImageOutputs(f.c)
		(*result).ImageEndpoint = (*result).ImageCount > 0
		(*result).ImageInputSize = f.imageInputSize
		(*result).ImageSize = normalizeOpenAIImageSizeTier(f.imageInputSize)
		if a := f.current; a != nil && a.resp != nil {
			(*result).RequestID = a.resp.Header.Get("x-request-id")
			(*result).UpstreamHeaders = a.resp.Header
		}
	}
}

func (f *incompleteForwardCapture) finishOpenAI(result **OpenAIForwardResult, err *error) {
	defer f.restoreWriter()
	var usage *ClaudeUsage
	disconnected := false
	if *result != nil {
		u := (*result).Usage
		usage = &ClaudeUsage{InputTokens: max(0, u.InputTokens-u.CacheReadInputTokens-u.CacheCreationInputTokens), OutputTokens: u.OutputTokens, CacheReadInputTokens: u.CacheReadInputTokens, CacheCreationInputTokens: u.CacheCreationInputTokens, OutputTokensEstimated: u.OutputTokensEstimated, InputTokensEstimated: u.InputTokensEstimated, CacheReadTokensEstimated: u.CacheReadTokensEstimated, UsageEstimationReason: u.UsageEstimationReason}
		disconnected = (*result).ClientDisconnect
		(*result).ClientDisconnect = f.clientDisconnected(disconnected)
	}
	u, billable := f.usage(usage, *err, disconnected)
	if *result != nil && u != nil {
		// Preserve image modality and cache-duration buckets.
		dst := &(*result).Usage
		dst.InputTokens = u.InputTokens + u.CacheReadInputTokens + u.CacheCreationInputTokens
		dst.OutputTokens = u.OutputTokens
		dst.CacheReadInputTokens = u.CacheReadInputTokens
		dst.CacheCreationInputTokens = u.CacheCreationInputTokens
		dst.InputTokensEstimated = u.InputTokensEstimated
		dst.CacheReadTokensEstimated = u.CacheReadTokensEstimated
		dst.OutputTokensEstimated = u.OutputTokensEstimated
		dst.UsageEstimationReason = u.UsageEstimationReason
	}
	if *result == nil && billable {
		*result = &OpenAIForwardResult{Model: f.model, UpstreamModel: f.upstreamModel, Stream: f.stream, Duration: time.Since(f.start), ClientDisconnect: f.clientGone(), Usage: claudeUsageToOpenAIUsage(u), BillingModel: f.billingModel, UpstreamEndpoint: GetActualOpenAIUpstreamEndpoint(f.c), UpstreamResponseModel: observedUpstreamResponseModel(f.c), UpstreamResponseModelConflict: observedUpstreamResponseModelConflict(f.c), UpstreamResponseServiceTier: observedUpstreamResponseServiceTier(f.c)}
		if a := f.current; a != nil && a.resp != nil {
			(*result).RequestID = a.resp.Header.Get("x-request-id")
			(*result).UpstreamHeaders = a.resp.Header
		}
	}
}

// Observe complete upstream JSON payloads, including buffered frames already
// received when cancellation closes the connection. Never retain an unbounded
// response: one SSE line / nonstream JSON is bounded independently of output.
type incompleteObservedBody struct {
	io.ReadCloser
	progress  *incompleteProtocolProgress
	pending   []byte
	oversize  bool
	streaming bool
	owner     *incompleteForwardCapture
	attempt   *incompleteForwardAttempt
}

func (b *incompleteObservedBody) Read(dst []byte) (int, error) {
	n, err := b.ReadCloser.Read(dst)
	if err != nil && b.owner != nil && !b.owner.clientGone() {
		b.attempt.upstreamEndedBeforeCancel.Store(true)
	}
	if !b.streaming {
		if len(b.pending)+n > 4<<20 {
			b.pending = nil
			b.oversize = true
		}
		if !b.oversize {
			b.pending = append(b.pending, dst[:n]...)
		}
		if err != nil {
			b.observeLine()
		}
		return n, err
	}
	for _, ch := range dst[:n] {
		if ch == '\n' {
			b.observeLine()
			continue
		}
		if len(b.pending) >= 4<<20 {
			b.pending = nil
			b.oversize = true
		}
		if !b.oversize {
			b.pending = append(b.pending, ch)
		}
	}
	if err == io.EOF {
		b.observeLine()
	}
	return n, err
}
func (b *incompleteObservedBody) observeLine() {
	if !b.oversize {
		line := bytes.TrimSpace(b.pending)
		if bytes.HasPrefix(line, []byte("data:")) {
			line = bytes.TrimSpace(line[5:])
		}
		if gjson.ValidBytes(line) {
			b.progress.observe(line)
		}
	}
	b.pending = b.pending[:0]
	b.oversize = false
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

type incompleteProtocolProgress struct {
	outputReported, cacheReadReported, cacheCreationReported    bool
	geminiCumulative                                            bool
	geminiText, geminiThought, geminiTool                       geminiTextDelta
	lastToolName                                                string
	mu                                                          sync.Mutex
	usage                                                       ClaudeUsage
	inputReported, cacheReported, terminal, rejected, anthropic bool
	anthropicOutput                                             anthropicStreamOutputUsage
	output                                                      incompleteOutputUsage
	thoughtOutput                                               incompleteOutputUsage
	thoughtTokens, candidateTokens                              int
	thoughtReported, candidateReported                          bool
	geminiPromptTokens                                          int
}

func (p *incompleteProtocolProgress) mergeActualInput(existing *ClaudeUsage) {
	if p.inputReported {
		existing.InputTokens = p.usage.InputTokens
		existing.InputTokensEstimated = false
	}
	if p.cacheReadReported {
		existing.CacheReadInputTokens = p.usage.CacheReadInputTokens
		existing.CacheReadTokensEstimated = false
	}
	if p.cacheCreationReported {
		existing.CacheCreationInputTokens = p.usage.CacheCreationInputTokens
		existing.CacheCreation5mTokens = p.usage.CacheCreation5mTokens
		existing.CacheCreation1hTokens = p.usage.CacheCreation1hTokens
	}
}

func (p *incompleteProtocolProgress) applyGeminiOutput(existing int) (int, bool) {
	ordinaryExisting := max(0, existing-p.thoughtTokens)
	if p.thoughtReported {
		// Once thinking has its own real counter, the parser's aggregate can
		// still reflect an older frame. Only the observed candidate bucket can
		// seed ordinary output; otherwise an old thinking value is counted twice.
		ordinaryExisting = p.candidateTokens
	}
	ordinary, ordinaryEstimated := p.output.Apply(ordinaryExisting)
	thoughts, thoughtsEstimated := p.thoughtOutput.Apply(p.thoughtTokens)
	return ordinary + thoughts, ordinaryEstimated || thoughtsEstimated
}

func (p *incompleteProtocolProgress) hasOutput() bool {
	return p.anthropicOutput.HasOutput() || p.output.HasOutput() || p.thoughtOutput.HasOutput() || p.usage.ImageOutputTokens > 0 ||
		(p.anthropic && p.outputReported && p.usage.OutputTokens > 0) ||
		(p.candidateReported && p.candidateTokens > 0) || (p.thoughtReported && p.thoughtTokens > 0)
}
func (p *incompleteProtocolProgress) observe(data []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()
	root := gjson.ParseBytes(data)
	if inner := root.Get("response"); inner.IsObject() {
		root = inner
	}
	typ := root.Get("type").String()
	if typ == "error" || root.Get("error").Exists() {
		p.rejected = !p.hasOutput()
		return
	}
	if strings.HasPrefix(typ, "message_") || strings.HasPrefix(typ, "content_block_") || typ == "message" {
		p.anthropic = true
		p.anthropicOutput.Observe(root.Raw)
		if typ == "message" {
			root.Get("content").ForEach(func(_, block gjson.Result) bool {
				p.anthropicOutput.Observe(`{"type":"content_block_start","content_block":` + block.Raw + `}`)
				return true
			})
		}
		u := root.Get("usage")
		if typ == "message_start" {
			u = root.Get("message.usage")
		}
		if v, ok := streamOutputUsageTokenCount(u.Get("input_tokens")); ok {
			p.inputReported = true
			// Compatible providers repeat a zero placeholder while updating
			// independent cache buckets. Keep the prior uncached input in that
			// shape; authoritative prompt/miss fields below can confirm zero.
			cacheOnlyDelta := typ == "message_delta" && v == 0 && p.usage.InputTokens > 0 &&
				(u.Get("cache_read_input_tokens").Exists() || u.Get("cache_creation_input_tokens").Exists())
			if !cacheOnlyDelta {
				p.usage.InputTokens = v
			}
		}
		if v, ok := streamOutputUsageTokenCount(u.Get("cache_read_input_tokens")); ok {
			p.cacheReported = true
			p.cacheReadReported = true
			p.usage.CacheReadInputTokens = v
		}
		if v, ok := streamOutputUsageTokenCount(u.Get("cache_creation_input_tokens")); ok {
			p.cacheReported = true
			p.cacheCreationReported = true
			p.usage.CacheCreationInputTokens = v
			p.usage.CacheCreation5mTokens = int(u.Get("cache_creation.ephemeral_5m_input_tokens").Int())
			p.usage.CacheCreation1hTokens = int(u.Get("cache_creation.ephemeral_1h_input_tokens").Int())
		}
		// Reuse the same provider normalization as the ordinary parsers; raw
		// input_tokens can be a total when OpenAI/DeepSeek aliases are present.
		for _, path := range []string{"cached_tokens", "prompt_tokens_details.cached_tokens", "prompt_cache_hit_tokens"} {
			if v, ok := streamOutputUsageTokenCount(u.Get(path)); ok {
				if !u.Get("cache_read_input_tokens").Exists() {
					p.usage.CacheReadInputTokens = v
				}
				p.cacheReported, p.cacheReadReported = true, true
				break
			}
		}
		if normalizeAnthropicCompatiblePromptUsage(u, &p.usage) {
			p.inputReported = true
		}
		if v, ok := streamOutputUsageTokenCount(u.Get("output_tokens")); ok {
			p.usage.OutputTokens = v
			if typ != "message_start" {
				p.outputReported = true
			}
		}
		if typ == "message_stop" || typ == "message" || root.Get("delta.stop_reason").String() != "" {
			p.terminal = true
			if v, ok := streamOutputUsageTokenCount(u.Get("output_tokens")); ok {
				p.anthropicOutput.ReportFinal(v, false)
			}
		}
		return
	}
	if !root.Get("candidates").Exists() && !root.Get("usageMetadata").Exists() {
		return
	}
	root.Get("candidates").ForEach(func(_, candidate gjson.Result) bool {
		if candidate.Get("finishReason").String() != "" {
			p.terminal = true
		}
		candidate.Get("content.parts").ForEach(func(_, part gjson.Result) bool {
			text := part.Get("text").String()
			if p.geminiCumulative {
				if part.Get("thought").Bool() {
					text = p.geminiThought.next(text)
				} else {
					text = p.geminiText.next(text)
				}
			}
			if part.Get("thought").Bool() {
				p.thoughtOutput.AddText(text)
			} else {
				p.output.AddText(text)
			}
			if call := part.Get("functionCall"); call.Exists() {
				name := call.Get("name").String()
				if !p.geminiCumulative || p.lastToolName != name {
					p.output.AddText(name)
					p.geminiTool.reset()
				}
				p.lastToolName = name
				if args := call.Get("args"); args.Exists() {
					text := string(stripOpaqueBlobs([]byte(args.Raw)))
					if p.geminiCumulative {
						text = p.geminiTool.next(text)
					}
					p.output.AddText(text)
				}
			}
			p.output.AddText(part.Get("executableCode.code").String())
			p.output.AddText(part.Get("codeExecutionResult.output").String())
			return true
		})
		return true
	})
	u := root.Get("usageMetadata")
	if v, ok := streamOutputUsageTokenCount(u.Get("promptTokenCount")); ok && (v > 0 || p.terminal) {
		p.inputReported = true
		p.geminiPromptTokens = v
	}
	if v, ok := streamOutputUsageTokenCount(u.Get("cachedContentTokenCount")); ok && (v > 0 || p.terminal) {
		p.cacheReported = true
		p.cacheReadReported = true
		p.usage.CacheReadInputTokens = v
	}
	if p.inputReported {
		// Gemini reports a total prompt count. Cache usage may arrive in a
		// later frame, so derive the fresh bucket from the retained total.
		p.usage.InputTokens = max(0, p.geminiPromptTokens-p.usage.CacheReadInputTokens)
	}
	if v, ok := streamOutputUsageTokenCount(u.Get("candidatesTokenCount")); ok {
		if v > 0 || p.terminal {
			p.candidateReported = true
		}
		if p.terminal {
			p.output.ReportFinal(v, false)
		} else {
			p.output.Report(v, v == 0, false)
		}
		if p.output.authoritative {
			p.candidateTokens = p.output.authoritativeOutput
		}
	}
	if v, ok := streamOutputUsageTokenCount(u.Get("thoughtsTokenCount")); ok {
		if v > 0 || p.terminal {
			p.thoughtReported = true
		}
		if p.terminal {
			p.thoughtOutput.ReportFinal(v, false)
		} else {
			p.thoughtOutput.Report(v, v == 0, false)
		}
		if p.thoughtOutput.authoritative {
			p.thoughtTokens = p.thoughtOutput.authoritativeOutput
		}
	}
	p.outputReported = p.candidateReported && (!p.thoughtOutput.HasOutput() || p.thoughtReported)
	p.usage.OutputTokens = p.candidateTokens + p.thoughtTokens
}
