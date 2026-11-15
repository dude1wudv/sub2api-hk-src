package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

func deepSeekSuiteKey(c *gin.Context) (*service.APIKey, bool) {
	key, ok := middleware2.GetAPIKeyFromContext(c)
	return key, ok && key != nil && key.User != nil && key.GroupID != nil && key.Group != nil &&
		len(key.RoutingGroupIDs) == 0 && (key.Group.Platform == service.PlatformOpenAI || key.Group.Platform == service.PlatformDeepseek)
}

// TryDeepSeekSuiteConversation runs after the normal gateway group/model
// middleware, and restores the body for ordinary requests.
func (h *OpenAIGatewayHandler) TryDeepSeekSuiteConversation(c *gin.Context, kind string) bool {
	if _, ok := deepSeekSuiteKey(c); !ok {
		return false
	}
	body, err := pkghttputil.ReadRequestBodyWithPrealloc(c.Request)
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	if err != nil {
		if !middleware2.RejectRequestBodyReadError(c, "handler.deepseek_suite", err) {
			h.errorResponse(c, 400, "invalid_request_error", "Unable to read request body")
		}
		return true
	}
	if !strings.HasPrefix(strings.ToLower(gjson.GetBytes(body, "model").String()), "deepseek") ||
		!service.DeepSeekSuiteConversation(body) {
		return false
	}
	h.handleDeepSeekSuite(c, kind, false)
	return true
}

func (h *OpenAIGatewayHandler) DeepSeekSuiteTool(c *gin.Context) {
	h.handleDeepSeekSuite(c, strings.TrimPrefix(c.FullPath(), "/v1/tools/"), false)
}

func (h *OpenAIGatewayHandler) TryDeepSeekAudio(c *gin.Context, kind string) bool {
	if _, ok := deepSeekSuiteKey(c); !ok {
		return false
	}
	raw, err := pkghttputil.ReadRequestBodyWithPrealloc(c.Request)
	c.Request.Body = io.NopCloser(bytes.NewReader(raw))
	if err != nil {
		if !middleware2.RejectRequestBodyReadError(c, "handler.deepseek_audio", err) {
			h.errorResponse(c, 400, "invalid_request_error", "Unable to read audio request")
		}
		return true
	}
	model := ""
	if kind == "tts" {
		model = gjson.GetBytes(raw, "model").String()
	} else if _, params, parseErr := mime.ParseMediaType(c.GetHeader("Content-Type")); parseErr == nil {
		reader := multipart.NewReader(bytes.NewReader(raw), params["boundary"])
		for {
			part, nextErr := reader.NextPart()
			if nextErr != nil {
				break
			}
			if part.FormName() == "model" {
				v, _ := io.ReadAll(io.LimitReader(part, 1024))
				model = string(v)
				break
			}
		}
	}
	if model != "deepseek-"+kind {
		return false
	}
	h.handleDeepSeekSuite(c, kind, true)
	return true
}

