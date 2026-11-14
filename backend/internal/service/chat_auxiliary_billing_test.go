//go:build unit

package service

import (
	"context"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestChatAuxiliaryBillingPricesAndUnicode(t *testing.T) {
	seconds, err := chatAuxiliaryASRSeconds(3200)
	require.NoError(t, err)
	require.EqualValues(t, 4, seconds)
	characters, err := ChatAuxiliaryTTSCharacters("你好A")
	require.NoError(t, err)
	require.EqualValues(t, 5, characters)
	extra, err := ChatAuxiliaryTTSCharacters("𠀀😀A")
	require.NoError(t, err)
	require.EqualValues(t, 4, extra)
	for _, tc := range []struct {
		kind     string
		quantity int64
		want     float64
	}{{"search", 4, .02}, {"fetch", 0, 0}, {"ocr", 2, .003}, {"asr", seconds, .00086667}, {"tts", characters, .0015}} {
		u, err := chatAuxiliaryUsage(tc.kind, &tc.quantity, "confirmed")
		require.NoError(t, err)
		cost, err := u.cost()
		require.NoError(t, err)
		require.Equal(t, tc.want, cost)
		if tc.kind == "tts" {
			require.Equal(t, "request_text", u.Items[0].QuantitySource)
		}
	}
	u, err := chatAuxiliaryUsage("native_files", nil, "unreported")
	require.NoError(t, err)
	require.Nil(t, u.Items[0].Quantity)
	cost, err := u.cost()
	require.NoError(t, err)
	require.Zero(t, cost)
	require.Equal(t, "provider_unreported", u.Items[0].QuantitySource)
	require.Equal(t, "chat_auxiliary:operation", resolveUsageBillingRequestID(context.WithValue(t.Context(), ctxkey.ClientRequestID, "client-reused"), "chat_auxiliary:operation"))
}

func TestChatAuxiliaryBillingNeverUsesMultipliers(t *testing.T) {
	for _, rate := range []float64{0, .4, 2, 9} {
		quantity := int64(1)
		usage, err := chatAuxiliaryUsage("search", &quantity, "confirmed")
		require.NoError(t, err)
		logRepo := &openAIRecordUsageLogRepoStub{inserted: true}
		billingRepo := &openAIRecordUsageBillingRepoStub{result: &UsageBillingApplyResult{Applied: true}}
		svc := newOpenAIRecordUsageServiceWithBillingRepoForTest(logRepo, billingRepo, &openAIRecordUsageUserRepoStub{}, &openAIRecordUsageSubRepoStub{}, nil)
		groupID := int64(11)
		price := 99.0
		key := &APIKey{ID: 2, UserID: 7, GroupID: &groupID, Group: &Group{ID: groupID, Platform: PlatformOpenAI, RateMultiplier: rate, WebSearchPricePerCall: &price}}
		err = svc.RecordUsage(t.Context(), &OpenAIRecordUsageInput{Result: &OpenAIForwardResult{RequestID: "chat_auxiliary:synthetic", Model: "deepseek-search", chatAuxiliaryOnly: true, ChatAuxiliaryUsage: usage}, APIKey: key, User: &User{ID: 7}, Account: &Account{ID: 23, Platform: PlatformOpenAI}})
		require.NoError(t, err)
		require.Equal(t, .005, logRepo.lastLog.ActualCost)
		require.Equal(t, .005, logRepo.lastLog.TotalCost)
		require.Equal(t, 1.0, logRepo.lastLog.RateMultiplier)
		require.Equal(t, .005, billingRepo.lastCmd.BalanceCost)
		require.NotNil(t, logRepo.lastLog.ChatAuxiliaryUsage)
	}
}

func TestChatAuxiliaryMixedNativeAndMissingModelPrice(t *testing.T) {
	for _, tc := range []struct {
		model string
		want  float64
	}{{"jev-latest", .42}, {"synthetic-unpriced-chat", .02}} {
		account := newDeepSeekToolsAccount()
		account.Platform = PlatformDeepseek
		account.Credentials["base_url"] = "https://api.deepseek.com/anthropic"
		account.Credentials["api_protocol"] = APIProtocolAnthropic
		ctx := context.WithValue(t.Context(), chatNativeContextKey{}, &chatNativeRequest{extensions: true})
		usage := chatNativeAuxiliaryUsage(ctx, account, []byte(`{"type":"message","usage":{"server_tool_use":{"tool_search":2,"tool_fetch":1,"tool_image_search":1,"web_search_requests":9,"web_fetch_requests":7}}}`))
		cost, err := usage.cost()
		require.NoError(t, err)
		require.Equal(t, .02, cost)
		logs := &openAIRecordUsageLogRepoStub{inserted: true}
		bills := &openAIRecordUsageBillingRepoStub{result: &UsageBillingApplyResult{Applied: true}}
		svc := newOpenAIRecordUsageServiceWithBillingRepoForTest(logs, bills, &openAIRecordUsageUserRepoStub{}, &openAIRecordUsageSubRepoStub{}, nil)
		svc.billingService.fallbackPrices["jev-latest"] = &ModelPricing{InputPricePerToken: 1}
		priced, pricingErr := svc.billingService.GetModelPricing("jev-latest")
		require.NoError(t, pricingErr)
		require.Equal(t, 1.0, priced.InputPricePerToken)
		groupID := int64(11)
		err = svc.RecordUsage(t.Context(), &OpenAIRecordUsageInput{Result: &OpenAIForwardResult{RequestID: "synthetic-native", Model: tc.model, Usage: OpenAIUsage{InputTokens: 1}, ChatAuxiliaryUsage: usage}, APIKey: &APIKey{ID: 2, UserID: 7, GroupID: &groupID, Group: &Group{ID: groupID, Platform: PlatformOpenAI, RateMultiplier: .4}}, User: &User{ID: 7}, Account: account})
		require.NoError(t, err)
		require.InDelta(t, tc.want, logs.lastLog.ActualCost, 1e-12)
		require.InDelta(t, tc.want, bills.lastCmd.BalanceCost, 1e-12)
	}
}

func TestChatAuxiliaryNativeStreamUsesTerminalCumulativeUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, terminal := range []string{"message_delta", "error"} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
		a := newDeepSeekToolsAccount()
		a.Platform = PlatformDeepseek
		a.Credentials["base_url"] = "https://api.deepseek.com/anthropic"
		a.Credentials["api_protocol"] = APIProtocolAnthropic
		ctx := context.WithValue(t.Context(), chatNativeContextKey{}, &chatNativeRequest{extensions: true})
		wire := "data: {\"type\":\"message_start\",\"message\":{\"usage\":{\"server_tool_use\":{\"tool_search\":100}}}}\n\n" +
			"data: {\"type\":\"" + terminal + "\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"server_tool_use\":{\"tool_search\":2,\"tool_fetch\":1,\"tool_image_search\":1,\"web_search_requests\":9}}}\n\n" +
			"data: {\"type\":\"message_stop\"}\n\n"
		svc := &OpenAIGatewayService{}
		result, _ := svc.handleNativeAnthropicStreamingResponse(ctx, &http.Response{Header: http.Header{}, Body: io.NopCloser(strings.NewReader(wire))}, c, a, "synthetic", "synthetic", "synthetic", nil, time.Now())
		require.NotNil(t, result)
		cost, err := result.ChatAuxiliaryUsage.cost()
		require.NoError(t, err)
		require.Equal(t, .02, cost)
		require.Nil(t, chatNativeAuxiliaryUsage(t.Context(), a, []byte(`{"usage":{"server_tool_use":{"tool_search":99}}}`)), "public API fields cannot opt into chat pricing")
	}
}

