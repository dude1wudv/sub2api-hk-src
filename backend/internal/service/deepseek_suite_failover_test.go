//go:build unit

package service

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestDeepSeekSuiteRetryOnlyUnmeteredRejections(t *testing.T) {
	for _, status := range []int{400, 401, 403, 404, 405, 422, 429, 501} {
		require.True(t, deepSeekSuiteRetryableRejection(status, []byte(`{"error":"unsupported"}`)))
	}
	for _, status := range []int{200, 408, 500, 502, 503, 504} {
		require.False(t, deepSeekSuiteRetryableRejection(status, []byte(`{"error":"unknown"}`)))
	}
	for _, raw := range []string{
		`{"usage":{}}`, `{"response":{"usage":{"input_tokens":1}}}`,
		`{"page_count":1}`, `{"pages_count":1}`, `{"duration":3}`, `{"duration_ms":1000}`,
		`{"id":"file-owned"}`, `{"num_chars":2}`,
	} {
		require.False(t, deepSeekSuiteRetryableRejection(404, []byte(raw)))
	}
}

func TestDeepSeekSuiteMeteredErrorRemainsPending(t *testing.T) {
	for _, body := range []string{`{"error":"partial","duration_ms":1000}`, `{"error":"partial","pages_count":1}`} {
		account := suiteTestAccount()
		upstream := &deepSeekToolsUpstream{response: &http.Response{StatusCode: 400,
			Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}}
		svc := &OpenAIGatewayService{cfg: &config.Config{},
			accountRepo: &deepSeekToolsAccountRepo{latest: account}, httpUpstream: upstream}
		c, recorder := newDeepSeekToolsContext(nil)
		op := &DeepSeekSuiteOperation{ID: "metered-rejection", Status: "started"}
		result, _, err := svc.ForwardDeepSeekSuite(t.Context(), c, suiteTestKey(), account,
			&DeepSeekSuiteInput{Kind: "search", Body: []byte(`{"query":"test"}`)}, op)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotEqual(t, "failed", op.Status, "metered errors must not discard durable reconciliation")
		require.Equal(t, http.StatusBadRequest, recorder.Code)
	}
}

func TestDeepSeekSuiteRejectionLeavesClientUnwritten(t *testing.T) {
	account := suiteTestAccount()
	upstream := &deepSeekToolsUpstream{response: &http.Response{StatusCode: 404,
		Header: http.Header{"X-Request-Id": {"rejected-attempt"}},
		Body:   io.NopCloser(strings.NewReader(`{"error":"unsupported"}`))}}
	svc := &OpenAIGatewayService{cfg: &config.Config{},
		accountRepo: &deepSeekToolsAccountRepo{latest: account}, httpUpstream: upstream}
	c, recorder := newDeepSeekToolsContext(nil)
	op := &DeepSeekSuiteOperation{ID: "rejected"}
	_, _, err := svc.ForwardDeepSeekSuite(t.Context(), c, suiteTestKey(), account,
		&DeepSeekSuiteInput{Kind: "search", Body: []byte(`{"query":"test"}`)}, op)
	var rejection *DeepSeekSuiteRejectedError
	require.ErrorAs(t, err, &rejection)
	require.Equal(t, 404, rejection.StatusCode)
	require.Equal(t, "failed", op.Status)
	require.False(t, c.Writer.Written())
	require.Empty(t, recorder.Header().Get("X-Request-Id"))
}

func TestDeepSeekSuiteFallbackRespectsFileAffinity(t *testing.T) {
	account := suiteTestAccount()
	account.Extra[DeepSeekSuiteFallbackAccountKey] = float64(1182)
	require.EqualValues(t, 1182, account.DeepSeekSuiteFallbackAccountID(0))
	require.Zero(t, account.DeepSeekSuiteFallbackAccountID(account.ID))
	for _, value := range []any{nil, "1182", float64(1182.5), -1, account.ID} {
		account.Extra[DeepSeekSuiteFallbackAccountKey] = value
		require.Zero(t, account.DeepSeekSuiteFallbackAccountID(0))
	}
}

func TestDeepSeekSuiteOfficialUpgradePreservesAuxiliaryEligibility(t *testing.T) {
	account := suiteTestAccount()
	account.ID = 882
	account.Credentials["base_url"] = "https://api.deepseek.com"
	account.Extra[deepSeekToolsGroupIDsExtraKey] = []any{float64(81), float64(223)}
	require.True(t, chatAuxiliaryAccountValid(account))
	for _, group := range []int64{81, 223} {
		require.True(t, account.DeepSeekSuiteAllowed(&group, "files"))
		require.True(t, account.DeepSeekSuiteAllowed(&group, "tts_ws"))
	}
	require.False(t, account.DeepSeekSuiteAllowed(i64p(246), "files"))
}
