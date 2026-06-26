package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Images handles OpenAI Images API requests.
// POST /v1/images/generations
// POST /v1/images/edits
func (h *OpenAIGatewayHandler) Images(c *gin.Context) {
	streamStarted := false
	defer h.recoverResponsesPanic(c, &streamStarted)

	requestStart := time.Now()

	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}

	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusInternalServerError, "api_error", "User context not found")
		return
	}
	reqLog := requestLogger(
		c,
		"handler.openai_gateway.images",
		zap.Int64("user_id", subject.UserID),
		zap.Int64("api_key_id", apiKey.ID),
		zap.Any("group_id", apiKey.GroupID),
	)
	if !h.ensureResponsesDependencies(c, reqLog) {
		return
	}

	body, err := pkghttputil.ReadRequestBodyWithPrealloc(c.Request)
	if err != nil {
		if maxErr, ok := extractMaxBytesError(err); ok {
			h.errorResponse(c, http.StatusRequestEntityTooLarge, "invalid_request_error", buildBodyTooLargeMessage(maxErr.Limit))
			return
		}
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return
	}
	if len(body) == 0 {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Request body is empty")
		return
	}

	if isMultipartImagesContentType(c.GetHeader("Content-Type")) {
		setOpsRequestContext(c, "", false)
	} else {
		setOpsRequestContext(c, "", false)
	}

	parsed, err := h.gatewayService.ParseOpenAIImagesRequest(c, body)
	if err != nil {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}
	requestModel := parsed.Model

	reqLog = reqLog.With(
		zap.String("model", requestModel),
		zap.Bool("stream", parsed.Stream),
		zap.Bool("multipart", parsed.Multipart),
		zap.String("capability", string(parsed.RequiredCapability)),
	)

	if !service.GroupAllowsImageGeneration(apiKey.Group) {
		h.errorResponse(c, http.StatusForbidden, "permission_error", service.ImageGenerationPermissionMessage())
		return
	}
	if decision := h.checkContentModeration(c, reqLog, apiKey, subject, service.ContentModerationProtocolOpenAIImages, requestModel, parsed.ModerationBody()); decision != nil && decision.Blocked {
		h.errorResponse(c, contentModerationStatus(decision), contentModerationErrorCode(decision), decision.Message)
		return
	}
	imageReleaseFunc, acquired := h.acquireImageGenerationSlot(c, streamStarted)
	if !acquired {
		return
	}
	if imageReleaseFunc != nil {
		defer imageReleaseFunc()
	}

	if parsed.Multipart {
		setOpsRequestContext(c, requestModel, parsed.Stream)
	} else {
		setOpsRequestContext(c, requestModel, parsed.Stream)
	}
	setOpsEndpointContext(c, "", int16(service.RequestTypeFromLegacy(parsed.Stream, false)))

	channelMapping, _ := h.gatewayService.ResolveChannelMappingAndRestrict(c.Request.Context(), apiKey.GroupID, requestModel)

	if h.errorPassthroughService != nil {
		service.BindErrorPassthroughService(c, h.errorPassthroughService)
	}

	subscription, _ := middleware2.GetSubscriptionFromContext(c)

	service.SetOpsLatencyMs(c, service.OpsAuthLatencyMsKey, time.Since(requestStart).Milliseconds())
	routingStart := time.Now()

	userReleaseFunc, acquired := h.acquireResponsesUserSlot(c, subject.UserID, subject.Concurrency, parsed.Stream, &streamStarted, reqLog)
	if !acquired {
		return
	}
	if userReleaseFunc != nil {
		defer userReleaseFunc()
	}

	if err := h.billingCacheService.CheckBillingEligibility(c.Request.Context(), apiKey.User, apiKey, apiKey.Group, subscription, service.QuotaPlatform(c.Request.Context(), apiKey)); err != nil {
		reqLog.Info("openai.images.billing_eligibility_check_failed", zap.Error(err))
		status, code, message, retryAfter := billingErrorDetails(err)
		if retryAfter > 0 {
			c.Header("Retry-After", strconv.Itoa(retryAfter))
		}
		h.handleStreamingAwareError(c, status, code, message, streamStarted)
		return
	}

	sessionHash := h.gatewayService.GenerateExplicitSessionHash(c, body)
	requestCtx := service.WithOpenAIImageGenerationIntent(c.Request.Context())

	if h.tryForwardImagesFanout(
		c,
		reqLog,
		apiKey,
		subscription,
		parsed,
		body,
		sessionHash,
		channelMapping,
		routingStart,
	) {
		return
	}

	maxAccountSwitches := h.maxAccountSwitches
	switchCount := 0
	failedAccountIDs := make(map[int64]struct{})
	sameAccountRetryCount := make(map[int64]int)
	var lastFailoverErr *service.UpstreamFailoverError

	for {
		reqLog.Debug("openai.images.account_selecting", zap.Int("excluded_account_count", len(failedAccountIDs)))
		selection, scheduleDecision, err := h.gatewayService.SelectAccountWithSchedulerForImages(
			requestCtx,
			apiKey.GroupID,
			sessionHash,
			requestModel,
			failedAccountIDs,
			parsed.RequiredCapability,
		)
		if err != nil {
			reqLog.Warn("openai.images.account_select_failed",
				zap.Error(err),
				zap.Int("excluded_account_count", len(failedAccountIDs)),
			)
			if len(failedAccountIDs) == 0 {
				cls := classifyNoAccountErrorFromGin(c, h.gatewayService, apiKey, requestModel, requestModel, service.PlatformOpenAI)
				if !cls.ModelNotFound {
					markOpsRoutingCapacityLimitedIfNoAvailable(c, err)
				}
				message := cls.Message
				if !cls.ModelNotFound {
					message = "No available compatible accounts"
				}
				h.handleStreamingAwareError(c, cls.Status, cls.ErrType, message, streamStarted)
				return
			}
			if lastFailoverErr != nil {
				h.handleFailoverExhausted(c, lastFailoverErr, streamStarted)
			} else {
				h.handleFailoverExhaustedSimple(c, 502, streamStarted)
			}
			return
		}
		if selection == nil || selection.Account == nil {
			cls := classifyNoAccountErrorFromGin(c, h.gatewayService, apiKey, requestModel, requestModel, service.PlatformOpenAI)
			if !cls.ModelNotFound {
				markOpsRoutingCapacityLimited(c)
			}
			message := cls.Message
			if !cls.ModelNotFound {
				message = "No available compatible accounts"
			}
			h.handleStreamingAwareError(c, cls.Status, cls.ErrType, message, streamStarted)
			return
		}

		reqLog.Debug("openai.images.account_schedule_decision",
			zap.String("layer", scheduleDecision.Layer),
			zap.Bool("sticky_session_hit", scheduleDecision.StickySessionHit),
			zap.Int("candidate_count", scheduleDecision.CandidateCount),
			zap.Int("top_k", scheduleDecision.TopK),
			zap.Int64("latency_ms", scheduleDecision.LatencyMs),
			zap.Float64("load_skew", scheduleDecision.LoadSkew),
		)

		account := selection.Account
		sessionHash = ensureOpenAIPoolModeSessionHash(sessionHash, account)
		reqLog.Debug("openai.images.account_selected", zap.Int64("account_id", account.ID), zap.String("account_name", account.Name))
		setOpsSelectedAccount(c, account.ID, account.Platform)

		accountReleaseFunc, acquired := h.acquireResponsesAccountSlot(c, apiKey.GroupID, sessionHash, selection, parsed.Stream, &streamStarted, reqLog)
		if !acquired {
			return
		}

		service.SetOpsLatencyMs(c, service.OpsRoutingLatencyMsKey, time.Since(routingStart).Milliseconds())
		forwardStart := time.Now()
		writerSizeBeforeForward := c.Writer.Size()
		result, err := func() (*service.OpenAIForwardResult, error) {
			defer func() {
				if accountReleaseFunc != nil {
					accountReleaseFunc()
				}
			}()
			return h.gatewayService.ForwardImages(requestCtx, c, account, body, parsed, channelMapping.MappedModel)
		}()
		forwardDurationMs := time.Since(forwardStart).Milliseconds()
		upstreamLatencyMs, _ := getContextInt64(c, service.OpsUpstreamLatencyMsKey)
		responseLatencyMs := forwardDurationMs
		if upstreamLatencyMs > 0 && forwardDurationMs > upstreamLatencyMs {
			responseLatencyMs = forwardDurationMs - upstreamLatencyMs
		}
		service.SetOpsLatencyMs(c, service.OpsResponseLatencyMsKey, responseLatencyMs)
		if result != nil && result.FirstTokenMs != nil {
			service.SetOpsLatencyMs(c, service.OpsTimeToFirstTokenMsKey, int64(*result.FirstTokenMs))
		}
		if err != nil {
			if result != nil && result.ImageCount > 0 {
				reqLog.Warn("openai.images.forward_partial_error_with_image_result",
					zap.Int64("account_id", account.ID),
					zap.Int("image_count", result.ImageCount),
					zap.Error(err),
				)
			} else {
				var imageUpstreamErr *service.OpenAIImagesUpstreamError
				if errors.As(err, &imageUpstreamErr) {
					retryableServerError := service.IsOpenAIImagesRetryableUpstreamError(imageUpstreamErr)
					h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, !retryableServerError, nil)
					logEvent := "openai.images.upstream_user_error"
					if retryableServerError {
						logEvent = "openai.images.upstream_server_error_after_flush"
					}
					reqLog.Warn(logEvent,
						zap.Int64("account_id", account.ID),
						zap.Int("status_code", imageUpstreamErr.StatusCode),
						zap.String("error_type", imageUpstreamErr.ErrorType),
						zap.String("error_code", imageUpstreamErr.Code),
						zap.Error(err),
					)
					return
				}
				var failoverErr *service.UpstreamFailoverError
				if errors.As(err, &failoverErr) {
					h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
					if c.Writer.Size() != writerSizeBeforeForward {
						reqLog.Warn("openai.images.upstream_failover_skipped_after_flush",
							zap.Int64("account_id", account.ID),
							zap.Int("upstream_status", failoverErr.StatusCode),
						)
						h.handleFailoverExhausted(c, failoverErr, true)
						return
					}
					if failoverErr.RetryableOnSameAccount {
						retryLimit := account.GetPoolModeRetryCount()
						if sameAccountRetryCount[account.ID] < retryLimit {
							sameAccountRetryCount[account.ID]++
							reqLog.Warn("openai.images.pool_mode_same_account_retry",
								zap.Int64("account_id", account.ID),
								zap.Int("upstream_status", failoverErr.StatusCode),
								zap.Int("retry_limit", retryLimit),
								zap.Int("retry_count", sameAccountRetryCount[account.ID]),
							)
							select {
							case <-requestCtx.Done():
								return
							case <-time.After(sameAccountRetryDelay):
							}
							continue
						}
					}
					h.gatewayService.RecordOpenAIAccountSwitch()
					failedAccountIDs[account.ID] = struct{}{}
					lastFailoverErr = failoverErr
					if switchCount >= maxAccountSwitches {
						h.handleFailoverExhausted(c, failoverErr, streamStarted)
						return
					}
					switchCount++
					if h.gatewayService.ShouldStopOpenAIOAuth429Failover(account, failoverErr.StatusCode, switchCount) {
						h.handleFailoverExhausted(c, failoverErr, streamStarted)
						return
					}
					reqLog.Warn("openai.images.upstream_failover_switching",
						zap.Int64("account_id", account.ID),
						zap.Int("upstream_status", failoverErr.StatusCode),
						zap.Int("switch_count", switchCount),
						zap.Int("max_switches", maxAccountSwitches),
					)
					continue
				}
				h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
				upstreamErrorAlreadyCommunicated := openAIForwardErrorAlreadyCommunicated(c, writerSizeBeforeForward, err)
				wroteFallback := false
				if !upstreamErrorAlreadyCommunicated {
					wroteFallback = h.ensureForwardErrorResponse(c, streamStarted)
				}
				fields := []zap.Field{
					zap.Int64("account_id", account.ID),
					zap.Bool("fallback_error_response_written", wroteFallback),
					zap.Bool("upstream_error_response_already_written", upstreamErrorAlreadyCommunicated),
					zap.Error(err),
				}
				if shouldLogOpenAIForwardFailureAsWarn(c, wroteFallback) {
					reqLog.Warn("openai.images.forward_failed", fields...)
					return
				}
				reqLog.Error("openai.images.forward_failed", fields...)
				return
			}
		}
		if result != nil {
			if account.Type == service.AccountTypeOAuth {
				h.gatewayService.UpdateCodexUsageSnapshotFromHeaders(c.Request.Context(), account.ID, result.ResponseHeaders)
			}
			h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, true, result.FirstTokenMs)
		} else {
			h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, true, nil)
		}

		userAgent := c.GetHeader("User-Agent")
		clientIP := ip.GetClientIP(c)
		requestPayloadHash := service.HashUsageRequestPayload(body)
		if parsed.Multipart {
			requestPayloadHash = service.HashUsageRequestPayload([]byte(parsed.StickySessionSeed()))
		}
		inboundEndpoint := GetInboundEndpoint(c)
		upstreamEndpoint := GetUpstreamEndpoint(c, account.Platform)

		upstreamModel := ""
		if result != nil {
			upstreamModel = result.UpstreamModel
		}
		h.submitMandatoryUsageRecordTask(c.Request.Context(), func(ctx context.Context) {
			if err := h.gatewayService.RecordUsage(ctx, &service.OpenAIRecordUsageInput{
				Result:             result,
				APIKey:             apiKey,
				User:               apiKey.User,
				Account:            account,
				Subscription:       subscription,
				InboundEndpoint:    inboundEndpoint,
				UpstreamEndpoint:   upstreamEndpoint,
				UserAgent:          userAgent,
				IPAddress:          clientIP,
				RequestPayloadHash: requestPayloadHash,
				APIKeyService:      h.apiKeyService,
				ChannelUsageFields: channelMapping.ToUsageFields(requestModel, upstreamModel),
			}); err != nil {
				logger.L().With(
					zap.String("component", "handler.openai_gateway.images"),
					zap.Int64("user_id", subject.UserID),
					zap.Int64("api_key_id", apiKey.ID),
					zap.Any("group_id", apiKey.GroupID),
					zap.String("model", requestModel),
					zap.Int64("account_id", account.ID),
				).Error("openai.images.record_usage_failed", zap.Error(err))
			}
		})

		reqLog.Debug("openai.images.request_completed",
			zap.Int64("account_id", account.ID),
			zap.Int("switch_count", switchCount),
		)
		return
	}
}

