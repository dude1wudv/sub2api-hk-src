package apicompat

import (
	"strconv"
	"strings"
)

// RepairDeepSeekReasoning preserves the caller's tool pairs and opaque tool
// fields. Only reasoning items carry the incompatible OpenAI encrypted field.
// Existing IDs are reserved before generating stable request-local IDs.
func RepairDeepSeekReasoning(input any) (any, bool) {
	items, ok := input.([]any)
	if !ok {
		return input, false
	}
	used := map[string]bool{}
	for _, raw := range items {
		if item, ok := raw.(map[string]any); ok {
			if id, ok := item["id"].(string); ok {
				used[id] = true
			}
		}
	}
	seq := 0
	nextID := func() string {
		for {
			seq++
			id := "rs_sub2api_" + strconv.Itoa(seq)
			if !used[id] {
				used[id] = true
				return id
			}
		}
	}
	changed, reasoning := false, false
	out := make([]any, 0, len(items))
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			out = append(out, raw)
			continue
		}
		if item["role"] == "user" {
			reasoning = false
		}
		switch item["type"] {
		case "reasoning":
			reasoning = true
			if id, _ := item["id"].(string); strings.TrimSpace(id) == "" {
				item["id"] = nextID()
				changed = true
			}
			if _, exists := item["encrypted_content"]; exists {
				delete(item, "encrypted_content")
				changed = true
			}
		case "function_call", "custom_tool_call", "local_shell_call":
			if !reasoning {
				out = append(out, map[string]any{
					"type": "reasoning", "id": nextID(),
					"summary": []any{}, "content": []any{},
				})
				reasoning, changed = true, true
			}
		}
		out = append(out, item)
	}
	return out, changed
}
