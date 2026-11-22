//go:build unit

package repository

import (
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

// buildSchedulerMetadataAccount 是调度候选的唯一投影出口，openAIAccountSkipReason
// 按 SupportsOpenAIEndpointCapability 判 capability_mismatch。Seedance 分支直接读
// credentials.openai_capabilities 且不走「未配置即放行」，所以该键一旦被裁掉，
// Seedance 账号在选号阶段必然被剔除，而 /v1/models 走 DB 全量仍会列出模型。
func TestSchedulerMetadataKeepsOpenAICapabilities(t *testing.T) {
	account := service.Account{
		ID:       1,
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":             "sk-test",
			"base_url":            "https://relay.example.com/api/v3",
			"openai_capabilities": []string{"seedance"},
		},
	}

	projected := buildSchedulerMetadataAccount(account)

	require.Equal(t, []string{"seedance"}, projected.Credentials["openai_capabilities"])
	require.Equal(t, "https://relay.example.com/api/v3", projected.Credentials["base_url"])
	require.True(t, projected.SupportsOpenAIEndpointCapability(service.OpenAIEndpointCapabilitySeedance))
}

func TestSchedulerMetadataDropsUnlistedCredentials(t *testing.T) {
	account := service.Account{
		ID:       2,
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key":             "sk-test",
			"base_url":            "https://relay.example.com/api/v3",
			"openai_capabilities": []string{"seedance"},
			"model_mapping":       map[string]string{"seedance-2-5": "doubao-seedance-2-5-260628"},
			"unlisted_secret":     "drop-me",
		},
	}

	projected := buildSchedulerMetadataAccount(account)

	require.Equal(t, map[string]string{"seedance-2-5": "doubao-seedance-2-5-260628"}, projected.Credentials["model_mapping"])
	require.NotContains(t, projected.Credentials, "unlisted_secret")
}

func TestSchedulerMetadataKeepsHKDeepSeekSuiteRouting(t *testing.T) {
	account := service.Account{
		ID:       1112,
		Platform: service.PlatformOpenAI,
		Type:     service.AccountTypeAPIKey,
		Credentials: map[string]any{
			"openai_capabilities": []string{"chat", "responses"},
		},
		Extra: map[string]any{
			"deepseek_tools_group_ids":           []int64{81, 223},
			service.DeepSeekSuiteCapabilitiesKey: []string{"chat", "responses", "files", "ocr"},
			"deepseek_suite_fallback_account_id": int64(1182),
			"openai_passthrough":                 true,
			"openai_passthrough_model_filter":    true,
			"unlisted_extra":                     "drop-me",
		},
	}

	payload, err := json.Marshal(buildSchedulerMetadataAccount(account))
	require.NoError(t, err)
	var restored service.Account
	require.NoError(t, json.Unmarshal(payload, &restored))

	require.Equal(t, []any{"chat", "responses"}, restored.Credentials["openai_capabilities"])
	require.Equal(t, []any{float64(81), float64(223)}, restored.Extra["deepseek_tools_group_ids"])
	require.Equal(t, []any{"chat", "responses", "files", "ocr"}, restored.Extra[service.DeepSeekSuiteCapabilitiesKey])
	require.NotContains(t, restored.Extra, "deepseek_suite_fallback_account_id")
	require.Equal(t, true, restored.Extra["openai_passthrough_model_filter"])
	require.NotContains(t, restored.Extra, "unlisted_extra")
}
