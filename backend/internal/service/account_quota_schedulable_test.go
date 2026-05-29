//go:build unit

package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAccountIsSchedulable_QuotaExceeded(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		account *Account
		want    bool
	}{
		{
			name: "apikey daily quota exceeded",
			account: &Account{
				Status:      StatusActive,
				Schedulable: true,
				Type:        AccountTypeAPIKey,
				Extra: map[string]any{
					"quota_daily_limit": 10.0,
					"quota_daily_used":  10.0,
					"quota_daily_start": now.Add(-1 * time.Hour).Format(time.RFC3339),
				},
			},
			want: false,
		},
		{
			name: "apikey weekly quota exceeded",
			account: &Account{
				Status:      StatusActive,
				Schedulable: true,
				Type:        AccountTypeAPIKey,
				Extra: map[string]any{
					"quota_weekly_limit": 50.0,
					"quota_weekly_used":  50.0,
					"quota_weekly_start": now.Add(-2 * 24 * time.Hour).Format(time.RFC3339),
				},
			},
			want: false,
		},
		{
			name: "apikey total quota exceeded",
			account: &Account{
				Status:      StatusActive,
				Schedulable: true,
				Type:        AccountTypeAPIKey,
				Extra: map[string]any{
					"quota_limit": 100.0,
					"quota_used":  100.0,
				},
			},
			want: false,
		},
		{
			name: "apikey quota not exceeded",
			account: &Account{
				Status:      StatusActive,
				Schedulable: true,
				Type:        AccountTypeAPIKey,
				Extra: map[string]any{
					"quota_daily_limit": 10.0,
					"quota_daily_used":  5.0,
					"quota_daily_start": now.Add(-1 * time.Hour).Format(time.RFC3339),
				},
			},
			want: true,
		},
		{
			name: "apikey expired daily period restores schedulable",
			account: &Account{
				Status:      StatusActive,
				Schedulable: true,
				Type:        AccountTypeAPIKey,
				Extra: map[string]any{
					"quota_daily_limit": 10.0,
					"quota_daily_used":  10.0,
					"quota_daily_start": now.Add(-25 * time.Hour).Format(time.RFC3339),
				},
			},
			want: true,
		},
		{
			name: "oauth ignores quota exceeded",
			account: &Account{
				Status:      StatusActive,
				Schedulable: true,
				Type:        AccountTypeOAuth,
				Extra: map[string]any{
					"quota_daily_limit": 10.0,
					"quota_daily_used":  10.0,
					"quota_daily_start": now.Add(-1 * time.Hour).Format(time.RFC3339),
				},
			},
			want: true,
		},
		{
			name: "bedrock quota exceeded",
			account: &Account{
				Status:      StatusActive,
				Schedulable: true,
				Type:        AccountTypeBedrock,
				Extra: map[string]any{
					"quota_limit": 200.0,
					"quota_used":  200.0,
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.account.IsSchedulable())
		})
	}
}

func TestAccountIsOpenAICodexQuotaDraining(t *testing.T) {
	tests := []struct {
		name    string
		account *Account
		want    bool
	}{
		{
			name: "openai oauth below drain threshold",
			account: &Account{
				Platform: PlatformOpenAI,
				Type:     AccountTypeOAuth,
				Extra: map[string]any{
					openAICodexPrimaryUsedPercentExtraKey: 98.9,
				},
			},
			want: false,
		},
		{
			name: "openai oauth at drain threshold",
			account: &Account{
				Platform: PlatformOpenAI,
				Type:     AccountTypeOAuth,
				Extra: map[string]any{
					openAICodexPrimaryUsedPercentExtraKey: 99.0,
				},
			},
			want: true,
		},
		{
			name: "openai oauth above drain threshold string",
			account: &Account{
				Platform: PlatformOpenAI,
				Type:     AccountTypeOAuth,
				Extra: map[string]any{
					openAICodexPrimaryUsedPercentExtraKey: "99.5",
				},
			},
			want: true,
		},
		{
			name: "openai setup token at drain threshold",
			account: &Account{
				Platform: PlatformOpenAI,
				Type:     AccountTypeSetupToken,
				Extra: map[string]any{
					openAICodexPrimaryUsedPercentExtraKey: 99.0,
				},
			},
			want: true,
		},
		{
			name: "openai api key ignored",
			account: &Account{
				Platform: PlatformOpenAI,
				Type:     AccountTypeAPIKey,
				Extra: map[string]any{
					openAICodexPrimaryUsedPercentExtraKey: 100.0,
				},
			},
			want: false,
		},
		{
			name: "non openai oauth ignored",
			account: &Account{
				Platform: PlatformGemini,
				Type:     AccountTypeOAuth,
				Extra: map[string]any{
					openAICodexPrimaryUsedPercentExtraKey: 100.0,
				},
			},
			want: false,
		},
		{name: "nil account", account: nil, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.account.IsOpenAICodexQuotaDraining())
		})
	}
}

func TestOpenAICodexUsageExhausted(t *testing.T) {
	require.False(t, openAICodexUsageExhausted(map[string]any{
		openAICodex5hUsedPercentExtraKey: 98.9,
		openAICodex7dUsedPercentExtraKey: 0,
	}))
	require.True(t, openAICodexUsageExhausted(map[string]any{
		openAICodex5hUsedPercentExtraKey: 99.0,
	}))
	require.True(t, openAICodexUsageExhausted(map[string]any{
		openAICodex7dUsedPercentExtraKey: "99.5",
	}))
}

func TestAccountOpenAICodexUsageRecoveredAfterResetAt(t *testing.T) {
	past := time.Now().Add(-time.Minute).Format(time.RFC3339)
	future := time.Now().Add(time.Hour).Format(time.RFC3339)

	account := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Extra: map[string]any{
			openAICodexPrimaryUsedPercentExtraKey: 100,
			openAICodexPrimaryResetAtExtraKey:     past,
			openAICodex5hUsedPercentExtraKey:      100,
			openAICodex5hResetAtExtraKey:          past,
			openAICodex7dUsedPercentExtraKey:      100,
			openAICodex7dResetAtExtraKey:          past,
		},
	}
	require.False(t, account.IsOpenAICodexQuotaDraining())
	require.False(t, account.IsOpenAICodexUsageExhausted())
	require.True(t, account.IsOpenAICodexUsageRecovered())

	account.Extra[openAICodex7dResetAtExtraKey] = future
	require.True(t, account.IsOpenAICodexUsageExhausted())
	require.False(t, account.IsOpenAICodexUsageRecovered())
}
