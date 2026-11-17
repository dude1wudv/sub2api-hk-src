//go:build unit

package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestOpenAICompactFallbackReevaluatesForcedTierAndBilling(t *testing.T) {
	for _, passthrough := range []bool{false, true} {
		for _, tc := range []struct {
			name, clientTier, want string
			fast                   bool
		}{
			{name: "both flags fall back to fast", fast: true, want: "priority"},
			{name: "ultra alone restores absence"},
			{name: "ultra alone restores flex", clientTier: "flex", want: "flex"},
			{name: "ultra alone preserves explicit client ultra", clientTier: "ultrafast", want: "ultrafast"},
		} {
			t.Run(fmt.Sprintf("passthrough=%t/%s", passthrough, tc.name), func(t *testing.T) {
				group := forcedOpenAITierTestGroup("ultrafast")
				group.ForceOpenAIFast = tc.fast
				ctx := context.WithValue(context.Background(), ctxkey.Group, group)
				tierField := ""
				if tc.clientTier != "" {
					tierField = fmt.Sprintf(`,"service_tier":%q`, tc.clientTier)
				}
				body := []byte(`{"model":"gpt-6.1-sol","stream":false,"instructions":"compact-test","input":[{"type":"compaction_trigger"}]` + tierField + `}`)
				c := newOpenAICompactFallbackTestContext(t, "/v1/responses")
				c.Request = c.Request.WithContext(ctx)
				MarkOpenAINativeCompactionV2(c)
				upstream := &httpUpstreamRecorder{responses: []*http.Response{
					{
						StatusCode: http.StatusBadRequest, Header: http.Header{"Content-Type": {"application/json"}},
						Body: io.NopCloser(strings.NewReader(`{"error":{"code":"context_length_exceeded","message":"context window exceeded"}}`)),
					},
					{
						StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"application/json"}},
						Body: io.NopCloser(strings.NewReader(`{"id":"resp_compact","status":"completed","model":"gpt-5.6-sol","service_tier":"default","output":[],"usage":{"input_tokens":10,"output_tokens":1}}`)),
					},
				}}
				svc := newOpenAIGatewayServiceWithSettings(t, DefaultOpenAIFastPolicySettings())
				svc.cfg = &config.Config{Gateway: config.GatewayConfig{OpenAICompactModel: "gpt-5.6-sol"}}
				svc.httpUpstream = upstream
				account := &Account{
					ID: 1, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Concurrency: 1,
					Credentials: map[string]any{"access_token": "test-token", "chatgpt_account_id": "test-account"},
					Extra:       map[string]any{"openai_passthrough": passthrough},
					Status:      StatusActive, Schedulable: true,
				}
				result, err := svc.Forward(ctx, c, account, body)
				require.NoError(t, err)
				require.Len(t, upstream.bodies, 2)
				require.Equal(t, "ultrafast", gjson.GetBytes(upstream.bodies[0], "service_tier").String())
				require.Equal(t, "gpt-5.6-sol", gjson.GetBytes(upstream.bodies[1], "model").String())
				require.Equal(t, tc.want, gjson.GetBytes(upstream.bodies[1], "service_tier").String())
				require.Equal(t, tc.want != "", gjson.GetBytes(upstream.bodies[1], "service_tier").Exists())
				require.Equal(t, "gpt-5.6-sol", result.UpstreamModel)
				require.Equal(t, tc.want, optionalStringValue(result.ServiceTier))
				resolution := ApplyOpenAIGroupServiceTierBillingResolution(account, group, result)
				require.Equal(t, tc.want, resolution.Billing)
			})
		}
	}
}

func TestPrepareOpenAICompactFallbackReappliesGlobalPolicyForFallbackModel(t *testing.T) {
	account := &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey}
	group := forcedOpenAITierTestGroup("ultrafast")
	group.ForceOpenAIFast = false
	ctx := context.WithValue(context.Background(), ctxkey.Group, group)
	current := []byte(`{"model":"gpt-6.1-sol","service_tier":"ultrafast","input":[{"type":"compaction_trigger"}]}`)
	failure := []byte(`{"error":{"code":"context_length_exceeded","message":"context window exceeded"}}`)
	for _, action := range []string{BetaPolicyActionFilter, BetaPolicyActionBlock} {
		t.Run(action, func(t *testing.T) {
			svc := newOpenAIGatewayServiceWithSettings(t, &OpenAIFastPolicySettings{Rules: []OpenAIFastPolicyRule{{
				ServiceTier: OpenAIFastTierPriority, Action: action, Scope: BetaPolicyScopeAll,
			}}})
			svc.cfg = &config.Config{Gateway: config.GatewayConfig{OpenAICompactModel: "gpt-5.6-sol"}}
			c := newOpenAICompactFallbackTestContext(t, "/v1/responses")
			MarkOpenAINativeCompactionV2(c)
			body, model, retry := svc.prepareOpenAICompactFallbackRetry(
				ctx, c, account, "gpt-6.1-sol", current, openAICompactClientTier{raw: `"priority"`},
				http.StatusBadRequest, "context window exceeded", failure, false)
			if action == BetaPolicyActionBlock {
				require.False(t, retry)
				require.Empty(t, model)
				require.Equal(t, current, body)
			} else {
				require.True(t, retry)
				require.Equal(t, "gpt-5.6-sol", model)
				require.False(t, gjson.GetBytes(body, "service_tier").Exists())
			}
		})
	}
}

func TestOpenAICompactClientTierSnapshotBounded(t *testing.T) {
	tier := snapshotOpenAICompactClientTier([]byte(`{"service_tier":"priority"}`))
	require.Equal(t, `"priority"`, tier.raw)
	require.False(t, tier.unavailable)
	tier = snapshotOpenAICompactClientTier([]byte(`{"service_tier":"` + strings.Repeat("x", 1024) + `"}`))
	require.True(t, tier.unavailable)
	require.Empty(t, tier.raw)
}
