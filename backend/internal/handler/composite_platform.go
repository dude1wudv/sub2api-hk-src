package handler

import (
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

func ensureCompositeTargetPlatform(c *gin.Context, apiKey *service.APIKey, model string) {
	if c == nil || c.Request == nil || apiKey == nil || apiKey.Group == nil || apiKey.Group.Platform != service.PlatformComposite {
		return
	}
	if _, ok := service.ResolvedTargetPlatformFromContext(c.Request.Context()); ok {
		return
	}
	if platform, ok := service.DetectModelPlatform(model); ok {
		c.Request = c.Request.WithContext(service.WithResolvedTargetPlatform(c.Request.Context(), platform))
	}
}

func compositeTargetPlatformAllowed(c *gin.Context, apiKey *service.APIKey, model string, allowed ...string) bool {
	if c == nil || c.Request == nil || apiKey == nil || apiKey.Group == nil || apiKey.Group.Platform != service.PlatformComposite {
		return true
	}
	ensureCompositeTargetPlatform(c, apiKey, model)
	platform, ok := service.ResolvedTargetPlatformFromContext(c.Request.Context())
	if !ok {
		return false
	}
	for _, allowedPlatform := range allowed {
		if platform == allowedPlatform {
			return true
		}
	}
	return false
}

func compositeTargetPlatformResolved(c *gin.Context, apiKey *service.APIKey, model string) bool {
	if c == nil || c.Request == nil || apiKey == nil || apiKey.Group == nil || apiKey.Group.Platform != service.PlatformComposite {
		return true
	}
	ensureCompositeTargetPlatform(c, apiKey, model)
	_, ok := service.ResolvedTargetPlatformFromContext(c.Request.Context())
	return ok
}

func effectiveAPIKeyPlatform(c *gin.Context, apiKey *service.APIKey) string {
	if c != nil && c.Request != nil {
		if platform, ok := service.ResolvedTargetPlatformFromContext(c.Request.Context()); ok {
			return platform
		}
	}
	if apiKey == nil || apiKey.Group == nil {
		return ""
	}
	return apiKey.Group.Platform
}

func openAIReasoningEffortPolicyForRequest(c *gin.Context, apiKey *service.APIKey) (string, []service.ReasoningEffortMapping, string, bool) {
	if apiKey == nil || apiKey.Group == nil {
		return "", nil, "", false
	}
	if apiKey.Group.Platform != service.PlatformAnthropic && apiKey.Group.Platform != service.PlatformOpenAI && apiKey.Group.Platform != service.PlatformComposite {
		return "", nil, "", false
	}
	effectivePlatform := effectiveAPIKeyPlatform(c, apiKey)
	if effectivePlatform != service.PlatformAnthropic && effectivePlatform != service.PlatformOpenAI {
		return "", nil, "", false
	}
	maxEffort, mappings := apiKey.Group.MaxReasoningEffort, apiKey.Group.ReasoningEffortMappings
	if effectivePlatform == service.PlatformAnthropic {
		maxEffort, mappings = anthropicCompatibleReasoningEffortPolicy(maxEffort, mappings)
	}
	return maxEffort, mappings, apiKey.Group.MaxReasoningEffortOverLimit, true
}

func anthropicReasoningEffortPolicyForRequest(c *gin.Context, apiKey *service.APIKey) (string, []service.ReasoningEffortMapping, string, bool) {
	if apiKey == nil || apiKey.Group == nil {
		return "", nil, "", false
	}
	if apiKey.Group.Platform != service.PlatformAnthropic && apiKey.Group.Platform != service.PlatformComposite {
		return "", nil, "", false
	}
	if effectiveAPIKeyPlatform(c, apiKey) != service.PlatformAnthropic {
		return "", nil, "", false
	}
	maxEffort, mappings := anthropicCompatibleReasoningEffortPolicy(apiKey.Group.MaxReasoningEffort, apiKey.Group.ReasoningEffortMappings)
	return maxEffort, mappings, apiKey.Group.MaxReasoningEffortOverLimit, true
}

func anthropicCompatibleReasoningEffortPolicy(maxEffort string, mappings []service.ReasoningEffortMapping) (string, []service.ReasoningEffortMapping) {
	if service.NormalizeMaxReasoningEffort(maxEffort) == "minimal" {
		maxEffort = "low"
	}
	normalizedMappings := append([]service.ReasoningEffortMapping(nil), mappings...)
	for i := range normalizedMappings {
		if service.NormalizeMaxReasoningEffort(normalizedMappings[i].To) == "minimal" {
			normalizedMappings[i].To = "low"
		}
	}
	return maxEffort, normalizedMappings
}

func bindRequestedReasoningEffort(c *gin.Context, body []byte, model string) {
	if c == nil || c.Request == nil {
		return
	}
	effort := service.CanonicalRequestedReasoningEffort(body, model)
	if effort == nil {
		return
	}
	c.Request = c.Request.WithContext(service.WithRequestedReasoningEffort(c.Request.Context(), *effort))
}

func stampOpenAIRequestedReasoningEffort(result *service.OpenAIForwardResult, c *gin.Context) {
	if result == nil || c == nil || c.Request == nil {
		return
	}
	source := service.ReasoningEffortSourceFromContext(c.Request.Context())
	if source == service.ReasoningEffortSourceDefault {
		result.RequestedReasoningEffort = nil
	} else if result.RequestedReasoningEffort == nil {
		result.RequestedReasoningEffort = service.RequestedReasoningEffortFromContext(c.Request.Context())
	}
	if result.ReasoningEffortSource == nil && source != "" {
		result.ReasoningEffortSource = &source
	}
}

func stampForwardRequestedReasoningEffort(result *service.ForwardResult, requested *string) {
	if result == nil {
		return
	}
	if result.RequestedReasoningEffort == nil {
		result.RequestedReasoningEffort = requested
	}
}

func stampForwardReasoningEffortSource(result *service.ForwardResult, c *gin.Context) {
	if result == nil || c == nil || c.Request == nil || result.ReasoningEffortSource != nil {
		return
	}
	if source := service.ReasoningEffortSourceFromContext(c.Request.Context()); source != "" {
		result.ReasoningEffortSource = &source
		if source == service.ReasoningEffortSourceDefault {
			result.RequestedReasoningEffort = nil
		}
	}
}

func applyReasoningEffortDefaultForRequest(c *gin.Context, settingService *service.SettingService, body []byte, inboundProtocol, platform, model string) ([]byte, string, error) {
	cfg := service.ReasoningEffortDefaultConfig{}
	if settingService != nil {
		cfg = settingService.GetGatewayReasoningEffortDefault(c.Request.Context())
	}
	updated, source, err := service.ResolveReasoningEffortDefault(body, inboundProtocol, platform, model, cfg)
	if err != nil {
		return body, "", err
	}
	c.Request = c.Request.WithContext(service.WithReasoningEffortSource(c.Request.Context(), source))
	if source != service.ReasoningEffortSourceDefault {
		bindRequestedReasoningEffort(c, body, model)
	}
	return updated, source, nil
}

func applyOpenAIReasoningEffortPolicyForRequest(c *gin.Context, apiKey *service.APIKey, body []byte) ([]byte, bool, error) {
	maxEffort, mappings, overLimit, ok := openAIReasoningEffortPolicyForRequest(c, apiKey)
	if service.ReasoningEffortSourceFromContext(c.Request.Context()) != service.ReasoningEffortSourceDefault {
		bindRequestedReasoningEffort(c, body, strings.TrimSpace(gjson.GetBytes(body, "model").String()))
	}
	if !ok {
		return body, false, nil
	}
	return service.ApplyReasoningEffortPolicyWithSource(body, maxEffort, mappings, overLimit, service.ReasoningEffortSourceFromContext(c.Request.Context()))
}

func respondOpenAIReasoningEffortPolicyError(c *gin.Context, err error, write func(*gin.Context, int, string, string)) {
	if c == nil || err == nil || write == nil {
		return
	}
	service.MarkOpsClientBusinessLimited(c, service.OpsClientBusinessLimitedReasonLocalPolicyDenied)
	write(c, http.StatusForbidden, "permission_error", err.Error())
}

func applyAnthropicReasoningEffortPolicyForRequest(c *gin.Context, apiKey *service.APIKey, body []byte) ([]byte, bool, error) {
	maxEffort, mappings, overLimit, ok := anthropicReasoningEffortPolicyForRequest(c, apiKey)
	if service.ReasoningEffortSourceFromContext(c.Request.Context()) != service.ReasoningEffortSourceDefault {
		bindRequestedReasoningEffort(c, body, strings.TrimSpace(gjson.GetBytes(body, "model").String()))
	}
	if !ok {
		return body, false, nil
	}
	return service.ApplyReasoningEffortPolicyWithSource(body, maxEffort, mappings, overLimit, service.ReasoningEffortSourceFromContext(c.Request.Context()))
}

func bindOpenAIReasoningEffortPolicyForMessagesRequest(c *gin.Context, apiKey *service.APIKey, body []byte) {
	if c == nil || c.Request == nil {
		return
	}
	if service.ReasoningEffortSourceFromContext(c.Request.Context()) != service.ReasoningEffortSourceDefault {
		bindRequestedReasoningEffort(c, body, strings.TrimSpace(gjson.GetBytes(body, "model").String()))
	}
	// Apply policy to a client value or an injected default. When no rule
	// matched, preserve the bridge's existing synthesized effort behavior.
	effort := gjson.GetBytes(body, "output_config.effort")
	if !effort.Exists() || effort.Type != gjson.String || strings.TrimSpace(effort.String()) == "" {
		return
	}
	maxEffort, mappings, overLimit, ok := openAIReasoningEffortPolicyForRequest(c, apiKey)
	if !ok {
		return
	}
	c.Request = c.Request.WithContext(service.WithOpenAIReasoningEffortPolicy(c.Request.Context(), maxEffort, mappings, overLimit))
}
