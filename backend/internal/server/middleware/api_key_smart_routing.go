package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/requestmodel"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

func resolveSmartRoutingKey(c *gin.Context, keys *service.APIKeyService, gateways smartRoutingGateways, key *service.APIKey, metadataBridgeEnabled bool) (*service.APIKey, bool) {
	gateway := gateways.openai
	if gateway == nil && gateways.anthropic == nil {
		abortSmartRoutingError(c, 503, "SMART_ROUTING_UNAVAILABLE", "Smart routing is temporarily unavailable")
		return nil, false
	}
	path := c.Request.URL.Path
	for _, prefix := range []string{"/backend-api/codex", "/v1"} {
		path = strings.TrimPrefix(path, prefix)
	}
	discovery := c.Request.Method == http.MethodGet && path == "/models"
	inspection := c.Request.Method == http.MethodGet && (path == "/usage" || path == "/sub2api/billing")
	messages := path == "/messages" || path == "/messages/count_tokens"
	countTokens := path == "/messages/count_tokens"
	textRequest := c.Request.Method == http.MethodPost && (messages || path == "/chat/completions" || path == "/responses" || path == "/responses/compact")
	if !discovery && !inspection && !textRequest {
		abortSmartRoutingError(c, 400, "SMART_ROUTING_ENDPOINT_UNSUPPORTED", "Smart routing supports HTTP Messages, token counting, Chat Completions and Responses; use a fixed-group key for other endpoints")
		return nil, false
	}
	model, previousResponseID := "", ""
	compact := path == "/responses/compact"
	needsResponses := compact
	imageIntent := false
	var parsed *service.ParsedRequest
	if textRequest {
		body, err := httputil.ReadRequestBodyWithPrealloc(c.Request)
		if err != nil {
			status := http.StatusBadRequest
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				status = http.StatusRequestEntityTooLarge
			}
			abortSmartRoutingError(c, status, "INVALID_REQUEST", "Unable to read request body")
			return nil, false
		}
		requestmodel.ResetRequestBody(c.Request, body)
		value := gjson.GetBytes(body, "model")
		if !gjson.ValidBytes(body) || value.Type != gjson.String || strings.TrimSpace(value.String()) == "" {
			abortSmartRoutingError(c, 400, "INVALID_REQUEST", "A non-empty model is required for smart routing")
			return nil, false
		}
		model = value.String()
		// Reject ambiguous model fields before choosing any billing group.
		for _, candidate := range requestmodel.FromBodyCandidates(c.FullPath(), c.GetHeader("Content-Type"), body) {
			if candidate != model {
				abortSmartRoutingError(c, 400, "INVALID_REQUEST", "Conflicting model fields")
				return nil, false
			}
		}
		if path == "/responses" || compact {
			previousResponseID = strings.TrimSpace(gjson.GetBytes(body, "previous_response_id").String())
		}
		imageIntent = service.IsExplicitImageGenerationIntent(path, model, body)
		if imageIntent {
			abortSmartRoutingError(c, 400, "SMART_ROUTING_ENDPOINT_UNSUPPORTED", "Use a fixed-group key for image generation")
			return nil, false
		}
		if path == "/responses" && service.HasCompactionTriggerInInput(body) {
			needsResponses = true
			compact = !gjson.GetBytes(body, "stream").Bool()
		}
		protocol := "chat_completions"
		if messages {
			protocol = service.PlatformAnthropic
		} else if path == "/responses" || compact {
			protocol = "responses"
		}
		parsed, err = service.ParseGatewayRequest(service.NewRequestBodyRef(body), protocol)
		if err != nil {
			abortSmartRoutingError(c, 400, "INVALID_REQUEST", "Failed to parse request body")
			return nil, false
		}
		parsed.SessionContext = &service.SessionContext{
			ClientIP: ip.GetClientIP(c), UserAgent: c.GetHeader("User-Agent"), APIKeyID: key.ID,
		}
		if messages {
			service.SetClaudeCodeClientContext(c, body, parsed)
		} else {
			c.Request = c.Request.WithContext(service.SetClaudeCodeClient(c.Request.Context(), false))
		}
		c.Request = c.Request.WithContext(service.WithThinkingEnabled(c.Request.Context(), parsed.ThinkingEnabled, metadataBridgeEnabled))
	}
	candidates, err := keys.SmartRoutingKeys(c.Request.Context(), key)
	if err != nil {
		abortSmartRoutingError(c, 503, "SMART_ROUTING_UNAVAILABLE", "Unable to load routing groups")
		return nil, false
	}
	if len(candidates) == 0 {
		abortSmartRoutingError(c, 403, "SMART_ROUTING_NO_ACCESS", "No accessible routing groups")
		return nil, false
	}
	if inspection {
		if path == "/sub2api/billing" && len(candidates) > 1 {
			abortSmartRoutingError(c, 400, "SMART_ROUTING_BILLING_UNSUPPORTED", "Billing introspection requires a fixed-group API key")
			return nil, false
		}
		return candidates[0], true
	}
	if discovery {
		models := make([]string, 0)
		for _, candidate := range candidates {
			var ids []string
			var err error
			if candidate.Group.Platform == service.PlatformAnthropic {
				if gateways.anthropic == nil {
					abortSmartRoutingError(c, 503, "SMART_ROUTING_UNAVAILABLE", "Claude routing is temporarily unavailable")
					return nil, false
				}
				ids, err = gateways.anthropic.SmartRoutingModels(c.Request.Context(), candidate)
			} else {
				if gateway == nil {
					abortSmartRoutingError(c, 503, "SMART_ROUTING_UNAVAILABLE", "OpenAI routing is temporarily unavailable")
					return nil, false
				}
				ids, err = gateway.SmartRoutingModels(c.Request.Context(), candidate)
			}
			if err != nil {
				abortSmartRoutingError(c, 503, "SMART_ROUTING_UNAVAILABLE", "Unable to load routing models")
				return nil, false
			}
			models = append(models, ids...)
		}
		c.Request = c.Request.WithContext(service.WithSmartRoutingModels(c.Request.Context(), models))
		return candidates[0], true
	}
	for _, candidate := range candidates {
		if candidate.Group.Platform == service.PlatformAnthropic {
			if previousResponseID != "" || needsResponses {
				continue
			}
			if gateways.anthropic == nil {
				abortSmartRoutingError(c, 503, "SMART_ROUTING_UNAVAILABLE", "Claude routing is temporarily unavailable")
				return nil, false
			}
			prepared, selected, err := gateways.anthropic.PrepareSmartRouting(c.Request.Context(), candidate, parsed, countTokens)
			if err != nil {
				abortSmartRoutingError(c, 503, "SMART_ROUTING_UNAVAILABLE", "Unable to select a Claude routing account")
				return nil, false
			}
			if selected {
				c.Request = c.Request.WithContext(prepared)
				return candidate, true
			}
			continue
		}
		if messages {
			continue
		}
		if gateway == nil {
			abortSmartRoutingError(c, 503, "SMART_ROUTING_UNAVAILABLE", "OpenAI routing is temporarily unavailable")
			return nil, false
		}
		if previousResponseID != "" {
			owned, err := gateway.ValidateOpenAIHTTPResponseOwner(c.Request.Context(), *candidate.GroupID, previousResponseID, key.UserID, key.ID)
			if err != nil {
				abortSmartRoutingError(c, 503, "SMART_ROUTING_UNAVAILABLE", "Unable to verify response ownership")
				return nil, false
			}
			if !owned {
				continue
			}
		}
		capability := service.OpenAIEndpointCapabilityChatCompletions
		if needsResponses && candidate.Group.Platform == service.PlatformOpenAI {
			capability = service.OpenAIEndpointCapabilityResponses
		}
		available, err := gateway.SmartRoutingAvailable(c.Request.Context(), candidate, model, capability, compact)
		if err != nil {
			abortSmartRoutingError(c, 503, "SMART_ROUTING_UNAVAILABLE", "Unable to check routing availability")
			return nil, false
		}
		if available {
			prepared, selected, err := gateway.PrepareSmartRouting(c.Request.Context(), candidate, model, previousResponseID, capability, compact)
			if err != nil {
				abortSmartRoutingError(c, 503, "SMART_ROUTING_UNAVAILABLE", "Unable to select a routing account")
				return nil, false
			}
			if !selected {
				continue
			}
			c.Request = c.Request.WithContext(prepared)
			return candidate, true
		}
	}
	abortSmartRoutingError(c, 503, "SMART_ROUTING_NO_AVAILABLE_GROUP", "No routing group currently has an available account for this model")
	return nil, false
}

// Messages clients expect Anthropic errors even when routing fails before a handler.
func abortSmartRoutingError(c *gin.Context, status int, code, message string) {
	if strings.HasSuffix(c.Request.URL.Path, "/messages") || strings.HasSuffix(c.Request.URL.Path, "/messages/count_tokens") {
		errorType := "api_error"
		switch status {
		case http.StatusBadRequest, http.StatusRequestEntityTooLarge:
			errorType = "invalid_request_error"
		case http.StatusForbidden:
			errorType = "permission_error"
		case http.StatusTooManyRequests:
			errorType = "rate_limit_error"
		case http.StatusServiceUnavailable:
			errorType = "overloaded_error"
		}
		c.AbortWithStatusJSON(status, gin.H{"type": "error", "error": gin.H{"type": errorType, "message": message, "code": code}})
		return
	}
	AbortWithError(c, status, code, message)
}
