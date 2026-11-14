package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (h *OpenAIGatewayHandler) ChatAuxiliary(c *gin.Context) {
	if !service.HasChatAuxiliaryScope(c.Request.Context()) {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	streamStarted := false
	defer h.recoverResponsesPanic(c, &streamStarted)
	setOpenAIClientTransportHTTP(c)
	started := time.Now()
	kind := c.Param("kind")
	key, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok || key == nil || key.Group == nil {
		h.errorResponse(c, 401, "authentication_error", "A fixed-group API key is required")
		return
	}
	id, err := h.gatewayService.ValidateChatAuxiliaryScope(c.Request.Context(), key, kind)
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID != key.UserID {
		h.errorResponse(c, 403, "permission_error", "User scope does not match")
		return
	}
	reqLog := requestLogger(c, "handler.chat_auxiliary", zap.Int64("user_id", subject.UserID), zap.String("kind", kind))
	if !h.ensureResponsesDependencies(c, reqLog) {
		return
	}
	if kind != "search" && kind != "fetch" && kind != "asr" && kind != "tts" && kind != "ocr" {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	inputBody, err := service.ParseChatAuxiliaryInput(kind, c.Request)
	if err != nil {
		writeDeepSeekToolValidationError(c, 400, "Invalid auxiliary request")
		return
	}
	body, auditText := inputBody.Body, inputBody.AuditText
	model := "deepseek-" + kind
	endpoint := "/v1/tools/" + kind
	setOpsRequestContext(c, model, kind == "tts")
	requestType := service.RequestTypeSync
	if kind == "tts" {
		requestType = service.RequestTypeStream
	}
	setOpsEndpointContext(c, endpoint, int16(requestType))
	if auditText != "" {
		auditBody, _ := json.Marshal(map[string]any{"messages": []map[string]any{{"role": "user", "content": auditText}}})
		if decision := h.checkSecurityAudit(c, reqLog, key, subject, service.ContentModerationProtocolOpenAIChat, model, auditBody); decision != nil && !decision.AllowNextStage {
			h.openAISecurityAuditError(c, decision)
			return
		}
	}
	subscription, _ := middleware2.GetSubscriptionFromContext(c)
	if err := h.billingCacheService.CheckBillingEligibility(c.Request.Context(), key.User, key, key.Group, subscription, service.QuotaPlatform(c.Request.Context(), key)); err != nil {
		h.deepSeekToolBillingError(c, err)
		return
	}
	reservedCtx, reservation, err := h.billingCacheService.ReserveChatAuxiliary(c.Request.Context(), key.User, key, key.Group, subscription, inputBody.ReserveCost)
	if err != nil {
		h.deepSeekToolBillingError(c, err)
		return
	}
	defer reservation.HandlerDone()
	c.Request = c.Request.WithContext(reservedCtx)
	account, err := h.gatewayService.SelectChatAuxiliaryAccount(c.Request.Context(), id)
	if err != nil {
		h.errorResponse(c, 503, "api_error", "Shared auxiliary account is unavailable")
		return
	}
	keyRelease, err := h.concurrencyHelper.AcquireAPIKeySlot(c.Request.Context(), key.ID, key.ConcurrencyLimit)
	if err != nil {
		h.handleConcurrencyError(c, err, "API key", false)
		return
	}
	defer keyRelease()
	lease, err := h.concurrencyHelper.concurrencyService.AcquireChatAuxiliarySlots(c.Request.Context(), subject.UserID, subject.Concurrency, account.ID, account.Concurrency, kind, 0)
	if err != nil {
		if errors.Is(err, service.ErrChatAuxiliarySlotsFull) {
			h.errorResponse(c, 429, "rate_limit_error", "Auxiliary concurrency limit reached")
		} else {
			h.errorResponse(c, 503, "api_error", "Auxiliary concurrency service unavailable")
		}
		return
	}
	defer lease.Release()
	c.Request = c.Request.WithContext(lease.Context())
	if _, err := h.gatewayService.ValidateChatAuxiliaryScope(lease.Context(), key, kind); err != nil {
		h.errorResponse(c, 403, "permission_error", "Auxiliary permission changed")
		return
	}
	account, err = h.gatewayService.SelectChatAuxiliaryAccount(lease.Context(), id)
	if err != nil {
		h.errorResponse(c, 503, "api_error", "Shared auxiliary account is unavailable")
		return
	}
	setOpsSelectedAccount(c, account.ID, account.Platform, account.Name)
	var result *service.OpenAIForwardResult
	var forwardErr error
	var parsed *service.ChatOCRResult
	switch kind {
	case "ocr":
		result, parsed, forwardErr = h.gatewayService.ForwardChatOCR(lease.Context(), key, account)
	case "asr":
		var recognized []byte
		result, recognized, forwardErr = h.gatewayService.ForwardChatASR(lease.Context(), key, account, inputBody)
		if forwardErr == nil {
			var payload struct {
				Text string `json:"text"`
			}
			_ = json.Unmarshal(recognized, &payload)
			audit, _ := json.Marshal(map[string]any{"messages": []map[string]any{{"role": "user", "content": payload.Text}}})
			if decision := h.checkSecurityAudit(c, reqLog, key, subject, service.ContentModerationProtocolOpenAIChat, model, audit); decision != nil && !decision.AllowNextStage {
				h.openAISecurityAuditError(c, decision)
			} else {
				c.Data(http.StatusOK, "application/json", recognized)
			}
		}
	case "tts":
		streamStarted = true
		result, forwardErr = h.gatewayService.ForwardChatTTS(lease.Context(), c, key, account, inputBody)
	default:
		result, forwardErr = h.gatewayService.ForwardChatAuxiliaryJSON(lease.Context(), c, key, account, kind, body)
	}
	if result != nil {
		input := &service.OpenAIRecordUsageInput{Result: result, APIKey: key, User: key.User, Account: account, Subscription: subscription, InboundEndpoint: "/internal/chat/auxiliary/" + kind, UpstreamEndpoint: endpoint, UserAgent: c.GetHeader("User-Agent"), IPAddress: ip.GetClientIP(c), SessionID: service.ExtractClientSessionID(c), RequestPayloadHash: service.HashUsageRequestPayload(body), APIKeyService: h.apiKeyService, QuotaPlatform: service.QuotaPlatform(c.Request.Context(), key), PricingAt: started}
		if kind == "ocr" {
			settlement, commitErr := h.gatewayService.CommitChatOCR(lease.Context(), input, parsed)
			if commitErr != nil {
				reqLog.Error("chat_auxiliary.persist_ocr_failed", zap.Error(commitErr))
				h.errorResponse(c, 502, "upstream_error", "OCR result could not be persisted")
				return
			}
			h.submitMandatoryUsageRecordTask(c.Request.Context(), func(ctx context.Context) {
				if err := settlement.Apply(ctx, h.apiKeyService); err != nil {
					reqLog.Error("chat_auxiliary.settle_ocr_failed", zap.Error(err))
				}
			})
			c.JSON(http.StatusOK, gin.H{"status": "settling"})
			return
		}
		h.submitMandatoryUsageRecordTask(c.Request.Context(), func(ctx context.Context) {
			if err := h.gatewayService.RecordUsage(ctx, input); err != nil {
				reqLog.Error("chat_auxiliary.record_usage_failed", zap.Error(err))
			}
		})
	}
	if forwardErr != nil && !c.Writer.Written() && c.Request.Context().Err() == nil {
		h.errorResponse(c, 502, "upstream_error", "Auxiliary request failed")
	}
}
