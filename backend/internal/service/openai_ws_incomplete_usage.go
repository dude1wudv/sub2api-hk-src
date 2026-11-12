package service

import (
	"context"
	"errors"
	"sync"
	"time"

	openaiwsv2 "github.com/Wei-Shaw/sub2api/internal/service/openai_ws_v2"
	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
)

// A single reader lives for the downstream connection, across account failover.
// It detects a close while the upstream is silent; requests queued by a client
// remain bounded and are consumed only after the current turn finishes.
type openAIWSClientReadPump struct {
	ctx      context.Context
	conn     *coderws.Conn
	cancel   context.CancelFunc
	messages chan openAIWSClientRead
}
type openAIWSClientRead struct {
	kind coderws.MessageType
	body []byte
	err  error
}

func openAIWSIncompleteClientReader(c *gin.Context, conn *coderws.Conn) *openAIWSClientReadPump {
	const key = "openai.incomplete.client_reader"
	if value, ok := c.Get(key); ok {
		if pump, ok := value.(*openAIWSClientReadPump); ok {
			return pump
		}
	}
	// Keep the physical reader alive until a control close frame is sent.
	// Canceling coder/websocket Read directly would close the socket first.
	ctx, cancel := context.WithCancel(context.WithoutCancel(c.Request.Context()))
	p := &openAIWSClientReadPump{conn: conn, ctx: ctx, cancel: cancel, messages: make(chan openAIWSClientRead, 32)}
	c.Set(key, p)
	go func() {
		defer cancel()
		defer close(p.messages)
		for {
			kind, body, err := conn.Read(ctx)
			select {
			case p.messages <- openAIWSClientRead{kind, body, err}:
			case <-ctx.Done():
				return
			default:
				// Bound pipelined requests without blocking disconnect detection.
				// Normal multi-turn clients may queue up to 32 future frames.
				// Close overload immediately rather than retaining unbounded
				// requests or continuing a now-unobservable billable turn.
				cancel()
				_ = conn.Close(coderws.StatusPolicyViolation, "too many queued websocket requests (maximum 32)")
				_ = conn.CloseNow()
				return
			}
			if err != nil {
				return
			}
		}
	}()
	return p
}
func (p *openAIWSClientReadPump) Next(ctx context.Context, timeout time.Duration) (coderws.MessageType, []byte, error) {
	if ctx.Err() != nil {
		return p.closeForControl(ctx)
	}
	var deadline <-chan time.Time
	if timeout > 0 {
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		deadline = timer.C
	}

	select {
	case msg, ok := <-p.messages:
		if !ok {
			return coderws.MessageText, nil, context.Canceled
		}
		return msg.kind, msg.body, msg.err
	case <-ctx.Done():
		return p.closeForControl(ctx)
	case <-deadline:
		_ = p.conn.Close(coderws.StatusNormalClosure, "websocket idle timeout")
		return coderws.MessageText, nil, NewOpenAIWSClientCloseError(coderws.StatusNormalClosure, "websocket idle timeout", context.DeadlineExceeded)
	}
}

func (p *openAIWSClientReadPump) closeForControl(ctx context.Context) (coderws.MessageType, []byte, error) {
	cause := context.Cause(ctx)
	status, reason := coderws.StatusGoingAway, "websocket request canceled"
	if errors.Is(cause, ErrOpenAIWSIngressLeaseLost) {
		status, reason = coderws.StatusTryAgainLater, "websocket ingress capacity lease lost; please reconnect"
	}
	_ = p.conn.Close(status, reason)
	_ = p.conn.CloseNow()
	p.cancel()
	return coderws.MessageText, nil, NewOpenAIWSClientCloseError(status, reason, cause)
}

// Per-turn WS usage state. It is not shared with a previous or retried turn.
type openAIWSIncompleteTurn struct {
	mu       sync.Mutex
	tracker  openAIIncompleteUsage
	observer upstreamResponseModelObserver
	images   *openAIImageOutputCounter
	input    int
	request  []byte
	sent     bool
	started  time.Time
}

