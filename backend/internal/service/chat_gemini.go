package service

import (
	"encoding/json"
	"strings"
)

const chatGeminiPrefix = "/v1beta/models/"
const chatGeminiSuffix = ":streamGenerateContent?alt=sse"

func isChatGeminiPath(path string) bool {
	if !strings.HasPrefix(path, chatGeminiPrefix) || !strings.HasSuffix(path, chatGeminiSuffix) {
		return false
	}
	model := strings.TrimSuffix(strings.TrimPrefix(path, chatGeminiPrefix), chatGeminiSuffix)
	return model == strings.TrimSpace(model) && IsSafeGeminiModelPathSegment(model)
}

func chatGeminiContents(messages []any) []any {
	contents := make([]any, 0, len(messages))
	for _, value := range messages {
		message := chatObject(value)
		role := chatString(message["role"])
		if role == "assistant" {
			role = "model"
		}
		parts, ok := message["content"].([]any)
		if !ok {
			parts = []any{map[string]any{"text": chatString(message["content"])}}
		}
		contents = append(contents, map[string]any{"role": role, "parts": parts})
	}
	return contents
}

func chatGeminiAssistantParts(message ChatMessage) []any {
	parts := []any{}
	for _, raw := range message.Blocks {
		var block map[string]any
		if json.Unmarshal(raw, &block) != nil || block["isTemplate"] == true {
			continue
		}
		part := map[string]any{}
		switch block["type"] {
		case "text":
			part["text"] = chatString(block["text"])
		case "thinking":
			part["text"], part["thought"] = chatString(block["thinking"]), true
		default:
			continue
		}
		if signature := chatString(block["signature"]); signature != "" {
			part["thoughtSignature"] = signature
		}
		parts = append(parts, part)
	}
	return parts
}

func chatGeminiThinking(model string, enabled bool) map[string]any {
	param := geminiThinkingParamForModel(model)
	if param == geminiThinkingParamNone {
		return nil
	}
	result := map[string]any{"includeThoughts": enabled}
	intent := claudeThinkingIntent{}
	if !enabled {
		intent.thinkingType, intent.tier = "disabled", "minimal"
	}
	if !geminiModelNameCarriesThinkingTier(model) {
		switch param {
		case geminiThinkingParamLevel:
			if level := geminiThinkingLevelFor(model, intent); level != "" {
				result["thinkingLevel"] = level
			}
		case geminiThinkingParamBudget:
			if budget, ok := geminiThinkingBudgetFor(model, intent); ok {
				result["thinkingBudget"] = budget
			}
		}
	}
	return result
}

// The native stream has no [DONE] marker. Require a terminal candidate (or a
// prompt refusal), then consume optional trailing usage before EOF completes it.
// No Gemini tools are advertised: existing tool authorization is scoped to the
// OpenAI/DeepSeek gateway. Unsolicited calls fail closed, never execute locally.
func (d *chatStreamDecoder) gemini(event map[string]any) error {
	if usage := chatObject(event["usageMetadata"]); usage != nil {
		cached := max(0, chatInteger(usage["cachedContentTokenCount"]))
		d.geminiUsage = map[string]any{
			"input_tokens":                max(0, chatInteger(usage["promptTokenCount"])-cached),
			"cache_read_input_tokens":     cached,
			"cache_creation_input_tokens": 0,
			"output_tokens":               max(0, chatInteger(usage["candidatesTokenCount"])) + max(0, chatInteger(usage["thoughtsTokenCount"])),
		}
	}
	if reason := chatString(chatObject(event["promptFeedback"])["blockReason"]); reason != "" && reason != "BLOCKED_REASON_UNSPECIFIED" {
		d.stopReason = "refusal"
		d.finished = true
		return nil
	}
	candidates, _ := event["candidates"].([]any)
	for _, value := range candidates {
		candidate := chatObject(value)
		if chatInteger(candidate["index"]) != 0 {
			return ErrChatStream
		}
		parts, _ := chatObject(candidate["content"])["parts"].([]any)
		if d.finished && (len(parts) > 0 || chatString(candidate["finishReason"]) != "") {
			return ErrChatStream
		}
		for _, value := range parts {
			part := chatObject(value)
			if part["functionCall"] != nil || part["inlineData"] != nil || part["fileData"] != nil {
				return ErrChatCapability
			}
			kind := "text"
			if part["thought"] == true {
				kind = "thinking"
			}
			if err := d.sequentialText(kind, chatString(part["text"])); err != nil {
				return err
			}
			if signature := chatString(part["thoughtSignature"]); signature != "" {
				if d.lastIndex < 0 {
					return ErrChatStream
				}
				if err := d.delta(d.lastIndex, map[string]any{"type": "signature_delta", "signature": signature}); err != nil {
					return err
				}
				// A signed part must not merge with the next independently signed part.
				d.lastKind = ""
			}
		}
		switch chatString(candidate["finishReason"]) {
		case "", "FINISH_REASON_UNSPECIFIED":
			continue
		case "STOP":
			d.stopReason = "end_turn"
		case "MAX_TOKENS":
			d.stopReason = "max_tokens"
		default:
			if !isGeminiContentFilterFinishReason(chatString(candidate["finishReason"])) {
				return ErrChatStream
			}
			d.stopReason = "refusal"
		}
		d.finished = true
	}
	return nil
}
