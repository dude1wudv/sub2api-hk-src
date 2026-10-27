package handler

import (
	"context"
	"encoding/json"
	"mime"
	"net/http"
	"strconv"
	"time"

	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (h *OpenAIGatewayHandler) DeepSeekSearch(c *gin.Context) {
	h.handleDeepSeekTool(c, service.DeepSeekToolSearch)
}

func (h *OpenAIGatewayHandler) DeepSeekFetch(c *gin.Context) {
	h.handleDeepSeekTool(c, service.DeepSeekToolFetch)
}

func (h *OpenAIGatewayHandler) handleDeepSeekTool(c *gin.Context, kind service.DeepSeekToolKind) {
	streamStarted := false
	defer h.recoverResponsesPanic(c, &streamStarted)
	setOpenAIClientTransportHTTP(c)
	started := time.Now()
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok || apiKey == nil || apiKey.GroupID == nil || apiKey.Group == nil {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "A fixed-group API key is required")
		return
	}
	if len(apiKey.RoutingGroupIDs) > 0 || (apiKey.Group.Platform != service.PlatformOpenAI && apiKey.Group.Platform != service.PlatformDeepseek) {
		h.errorResponse(c, http.StatusNotFound, "not_found_error", "DeepSeek tools require a supported fixed-group key")
		return
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusInternalServerError, "api_error", "User context not found")
		return
	}
	reqLog := requestLogger(c, "handler.openai_gateway.deepseek_tools",
		zap.Int64("user_id", subject.UserID), zap.Int64("api_key_id", apiKey.ID),
		zap.Int64("group_id", *apiKey.GroupID), zap.String("tool", string(kind)))
	if !h.ensureResponsesDependencies(c, reqLog) {
		return
	}
	contentType, _, contentTypeErr := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if contentTypeErr != nil || contentType != "application/json" {
		writeDeepSeekToolValidationError(c, http.StatusBadRequest, "Content-Type must be application/json")
		return
	}
	body, err := pkghttputil.ReadRequestBodyWithPrealloc(c.Request)
	if err != nil {
		if maxErr, ok := extractMaxBytesError(err); ok {
			writeDeepSeekToolValidationError(c, http.StatusRequestEntityTooLarge, buildBodyTooLargeMessage(maxErr.Limit))
		} else {
			writeDeepSeekToolValidationError(c, http.StatusBadRequest, "Failed to read request body")
		}
		return
	}
	auditText, err := service.ValidateDeepSeekToolRequest(kind, body)
	if err != nil {
		writeDeepSeekToolValidationError(c, http.StatusBadRequest, err.Error())
		return
	}
	setOpsRequestContext(c, kind.Model(), false)
	setOpsEndpointContext(c, kind.Endpoint(), int16(service.RequestTypeSync))
	auditBody, _ := json.Marshal(map[string]any{"messages": []map[string]any{{"role": "user", "content": auditText}}})
	if decision := h.checkSecurityAudit(c, reqLog, apiKey, subject, service.ContentModerationProtocolOpenAIChat, kind.Model(), auditBody); decision != nil && !decision.AllowNextStage {
		h.openAISecurityAuditError(c, decision)
		return
	}
	subscription, _ := middleware2.GetSubscriptionFromContext(c)
	if err := h.billingCacheService.CheckBillingEligibility(c.Request.Context(), apiKey.User, apiKey, apiKey.Group, subscription, service.QuotaPlatform(c.Request.Context(), apiKey)); err != nil {
		h.deepSeekToolBillingError(c, err)
		return
	}
	inflightDone, err := reserveInflightBalance(c, h.billingCacheService, h.gatewayService, apiKey, subscription,
		service.InflightEstimateRequest{Kind: service.InflightEstimateDeepSeekTool})
	if err != nil {
		h.deepSeekToolBillingError(c, err)
		return
	}
	defer inflightDone()
	userRelease, acquired := h.acquireResponsesUserSlot(c, subject.UserID, subject.Concurrency, false, &streamStarted, reqLog)
	if !acquired {
		return
	}
	if userRelease != nil {
		defer userRelease()
	}
	service.SetOpsLatencyMs(c, service.OpsAuthLatencyMsKey, time.Since(started).Milliseconds())
	// Model-based profit checks cannot price a model-free standalone tool.
	c.Request = c.Request.WithContext(service.WithOpenAIProfitControlSuppressed(c.Request.Context()))
	selection, _, err := h.gatewayService.SelectAccountWithSchedulerForCapability(
		c.Request.Context(), apiKey.GroupID, "", "", "", nil,
		service.OpenAIUpstreamTransportHTTPSSE, service.OpenAIEndpointCapabilityDeepSeekTools,
		false, false, false, apiKey.Group.Platform)
	if err != nil || selection == nil || selection.Account == nil {
		markOpsRoutingCapacityLimitedIfNoAvailable(c, err)
		h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "No available DeepSeek tool accounts for this group")
		return
	}
	account := selection.Account
	setOpsSelectedAccount(c, account.ID, account.Platform)
	accountRelease, slotStatus := h.acquireResponsesAccountSlot(c, apiKey.GroupID, "", selection, false, &streamStarted, reqLog)
	if slotStatus != openAISlotAcquireOK {
		if slotStatus == openAISlotAcquireProfitVetoed {
			h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "DeepSeek tool account is unavailable")
		}
		return
	}
	if accountRelease != nil {
		defer accountRelease()
	}
	account = selection.Account
	service.SetOpsLatencyMs(c, service.OpsRoutingLatencyMsKey, time.Since(started).Milliseconds())
	result, forwardErr := h.gatewayService.ForwardDeepSeekTool(c.Request.Context(), c, account, *apiKey.GroupID, kind, body)
	if forwardErr != nil {
		if !c.Writer.Written() && c.Request.Context().Err() == nil {
			h.errorResponse(c, http.StatusBadGateway, "upstream_error", "Upstream tool request failed")
		}
		return
	}
	if result == nil {
		return // Non-200 upstream response was already forwarded, with no bill.
	}
	userAgent := c.GetHeader("User-Agent")
	clientIP := ip.GetClientIP(c)
	sessionID := service.ExtractClientSessionID(c)
	payloadHash := service.HashUsageRequestPayload(body)
	quotaPlatform := service.QuotaPlatform(c.Request.Context(), apiKey)
	// Mandatory work retains the reservation until settlement finishes, even
	// when the request is cancelled or the ordinary usage queue is full.
	h.submitMandatoryUsageRecordTask(c.Request.Context(), func(ctx context.Context) {
		if err := h.gatewayService.RecordUsage(ctx, &service.OpenAIRecordUsageInput{
			Result: result, APIKey: apiKey, User: apiKey.User, Account: account,
			Subscription: subscription, InboundEndpoint: kind.Endpoint(), UpstreamEndpoint: kind.Endpoint(),
			UserAgent: userAgent, IPAddress: clientIP, SessionID: sessionID,
			RequestPayloadHash: payloadHash, APIKeyService: h.apiKeyService,
			QuotaPlatform: quotaPlatform, PricingAt: started,
		}); err != nil {
			reqLog.Error("deepseek_tools.record_usage_failed", zap.Error(err))
		}
	})
}

func (h *OpenAIGatewayHandler) deepSeekToolBillingError(c *gin.Context, err error) {
	status, code, message, retryAfter := billingErrorDetails(err)
	if retryAfter > 0 {
		c.Header("Retry-After", strconv.Itoa(retryAfter))
	}
	h.errorResponse(c, status, code, message)
}

func writeDeepSeekToolValidationError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": gin.H{"message": message, "type": "invalid_request_error", "param": nil, "code": "invalid_request_error"}})
}