func (t *openAIWSIncompleteTurn) Sent(payload []byte) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.tracker = openAIIncompleteUsage{}
	t.observer = upstreamResponseModelObserver{}
	t.images = newOpenAIImageOutputCounter()
	t.input = 0
	t.request = payload
	t.sent = true
	t.started = time.Now()
}
func (t *openAIWSIncompleteTurn) Observe(payload []byte) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.sent {
		t.tracker.Observe(payload)
		t.observer.ObserveOpenAI(payload, gjson.GetBytes(payload, "type").String())
		t.images.AddSSEData(payload)
	}
}
func (t *openAIWSIncompleteTurn) Apply(result *OpenAIForwardResult, gone bool, turnErr error) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.sent || result == nil {
		return false
	}
	gone = gone || result.ClientDisconnect
	var failover *UpstreamFailoverError
	if errors.As(turnErr, &failover) {
		return false
	}
	_, unmeteredTail := t.tracker.output.Apply(t.tracker.usage.OutputTokens)
	missing := !t.tracker.inputReported || !t.tracker.outputReported || unmeteredTail
	hasOutput := t.tracker.output.HasOutput() || t.tracker.usage.OutputTokens > 0
	estimate := (gone || turnErr != nil || !t.tracker.terminal || t.tracker.failed || missing) && (gone || hasOutput) && (!t.tracker.failed || hasOutput)
	reason := openAIIncompleteReason(turnErr, gone, missing && t.tracker.terminal)
	if gone {
		reason = "client_cancelled"
		result.ClientDisconnect = true
	}
	if estimate && !t.tracker.inputReported && result.Usage.InputTokens == 0 && len(t.request) <= 32<<20 {
		t.input = estimateRequestInputTokens(t.request)
	}
	t.tracker.Apply(&result.Usage, t.input, estimate, reason)
	if result.UpstreamResponseModel == "" {
		result.UpstreamResponseModel = t.observer.Model()
	}
	result.UpstreamResponseModelConflict = result.UpstreamResponseModelConflict || t.observer.Conflict()
	if result.UpstreamResponseServiceTier == "" {
		result.UpstreamResponseServiceTier = t.observer.ServiceTier()
	}
	if t.images.Count() > result.ImageCount {
		result.ImageCount = t.images.Count()
		result.ImageOutputSizes = t.images.Sizes()
	}

	if result.RequestID == "" {
		result.RequestID = t.tracker.responseID
	}
	if result.RequestID == "" {
		result.RequestID = uuid.NewString()
	}
	if result.ResponseID == "" {
		result.ResponseID = t.tracker.responseID
	}
	return result.ImageCount > 0 || openAIUsageHasTokens(&result.Usage) || result.Usage.InputTokensEstimated || result.Usage.OutputTokensEstimated
}
func (t *openAIWSIncompleteTurn) Complete() {
	t.mu.Lock()
	t.sent = false
	t.request = nil
	t.mu.Unlock()
}
func (t *openAIWSIncompleteTurn) Active() bool { t.mu.Lock(); defer t.mu.Unlock(); return t.sent }
func openAIWSIsResponseCreate(payload []byte) bool {
	return gjson.GetBytes(payload, "type").String() == "response.create"
}

type openAIWSIncompleteFrameConn struct {
	openaiwsv2.FrameConn
	mu   sync.Mutex
	turn *openAIWSIncompleteTurn
}

func (c *openAIWSIncompleteFrameConn) WriteFrame(ctx context.Context, kind coderws.MessageType, payload []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	err := c.FrameConn.WriteFrame(ctx, kind, payload)
	if err == nil && (kind == coderws.MessageText || kind == coderws.MessageBinary) && openAIWSIsResponseCreate(payload) {
		c.turn.Sent(payload)
	}
	return err
}
func (c *openAIWSIncompleteFrameConn) ReadFrame(ctx context.Context) (coderws.MessageType, []byte, error) {
	kind, payload, err := c.FrameConn.ReadFrame(ctx)
	if err == nil {
		c.mu.Lock()
		c.turn.Observe(payload)
		c.mu.Unlock()
	}
	return kind, payload, err
}
