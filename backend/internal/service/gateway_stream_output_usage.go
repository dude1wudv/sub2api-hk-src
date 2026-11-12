package service

import (
	"math"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
)

// Bound the streaming counters without retaining generated text. Output uses
// the shared English/CJK heuristic; byte counters are only recovery diagnostics.
const (
	streamOutputEstimateBytesPerToken = uint64(4)
	maxStreamOutputUsageTokens        = math.MaxInt32
	maxStreamOutputUsageBytes         = uint64(maxStreamOutputUsageTokens) * streamOutputEstimateBytesPerToken
)

type anthropicStreamOutputUsage struct{ incompleteOutputUsage }

// Observe accepts only complete JSON events received from the upstream. Call it
// once per event, before any locally generated keepalive or response rewrite.
func (u *anthropicStreamOutputUsage) Observe(data string) {
	if u == nil || !gjson.Valid(data) {
		return
	}
	event := gjson.Parse(data)
	switch event.Get("type").String() {
	case "message_start":
		if tokens, ok := streamOutputUsageTokenCount(event.Get("message.usage.output_tokens")); ok {
			u.Report(tokens, true, event.Get("message.usage.output_tokens_estimated").Bool())
		}
	case "message_delta", "error":
		if tokens, ok := streamOutputUsageTokenCount(event.Get("usage.output_tokens")); ok {
			u.Report(tokens, false, event.Get("usage.output_tokens_estimated").Bool())
		}
	case "content_block_start":
		block := event.Get("content_block")
		switch block.Get("type").String() {
		case "text":
			u.addString(block.Get("text"))
		case "thinking":
			u.addString(block.Get("thinking"))
		case "tool_use", "server_tool_use":
			u.addString(block.Get("name"))
			input := block.Get("input")
			if input.IsObject() || input.IsArray() {
				hasValue := false
				input.ForEach(func(_, _ gjson.Result) bool {
					hasValue = true
					return false
				})
				if hasValue {
					u.AddText(string(stripOpaqueBlobs([]byte(input.Raw))))
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
		u.AddText(value.String())
	}
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
	usage.OutputTokens, usage.OutputTokensEstimated = u.Apply(usage.OutputTokens)
}
