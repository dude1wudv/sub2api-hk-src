package service

import (
	"context"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/gin-gonic/gin"
)

const (
	AnthropicCacheTTLModeInherit = "inherit"
	AnthropicCacheTTLModeOff     = "off"
	AnthropicCacheTTLMode5m      = "5m"
	AnthropicCacheTTLMode1h      = "1h"
)

// NormalizeAnthropicCacheTTLMode also supports keys created before this policy.
func NormalizeAnthropicCacheTTLMode(mode string) string {
	switch mode {
	case AnthropicCacheTTLModeOff, AnthropicCacheTTLMode5m, AnthropicCacheTTLMode1h:
		return mode
	}
	return AnthropicCacheTTLModeInherit
}

func validateAnthropicCacheTTLMode(mode string, allowEmpty bool) error {
	switch mode {
	case AnthropicCacheTTLModeInherit, AnthropicCacheTTLModeOff, AnthropicCacheTTLMode5m, AnthropicCacheTTLMode1h:
		return nil
	case "":
		if allowEmpty {
			return nil
		}
	}
	return infraerrors.BadRequest("API_KEY_CACHE_TTL_INVALID", "anthropic_cache_ttl_mode must be inherit, off, 5m or 1h")
}

// The selected account is authoritative, including when an Anthropic group
// routes to another platform. Use routing/group context only before selection.
func anthropicTargetForBilling(ctx context.Context, apiKey *APIKey, account *Account) bool {
	if account != nil {
		return account.Platform == PlatformAnthropic && !account.IsBedrock()
	}
	if ctx != nil {
		if platform, ok := ResolvedTargetPlatformFromContext(ctx); ok {
			return platform == PlatformAnthropic
		}
	}
	return apiKey != nil && apiKey.Group != nil && apiKey.Group.Platform == PlatformAnthropic
}

// Both request and billing use Key > Account > Global. The inherited global
// policy retains its existing 1h request / 5m billing classification.
func (s *GatewayService) resolveAnthropicCacheTTLPolicy(ctx context.Context, apiKey *APIKey, account *Account) (requestTTL, billingTTL string, apply bool) {
	if !anthropicTargetForBilling(ctx, apiKey, account) {
		return "", "", false
	}
	mode := AnthropicCacheTTLModeInherit
	if apiKey != nil {
		mode = NormalizeAnthropicCacheTTLMode(apiKey.AnthropicCacheTTLMode)
	}
	switch mode {
	case AnthropicCacheTTLModeOff:
		return "", "", false
	case AnthropicCacheTTLMode5m, AnthropicCacheTTLMode1h:
		return mode, mode, true
	}
	if account != nil && account.IsCacheTTLOverrideEnabled() {
		target := account.GetCacheTTLOverrideTarget()
		return target, target, true
	}
	if s.shouldInjectAnthropicCacheTTL1h(ctx, account) {
		return cacheTTLTarget1h, cacheTTLTarget5m, true
	}
	return "", "", false
}

func (s *GatewayService) applyAnthropicCacheTTLForRequest(ctx context.Context, c *gin.Context, account *Account, body []byte) []byte {
	var apiKey *APIKey
	if c != nil {
		apiKey = getAPIKeyFromContext(c)
	}
	target, _, ok := s.resolveAnthropicCacheTTLPolicy(ctx, apiKey, account)
	if !ok {
		return body
	}
	return forceEphemeralCacheControlTTL(body, target)
}
