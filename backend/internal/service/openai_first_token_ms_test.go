//go:build unit

package service

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOpenAIFirstTokenElapsedMs_UsesFastestOfWallClockAndCreated(t *testing.T) {
	t.Parallel()

	observedAt := time.Now()
	firstTokenStart := observedAt.Add(-5 * time.Second)
	createdAt := observedAt.Add(-800 * time.Millisecond).Unix()

	ms := openAIFirstTokenElapsedMs(
		firstTokenStart,
		observedAt,
		[]byte(fmt.Sprintf(`{"id":"chatcmpl_1","object":"chat.completion.chunk","created":%d,"choices":[{"delta":{"content":"h"}}]}`, createdAt)),
	)
	require.GreaterOrEqual(t, ms, 500)
	require.Less(t, ms, 2000, "should prefer upstream created over 5s wall-clock")
}

func TestOpenAIFirstTokenElapsedMs_FallsBackToWallClockWithoutCreated(t *testing.T) {
	t.Parallel()

	observedAt := time.Now()
	firstTokenStart := observedAt.Add(-1200 * time.Millisecond)

	ms := openAIFirstTokenElapsedMs(
		firstTokenStart,
		observedAt,
		[]byte(`{"type":"response.output_text.delta","delta":"h"}`),
	)
	require.GreaterOrEqual(t, ms, 1000)
	require.Less(t, ms, 2000)
}

func TestOpenAIResponseCreatedAt_AcceptsChatCompletionsCreated(t *testing.T) {
	t.Parallel()

	observedAt := time.Now()
	created := observedAt.Add(-2 * time.Second).Unix()
	got, ok := openAIResponseCreatedAt(
		[]byte(fmt.Sprintf(`{"id":"chatcmpl_1","created":%d,"choices":[{"delta":{"reasoning_content":"x"}}]}`, created)),
		observedAt,
	)
	require.True(t, ok)
	require.Equal(t, created, got.Unix())
}
