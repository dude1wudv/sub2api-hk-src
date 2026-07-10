package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

var openAIPromptCacheKeyPartUnsafe = regexp.MustCompile(`[^A-Za-z0-9_-]+`)

func accountConfigBool(account *Account, key string) bool {
	if account == nil {
		return false
	}
	for _, source := range []map[string]any{account.Extra, account.Credentials} {
		if source == nil {
			continue
		}
		switch v := source[key].(type) {
		case bool:
			return v
		case string:
			switch strings.ToLower(strings.TrimSpace(v)) {
			case "1", "true", "yes", "on", "enabled":
				return true
			}
		}
	}
	return false
}

func accountConfigString(account *Account, key string) string {
	if account == nil {
		return ""
	}
	for _, source := range []map[string]any{account.Extra, account.Credentials} {
		if source == nil {
			continue
		}
		if v, ok := source[key].(string); ok {
			if trimmed := strings.TrimSpace(v); trimmed != "" {
				return trimmed
			}
		}
	}
	return ""
}

func openAIPromptCacheKeyPart(raw string) string {
	part := strings.TrimSpace(raw)
	if part == "" {
		return "unknown"
	}
	part = openAIPromptCacheKeyPartUnsafe.ReplaceAllString(part, "_")
	part = strings.Trim(part, "_-")
	if part == "" {
		return "unknown"
	}
	if len(part) > 48 {
		return part[:48]
	}
	return part
}

func shouldInjectConfiguredOpenAIPromptCacheKey(account *Account) bool {
	return account != nil &&
		account.Platform == PlatformOpenAI &&
		account.Type == AccountTypeAPIKey &&
		accountConfigBool(account, "openai_prompt_cache_key_enabled")
}

func shouldPreserveConfiguredOpenAIPromptCacheRetention(account *Account) bool {
	return account != nil &&
		account.Platform == PlatformOpenAI &&
		account.Type == AccountTypeAPIKey &&
		(accountConfigBool(account, "openai_preserve_prompt_cache_retention") ||
			strings.TrimSpace(accountConfigString(account, "openai_prompt_cache_retention")) != "")
}

func configuredOpenAIPromptCacheRetention(account *Account) string {
	retention := strings.TrimSpace(accountConfigString(account, "openai_prompt_cache_retention"))
	switch retention {
	case "24h", "in_memory":
		return retention
	default:
		return ""
	}
}

// openAIModelUsesPromptCacheOptionsTTL reports whether the model follows the
// GPT-5.6+ prompt cache API: prompt_cache_options.ttl (minimum lifetime), not
// the older prompt_cache_retention max-retention policy.
func openAIModelUsesPromptCacheOptionsTTL(model string) bool {
	normalized := strings.ToLower(strings.TrimSpace(model))
	switch {
	case strings.HasPrefix(normalized, "gpt-5.6"):
		return true
	default:
		return false
	}
}

// shouldInjectOpenAIPromptCacheOptionsTTL is true only for non-OAuth accounts
// (API Key). ChatGPT internal Codex rejects prompt_cache_options.
func shouldInjectOpenAIPromptCacheOptionsTTL(account *Account, model string) bool {
	return account != nil &&
		account.Type != AccountTypeOAuth &&
		openAIModelUsesPromptCacheOptionsTTL(model)
}

// openAIPromptCacheOptionsTTL is the only GPT-5.6+ TTL OpenAI currently supports.
const openAIPromptCacheOptionsTTL = "30m"

func ensureOpenAIPromptCacheOptionsTTL(body map[string]any) bool {
	if body == nil {
		return false
	}
	raw, ok := body["prompt_cache_options"]
	if !ok || raw == nil {
		body["prompt_cache_options"] = map[string]any{"ttl": openAIPromptCacheOptionsTTL}
		return true
	}
	opts, ok := raw.(map[string]any)
	if !ok {
		body["prompt_cache_options"] = map[string]any{"ttl": openAIPromptCacheOptionsTTL}
		return true
	}
	ttl, _ := opts["ttl"].(string)
	if strings.TrimSpace(ttl) == openAIPromptCacheOptionsTTL {
		return false
	}
	opts["ttl"] = openAIPromptCacheOptionsTTL
	body["prompt_cache_options"] = opts
	return true
}

func buildConfiguredOpenAIPromptCacheKey(c *gin.Context, body []byte, account *Account, model string) string {
	if !shouldInjectConfiguredOpenAIPromptCacheKey(account) {
		return ""
	}

	apiKeyID := getAPIKeyIDFromContext(c)
	seed := explicitOpenAISessionID(c, body)
	if seed == "" {
		if apiKeyID > 0 {
			seed = fmt.Sprintf("api_key:%d", apiKeyID)
		} else {
			seed = deriveOpenAIContentSessionSeed(body)
		}
	}
	if strings.TrimSpace(seed) == "" {
		return ""
	}

	prefix := strings.TrimSpace(accountConfigString(account, "openai_prompt_cache_key_prefix"))
	if prefix == "" {
		prefix = "sub2api"
	}
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%d\x00%s\x00%s", prefix, apiKeyID, strings.TrimSpace(model), seed)))
	return fmt.Sprintf(
		"%s_%s_k%d_%s",
		openAIPromptCacheKeyPart(prefix),
		openAIPromptCacheKeyPart(model),
		apiKeyID,
		hex.EncodeToString(sum[:12]),
	)
}
