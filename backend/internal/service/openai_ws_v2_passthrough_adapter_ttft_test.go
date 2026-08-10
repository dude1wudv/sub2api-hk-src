package service

import (
	"testing"
	"time"

	openaiwsv2 "github.com/Wei-Shaw/sub2api/internal/service/openai_ws_v2"
	coderws "github.com/coder/websocket"
	"github.com/stretchr/testify/require"
)

func TestOpenAIWSPassthroughSemanticOutputClassification(t *testing.T) {
	t.Parallel()

	cases := []struct {
		eventType string
		semantic  bool
		terminal  bool
	}{
		{eventType: "response.created"},
		{eventType: "response.in_progress"},
		{eventType: "response.output_item.added"},
		{eventType: "response.output_item.done"},
		{eventType: "response.completed", terminal: true},
		{eventType: "response.done", terminal: true},
		{eventType: "response.failed", terminal: true},
		{eventType: "response.incomplete", terminal: true},
		{eventType: "response.cancelled", terminal: true},
		{eventType: "response.canceled", terminal: true},
		{eventType: "response.output_text.delta", semantic: true},
		{eventType: "response.reasoning_summary_text.delta", semantic: true},
		{eventType: "response.audio.delta", semantic: true},
		{eventType: "response.function_call_arguments.delta", semantic: true},
		{eventType: "response.custom_tool_call_input.delta", semantic: true},
		{eventType: "response.code_interpreter_call_code.delta", semantic: true},
		{eventType: "response.mcp_call_arguments.delta", semantic: true},
		{eventType: "response.future_output_channel.delta", semantic: true},
		{eventType: "response.output", semantic: true},
		{eventType: "response.output_text.done", semantic: true},
		{eventType: "response.refusal.done", semantic: true},
		{eventType: "response.reasoning_summary_text.done", semantic: true},
		{eventType: "response.reasoning_text.done", semantic: true},
		{eventType: "response.function_call_arguments.done", semantic: true},
		{eventType: "response.custom_tool_call_input.done", semantic: true},
		{eventType: "response.mcp_call_arguments.done", semantic: true},
		{eventType: "response.code_interpreter_call_code.done", semantic: true},
		{eventType: "response.output_audio.done"},
		{eventType: "response.output_text.annotation.added"},
		{eventType: "response.reasoning_summary_part.added"},
		{eventType: "error"},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.eventType, func(t *testing.T) {
			payload := []byte(`{"type":"` + tc.eventType + `"}`)
			require.Equal(t, tc.semantic, openAIWSPassthroughStartsSemanticOutput(payload), tc.eventType)
			require.Equal(t, tc.terminal, openAIWSPassthroughIsTerminalOutput(payload), tc.eventType)
			require.Equal(t, tc.semantic, openaiwsv2.IsSemanticOutputEvent(tc.eventType), tc.eventType)
			require.Equal(t, tc.terminal, openaiwsv2.IsTerminalEvent(tc.eventType), tc.eventType)
		})
	}
}

func TestOpenAIWSPassthroughFirstOutputDeadlineWaitsForSemanticOutput(t *testing.T) {
	t.Parallel()

	conn := &openAIWSPassthroughFirstOutputFrameConn{
		resolveDeadline: func([]byte) openAIWSPassthroughFirstOutputDeadline {
			return openAIWSPassthroughFirstOutputDeadline{timeout: time.Second}
		},
		activeReadTimeout: time.Second,
		deadlineChanged:   make(chan struct{}, 8),
	}

	generation := conn.armDeadline([]byte(`{"type":"response.create","model":"gpt-5"}`))
	require.NotZero(t, generation)
	state := conn.deadlineState()
	require.True(t, state.armed)
	require.Equal(t, openAIWSPassthroughDeadlinePhaseFirstSemantic, state.deadline.phase)

	for _, eventType := range []string{
		"response.created",
		"response.in_progress",
		"response.output_item.added",
		"response.output_item.done",
	} {
		conn.observeUpstreamActivity(coderws.MessageText, []byte(`{"type":"`+eventType+`"}`))
		state = conn.deadlineState()
		require.True(t, state.armed, eventType)
		require.Equal(t, openAIWSPassthroughDeadlinePhaseFirstSemantic, state.deadline.phase, eventType)
	}

	conn.observeUpstreamActivity(coderws.MessageText, []byte(`{"type":"response.output_text.delta","delta":"x"}`))
	state = conn.deadlineState()
	require.True(t, state.armed)
	require.Equal(t, openAIWSPassthroughDeadlinePhaseActiveRead, state.deadline.phase)

	conn.observeUpstreamActivity(coderws.MessageText, []byte(`{"type":"response.completed"}`))
	require.False(t, conn.deadlineState().armed, "terminal output should disarm the active-read deadline")
}

func TestOpenAIWSPassthroughTerminalOutputDisarmsFirstOutputDeadline(t *testing.T) {
	t.Parallel()

	for _, eventType := range []string{
		"response.completed",
		"response.done",
		"response.failed",
		"response.incomplete",
		"response.cancelled",
		"response.canceled",
	} {
		eventType := eventType
		t.Run(eventType, func(t *testing.T) {
			conn := &openAIWSPassthroughFirstOutputFrameConn{
				resolveDeadline: func([]byte) openAIWSPassthroughFirstOutputDeadline {
					return openAIWSPassthroughFirstOutputDeadline{timeout: time.Second}
				},
				deadlineChanged: make(chan struct{}, 2),
			}
			conn.armDeadline([]byte(`{"type":"response.create"}`))
			conn.observeUpstreamActivity(coderws.MessageText, []byte(`{"type":"`+eventType+`"}`))
			require.False(t, conn.deadlineState().armed)
		})
	}
}
