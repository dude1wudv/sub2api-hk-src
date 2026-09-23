package handler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

func (h *OpenAIGatewayHandler) StepFunAudio(c *gin.Context) {
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok || apiKey.Group == nil || apiKey.Group.Platform != service.PlatformStepFun {
		h.errorResponse(c, http.StatusNotFound, "not_found_error", "Audio API is not supported for this platform")
		return
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil || len(body) == 0 {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Request body is required")
		return
	}
	req, err := parseStepFunAudioRequest(c, body)
	if err != nil {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return
	}
	subscription, _ := middleware2.GetSubscriptionFromContext(c)
	if err := h.billingCacheService.CheckBillingEligibility(c.Request.Context(), apiKey.User, apiKey, apiKey.Group, subscription, service.QuotaPlatform(c.Request.Context(), apiKey)); err != nil {
		status, code, message, retryAfter := billingErrorDetails(err)
		if retryAfter > 0 {
			c.Header("Retry-After", strconv.Itoa(retryAfter))
		}
		h.errorResponse(c, status, code, message)
		return
	}
	selection, _, err := h.gatewayService.SelectAccountWithSchedulerForCapability(
		c.Request.Context(), apiKey.GroupID, "", req.BodyHash, req.Model, nil,
		service.OpenAIUpstreamTransportHTTPSSE, service.OpenAIEndpointCapabilityChatCompletions,
		false, false, false, service.PlatformStepFun,
	)
	if err != nil || selection == nil || selection.Account == nil {
		h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "No available StepFun accounts")
		return
	}
	var streamStarted bool
	release, acquired := h.acquireResponsesAccountSlot(c, apiKey.GroupID, req.BodyHash, selection, false, &streamStarted, nil)
	if acquired != openAISlotAcquireOK {
		return
	}
	if release != nil {
		defer release()
	}
	result, err := h.gatewayService.ForwardStepFunAudio(c.Request.Context(), c, selection.Account, req)
	if err != nil {
		var upstream *service.UpstreamFailoverError
		if errors.As(err, &upstream) {
			h.handleFailoverExhausted(c, upstream, false)
			return
		}
		h.errorResponse(c, http.StatusBadGateway, "upstream_error", "StepFun audio request failed")
		return
	}
	if result.AudioUsage == nil || result.AudioUsage.DurationOrUnits <= 0 {
		return
	}
	h.submitMandatoryUsageRecordTask(c.Request.Context(), func(ctx context.Context) {
		_ = h.gatewayService.RecordUsage(ctx, &service.OpenAIRecordUsageInput{
			Result: result, APIKey: apiKey, User: apiKey.User, Account: selection.Account, Subscription: subscription,
			InboundEndpoint: req.Endpoint, UpstreamEndpoint: req.Endpoint, RequestPayloadHash: req.BodyHash,
			APIKeyService: h.apiKeyService, QuotaPlatform: service.QuotaPlatform(c.Request.Context(), apiKey),
		})
	})
}

func parseStepFunAudioRequest(c *gin.Context, body []byte) (service.StepFunAudioRequest, error) {
	endpoint := "/v1/audio/transcriptions"
	if strings.HasSuffix(c.Request.URL.Path, "/speech") {
		endpoint = "/v1/audio/speech"
	}
	contentType := c.GetHeader("Content-Type")
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return service.StepFunAudioRequest{}, errors.New("invalid Content-Type")
	}
	model, input := "", ""
	if endpoint == "/v1/audio/speech" {
		if mediaType != "application/json" || !gjson.ValidBytes(body) {
			return service.StepFunAudioRequest{}, errors.New("speech requires a JSON request")
		}
		model = strings.TrimSpace(gjson.GetBytes(body, "model").String())
		input = gjson.GetBytes(body, "input").String()
		if !utf8.ValidString(input) || strings.TrimSpace(input) == "" {
			return service.StepFunAudioRequest{}, errors.New("input is required")
		}
	} else {
		if mediaType != "multipart/form-data" {
			return service.StepFunAudioRequest{}, errors.New("transcriptions requires multipart/form-data")
		}
		reader := multipart.NewReader(bytes.NewReader(body), params["boundary"])
		form, formErr := reader.ReadForm(32 << 20)
		if formErr != nil {
			return service.StepFunAudioRequest{}, errors.New("invalid multipart request")
		}
		defer func() { _ = form.RemoveAll() }()
		if values := form.Value["model"]; len(values) > 0 {
			model = strings.TrimSpace(values[0])
		}
		if len(form.File["file"]) == 0 {
			return service.StepFunAudioRequest{}, errors.New("file is required")
		}
	}
	want := "stepaudio-2.5-asr"
	if endpoint == "/v1/audio/speech" {
		want = "stepaudio-2.5-tts"
	}
	if model != want {
		return service.StepFunAudioRequest{}, errors.New("model must be " + want)
	}
	sum := sha256.Sum256(body)
	req := service.StepFunAudioRequest{Model: model, Endpoint: endpoint, ContentType: contentType, Body: body, BodyHash: hex.EncodeToString(sum[:8])}
	if endpoint == "/v1/audio/speech" {
		req.AudioUsage = &service.AudioUsage{Mode: "tts", DurationOrUnits: float64(utf8.RuneCountInString(input)) / 1_000_000}
	}
	return req, nil
}