func (h *OpenAIGatewayHandler) handleDeepSeekSuite(c *gin.Context, kind string, audioCompat bool) {
	streamStarted := false
	defer h.recoverResponsesPanic(c, &streamStarted)
	started := time.Now()
	key, ok := deepSeekSuiteKey(c)
	if !ok {
		h.errorResponse(c, 404, "not_found_error", "A supported fixed-group API key is required")
		return
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		h.errorResponse(c, 401, "authentication_error", "User context is unavailable")
		return
	}
	reqLog := requestLogger(c, "handler.deepseek_suite", zap.Int64("api_key_id", key.ID), zap.String("kind", kind))
	if !h.ensureResponsesDependencies(c, reqLog) {
		return
	}
	var input *service.DeepSeekSuiteInput
	var err error
	if kind == "tts_ws" {
		input = &service.DeepSeekSuiteInput{Kind: kind, ReserveCost: 3, Stream: true}
	} else {
		input, err = service.ParseDeepSeekSuiteInput(c.Request, kind, audioCompat)
	}
	if c.Request.MultipartForm != nil {
		form := c.Request.MultipartForm
		defer func() { _ = form.RemoveAll() }()
	}
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, service.ErrChatTooLarge) {
			status = http.StatusRequestEntityTooLarge
		}
		writeDeepSeekToolValidationError(c, status, err.Error())
		return
	}
	var boundAccount int64
	if input.Conversation {
		input.Body, boundAccount, err = h.gatewayService.ResolveDeepSeekFiles(c.Request.Context(), key, input.Body)
		if err != nil {
			h.errorResponse(c, 400, "invalid_request_error", "File references are unavailable in this user/group or span multiple accounts")
			return
		}
	}
	model := "deepseek-" + kind
	if input.Conversation {
		model = input.Model
	}
	setOpsRequestContext(c, model, input.Stream)
	setOpsEndpointContext(c, c.Request.URL.Path, int16(service.RequestTypeSync))
	audit := func(text string) bool {
		body, _ := json.Marshal(gin.H{"messages": []gin.H{{"role": "user", "content": text}}})
		protocol := service.ContentModerationProtocolOpenAIChat
		if input.Conversation {
			body = input.Body
			protocol = map[string]string{"chat": service.ContentModerationProtocolOpenAIChat, "responses": service.ContentModerationProtocolOpenAIResponses, "messages": service.ContentModerationProtocolAnthropicMessages}[kind]
		}
		decision := h.checkSecurityAudit(c, reqLog, key, subject, protocol, model, body)
		return decision == nil || decision.AllowNextStage
	}
	if input.AuditText != "" && !audit(input.AuditText) {
		h.errorResponse(c, 403, "permission_error", "Request rejected by content policy")
		return
	}
	subscription, _ := middleware2.GetSubscriptionFromContext(c)
	if err := h.billingCacheService.CheckBillingEligibility(c.Request.Context(), key.User, key, key.Group, subscription, service.QuotaPlatform(c.Request.Context(), key)); err != nil {
		h.deepSeekToolBillingError(c, err)
		return
	}
	ctx := c.Request.Context()
	if kind == "files" || kind == "ocr" {
		// Upload/parse can complete and charge after the caller disconnects.
		ctx = context.WithoutCancel(ctx)
	}
	timeout := 5 * time.Minute
	if kind == "tts" || kind == "tts_ws" {
		timeout = 30 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	amount := input.ReserveCost
	if input.Conversation {
		var priced bool
		maxTokens := int(gjson.GetBytes(input.Body, "max_output_tokens").Int())
		if maxTokens == 0 {
			maxTokens = int(gjson.GetBytes(input.Body, "max_tokens").Int())
		}
		amount, priced = h.gatewayService.EstimateInflightReservation(ctx, key, service.InflightEstimateRequest{
			Model: model, BodyBytes: len(input.Body), Kind: service.InflightEstimateToken, MaxTokens: maxTokens, ChatAuxiliaryCost: input.ReserveCost,
		})
		if !priced {
			h.errorResponse(c, 503, "api_error", "Model pricing is unavailable")
			return
		}
	}
	ctx, reservation, err := h.billingCacheService.ReserveChatAuxiliary(ctx, key.User, key, key.Group, subscription, amount)
	if err != nil {
		h.deepSeekToolBillingError(c, err)
		return
	}
	defer reservation.HandlerDone()
	ctx = service.WithDeepSeekSuiteScope(ctx, kind, *key.GroupID, boundAccount)
	if !input.Conversation {
		ctx = service.WithOpenAIProfitControlSuppressed(ctx)
	}
	c.Request = c.Request.WithContext(ctx)
	requestedModel := ""
	if input.Conversation {
		requestedModel = model
	}
	keyRelease, err := h.concurrencyHelper.AcquireAPIKeySlot(ctx, key.ID, key.ConcurrencyLimit)
	if err != nil {
		h.handleConcurrencyError(c, err, "API key", false)
		return
	}
	defer keyRelease()
	slotKind := kind
	if kind == "files" {
		slotKind = "ocr"
	} else if kind == "tts_ws" {
		slotKind = "tts"
	} else if input.Conversation {
		slotKind = "search"
	}
	// One explicit fallback only; every attempt keeps its own durable receipt.
	for attempt := 0; attempt < 2; attempt++ {
		selection, _, err := h.gatewayService.SelectAccountWithSchedulerForCapability(ctx, key.GroupID, "", "", requestedModel, nil,
			service.OpenAIUpstreamTransportHTTPSSE, service.OpenAIEndpointCapabilityDeepSeekTools, false, false, false, key.Group.Platform)
		if err != nil || selection == nil || selection.Account == nil {
			h.errorResponse(c, 503, "api_error", "No enabled DeepSeek account supports this operation and group")
			return
		}
		if selection.ReleaseFunc != nil {
			selection.ReleaseFunc()
		}
		account := selection.Account
		lease, err := h.concurrencyHelper.concurrencyService.AcquireChatAuxiliarySlots(ctx, subject.UserID, subject.Concurrency, account.ID, account.Concurrency, slotKind, 0)
		if err != nil {
			h.errorResponse(c, 429, "rate_limit_error", "DeepSeek concurrency limit reached")
			return
		}
		defer lease.Release()
		attemptCtx := lease.Context()
		c.Request = c.Request.WithContext(attemptCtx)
		if _, _, err := h.concurrencyHelper.concurrencyService.RevalidateAPIKeyQueueTurn(attemptCtx); err != nil {
			h.errorResponse(c, 403, "permission_error", "API key permissions changed")
			return
		}
		freshKey, valid := deepSeekSuiteKey(c)
		if !valid || freshKey.ID != key.ID || *freshKey.GroupID != *key.GroupID {
			h.errorResponse(c, 403, "permission_error", "API key group changed")
			return
		}
		key = freshKey
		if (input.Conversation || audioCompat) && key.Group.ModelAllowlistEnabled() && !key.Group.ModelAllowlist.Allows(model) {
			h.errorResponse(c, 404, "not_found_error", "Model is not allowed for this group")
			return
		}
		account, err = h.gatewayService.DeepSeekSuiteAccount(attemptCtx, key, account, kind)
		if err != nil {
			h.errorResponse(c, 503, "api_error", "DeepSeek capability or group permission changed")
			return
		}
		setOpsSelectedAccount(c, account.ID, account.Platform, account.Name)
		op, err := h.gatewayService.NewDeepSeekOperation(attemptCtx, key, account, kind)
		if err != nil {
			h.errorResponse(c, 503, "api_error", "Durable DeepSeek operation storage is unavailable")
			return
		}
		record := &service.OpenAIRecordUsageInput{APIKey: key, User: key.User, Account: account, Subscription: subscription,
			InboundEndpoint: c.Request.URL.Path, UserAgent: c.GetHeader("User-Agent"), IPAddress: ip.GetClientIP(c),
			SessionID: service.ExtractClientSessionID(c), RequestPayloadHash: service.HashUsageRequestPayload(input.Body),
			APIKeyService: h.apiKeyService, QuotaPlatform: service.QuotaPlatform(attemptCtx, key), PricingAt: started}
		if err := h.gatewayService.SeedDeepSeekSuiteBilling(attemptCtx, op, input, record); err != nil {
			h.errorResponse(c, 503, "api_error", "DeepSeek billing could not be prepared")
			return
		}
		c.Header("X-DeepSeek-Operation-ID", op.ID)
		var payload []byte
		if kind == "tts_ws" {
			streamStarted = true
			record.Result, err = h.gatewayService.ForwardDeepSeekTTSWebSocket(attemptCtx, c, key, account, op, audit)
		} else {
			attemptInput := *input
			attemptInput.Body = bytes.Clone(input.Body)
			record.Result, payload, err = h.gatewayService.ForwardDeepSeekSuite(attemptCtx, c, key, account, &attemptInput, op)
		}
		forwardErr := err
		persist, stopPersist := context.WithTimeout(context.WithoutCancel(attemptCtx), 30*time.Second)
		defer stopPersist()
		if record.Result == nil || op.Status == "failed" {
			if op.Status != "failed" {
				op.Status = "unknown"
			}
			repo, _ := h.gatewayService.DeepSeekSuiteRepository()
			err = repo.SaveDeepSeekOperation(persist, op)
		} else {
			record.UpstreamEndpoint = record.Result.UpstreamEndpoint
			err = h.gatewayService.CommitDeepSeekSuite(persist, op, record)
		}
		if err != nil {
			reqLog.Error("deepseek_suite.persistence_pending", zap.String("operation_id", op.ID), zap.Error(err))
		}
		var rejection *service.DeepSeekSuiteRejectedError
		if err == nil && errors.As(forwardErr, &rejection) && op.Status == "failed" && !c.Writer.Written() {
			if fallbackID := account.DeepSeekSuiteFallbackAccountID(boundAccount); attempt == 0 && fallbackID > 0 {
				lease.Release()
				stopPersist()
				reqLog.Info("deepseek_suite.failover", zap.Int64("account_id", account.ID), zap.Int64("fallback_account_id", fallbackID), zap.Int("status", rejection.StatusCode))
				ctx = service.WithDeepSeekSuiteScope(ctx, kind, *key.GroupID, fallbackID)
				c.Request = c.Request.WithContext(ctx)
				continue
			}
			if len(rejection.Body) > 0 {
				c.Data(rejection.StatusCode, "application/json", rejection.Body)
			} else {
				h.errorResponse(c, rejection.StatusCode, "upstream_error", "DeepSeek upstream rejected the operation")
			}
			return
		}
		if err != nil || forwardErr != nil {
			if !c.Writer.Written() {
				h.errorResponse(c, 502, "upstream_error", "DeepSeek operation did not complete; its durable record will be reconciled")
			}
			return
		}
		if !c.Writer.Written() && payload != nil {
			c.Header("X-DeepSeek-Metering-Status", op.Status)
			if audioCompat && input.ResponseFormat == "text" {
				c.Data(200, "text/plain; charset=utf-8", []byte(gjson.GetBytes(payload, "text").String()))
			} else {
				c.Data(200, "application/json", payload)
			}
		}
		return
	}
}

