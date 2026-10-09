package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestHKProviderProfilesPreserveRoutingAcrossIngressProtocols(t *testing.T) {
	for _, platform := range []string{PlatformStepFun, PlatformMirasim} {
		require.True(t, domain.IsConcretePlatform(platform))
		require.True(t, domain.UsesOpenAIGateway(platform))
		require.Contains(t, AllowedQuotaPlatforms, platform)
	}
	protocols := []string{APIProtocolChatCompletions, APIProtocolResponses, APIProtocolAnthropic}
	for _, mode := range []string{AccountModePayG, AccountModeStepPlan} {
		account := &Account{Platform: PlatformStepFun, Type: AccountTypeAPIKey, Credentials: map[string]any{
			"api_protocol": APIProtocolAdaptive, "account_mode": mode,
		}}
		for _, inbound := range protocols {
			require.Equal(t, inbound, resolveUpstreamProtocol(account, inbound, "step-router-v1", nil))
			require.NotEmpty(t, account.GetCNProtocolBaseURL(inbound))
		}
	}
	for _, tc := range []struct{ model, want string }{
		{"glm-5.3-flash", APIProtocolChatCompletions},
		{"mirasim/kimi-k3", APIProtocolChatCompletions},
		{"claude-sonnet-5", APIProtocolAnthropic},
		{"mirasim/claude-sonnet-5", APIProtocolAnthropic},
		{"gpt-6-astra", APIProtocolResponses},
		{"mirasim/gpt-6-astra", APIProtocolResponses},
	} {
		account := &Account{Platform: PlatformMirasim, Type: AccountTypeAPIKey}
		for _, inbound := range protocols {
			require.Equal(t, tc.want, resolveUpstreamProtocol(account, inbound, tc.model, nil), "%s/%s", tc.model, inbound)
		}
	}
}
