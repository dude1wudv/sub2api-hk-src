package service

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestConfiguredOpenAIPromptCacheKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/responses", nil)
	c.Request.Header.Set("session_id", "stable-session")
	c.Set("api_key", &APIKey{ID: 27})

	account := &Account{
		ID:       245,
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Extra: map[string]any{
			"openai_prompt_cache_key_enabled": true,
			"openai_prompt_cache_key_prefix":  "token101",
		},
	}

	body := []byte(`{"model":"gpt-5.5","input":"hello"}`)
	got := buildConfiguredOpenAIPromptCacheKey(c, body, account, "gpt-5.5")
	require.NotEmpty(t, got)
	require.Contains(t, got, "token101_gpt-5_5_k27_")

	gotAgain := buildConfiguredOpenAIPromptCacheKey(c, body, account, "gpt-5.5")
	require.Equal(t, got, gotAgain)
}

func TestConfiguredOpenAIPromptCacheKeyFallsBackToAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/responses", nil)
	c.Set("api_key", &APIKey{ID: 27})

	account := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Extra: map[string]any{
			"openai_prompt_cache_key_enabled": true,
			"openai_prompt_cache_key_prefix":  "token101",
		},
	}

	first := buildConfiguredOpenAIPromptCacheKey(c, []byte(`{"model":"gpt-5.5","input":"first task"}`), account, "gpt-5.5")
	second := buildConfiguredOpenAIPromptCacheKey(c, []byte(`{"model":"gpt-5.5","input":"second task"}`), account, "gpt-5.5")

	require.NotEmpty(t, first)
	require.Equal(t, first, second)
}

func TestConfiguredOpenAIPromptCacheRetention(t *testing.T) {
	account := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Extra: map[string]any{
			"openai_prompt_cache_retention": "24h",
		},
	}

	require.True(t, shouldPreserveConfiguredOpenAIPromptCacheRetention(account))
	require.Equal(t, "24h", configuredOpenAIPromptCacheRetention(account))
}

func TestOpenAIModelUsesPromptCacheOptionsTTL(t *testing.T) {
	require.True(t, openAIModelUsesPromptCacheOptionsTTL("gpt-5.6-sol"))
	require.True(t, openAIModelUsesPromptCacheOptionsTTL("gpt-5.6-terra"))
	require.True(t, openAIModelUsesPromptCacheOptionsTTL("GPT-5.6-luna"))
	require.False(t, openAIModelUsesPromptCacheOptionsTTL("gpt-5.5"))
	require.False(t, openAIModelUsesPromptCacheOptionsTTL("gpt-5.4"))
}

func TestEnsureOpenAIPromptCacheOptionsTTL(t *testing.T) {
	body := map[string]any{"model": "gpt-5.6-sol"}
	require.True(t, ensureOpenAIPromptCacheOptionsTTL(body))
	opts := body["prompt_cache_options"].(map[string]any)
	require.Equal(t, "30m", opts["ttl"])

	require.False(t, ensureOpenAIPromptCacheOptionsTTL(body))

	body["prompt_cache_options"] = map[string]any{"mode": "explicit", "ttl": "24h"}
	require.True(t, ensureOpenAIPromptCacheOptionsTTL(body))
	opts = body["prompt_cache_options"].(map[string]any)
	require.Equal(t, "30m", opts["ttl"])
	require.Equal(t, "explicit", opts["mode"])
}
