package domain

import (
	"fmt"
	"strings"
)

// ChatModelPolicy permits an adapter, never account or group access.
// Policies are keyed by the exact public model name, without wildcard precedence.
type ChatModelPolicy struct {
	Transport                string `json:"transport"`
	ImageInput               bool   `json:"image_input"`
	DocumentInput            bool   `json:"document_input"`
	NativeDeepSeekExtensions bool   `json:"native_deepseek_extensions"`
	NativeFiles              bool   `json:"native_files"`
	NativeSearch             bool   `json:"native_search"`
	NativeReadURL            bool   `json:"native_read_url"`
}

func ValidateChatModelPolicies(policies map[string]ChatModelPolicy) error {
	if len(policies) > 1000 {
		return fmt.Errorf("too many chat model policies")
	}
	for model, p := range policies {
		if model == "" || len(model) > 256 || model != strings.TrimSpace(model) || strings.ContainsAny(model, "*?\\\r\n") {
			return fmt.Errorf("chat model policy requires an exact public model name")
		}
		switch p.Transport {
		case "chat_completions", "messages", "responses":
		default:
			return fmt.Errorf("invalid chat transport for %q", model)
		}
		native := p.NativeDeepSeekExtensions || p.NativeFiles || p.NativeSearch || p.NativeReadURL
		if native && p.Transport != "messages" {
			return fmt.Errorf("native DeepSeek capabilities require messages transport")
		}
		if p.DocumentInput && !p.NativeFiles {
			return fmt.Errorf("document input requires native files")
		}
	}
	return nil
}