func TestChatAuxiliaryInflightSurvivesFreeGroupAndEmptyModel(t *testing.T) {
	svc := &OpenAIGatewayService{}
	key := &APIKey{User: &User{ID: 7}, Group: &Group{RateMultiplier: 0}, GroupID: i64p(11)}
	for _, kind := range []InflightEstimateKind{InflightEstimateToken, InflightEstimateChatAuxiliary} {
		amount, ok := svc.EstimateInflightReservation(t.Context(), key, InflightEstimateRequest{Kind: kind, ChatAuxiliaryCost: .1})
		require.True(t, ok)
		require.Equal(t, .1, amount)
	}
	for _, amount := range []float64{-1, math.NaN(), math.Inf(1)} {
		_, ok := svc.EstimateInflightReservation(t.Context(), key, InflightEstimateRequest{Kind: InflightEstimateChatAuxiliary, ChatAuxiliaryCost: amount})
		require.False(t, ok)
	}
}

func TestChatAuxiliaryNativeMeteringUsesAnthropicEndpoint(t *testing.T) {
	ctx := context.WithValue(t.Context(), chatNativeContextKey{}, &chatNativeRequest{extensions: true})
	for _, native := range []bool{false, true} {
		account := newDeepSeekToolsAccount()
		account.Platform = PlatformDeepseek
		account.Credentials["api_protocol"] = APIProtocolAdaptive
		openaiBase, anthropicBase := "https://api.deepseek.com", "https://third-party.example/anthropic"
		if native {
			openaiBase, anthropicBase = "https://third-party.example", "https://api.deepseek.com/anthropic"
		}
		account.Credentials["api_base_urls"] = map[string]any{APIProtocolChatCompletions: openaiBase, APIProtocolAnthropic: anthropicBase}
		usage := chatNativeAuxiliaryUsage(ctx, account, []byte(`{"usage":{"server_tool_use":{"tool_search":1}}}`))
		if native {
			require.NotNil(t, usage)
			cost, err := usage.cost()
			require.NoError(t, err)
			require.Equal(t, .005, cost)
		} else {
			require.Nil(t, usage, "third-party Messages must not receive official native pricing")
		}
	}
}
