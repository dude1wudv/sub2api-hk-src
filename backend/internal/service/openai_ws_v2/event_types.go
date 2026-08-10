package openai_ws_v2

import "strings"

// IsTerminalEvent reports whether eventType ends the current Responses turn.
//
// Terminal classification is intentionally separate from semantic-output
// classification. A terminal frame must still be forwarded, have usage parsed,
// and close the turn, but it must never become the source of TTFT.
func IsTerminalEvent(eventType string) bool {
	switch strings.TrimSpace(eventType) {
	case "response.completed", "response.done", "response.failed", "response.incomplete", "response.cancelled", "response.canceled":
		return true
	default:
		return false
	}
}

// IsSemanticOutputEvent reports whether eventType carries semantic model output
// that is valid as the first-token/first-output timestamp.
//
// Lifecycle and structure events are explicitly excluded. For Responses event
// types, a .delta suffix is the semantic-output signal so newly added delta
// channels are not silently missed. Non-delta compatibility events are retained
// only when their payload carries completed semantic content.
func IsSemanticOutputEvent(eventType string) bool {
	eventType = strings.TrimSpace(eventType)
	switch eventType {
	case "":
		return false
	case
		// Response lifecycle and terminal events.
		"response.created",
		"response.in_progress",
		"response.completed",
		"response.done",
		"response.failed",
		"response.incomplete",
		"response.cancelled",
		"response.canceled",

		// Response item/content structure events.
		"response.output_item.added",
		"response.output_item.done",
		"response.content_part.added",
		"response.content_part.done",
		"response.output_text.annotation.added",
		"response.reasoning_summary_part.added",
		"response.reasoning_summary_part.done",

		// This only marks the end of an audio stream; it carries no audio content.
		"response.output_audio.done":
		return false
	case
		// Explicit compatibility event retained by this project. Its payload
		// contains the complete output array rather than an incremental delta.
		"response.output",

		// Non-delta events whose payload carries complete semantic content.
		"response.output_text.done",
		"response.refusal.done",
		"response.reasoning_summary_text.done",
		"response.reasoning_text.done",
		"response.function_call_arguments.done",
		"response.custom_tool_call_input.done",
		"response.mcp_call_arguments.done",
		"response.code_interpreter_call_code.done":
		return true
	default:
		return strings.HasPrefix(eventType, "response.") && strings.HasSuffix(eventType, ".delta")
	}
}
