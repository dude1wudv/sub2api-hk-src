package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

// Reproduce the production account shape: a third-party API key accepts a
// converted streaming Chat request and returns default despite the forced tier.
// Capture the final HTTP payload, then bill that exact forwarding result.
func TestForcedTierThirdPartyChatStreamingBillingAndObservation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tier := range []string{"priority", "ultrafast"} {
		for _, observed := range []string{"default", "priority", "ultrafast", ""} {
			for _, free := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/free=%t", tier, observed, free), func(t *testing.T) {
					group := forcedOpenAITierTestGroup(tier)
					group.RateMultiplier = 0.1
					group.FreeOpenAIFast, group.FreeOpenAIUltrafast = free, free
					group.ModelPricing = []ChannelModelPricing{{
						Models: []string{"gpt-6.1-sol"}, BillingMode: BillingModeToken,
						InputPrice: pricingMultiplier(0.000002), OutputPrice: pricingMultiplier(0.000010), CacheReadPrice: pricingMultiplier(0.0000001),
					}}
					ctx := context.WithValue(context.Background(), ctxkey.Group, group)
					body := []byte(`{"model":"gpt-6.1-sol","messages":[{"role":"user","content":"Reply OK"}],"stream":true}`)
					rec := httptest.NewRecorder()
					c, _ := gin.CreateTestContext(rec)
					c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(string(body))).WithContext(ctx)
					payload := fmt.Sprintf("data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_forced\",\"model\":\"gpt-6.1-sol\"}}\n\n"+
						"data: {\"type\":\"response.output_text.delta\",\"item_id\":\"it_1\",\"output_index\":0,\"delta\":\"OK\"}\n\n"+
						"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_forced\",\"model\":\"gpt-6.1-sol\",\"status\":\"completed\",\"service_tier\":%q,\"output\":[],\"usage\":{\"input_tokens\":4223,\"output_tokens\":13,\"input_tokens_details\":{\"cached_tokens\":4096}}}}\n\n", observed)
					upstream := &httpUpstreamRecorder{resp: &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(payload))}}
					forwarder := &OpenAIGatewayService{cfg: &config.Config{}, httpUpstream: upstream}
					account := &Account{ID: 1091, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Concurrency: 1,
						Credentials: map[string]any{"api_key": "synthetic-test-key", "base_url": "https://upstream.example.test"},
						Extra:       map[string]any{"openai_responses_supported": true}, Status: StatusActive, Schedulable: true}
					result, err := forwarder.ForwardAsChatCompletions(ctx, c, account, body, "", "")
					require.NoError(t, err)
					require.Equal(t, "/v1/responses", upstream.lastReq.URL.Path)
					require.Equal(t, tier, gjson.GetBytes(upstream.lastBody, "service_tier").String())
					require.Equal(t, tier, optionalStringValue(result.ServiceTier))
					require.Equal(t, observed, result.UpstreamResponseServiceTier)
					require.Equal(t, 4223, result.Usage.InputTokens)
					require.Equal(t, 4096, result.Usage.CacheReadInputTokens)

					usageRepo := &openAIRecordUsageLogRepoStub{inserted: true}
					billing := newOpenAIRecordUsageServiceForTest(usageRepo, &openAIRecordUsageUserRepoStub{}, &openAIRecordUsageSubRepoStub{}, nil)
					billing.resolver = NewModelPricingResolver(nil, billing.billingService)
					err = billing.RecordUsage(ctx, &OpenAIRecordUsageInput{Result: result, APIKey: &APIKey{ID: 261, GroupID: &group.ID, Group: group}, User: &User{ID: 1}, Account: account})
					require.NoError(t, err)
					got := usageRepo.lastLog
					require.Equal(t, tier, optionalStringValue(got.ServiceTier))
					require.Equal(t, tier, optionalStringValue(got.RequestedServiceTier))
					require.Equal(t, observed, optionalStringValue(got.UpstreamServiceTier))
					if observed == "" {
						require.Nil(t, got.ServiceTierDowngraded)
					} else {
						require.NotNil(t, got.ServiceTierDowngraded)
						require.Equal(t, observed == "default" || tier == "ultrafast" && observed == "priority", *got.ServiceTierDowngraded)
					}
					multiplier := 2.5
					if tier == "ultrafast" {
						multiplier = 8
					}
					const standard = 127*0.000002 + 13*0.000010 + 4096*0.0000001
					require.InDelta(t, standard*multiplier, got.TotalCost, 1e-12)
					if free {
						multiplier = 1
					}
					require.InDelta(t, standard*multiplier*0.1, got.ActualCost, 1e-12)
				})
			}
		}
	}
}

func TestForcedTierBillingDoesNotInventMissingOutboundOrWrongGroupTier(t *testing.T) {
	account := &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey}
	for _, tc := range []struct {
		name, requested, want string
		group                 *Group
	}{
		{name: "missing outbound", group: forcedOpenAITierTestGroup("priority")},
		{name: "different outbound", requested: "priority", want: "default", group: forcedOpenAITierTestGroup("ultrafast")},
		{name: "no forced group", requested: "priority", want: "default", group: &Group{ID: 1, Hydrated: true, Platform: PlatformOpenAI, Status: StatusActive}},
		{name: "untrusted group", requested: "priority", want: "default", group: &Group{ID: 1, Platform: PlatformOpenAI, Status: StatusActive, ForceOpenAIFast: true}},
		{name: "other group platform", requested: "priority", want: "default", group: &Group{ID: 1, Hydrated: true, Platform: PlatformAnthropic, Status: StatusActive, ForceOpenAIFast: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := &OpenAIForwardResult{ServiceTier: optionalTrimmedStringPtr(tc.requested), UpstreamResponseServiceTier: "default"}
			resolution := ApplyOpenAIGroupServiceTierBillingResolution(account, tc.group, result)
			require.Equal(t, tc.want, resolution.Billing)
			require.Equal(t, tc.want, optionalStringValue(result.ServiceTier))
		})
	}
}