func isMultipartImagesContentType(contentType string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(contentType)), "multipart/form-data")
}

type openAIImagesFanoutShard struct {
	account *service.Account
	n       int
	release func()
}

type openAIImagesFanoutPart struct {
	account *service.Account
	result  *service.OpenAIForwardResult
	body    []byte
	imageN  int
	err     error
}

func (h *OpenAIGatewayHandler) tryForwardImagesFanout(
	c *gin.Context,
	reqLog *zap.Logger,
	apiKey *service.APIKey,
	subscription *service.UserSubscription,
	parsed *service.OpenAIImagesRequest,
	body []byte,
	sessionHash string,
	channelMapping service.ChannelMappingResult,
	routingStart time.Time,
) bool {
	if parsed == nil || parsed.Stream || parsed.N <= 1 {
		return false
	}

	shards := h.selectOpenAIImagesFanoutShards(c, reqLog, apiKey.GroupID, sessionHash, parsed)
	if len(shards) < 2 {
		for _, shard := range shards {
			if shard.release != nil {
				shard.release()
			}
		}
		return false
	}
	for i := 0; i < parsed.N; i++ {
		shards[i%len(shards)].n++
	}

	service.SetOpsLatencyMs(c, service.OpsRoutingLatencyMsKey, time.Since(routingStart).Milliseconds())
	results := make([]openAIImagesFanoutPart, len(shards))
	var wg sync.WaitGroup
	for i, shard := range shards {
		partCtx := c.Copy()
		wg.Add(1)
		go func(index int, shard openAIImagesFanoutShard, partCtx *gin.Context) {
			defer wg.Done()
			if shard.release != nil {
				defer shard.release()
			}
			part := openAIImagesFanoutPart{
				account: shard.account,
				imageN:  shard.n,
			}
			partParsed := service.CloneOpenAIImagesRequestWithN(parsed, shard.n)
			collected, err := h.gatewayService.ForwardImagesOAuthNonStreamingCollect(
				c.Request.Context(),
				partCtx,
				shard.account,
				partParsed,
				channelMapping.MappedModel,
			)
			if err != nil {
				part.err = err
				results[index] = part
				return
			}
			if collected == nil || collected.Result == nil || len(collected.Body) == 0 {
				part.err = errors.New("empty fanout image response")
				results[index] = part
				return
			}
			part.result = collected.Result
			part.body = collected.Body
			results[index] = part
		}(i, shard, partCtx)
	}
	wg.Wait()

	var (
		bodies     [][]byte
		usages     []service.OpenAIUsage
		firstErr   error
		successes  int
		firstToken *int
	)
	for _, part := range results {
		if part.account == nil {
			continue
		}
		if part.err != nil {
			h.gatewayService.ReportOpenAIAccountScheduleResult(part.account.ID, false, nil)
			if firstErr == nil {
				firstErr = part.err
			}
			reqLog.Warn("openai.images.fanout_part_failed",
				zap.Int64("account_id", part.account.ID),
				zap.Int("n", part.imageN),
				zap.Error(part.err),
			)
			continue
		}
		successes++
		bodies = append(bodies, part.body)
		usages = append(usages, part.result.Usage)
		if firstToken == nil {
			firstToken = part.result.FirstTokenMs
		}
		if part.account.Type == service.AccountTypeOAuth {
			h.gatewayService.UpdateCodexUsageSnapshotFromHeaders(c.Request.Context(), part.account.ID, part.result.ResponseHeaders)
		}
		h.gatewayService.ReportOpenAIAccountScheduleResult(part.account.ID, true, part.result.FirstTokenMs)
	}

	if successes == 0 {
		reqLog.Warn("openai.images.fanout_all_failed", zap.Error(firstErr))
		return false
	}

	totalUsage := service.SumOpenAIUsage(usages...)
	responseBody, imageCount, err := service.BuildOpenAIImagesFanoutAPIResponse(bodies, totalUsage)
	if err != nil {
		reqLog.Warn("openai.images.fanout_response_build_failed", zap.Error(err))
		return false
	}
	if firstToken != nil {
		service.SetOpsLatencyMs(c, service.OpsTimeToFirstTokenMsKey, int64(*firstToken))
	}
	service.SetOpsLatencyMs(c, service.OpsResponseLatencyMsKey, 0)
	c.Header("X-Sub2API-Image-Fanout", strconv.Itoa(successes))
	c.Data(http.StatusOK, "application/json; charset=utf-8", responseBody)

	h.recordOpenAIImagesFanoutUsage(c, reqLog, apiKey, subscription, parsed, body, channelMapping, results)
	reqLog.Debug("openai.images.fanout_completed",
		zap.Int("requested_n", parsed.N),
		zap.Int("account_count", successes),
		zap.Int("image_count", imageCount),
	)
	return true
}

