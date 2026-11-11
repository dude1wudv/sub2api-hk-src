//go:build unit

package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestAnthropicKeyCacheTTLPolicy(t *testing.T) {
	repo := &gatewayTTLSettingRepo{data: map[string]string{SettingKeyEnableAnthropicCacheTTL1hInjection: "true"}}
	gatewayForwardingCache.Store(&cachedGatewayForwardingSettings{})
	t.Cleanup(func() { gatewayForwardingCache.Store(&cachedGatewayForwardingSettings{}) })
	svc := &GatewayService{settingService: NewSettingService(repo, &config.Config{})}
	accountOverride := func(target string) *Account {
		return &Account{Platform: PlatformAnthropic, Type: AccountTypeOAuth, Extra: map[string]any{"cache_ttl_override_enabled": true, "cache_ttl_override_target": target}}
	}
	for _, tt := range []struct {
		name, mode, platform, requestTTL, billingTTL string
		account                                      *Account
		apply                                        bool
	}{
		{"key 1h over account 5m", "1h", PlatformAnthropic, "1h", "1h", accountOverride("5m"), true},
		{"key 5m over account 1h and global", "5m", PlatformAnthropic, "5m", "5m", accountOverride("1h"), true},
		{"off over account and global", "off", PlatformAnthropic, "", "", accountOverride("1h"), false},
		{"inherit account 5m over global", "inherit", PlatformAnthropic, "5m", "5m", accountOverride("5m"), true},
		{"inherit legacy global", "inherit", PlatformAnthropic, "1h", "5m", &Account{Platform: PlatformAnthropic, Type: AccountTypeOAuth}, true},
		{"old key legacy global", "", PlatformAnthropic, "1h", "5m", &Account{Platform: PlatformAnthropic, Type: AccountTypeSetupToken}, true},
		{"apikey inherit has no global injection", "inherit", PlatformAnthropic, "", "", &Account{Platform: PlatformAnthropic, Type: AccountTypeAPIKey}, false},
		{"key works on native apikey", "1h", PlatformAnthropic, "1h", "1h", &Account{Platform: PlatformAnthropic, Type: AccountTypeAPIKey}, true},
		{"selected openai wins over anthropic group", "1h", PlatformAnthropic, "", "", &Account{Platform: PlatformOpenAI, Type: AccountTypeAPIKey}, false},
		{"selected gemini wins over anthropic group", "5m", PlatformAnthropic, "", "", &Account{Platform: PlatformGemini, Type: AccountTypeAPIKey}, false},
		{"bedrock is excluded", "1h", PlatformAnthropic, "", "", &Account{Platform: PlatformAnthropic, Type: AccountTypeBedrock}, false},
		{"selected anthropic wins over openai group", "5m", PlatformOpenAI, "5m", "5m", &Account{Platform: PlatformAnthropic, Type: AccountTypeAPIKey}, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			key := &APIKey{AnthropicCacheTTLMode: tt.mode, Group: &Group{Platform: tt.platform}}
			requestTTL, billingTTL, apply := svc.resolveAnthropicCacheTTLPolicy(context.Background(), key, tt.account)
			require.Equal(t, tt.apply, apply)
			require.Equal(t, tt.requestTTL, requestTTL)
			require.Equal(t, tt.billingTTL, billingTTL)
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Set("api_key", key)
			body := []byte(`{"z":{"extension":true},"cache_control":{"type":"ephemeral","ttl":"5m","scope":"global"},"system":[{"type":"text","text":"system","cache_control":{"type":"ephemeral"}}],"messages":[{"role":"user","content":[{"type":"text","text":"hi","cache_control":{"type":"ephemeral","ttl":"1h"}},{"type":"text","text":"plain"}]}],"tools":[{"name":"tool","cache_control":{"type":"ephemeral","ttl":"1h"}}],"omega":2}`)
			next := svc.applyAnthropicCacheTTLForRequest(context.Background(), ctx, tt.account, body)
			if !tt.apply {
				require.Equal(t, body, next, "off and non-native targets must preserve wire bytes")
				return
			}
			for _, path := range []string{"cache_control.ttl", "system.0.cache_control.ttl", "messages.0.content.0.cache_control.ttl", "tools.0.cache_control.ttl"} {
				require.Equal(t, tt.requestTTL, gjson.GetBytes(next, path).String(), path)
			}
			require.True(t, gjson.GetBytes(next, "z.extension").Bool())
			require.Equal(t, "global", gjson.GetBytes(next, "cache_control.scope").String())
			require.False(t, gjson.GetBytes(next, "messages.0.content.1.cache_control").Exists())
			assertJSONTokenOrder(t, string(next), `"z"`, `"cache_control"`, `"system"`, `"messages"`, `"tools"`, `"omega"`)
			require.Equal(t, next, svc.applyAnthropicCacheTTLForRequest(context.Background(), ctx, tt.account, next))
		})
	}
	key := &APIKey{AnthropicCacheTTLMode: "1h", Group: &Group{Platform: PlatformAnthropic}}
	_, _, ok := svc.resolveAnthropicCacheTTLPolicy(WithResolvedTargetPlatform(context.Background(), PlatformOpenAI), key, nil)
	require.False(t, ok, "smart routing target takes precedence over the original group")
	key.Group.Platform = PlatformOpenAI
	request, billing, ok := svc.resolveAnthropicCacheTTLPolicy(WithResolvedTargetPlatform(context.Background(), PlatformAnthropic), key, nil)
	require.True(t, ok)
	require.Equal(t, "1h", request)
	require.Equal(t, "1h", billing)
}

