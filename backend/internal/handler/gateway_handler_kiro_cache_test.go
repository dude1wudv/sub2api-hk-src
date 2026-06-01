package handler

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/stretchr/testify/require"
)

func TestIsKiroGatewayAnthropicAPIKeyAccount(t *testing.T) {
	account := &service.Account{
		ID:       238,
		Platform: service.PlatformAnthropic,
		Type:     service.AccountTypeAPIKey,
		Extra: map[string]any{
			"source": "kiro-gateway",
		},
		Credentials: map[string]any{
			"base_url": "http://kiro-gateway:8000",
		},
	}

	require.True(t, isKiroGatewayAnthropicAPIKeyAccount(account))

	account.Extra = nil
	require.True(t, isKiroGatewayAnthropicAPIKeyAccount(account))

	account.Credentials = map[string]any{"base_url": "https://api.anthropic.com"}
	require.False(t, isKiroGatewayAnthropicAPIKeyAccount(account))

	account.Extra = map[string]any{"source": "kiro-gateway"}
	account.Platform = service.PlatformOpenAI
	require.False(t, isKiroGatewayAnthropicAPIKeyAccount(account))

	require.False(t, isKiroGatewayAnthropicAPIKeyAccount(nil))
}
