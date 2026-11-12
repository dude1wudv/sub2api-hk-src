package service

import (
	"encoding/json"
	"strings"

	"github.com/tidwall/gjson"
)

// Policy adapted from kiss-kedaya/sub2api at 2fb59635e90f98e9977bb189c500ca251cb52dd3.
// These are discounted estimates, never evidence of an upstream cache hit.
const incompleteInputCacheReadPercent = 95
const maxIncompleteInputTokens = 2_000_000

type incompleteInputEstimate struct{ Total, CacheRead, Fresh int }

func splitIncompleteInput(total int) incompleteInputEstimate {
	if total <= 0 {
		return incompleteInputEstimate{}
	}
	total = clampIncompleteInputTokens(total)
	cached := total * incompleteInputCacheReadPercent / 100
	return incompleteInputEstimate{Total: total, CacheRead: cached, Fresh: total - cached}
}

func clampIncompleteInputTokens(tokens int) int { return min(maxIncompleteInputTokens, max(0, tokens)) }

// sanitizeIncompleteInput drops known non-text state even when it is shorter than
// the opaque-run cutoff. Blob folding is only an estimate safeguard: it does not
// mutate the actual upstream request or assign imaginary image/context tokens.
func sanitizeIncompleteInput(body []byte) []byte {
	var value any
	if json.Unmarshal(stripOpaqueBlobs(body), &value) != nil {
		return nil
	}
	var clean func(any)
	clean = func(v any) {
		switch v := v.(type) {
		case map[string]any:
			for key, child := range v {
				switch strings.ToLower(key) {
				case "encrypted_content", "signature", "thoughtsignature", "thinking_signature", "file_data", "inlinedata", "inline_data", "filedata", "file_data_uri":
					delete(v, key)
				default:
					if text, ok := child.(string); ok && len(text) >= 5 && strings.EqualFold(text[:5], "data:") &&
						strings.Contains(strings.ToLower(text), ";base64,") {
						// Even a small image/file data URL is opaque input. The
						// generic long-run folder alone would miss short blobs.
						v[key] = "[blob]"
					} else if key == "data" && (v["type"] == "base64" || v["encoding"] == "base64") {
						delete(v, key)
					} else {
						clean(child)
					}
				}
			}
		case []any:
			for _, child := range v {
				clean(child)
			}
		}
	}
	clean(value)
	out, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	return out
}

func EstimateAnthropicCountTokens(body []byte) (int, error) {
	return estimateAnthropicCountTokensLocally(body)
}

func EstimateOpenAIResponsesInputTokens(body []byte) (int, error) {
	var request openAIInputTokensCountRequest
	if err := json.Unmarshal(body, &request); err != nil {
		return 0, err
	}
	return estimateOpenAIInputTokens(request)
}

func estimateRequestInputTokens(body []byte) int {
	body = sanitizeIncompleteInput(body)
	if len(body) == 0 {
		return 0
	}
	root := gjson.ParseBytes(body)
	// Antigravity wraps the real Gemini request; wrapper IDs are not input.
	if request := root.Get("request"); request.IsObject() {
		root = request
		body = []byte(request.Raw)
	}
	var n int
	var err error
	switch {
	case root.Get("contents").Exists() || root.Get("systemInstruction").Exists():
		// Include function calls/responses and tool declarations as visible input,
		// not just .text. Opaque media and signatures were removed above.
		for _, path := range []string{"systemInstruction", "contents", "tools"} {
			if value := root.Get(path); value.Exists() {
				n += estimateTokensForText(value.Raw)
			}
		}
	case root.Get("input").Exists() || root.Get("instructions").Exists():
		n, err = EstimateOpenAIResponsesInputTokens(body)
	case root.Get("messages").Exists():
		// Chat and Messages differ in tool/message shapes; tokenizing only the
		// transmitted semantic fields covers both without inventing server state.
		codec, codecErr := openAIInputTokensCodecForModel(root.Get("model").String())
		if codecErr != nil {
			return 0
		}
		for _, path := range []string{"system", "messages", "tools", "tool_choice"} {
			value := root.Get(path)
			if !value.Exists() {
				continue
			}
			count, countErr := codec.Count(value.Raw)
			if countErr != nil {
				return 0
			}
			n += count
		}
	default:
		return 0
	}
	if err != nil {
		return 0
	}
	return clampIncompleteInputTokens(n)
}