func TestAnthropicKeyCacheTTLPassthroughWireBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, mode := range []string{"1h", "5m", "off"} {
		t.Run(mode, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
			c.Set("api_key", &APIKey{AnthropicCacheTTLMode: mode})
			upstream := &anthropicHTTPUpstreamRecorder{resp: &http.Response{
				StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}},
				Body: io.NopCloser(strings.NewReader(`{"usage":{"cache_creation_input_tokens":12,"output_tokens":1}}`)),
			}}
			svc := &GatewayService{cfg: &config.Config{}, httpUpstream: upstream, rateLimitService: &RateLimitService{}}
			body := []byte(`{"model":"claude-sonnet-4-6","system":[{"type":"text","text":"prefix","cache_control":{"type":"ephemeral","ttl":"5m"}}],"messages":[{"role":"user","content":"hi"}]}`)
			parsed := &ParsedRequest{Body: NewRequestBodyRef(body), Model: "claude-sonnet-4-6"}
			_, err := svc.Forward(context.Background(), c, newAnthropicAPIKeyAccountForTest(), parsed)
			require.NoError(t, err)
			want := mode
			if mode == "off" {
				want = "5m"
			}
			require.Equal(t, want, gjson.GetBytes(upstream.lastBody, "system.0.cache_control.ttl").String())
			require.Equal(t, upstream.lastBody, parsed.Body.Bytes(), "billing payload hash must reflect the wire body")
		})
	}
}

func TestAnthropicKeyCacheTTLRecordUsage(t *testing.T) {
	for _, tt := range []struct {
		mode       string
		five, hour int
	}{
		{"1h", 0, 12}, {"5m", 12, 0}, {"off", 7, 5}, {"inherit", 7, 5},
	} {
		t.Run(tt.mode, func(t *testing.T) {
			repo := &openAIRecordUsageLogRepoStub{inserted: true}
			userRepo := &openAIRecordUsageUserRepoStub{}
			svc := newGatewayRecordUsageServiceForTest(repo, userRepo, &openAIRecordUsageSubRepoStub{})
			result := &ForwardResult{RequestID: "ttl-" + tt.mode, Model: "claude-sonnet-4", Duration: time.Second, Usage: ClaudeUsage{CacheCreationInputTokens: 12, CacheCreation5mTokens: 7, CacheCreation1hTokens: 5}}
			err := svc.RecordUsage(context.Background(), &RecordUsageInput{Result: result, APIKey: &APIKey{ID: 1, AnthropicCacheTTLMode: tt.mode}, User: &User{ID: 2}, Account: &Account{ID: 3, Platform: PlatformAnthropic, Type: AccountTypeAPIKey}})
			require.NoError(t, err)
			require.Equal(t, tt.five, result.Usage.CacheCreation5mTokens)
			require.Equal(t, tt.hour, result.Usage.CacheCreation1hTokens)
			require.Equal(t, 1, repo.calls)
			require.Equal(t, tt.five, repo.lastLog.CacheCreation5mTokens)
			require.Equal(t, tt.hour, repo.lastLog.CacheCreation1hTokens)
			require.Equal(t, 1, userRepo.deductCalls)
			require.InDelta(t, repo.lastLog.ActualCost, userRepo.lastAmount, 1e-12)
			require.Greater(t, userRepo.lastAmount, float64(0))
		})
	}
}
