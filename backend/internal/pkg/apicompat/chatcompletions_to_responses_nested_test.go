package apicompat

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChatCompletionsToResponsesUsesNestedReasoning(t *testing.T) {
	out, err := ChatCompletionsToResponses(&ChatCompletionsRequest{
		Model:     "gpt-5.4",
		Messages:  []ChatMessage{{Role: "user", Content: []byte(`"hello"`)}},
		Reasoning: &ChatReasoning{Effort: "high", Summary: "detailed"},
	})
	require.NoError(t, err)
	require.NotNil(t, out.Reasoning)
	require.Equal(t, "high", out.Reasoning.Effort)
	require.Equal(t, "detailed", out.Reasoning.Summary)
}

func TestChatCompletionsToResponsesPrefersFlatReasoning(t *testing.T) {
	out, err := ChatCompletionsToResponses(&ChatCompletionsRequest{
		Model:           "gpt-5.4",
		Messages:        []ChatMessage{{Role: "user", Content: []byte(`"hello"`)}},
		ReasoningEffort: "low",
		Reasoning:       &ChatReasoning{Effort: "high", Summary: "detailed"},
	})
	require.NoError(t, err)
	require.Equal(t, "low", out.Reasoning.Effort)
	require.Equal(t, "detailed", out.Reasoning.Summary)
}
