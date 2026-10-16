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
	authoritative       bool
	authoritativeOutput int
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
		if tokens, ok := streamOutputUsageTokenCount(event.Get("message.usage.output_tokens")); ok && tokens > u.initialOutput {
			u.initialOutput = tokens
		}
	case "message_delta":
		if tokens, ok := streamOutputUsageTokenCount(event.Get("usage.output_tokens")); ok {
			u.authoritative = true
			u.authoritativeOutput = tokens
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

// ApplyFallback is called once forwarding has ended. A declared message_delta
// count always wins, including an explicit zero. Otherwise estimate only the
// semantic output actually received, rounding once across all fragments.
func (u *anthropicStreamOutputUsage) ApplyFallback(usage *ClaudeUsage) {
	if u == nil || usage == nil {
		return
	}
	if u.authoritative {
		usage.OutputTokens = u.authoritativeOutput
		usage.OutputTokensEstimated = false
		return
	}
	if usage.OutputTokens < u.initialOutput {
		usage.OutputTokens = u.initialOutput
	}
	if u.semanticBytes == 0 {
		return
	}
	estimated := int((u.semanticBytes + streamOutputEstimateBytesPerToken - 1) / streamOutputEstimateBytesPerToken)
	if estimated > usage.OutputTokens {
		usage.OutputTokens = estimated
	}
	usage.OutputTokensEstimated = true
}
