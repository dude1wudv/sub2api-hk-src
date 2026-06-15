package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAccountIsKiroGatewayAnthropicAPIKey(t *testing.T) {
	account := &Account{
		Platform: PlatformAnthropic,
		Type:     AccountTypeAPIKey,
		Extra: map[string]any{
			"source": "kiro-gateway",
		},
		Credentials: map[string]any{
			"base_url": "https://api.anthropic.com",
		},
	}

	require.True(t, account.IsKiroGatewayAnthropicAPIKey())

	account.Extra = nil
	account.Credentials = map[string]any{"base_url": "http://kiro-gateway:8000"}
	require.True(t, account.IsKiroGatewayAnthropicAPIKey())

	account.Credentials = map[string]any{"base_url": "https://api.anthropic.com"}
	require.False(t, account.IsKiroGatewayAnthropicAPIKey())

	account.Extra = map[string]any{"source": "kiro-gateway"}
	account.Platform = PlatformOpenAI
	require.False(t, account.IsKiroGatewayAnthropicAPIKey())

	var nilAccount *Account
	require.False(t, nilAccount.IsKiroGatewayAnthropicAPIKey())
}

func TestKiroGatewayAccountMaxConcurrency(t *testing.T) {
	kiro := &Account{
		Platform: PlatformAnthropic,
		Type:     AccountTypeAPIKey,
		Extra:    map[string]any{"source": "kiro-gateway"},
	}

	kiro.Concurrency = 12
	require.Equal(t, KiroGatewayMaxAccountConcurrency, kiro.GatewayMaxConcurrency())

	kiro.Concurrency = KiroGatewayMaxAccountConcurrency
	require.Equal(t, KiroGatewayMaxAccountConcurrency, kiro.GatewayMaxConcurrency())

	kiro.Concurrency = 2
	require.Equal(t, 2, kiro.GatewayMaxConcurrency())

	kiro.Concurrency = 0
	require.Equal(t, KiroGatewayMaxAccountConcurrency, kiro.GatewayMaxConcurrency())

	kiro.Concurrency = -1
	require.Equal(t, KiroGatewayMaxAccountConcurrency, kiro.GatewayMaxConcurrency())

	plain := &Account{Platform: PlatformAnthropic, Type: AccountTypeAPIKey, Concurrency: 12}
	require.Equal(t, 12, plain.GatewayMaxConcurrency())

	plain.Concurrency = 0
	require.Equal(t, 0, plain.GatewayMaxConcurrency())
}

func TestKiroGatewayLoadConcurrency(t *testing.T) {
	loadFactor := 10
	kiro := &Account{
		Platform:    PlatformAnthropic,
		Type:        AccountTypeAPIKey,
		Extra:       map[string]any{"source": "kiro-gateway"},
		Concurrency: 12,
		LoadFactor:  &loadFactor,
	}

	require.Equal(t, KiroGatewayMaxAccountConcurrency, kiro.GatewayLoadConcurrency())

	loadFactor = 2
	require.Equal(t, 2, kiro.GatewayLoadConcurrency())

	plain := &Account{Platform: PlatformAnthropic, Type: AccountTypeAPIKey, Concurrency: 12, LoadFactor: &loadFactor}
	require.Equal(t, 2, plain.GatewayLoadConcurrency())
}
