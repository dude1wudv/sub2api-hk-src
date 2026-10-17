package service

import (
	"math"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
)

// Match the existing local output estimate of four UTF-8 bytes per token.
// This is a heuristic, not the provider's tokenizer. Keep the counter bounded
// without retaining generated text, thinking, or tool arguments.
const (
	streamOutputEstimateBytesPerToken = uint64(4)
	maxStreamOutputUsageTokens        = math.MaxInt32
	maxStreamOutputUsageBytes         = uint64(maxStreamOutputUsageTokens) * streamOutputEstimateBytesPerToken
)

type anthropicStreamOutputUsage struct {
	semanticBytes       uint64
	initialOutput       int
	initialEstimated    bool
	authoritative       bool
	authoritativeOutput int
	reportedAtBytes     uint64
	reportedEstimated   bool
}

// Observe accepts only complete JSON events received from the upstream. Call it
// once per event, before any locally generated keepalive or response rewrite.
func (u *anthropicStreamOutputUsage) Observe(data string) {
	if u == nil || !gjson.Valid(data) {
		return
	}
	event := gjson.Parse(data)
	switch event.Get("type").String() {
	case "message_start":
		if tokens, ok := streamOutputUsageTokenCount(event.Get("message.usage.output_tokens")); ok && tokens >= u.initialOutput {
			u.initialOutput = tokens
			u.initialEstimated = event.Get("message.usage.output_tokens_estimated").Bool() && tokens > 0
		}
	case "message_delta", "error":
		if tokens, ok := streamOutputUsageTokenCount(event.Get("usage.output_tokens")); ok {
			estimated := event.Get("usage.output_tokens_estimated").Bool()
			// Usage is cumulative. A later placeholder zero or regressing
			// snapshot must not erase a previously observed positive count.
			// An exact count may replace a larger local/provider estimate.
			if u.authoritative && !u.reportedEstimated && tokens < u.authoritativeOutput {
				break
			}
			u.authoritative = true
			u.authoritativeOutput = tokens
			u.reportedAtBytes = u.semanticBytes
			u.reportedEstimated = estimated
		}
	case "content_block_start":
		block := event.Get("content_block")
		switch block.Get("type").String() {
		case "text":
			u.addString(block.Get("text"))
		case "thinking":
			u.addString(block.Get("thinking"))
		case "tool_use":
			input := block.Get("input")
			if input.IsObject() || input.IsArray() {
				hasValue := false
				input.ForEach(func(_, _ gjson.Result) bool {
					hasValue = true
					return false
				})
				if hasValue {
					u.addBytes(len(input.Raw))
				}
			}
		}
	case "content_block_delta":
		delta := event.Get("delta")
		switch delta.Get("type").String() {
		case "text_delta":
			u.addString(delta.Get("text"))
		case "thinking_delta":
			u.addString(delta.Get("thinking"))
		case "input_json_delta":
			u.addString(delta.Get("partial_json"))
		}
	}
}

func (u *anthropicStreamOutputUsage) addString(value gjson.Result) {
	if value.Type == gjson.String {
		u.addBytes(len(value.String()))
	}
}

func (u *anthropicStreamOutputUsage) addBytes(n int) {
	if n <= 0 {
		return
	}
	count := uint64(n)
	if count >= maxStreamOutputUsageBytes-u.semanticBytes {
		u.semanticBytes = maxStreamOutputUsageBytes
		return
	}
	u.semanticBytes += count
}

func streamOutputUsageTokenCount(value gjson.Result) (int, bool) {
	// Preserve the existing usage parser's numeric-string compatibility.
	if value.Type == gjson.String {
		n, err := strconv.ParseInt(strings.TrimSpace(value.String()), 10, 32)
		return int(n), err == nil && n >= 0
	}
	if value.Type != gjson.Number {
		return 0, false
	}
	n := value.Float()
	if n < 0 || n > maxStreamOutputUsageTokens || math.Trunc(n) != n {
		return 0, false
	}
	return int(n), true
}

// ApplyFallback can be called after every complete upstream event and on exit.
// Positive exact usage wins for the output it covers. If more output follows,
// retain that count and estimate the unmetered tail. A placeholder zero cannot
// make observed output free. No semantic output means no invented charge.
func (u *anthropicStreamOutputUsage) ApplyFallback(usage *ClaudeUsage) {
	if u == nil || usage == nil {
		return
	}
	if u.authoritative && !u.reportedEstimated {
		tailBytes := u.semanticBytes - u.reportedAtBytes
		if tailBytes == 0 && (u.authoritativeOutput > 0 || u.semanticBytes == 0) {
			usage.OutputTokens = u.authoritativeOutput
			usage.OutputTokensEstimated = false
			return
		}
		if u.authoritativeOutput > 0 {
			// Do not re-estimate the already metered prefix. In particular, a
			// tokenizer may report far fewer tokens than the byte heuristic.
			tailTokens := int((tailBytes + streamOutputEstimateBytesPerToken - 1) / streamOutputEstimateBytesPerToken)
			usage.OutputTokens = min(maxStreamOutputUsageTokens, u.authoritativeOutput+tailTokens)
			usage.OutputTokensEstimated = true
			return
		}
	}
	if usage.OutputTokens < u.initialOutput {
		usage.OutputTokens = u.initialOutput
	}
	if u.semanticBytes == 0 {
		if u.authoritative {
			usage.OutputTokens = max(usage.OutputTokens, u.authoritativeOutput)
			usage.OutputTokensEstimated = u.reportedEstimated
		} else if u.initialEstimated {
			usage.OutputTokensEstimated = true
		}
		return
	}
	estimated := int((u.semanticBytes + streamOutputEstimateBytesPerToken - 1) / streamOutputEstimateBytesPerToken)
	if u.authoritative {
		tailBytes := u.semanticBytes - u.reportedAtBytes
		tailTokens := int((tailBytes + streamOutputEstimateBytesPerToken - 1) / streamOutputEstimateBytesPerToken)
		estimated = max(estimated, min(maxStreamOutputUsageTokens, u.authoritativeOutput+tailTokens))
	}
	if estimated > usage.OutputTokens {
		usage.OutputTokens = estimated
	}
	usage.OutputTokensEstimated = true
}
