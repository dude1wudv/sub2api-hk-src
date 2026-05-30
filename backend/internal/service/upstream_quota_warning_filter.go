package service

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
)

func shouldSuppressUpstreamQuotaWarningText(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	return strings.Contains(normalized, "heads up") ||
		strings.Contains(normalized, "5h limit") ||
		strings.Contains(normalized, "/status for a breakdown")
}

func shouldSuppressAnthropicMapEvent(event map[string]any) bool {
	eventType, _ := event["type"].(string)
	if eventType != "content_block_delta" {
		return false
	}
	delta, _ := event["delta"].(map[string]any)
	text, _ := delta["text"].(string)
	return shouldSuppressUpstreamQuotaWarningText(text)
}

func shouldSuppressAnthropicCompatEvent(evt apicompat.AnthropicStreamEvent) bool {
	if evt.Type != "content_block_delta" || evt.Delta == nil {
		return false
	}
	return shouldSuppressUpstreamQuotaWarningText(evt.Delta.Text)
}