func (h *OpenAIGatewayHandler) selectOpenAIImagesFanoutShards(
	c *gin.Context,
	reqLog *zap.Logger,
	groupID *int64,
	sessionHash string,
	parsed *service.OpenAIImagesRequest,
) []openAIImagesFanoutShard {
	excluded := make(map[int64]struct{})
	shards := make([]openAIImagesFanoutShard, 0, parsed.N)
	maxAttempts := parsed.N*3 + 8
	for attempts := 0; attempts < maxAttempts && len(shards) < parsed.N; attempts++ {
		selection, _, err := h.gatewayService.SelectAccountWithSchedulerForImages(
			c.Request.Context(),
			groupID,
			fmt.Sprintf("%s|image-fanout-%d", sessionHash, len(shards)),
			parsed.Model,
			excluded,
			parsed.RequiredCapability,
		)
		if err != nil || selection == nil || selection.Account == nil {
			if err != nil {
				reqLog.Debug("openai.images.fanout_select_stopped", zap.Error(err))
			}
			break
		}
		account := selection.Account
		excluded[account.ID] = struct{}{}
		if account.Type != service.AccountTypeOAuth {
			if selection.ReleaseFunc != nil {
				selection.ReleaseFunc()
			}
			continue
		}
		if !selection.Acquired {
			if selection.ReleaseFunc != nil {
				selection.ReleaseFunc()
			}
			continue
		}
		shards = append(shards, openAIImagesFanoutShard{
			account: account,
			release: selection.ReleaseFunc,
		})
	}
	return shards
}

