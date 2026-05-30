package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/stretchr/testify/require"
)

func TestShouldSuppressUpstreamQuotaWarningText(t *testing.T) {
	require.True(t, shouldSuppressUpstreamQuotaWarningText("⚠ Heads up, you have less than 25% of your 5h limit left. Run /status for a breakdown."))
	require.True(t, shouldSuppressUpstreamQuotaWarningText("25% of your 5h limit left."))
	require.True(t, shouldSuppressUpstreamQuotaWarningText("Run /status for a breakdown."))
	require.False(t, shouldSuppressUpstreamQuotaWarningText("Note: here is the answer you asked for."))
	require.False(t, shouldSuppressUpstreamQuotaWarningText("You have less than 25% of your daily limit left."))
}

func TestShouldSuppressAnthropicEvents(t *testing.T) {
	event := map[string]any{
		"type": "content_block_delta",
		"delta": map[string]any{
			"type": "text_delta",
			"text": "⚠ Heads up, you have less than 25% of your 5h limit left. Run /status for a breakdown.",
		},
	}
	require.True(t, shouldSuppressAnthropicMapEvent(event))

	compatEvent := apicompat.AnthropicStreamEvent{
		Type: "content_block_delta",
		Delta: &apicompat.AnthropicDelta{
			Type: "text_delta",
			Text: "⚠ Heads up, you have less than 25% of your 5h limit left. Run /status for a breakdown.",
		},
	}
	require.True(t, shouldSuppressAnthropicCompatEvent(compatEvent))
}
