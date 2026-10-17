//go:build unit

package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAnthropicStreamOutputUsageEstimatesSemanticPayloadOnce(t *testing.T) {
	var observed anthropicStreamOutputUsage
	for _, event := range []string{
		`{"type":"content_block_delta","delta":{"type":"text_delta","text":"猫"}}`,
		`{"type":"content_block_delta","delta":{"type":"thinking_delta","thinking":"x"}}`,
		`{"type":"content_block_start","content_block":{"type":"tool_use","id":"tool_1","name":"lookup","input":{"q":"x"}}}`,
		`{"type":"content_block_delta","delta":{"type":"input_json_delta","partial_json":"{}"}}`,
	} {
		observed.Observe(event)
	}

	// 3 UTF-8 bytes for 猫, 1 for thinking, 9 for the tool input object,
	// and 2 for partial tool JSON: ceil(15 / 4) is 4, rounded once.
	usage := &ClaudeUsage{}
	observed.ApplyFallback(usage)
	require.Equal(t, 4, usage.OutputTokens)
	require.True(t, usage.OutputTokensEstimated)
}

func TestAnthropicStreamOutputUsageIgnoresInvalidAndNonOutputEvents(t *testing.T) {
	var observed anthropicStreamOutputUsage
	for _, event := range []string{
		"not json",
		"{\"type\":\"content_block_delta\",",
		`[DONE]`,
		`{"type":"ping","text":"should not count"}`,
		`{"type":"content_block_delta","delta":{"type":"signature_delta","signature":"signature must not count"}}`,
		`{"type":"content_block_start","content_block":{"type":"tool_use","input":{}}}`,
	} {
		observed.Observe(event)
	}

	usage := &ClaudeUsage{}
	observed.ApplyFallback(usage)
	require.Zero(t, usage.OutputTokens)
	require.False(t, usage.OutputTokensEstimated)
}

func TestAnthropicStreamOutputUsagePreservesExistingOutputAndMarksEstimate(t *testing.T) {
	var observed anthropicStreamOutputUsage
	observed.Observe(`{"type":"message_start","message":{"usage":{"output_tokens":7}}}`)
	observed.Observe(`{"type":"content_block_delta","delta":{"type":"text_delta","text":"x"}}`)

	usage := &ClaudeUsage{OutputTokens: 5}
	observed.ApplyFallback(usage)
	require.Equal(t, 7, usage.OutputTokens)
	require.True(t, usage.OutputTokensEstimated)
}

func TestAnthropicStreamOutputUsageZeroCannotEraseObservedOutput(t *testing.T) {
	var observed anthropicStreamOutputUsage
	observed.Observe(`{"type":"message_start","message":{"usage":{"output_tokens":11}}}`)
	observed.Observe(`{"type":"content_block_delta","delta":{"type":"text_delta","text":"substantial output"}}`)
	observed.Observe(`{"type":"message_delta","usage":{"output_tokens":0}}`)

	usage := &ClaudeUsage{}
	observed.ApplyFallback(usage)
	require.Equal(t, 11, usage.OutputTokens)
	require.True(t, usage.OutputTokensEstimated)
}

func TestAnthropicStreamOutputUsageEmptyZeroRemainsFree(t *testing.T) {
	var observed anthropicStreamOutputUsage
	observed.Observe(`{"type":"message_delta","usage":{"output_tokens":0}}`)
	usage := &ClaudeUsage{}
	observed.ApplyFallback(usage)
	require.Zero(t, usage.OutputTokens)
	require.False(t, usage.OutputTokensEstimated)
}

func TestAnthropicStreamOutputUsageAccumulatesAfterUsageSnapshot(t *testing.T) {
	var observed anthropicStreamOutputUsage
	usage := &ClaudeUsage{}
	observed.Observe(`{"type":"content_block_delta","delta":{"type":"text_delta","text":"abcd"}}`)
	observed.Observe(`{"type":"message_delta","usage":{"output_tokens":10}}`)
	observed.ApplyFallback(usage)
	require.Equal(t, 10, usage.OutputTokens)
	require.False(t, usage.OutputTokensEstimated)

	for range 5 {
		observed.Observe(`{"type":"content_block_delta","delta":{"type":"thinking_delta","thinking":"x"}}`)
		observed.ApplyFallback(usage)
	}
	require.Equal(t, 12, usage.OutputTokens, "round the unmetered tail once, not per fragment")
	require.True(t, usage.OutputTokensEstimated)

	observed.Observe(`{"type":"message_delta","usage":{"output_tokens":11}}`)
	observed.ApplyFallback(usage)
	require.Equal(t, 11, usage.OutputTokens, "a complete exact count replaces the estimate")
	require.False(t, usage.OutputTokensEstimated)
}

