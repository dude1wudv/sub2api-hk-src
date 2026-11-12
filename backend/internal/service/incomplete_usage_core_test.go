//go:build unit

package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIncompleteOutputUnicodeAndConfirmedTail(t *testing.T) {
	var u incompleteOutputUsage
	u.AddText(strings.Repeat("前", 100))
	n, estimated := u.Apply(0)
	require.Equal(t, 100, n)
	require.True(t, estimated)
	u.Report(4, false, false)
	for _, part := range []string{"你", "好", "世", "界"} {
		u.AddText(part)
	}
	n, estimated = u.Apply(4)
	require.Equal(t, 8, n, "estimate only the CJK tail after the four confirmed tokens")
	require.True(t, estimated)
	u.Report(6, false, false)
	n, estimated = u.Apply(n)
	require.Equal(t, 6, n)
	require.False(t, estimated)
}

func TestIncompleteOutputFragmentationAndZeros(t *testing.T) {
	var u incompleteOutputUsage
	u.Report(0, true, false)
	for _, part := range []string{"a", "b", "c", "d", "e"} {
		u.AddText(part)
	}
	n, estimated := u.Apply(0)
	require.Equal(t, 2, n)
	require.True(t, estimated)
	u.Report(0, false, false)
	n, estimated = u.Apply(0)
	require.Equal(t, 2, n, "intermediate placeholder cannot erase observed content")
	require.True(t, estimated)
	u.ReportFinal(0, false)
	n, estimated = u.Apply(2)
	require.Zero(t, n, "an explicitly confirmed terminal zero is not missing")
	require.False(t, estimated)
}

func TestIncompleteOutputRealTerminalUsageCorrectsEarlierSnapshot(t *testing.T) {
	for _, final := range []int{0, 3} {
		var u incompleteOutputUsage
		u.AddText("confirmed prefix")
		u.Report(12, false, false)
		u.AddText("additional output")
		u.Report(0, false, false)
		n, estimated := u.Apply(12)
		require.Greater(t, n, 12, "an intermediate zero must not erase a positive prefix or its tail")
		require.True(t, estimated)
		u.ReportFinal(final, false)
		n, estimated = u.Apply(n)
		require.Equal(t, final, n, "real terminal usage wins, including an explicit zero")
		require.False(t, estimated)
	}
}

func TestIncompleteInputDiscountAndBounds(t *testing.T) {
	require.Equal(t, incompleteInputEstimate{Total: 100, Fresh: 5, CacheRead: 95}, splitIncompleteInput(100))
	require.Equal(t, incompleteInputEstimate{Total: 1, Fresh: 1}, splitIncompleteInput(1))
	require.Equal(t, incompleteInputEstimate{}, splitIncompleteInput(-1))
	require.Equal(t, maxIncompleteInputTokens, splitIncompleteInput(maxIncompleteInputTokens+1).Total)
}

func TestIncompleteInputOpaqueAndHiddenState(t *testing.T) {
	base := `{"model":"gpt-4o","input":"Hello there"}`
	withState := `{"model":"gpt-4o","input":"Hello there","previous_response_id":"a hidden server conversation","encrypted_content":"` + strings.Repeat("a", 12000) + `"}`
	require.Equal(t, estimateRequestInputTokens([]byte(base)), estimateRequestInputTokens([]byte(withState)))
	require.Zero(t, estimateRequestInputTokens([]byte(`{"model":"gpt-4o","previous_response_id":"server history"}`)))
	require.Zero(t, estimateRequestInputTokens([]byte(`invalid`)))
	body := []byte(`{"model":"claude-sonnet","messages":[{"role":"user","content":[{"type":"text","text":"hello"},{"type":"image","source":{"type":"base64","data":"` + strings.Repeat("a", 12000) + `"}}]}]}`)
	estimated := estimateRequestInputTokens(body)
	require.Positive(t, estimated)
	require.Less(t, estimated, 200, "attachments are never tokenized as giant text")
	sanitized := sanitizeIncompleteInput(body)
	require.True(t, json.Valid(sanitized))
	require.NotContains(t, string(sanitized), strings.Repeat("a", 100))
	shortImage := []byte(`{"model":"gpt-4o","messages":[{"role":"user","content":[{"type":"text","text":"hello"},{"type":"image_url","image_url":{"url":"data:image/png;base64,aGVsbG8="}}]}]}`)
	otherImage := []byte(strings.ReplaceAll(string(shortImage), "aGVsbG8=", strings.Repeat("a", 600)))
	require.Equal(t, estimateRequestInputTokens(shortImage), estimateRequestInputTokens(otherImage),
		"short data URLs must not be tokenized as ordinary text either")
	require.NotContains(t, string(sanitizeIncompleteInput(shortImage)), "aGVsbG8=")
}

func TestIncompleteInputGeminiToolContentAndEnvelope(t *testing.T) {
	inner := `{"contents":[{"role":"user","parts":[{"functionResponse":{"name":"lookup","response":{"answer":"some visible output"}}}]}]}`
	n := estimateRequestInputTokens([]byte(inner))
	require.Positive(t, n)
	require.Equal(t, n, estimateRequestInputTokens([]byte(`{"project":"ignored","request":`+inner+`}`)))
}

func TestStripOpaqueBlobsKeepsValidEscapedJSON(t *testing.T) {
	body := []byte(`{"text":"hello","blob":"` + strings.Repeat(`a\/`, 1100) + `","tail":"world"}`)
	result := stripOpaqueBlobs(body)
	require.True(t, json.Valid(result))
	require.Contains(t, string(result), "hello")
	require.Contains(t, string(result), "world")
	require.Less(t, len(result), len(body))
}
