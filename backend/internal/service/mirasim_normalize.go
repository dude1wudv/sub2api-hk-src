package service

import (
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"
)

const mirasimClaudeFingerprint = "You are a Claude agent, built on Anthropic's Claude Agent SDK."

// normalizeMirasimBody applies the relay's strict request-shape requirements
// after Sub2API has selected and mapped the upstream model.
func normalizeMirasimBody(path string, body []byte) ([]byte, error) {
	var request map[string]any
	if err := json.Unmarshal(body, &request); err != nil {
		return nil, errors.New("invalid Mirasim request JSON")
	}
	model, _ := request["model"].(string)
	model = strings.TrimPrefix(model, "mirasim/")
	if model == "" {
		return nil, errors.New("Mirasim model is required")
	}
	request["model"] = model
	if strings.HasPrefix(model, "claude-") && path != "/v1/messages" {
		return nil, errors.New("Mirasim Claude models require /v1/messages")
	}
	if strings.HasPrefix(model, "gpt-") && path != "/v1/responses" {
		return nil, errors.New("Mirasim GPT models require /v1/responses")
	}
	switch path {
	case "/v1/messages":
		delete(request, "top_p")
		if messages, ok := request["messages"].([]any); ok {
			kept := make([]any, 0, len(messages))
			system := mirasimTextBlocks(request["system"])
			for _, item := range messages {
				message, ok := item.(map[string]any)
				if ok && message["role"] == "system" {
					system = append(system, mirasimTextBlocks(message["content"])...)
				} else {
					kept = append(kept, item)
				}
			}
			request["messages"] = kept
			if len(system) > 0 {
				request["system"] = system
			}
		}
		if strings.HasPrefix(model, "claude-") {
			system := mirasimTextBlocks(request["system"])
			if !mirasimHasFingerprint(system) {
				system = append([]any{mirasimTextBlock(mirasimClaudeFingerprint)}, system...)
			}
			request["system"] = mirasimTruncateSystem(system, 200)
		}
	case "/v1/responses":
		if input, ok := request["input"].(string); ok {
			request["input"] = []any{map[string]any{"role": "user", "content": []any{map[string]any{"type": "input_text", "text": input}}}}
		}
		if strings.HasPrefix(model, "gpt-") && request["stream"] != true {
			return nil, errors.New("Mirasim GPT models require stream=true")
		}
	}
	return json.Marshal(request)
}

func mirasimTextBlock(text string) map[string]any {
	return map[string]any{"type": "text", "text": text}
}

func mirasimTextBlocks(value any) []any {
	switch v := value.(type) {
	case string:
		return []any{mirasimTextBlock(v)}
	case []any:
		blocks := make([]any, 0, len(v))
		for _, item := range v {
			if text, ok := item.(string); ok {
				blocks = append(blocks, mirasimTextBlock(text))
			} else {
				blocks = append(blocks, item)
			}
		}
		return blocks
	default:
		return nil
	}
}

func mirasimHasFingerprint(blocks []any) bool {
	for _, block := range blocks {
		if value, ok := block.(map[string]any); ok {
			if text, _ := value["text"].(string); strings.Contains(text, mirasimClaudeFingerprint) {
				return true
			}
		}
	}
	return false
}

func mirasimTruncateSystem(blocks []any, limit int) []any {
	result := make([]any, 0, len(blocks))
	for _, block := range blocks {
		if limit <= 0 {
			break
		}
		value, ok := block.(map[string]any)
		if !ok {
			continue
		}
		text, _ := value["text"].(string)
		if len(text) > limit {
			text = text[:limit]
			for !utf8.ValidString(text) {
				text = text[:len(text)-1]
			}
		}
		if text != "" {
			result = append(result, mirasimTextBlock(text))
			limit -= len(text)
		}
	}
	return result
}