func (h *OpenAIGatewayHandler) DeepSeekTTSWebSocket(c *gin.Context) {
	h.handleDeepSeekSuite(c, "tts_ws", false)
}

func (h *OpenAIGatewayHandler) DeepSeekFiles(c *gin.Context) {
	if c.Request.Method == http.MethodPost {
		h.handleDeepSeekSuite(c, "files", false)
		return
	}
	key, ok := deepSeekSuiteKey(c)
	if !ok {
		h.errorResponse(c, 404, "not_found_error", "A supported fixed-group API key is required")
		return
	}
	repo, err := h.gatewayService.DeepSeekSuiteRepository()
	if err != nil {
		h.errorResponse(c, 503, "api_error", "File storage is unavailable")
		return
	}
	id := c.Param("file_id")
	anthropic := strings.HasPrefix(c.Request.URL.Path, "/anthropic/")
	if id != "" {
		op, err := repo.GetDeepSeekFile(c.Request.Context(), key.UserID, *key.GroupID, service.DeepSeekSuiteOperationID(id))
		if err != nil || !service.DeepSeekSuiteFileVisible(op) {
			h.errorResponse(c, 404, "not_found_error", "File not found")
			return
		}
		if _, err := h.gatewayService.DeepSeekFileAccount(c.Request.Context(), key, op); err != nil {
			h.errorResponse(c, 403, "permission_error", "File account is no longer available")
			return
		}
		if c.Request.Method == http.MethodDelete {
			if err := h.gatewayService.DeleteDeepSeekFile(c.Request.Context(), key, op); err != nil {
				h.errorResponse(c, 502, "upstream_error", "File deletion failed")
				return
			}
			if anthropic {
				c.JSON(200, gin.H{"id": id, "type": "file_deleted"})
			} else {
				c.JSON(200, gin.H{"id": id, "object": "file", "deleted": true})
			}
			return
		}
		c.Data(200, "application/json", service.DeepSeekSuiteFileObject(op, anthropic))
		return
	}
	limit := 100
	if value := c.Query("limit"); value != "" {
		n, parseErr := strconv.Atoi(value)
		if parseErr != nil || n < 1 || n > 10000 {
			h.errorResponse(c, 400, "invalid_request_error", "Invalid file limit")
			return
		}
		limit = n
	}
	query := service.DeepSeekFileListQuery{After: c.Query("after"), Purpose: c.Query("purpose"), Limit: limit}
	if anthropic {
		query.After, query.Before = c.Query("after_id"), c.Query("before_id")
	}
	if (query.After != "" && query.Before != "") || (c.Query("order") != "" && c.Query("order") != "asc" && c.Query("order") != "desc") {
		h.errorResponse(c, 400, "invalid_request_error", "Invalid file pagination")
		return
	}
	query.Ascending = c.Query("order") == "asc"
	for _, cursor := range []*string{&query.After, &query.Before} {
		if *cursor == "" {
			continue
		}
		*cursor = service.DeepSeekSuiteOperationID(*cursor)
		if _, err := repo.GetDeepSeekFile(c.Request.Context(), key.UserID, *key.GroupID, *cursor); err != nil {
			h.errorResponse(c, 400, "invalid_request_error", "File cursor not found")
			return
		}
	}
	files, err := repo.ListDeepSeekFiles(c.Request.Context(), key.UserID, *key.GroupID, query)
	if err != nil {
		h.errorResponse(c, 503, "api_error", "File listing is unavailable")
		return
	}
	hasMore := len(files) > limit
	if hasMore {
		files = files[:limit]
	}
	if query.Before != "" {
		slices.Reverse(files)
	}
	data := []json.RawMessage{}
	first, last := "", ""
	for _, op := range files {
		if !service.DeepSeekSuiteFileVisible(op) {
			continue
		}
		if first == "" {
			first = op.FileID()
		}
		last = op.FileID()
		data = append(data, service.DeepSeekSuiteFileObject(op, anthropic))
	}
	c.JSON(200, gin.H{"object": "list", "data": data, "has_more": hasMore, "first_id": first, "last_id": last})
}
