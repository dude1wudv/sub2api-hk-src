package service

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
)

func chatReasoningEfforts(model string, caps ChatCapabilities) []string {
	if !caps.Reasoning {
		return nil
	}
	name := strings.ToLower(model)
	if slash := strings.LastIndexByte(name, '/'); slash >= 0 {
		name = name[slash+1:]
	}
	// DeepSeek has low/high/max, not medium; this UI exposes shared profiles.
	if caps.NativeDeepSeekExtensions || strings.HasPrefix(name, "deepseek-") {
		return []string{"low", "high"}
	}
	return []string{"low", "medium", "high"}
}

// The toggle selects a profile; normal mode is low effort, not disabled thinking.
// Provider-specific normalization, model mappings and billing remain in the gateway.
func applyChatReasoning(prepared *chatPreparedRequest, model string) {
	if !prepared.caps.Reasoning {
		return
	}
	effort := prepared.options.ReasoningEffort
	switch prepared.caps.Transport {
	case "responses":
		reasoning := map[string]any{"effort": effort}
		if prepared.options.Thinking {
			reasoning["summary"] = "auto"
		}
		prepared.body["reasoning"] = reasoning
	case "chat_completions":
		prepared.body["reasoning_effort"] = effort
	case "messages":
		levels := claude.EffortLevelsForModel(model)
		name := strings.ToLower(model)
		if prepared.caps.NativeDeepSeekExtensions || !strings.Contains(name, "claude") {
			prepared.body["output_config"] = map[string]any{"effort": effort}
			return
		}
		if len(levels) > 0 {
			prepared.body["output_config"] = map[string]any{"effort": effort}
			// Opus 4.5 supports effort but still needs a manual thinking budget.
			if !strings.Contains(strings.ReplaceAll(name, ".", "-"), "opus-4-5") {
				prepared.body["thinking"] = map[string]any{"type": "adaptive"}
				return
			}
		}
		budget := 1024
		switch effort {
		case "medium":
			budget = 4096
		case "high":
			budget = 10240
		}
		prepared.body["thinking"] = map[string]any{"type": "enabled", "budget_tokens": budget}
		// Leave the existing answer allowance after the reasoning budget.
		prepared.body["max_tokens"] = budget + 8192
	case "gemini":
		if thinking := chatGeminiThinking(model, effort, prepared.options.Thinking); len(thinking) > 0 {
			generation := prepared.body["generationConfig"].(map[string]any)
			generation["thinkingConfig"] = thinking
			if budget, ok := thinking["thinkingBudget"].(int); ok {
				generation["maxOutputTokens"] = budget + 8192
			}
		}
	}
}