func TestAnthropicStreamOutputUsagePreservesProxyErrorEstimate(t *testing.T) {
	var observed anthropicStreamOutputUsage
	observed.Observe(`{"type":"content_block_delta","delta":{"type":"text_delta","text":"abcd"}}`)
	observed.Observe(`{"type":"error","error":{"type":"api_error"},"usage":{"output_tokens":8,"output_tokens_estimated":true}}`)
	usage := &ClaudeUsage{}
	observed.ApplyFallback(usage)
	require.Equal(t, 8, usage.OutputTokens)
	require.True(t, usage.OutputTokensEstimated)
}

func TestAnthropicStreamOutputUsageRegressingSnapshotDoesNotEraseTail(t *testing.T) {
	var observed anthropicStreamOutputUsage
	observed.Observe(`{"type":"message_delta","usage":{"output_tokens":10}}`)
	observed.Observe(`{"type":"content_block_delta","delta":{"type":"text_delta","text":"abcd"}}`)
	observed.Observe(`{"type":"message_delta","usage":{"output_tokens":0}}`)
	usage := &ClaudeUsage{}
	observed.ApplyFallback(usage)
	require.Equal(t, 11, usage.OutputTokens)
	require.True(t, usage.OutputTokensEstimated)
}

func TestAnthropicStreamOutputUsageDoesNotReestimateMeasuredPrefix(t *testing.T) {
	var observed anthropicStreamOutputUsage
	usage := &ClaudeUsage{}
	observed.Observe(`{"type":"content_block_delta","delta":{"type":"text_delta","text":"` + strings.Repeat("x", 100) + `"}}`)
	observed.ApplyFallback(usage)
	require.Equal(t, 25, usage.OutputTokens)
	observed.Observe(`{"type":"message_delta","usage":{"output_tokens":4}}`)
	observed.ApplyFallback(usage)
	require.Equal(t, 4, usage.OutputTokens)
	observed.Observe(`{"type":"content_block_delta","delta":{"type":"text_delta","text":"abcd"}}`)
	observed.ApplyFallback(usage)
	require.Equal(t, 5, usage.OutputTokens, "only the four unmetered tail bytes may be estimated")
	require.True(t, usage.OutputTokensEstimated)
}

func TestAnthropicStreamOutputUsageInitialEstimateIsAFloorNotAPrefix(t *testing.T) {
	var observed anthropicStreamOutputUsage
	observed.Observe(`{"type":"message_start","message":{"usage":{"output_tokens":3,"output_tokens_estimated":true}}}`)
	usage := &ClaudeUsage{}
	observed.ApplyFallback(usage)
	require.Equal(t, 3, usage.OutputTokens)
	require.True(t, usage.OutputTokensEstimated)
	observed.Observe(`{"type":"content_block_delta","delta":{"type":"text_delta","text":"abcdefghijkl"}}`)
	observed.ApplyFallback(usage)
	require.Equal(t, 3, usage.OutputTokens, "the buffered first block is already included in the start estimate")
	require.True(t, usage.OutputTokensEstimated)
}

func TestAnthropicStreamOutputUsageAuthoritativeCountWinsOverEstimate(t *testing.T) {
	var observed anthropicStreamOutputUsage
	observed.Observe(`{"type":"content_block_delta","delta":{"type":"text_delta","text":"a long generated answer"}}`)
	observed.Observe(`{"type":"message_delta","usage":{"output_tokens":15355}}`)

	usage := &ClaudeUsage{}
	observed.ApplyFallback(usage)
	require.Equal(t, 15355, usage.OutputTokens)
	require.False(t, usage.OutputTokensEstimated)
}
