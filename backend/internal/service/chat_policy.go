package service

import (
	"maps"
	"strings"
)

// Policy values contain only scalar fields; a map clone fully isolates snapshots.
func CloneChatModelPolicies(value map[string]ChatModelPolicy) map[string]ChatModelPolicy {
	return maps.Clone(value)
}

// Explicit per-model policies take precedence; family defaults grant no capabilities.
func defaultChatTransport(model string) string {
	name := strings.ToLower(model)
	if index := strings.LastIndexByte(name, '/'); index >= 0 {
		name = name[index+1:]
	}
	switch {
	case strings.HasPrefix(name, "claude-"), strings.HasPrefix(name, "anthropic.claude-"):
		return "messages"
	case strings.HasPrefix(name, "gemini-"):
		return "gemini"
	default:
		return "responses"
	}
}