func (h *OpenAIGatewayHandler) recordOpenAIImagesFanoutUsage(
	c *gin.Context,
	reqLog *zap.Logger,
	apiKey *service.APIKey,
	subscription *service.UserSubscription,
	parsed *service.OpenAIImagesRequest,
	body []byte,
	channelMapping service.ChannelMappingResult,
	parts []openAIImagesFanoutPart,
) {
	userAgent := c.GetHeader("User-Agent")
	clientIP := ip.GetClientIP(c)
	basePayloadHash := service.HashUsageRequestPayload(body)
	if parsed.Multipart {
		basePayloadHash = service.HashUsageRequestPayload([]byte(parsed.StickySessionSeed()))
	}
	inboundEndpoint := GetInboundEndpoint(c)

	for index, part := range parts {
		if part.account == nil || part.result == nil || part.err != nil {
			continue
		}
		account := part.account
		result := part.result
		requestPayloadHash := service.HashUsageRequestPayload([]byte(fmt.Sprintf("%s|image-fanout|%d|%d", basePayloadHash, index, part.imageN)))
		upstreamEndpoint := GetUpstreamEndpoint(c, account.Platform)
		upstreamModel := result.UpstreamModel
		h.submitMandatoryUsageRecordTask(c.Request.Context(), func(ctx context.Context) {
			if err := h.gatewayService.RecordUsage(ctx, &service.OpenAIRecordUsageInput{
				Result:             result,
				APIKey:             apiKey,
				User:               apiKey.User,
				Account:            account,
				Subscription:       subscription,
				InboundEndpoint:    inboundEndpoint,
				UpstreamEndpoint:   upstreamEndpoint,
				UserAgent:          userAgent,
				IPAddress:          clientIP,
				RequestPayloadHash: requestPayloadHash,
				APIKeyService:      h.apiKeyService,
				ChannelUsageFields: channelMapping.ToUsageFields(parsed.Model, upstreamModel),
			}); err != nil {
				reqLog.Error("openai.images.fanout_record_usage_failed",
					zap.Int64("account_id", account.ID),
					zap.Int("part_index", index),
					zap.Error(err),
				)
			}
		})
	}
}
