package handler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"mime"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	qwenAudioASRInboundEndpoint = "/v1/audio/transcriptions"
	qwenAudioTTSInboundEndpoint = "/v1/audio/speech"
	qwenAudioASRUpstreamPath    = "/api/v1/services/aigc/multimodal-generation/generation"
	qwenAudioTTSUpstreamPath    = "/api-ws/v1/inference"
)

var qwenAudioVoicePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,127}$`)

var errQwenAudioRequestBodyTooLarge = errors.New("audio request body exceeds 8 MiB")

type qwenAudioSpeechRequest struct {
	Model          string   `json:"model"`
	Input          string   `json:"input"`
	Voice          string   `json:"voice"`
	ResponseFormat string   `json:"response_format"`
	Speed          *float64 `json:"speed"`
	Instructions   string   `json:"instructions"`
	Stream         *bool    `json:"stream"`
}

type qwenAudioRequest struct {
	model        string
	bodyHash     string
	wav          []byte
	wavMeta      service.WAVMetadata
	speech       qwenAudioSpeechRequest
	inbound      string
	upstream     string
	audioMode    string
	usageUnits   float64
	responseMIME string
}

// QwenAudioTranscriptions implements OpenAI's multipart transcription shape
// while forwarding only validated PCM WAV to native Qwen ASR.
func (h *OpenAIGatewayHandler) QwenAudioTranscriptions(c *gin.Context) {
	req, err := parseQwenAudioTranscriptionRequest(c)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, errQwenAudioRequestBodyTooLarge) {
			status = http.StatusRequestEntityTooLarge
		}
		h.errorResponse(c, status, "invalid_request_error", err.Error())
		return
	}
	h.handleQwenAudio(c, req)
}

// QwenAudioSpeech implements non-streaming OpenAI-compatible MP3 speech.
func (h *OpenAIGatewayHandler) QwenAudioSpeech(c *gin.Context) {
	req, err := parseQwenAudioSpeechRequest(c)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, errQwenAudioRequestBodyTooLarge) {
			status = http.StatusRequestEntityTooLarge
		}
		h.errorResponse(c, status, "invalid_request_error", err.Error())
		return
	}
	h.handleQwenAudio(c, req)
}

func parseQwenAudioTranscriptionRequest(c *gin.Context) (qwenAudioRequest, error) {
	if c == nil || c.Request == nil {
		return qwenAudioRequest{}, errors.New("request is required")
	}
	mediaType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || !strings.EqualFold(mediaType, "multipart/form-data") {
		return qwenAudioRequest{}, errors.New("Content-Type must be multipart/form-data")
	}
	if err := c.Request.ParseMultipartForm(service.QwenAudioMaxRequestBodyBytes); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return qwenAudioRequest{}, errQwenAudioRequestBodyTooLarge
		}
		return qwenAudioRequest{}, errors.New("invalid multipart form")
	}
	if c.Request.MultipartForm != nil {
		defer func() { _ = c.Request.MultipartForm.RemoveAll() }()
	}
	model := strings.TrimSpace(c.Request.FormValue("model"))
	if model == "" {
		return qwenAudioRequest{}, errors.New("model is required")
	}
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		return qwenAudioRequest{}, errors.New("file is required")
	}
	defer func() { _ = file.Close() }()
	if header != nil && header.Size > service.QwenAudioMaxWAVBytes {
		return qwenAudioRequest{}, errors.New("WAV file exceeds 7 MiB")
	}
	wav, err := io.ReadAll(io.LimitReader(file, service.QwenAudioMaxWAVBytes+1))
	if err != nil {
		return qwenAudioRequest{}, errors.New("failed to read WAV file")
	}
	if int64(len(wav)) > service.QwenAudioMaxWAVBytes {
		return qwenAudioRequest{}, errors.New("WAV file exceeds 7 MiB")
	}
	meta, err := service.ParsePCM16WAV(wav)
	if err != nil {
		return qwenAudioRequest{}, err
	}
	return qwenAudioRequest{
		model: model, bodyHash: hashQwenAudioPayload(wav), wav: wav, wavMeta: meta,
		inbound: qwenAudioASRInboundEndpoint, upstream: qwenAudioASRUpstreamPath,
		audioMode: "stt", usageUnits: meta.Duration.Hours(), responseMIME: "application/json",
	}, nil
}

func parseQwenAudioSpeechRequest(c *gin.Context) (qwenAudioRequest, error) {
	if c == nil || c.Request == nil || c.Request.Body == nil {
		return qwenAudioRequest{}, errors.New("request body is required")
	}
	mediaType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || !strings.EqualFold(mediaType, "application/json") {
		return qwenAudioRequest{}, errors.New("Content-Type must be application/json")
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return qwenAudioRequest{}, errQwenAudioRequestBodyTooLarge
		}
		return qwenAudioRequest{}, errors.New("failed to read request body")
	}
	if len(body) == 0 {
		return qwenAudioRequest{}, errors.New("request body is required")
	}
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	decoder.DisallowUnknownFields()
	var speech qwenAudioSpeechRequest
	if err := decoder.Decode(&speech); err != nil {
		return qwenAudioRequest{}, errors.New("invalid JSON request body")
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return qwenAudioRequest{}, errors.New("invalid JSON request body")
	}
	speech.Model = strings.TrimSpace(speech.Model)
	speech.Voice = strings.TrimSpace(speech.Voice)
	if speech.Model == "" {
		return qwenAudioRequest{}, errors.New("model is required")
	}
	if !utf8.ValidString(speech.Input) || strings.TrimSpace(speech.Input) == "" {
		return qwenAudioRequest{}, errors.New("input must be non-empty UTF-8 text")
	}
	if utf8.RuneCountInString(speech.Input) > service.QwenAudioMaxTTSCharacters {
		return qwenAudioRequest{}, errors.New("input exceeds 4000 Unicode code points")
	}
	if !qwenAudioVoicePattern.MatchString(speech.Voice) {
		return qwenAudioRequest{}, errors.New("voice is invalid")
	}
	if speech.ResponseFormat == "" {
		speech.ResponseFormat = "mp3"
	}
	if !strings.EqualFold(strings.TrimSpace(speech.ResponseFormat), "mp3") {
		return qwenAudioRequest{}, errors.New("response_format must be mp3")
	}
	if speech.Stream != nil && *speech.Stream {
		return qwenAudioRequest{}, errors.New("streaming speech is not supported")
	}
	if !utf8.ValidString(speech.Instructions) {
		return qwenAudioRequest{}, errors.New("instructions must be valid UTF-8 text")
	}
	if utf8.RuneCountInString(speech.Instructions) > service.QwenAudioMaxInstructionCharacters {
		return qwenAudioRequest{}, errors.New("instructions exceed 4000 Unicode code points")
	}
	if speech.Speed == nil {
		value := 1.0
		speech.Speed = &value
	}
	if math.IsNaN(*speech.Speed) || math.IsInf(*speech.Speed, 0) || *speech.Speed < 0.5 || *speech.Speed > 2.0 {
		return qwenAudioRequest{}, errors.New("speed must be between 0.5 and 2.0")
	}
	return qwenAudioRequest{
		model: speech.Model, bodyHash: hashQwenAudioPayload(body), speech: speech,
		inbound: qwenAudioTTSInboundEndpoint, upstream: qwenAudioTTSUpstreamPath,
		audioMode: "tts", responseMIME: "audio/mpeg",
	}, nil
}

func (h *OpenAIGatewayHandler) handleQwenAudio(c *gin.Context, req qwenAudioRequest) {
	streamStarted := false
	defer h.recoverResponsesPanic(c, &streamStarted)
	requestStart := time.Now()
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}
	if apiKey.Group == nil || apiKey.Group.Platform != service.PlatformOpenAI {
		h.errorResponse(c, http.StatusNotFound, "not_found_error", "Audio API is not supported for this platform")
		return
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusInternalServerError, "api_error", "User context not found")
		return
	}
	reqLog := requestLogger(c, "handler.openai_gateway.qwen_audio",
		zap.Int64("user_id", subject.UserID), zap.Int64("api_key_id", apiKey.ID),
		zap.Any("group_id", apiKey.GroupID), zap.String("mode", req.audioMode), zap.String("model", req.model))
	if !h.ensureResponsesDependencies(c, reqLog) {
		return
	}
	if req.audioMode == "tts" {
		auditBody, _ := json.Marshal(map[string]any{"messages": []map[string]any{{"role": "user", "content": req.speech.Input}}})
		if decision := h.checkSecurityAudit(c, reqLog, apiKey, subject, service.ContentModerationProtocolOpenAIChat, req.model, auditBody); decision != nil && !decision.AllowNextStage {
			h.openAISecurityAuditError(c, decision)
			return
		}
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
	userRelease, acquired := h.acquireResponsesUserSlot(c, subject.UserID, subject.Concurrency, false, &streamStarted, reqLog)
	if !acquired {
		return
	}
	if userRelease != nil {
		defer userRelease()
	}
	service.SetOpsLatencyMs(c, service.OpsAuthLatencyMsKey, time.Since(requestStart).Milliseconds())
	requestCtx := service.WithOpenAIProfitControlSuppressed(c.Request.Context())
	sessionHash := h.gatewayService.GenerateExplicitSessionHash(c, []byte(req.bodyHash))
	failed := make(map[int64]struct{})
	maxSwitches := h.maxAccountSwitches
	if maxSwitches <= 0 {
		maxSwitches = 3
	}
	var lastFailover *service.UpstreamFailoverError
	switchCount := 0

	for {
		selection, _, selectErr := h.gatewayService.SelectAccountWithSchedulerForCapability(
			requestCtx, apiKey.GroupID, "", sessionHash, req.model, failed,
			service.OpenAIUpstreamTransportHTTPSSE, service.OpenAIEndpointCapabilityQwenAudio,
			false, false, false, service.PlatformOpenAI,
		)
		if selectErr != nil || selection == nil || selection.Account == nil {
			if lastFailover != nil {
				h.handleFailoverExhausted(c, lastFailover, false)
				return
			}
			cls := classifyNoAccountErrorFromGin(c, h.gatewayService, apiKey, req.model, req.model, service.PlatformOpenAI)
			if cls.ModelNotFound {
				h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "model is not available for Qwen audio")
				return
			}
			markOpsRoutingCapacityLimitedIfNoAvailable(c, selectErr)
			h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "No available Qwen audio accounts")
			return
		}
		account := selection.Account
		setOpsSelectedAccount(c, account.ID, account.Platform)
		release, slotStatus := h.acquireResponsesAccountSlot(c, apiKey.GroupID, sessionHash, selection, false, &streamStarted, reqLog)
		if slotStatus == openAISlotAcquireProfitVetoed {
			failed[account.ID] = struct{}{}
			continue
		}
		if slotStatus != openAISlotAcquireOK {
			return
		}
		var asr *service.QwenASRResult
		var tts *service.QwenTTSResult
		var forwardErr error
		func() {
			defer release()
			if req.audioMode == "stt" {
				asr, forwardErr = h.gatewayService.ForwardQwenASR(requestCtx, account, req.model, req.wav, req.wavMeta)
				return
			}
			tts, forwardErr = h.gatewayService.ForwardQwenTTS(requestCtx, account, service.QwenTTSRequest{
				Model: req.model, Input: req.speech.Input, Voice: req.speech.Voice,
				Speed: *req.speech.Speed, Instructions: req.speech.Instructions,
			})
		}()
		if forwardErr != nil {
			var failoverErr *service.UpstreamFailoverError
			if errors.As(forwardErr, &failoverErr) {
				if failoverErr.ShouldRetryNextAccount() {
					h.gatewayService.ReportOpenAIAccountScheduleResult(account, req.model, false, nil, forwardErr)
				} else {
					h.gatewayService.ReportOpenAIAccountScheduleResult(account, req.model, true, nil)
					h.handleFailoverExhausted(c, failoverErr, false)
					return
				}
				failed[account.ID] = struct{}{}
				lastFailover = failoverErr
				if switchCount >= maxSwitches {
					h.handleFailoverExhausted(c, failoverErr, false)
					return
				}
				switchCount++
				h.gatewayService.RecordOpenAIAccountSwitch()
				continue
			}
			h.gatewayService.ReportOpenAIAccountScheduleResult(account, req.model, false, nil, forwardErr)
			h.errorResponse(c, http.StatusBadGateway, "upstream_error", "Upstream audio request failed")
			return
		}

		h.gatewayService.ReportOpenAIAccountScheduleResult(account, req.model, true, nil)
		var result *service.OpenAIForwardResult
		if req.audioMode == "stt" {
			c.JSON(http.StatusOK, gin.H{"text": asr.Text})
			result = &service.OpenAIForwardResult{
				RequestID: service.StableQwenAudioBillingRequestID(asr.RequestID), Model: req.model,
				BillingModel: req.model, UpstreamModel: asr.UpstreamModel, UpstreamEndpoint: req.upstream,
				Duration: asr.Duration, AudioUsage: &service.AudioUsage{Mode: "stt", DurationOrUnits: req.usageUnits},
			}
		} else {
			c.Data(http.StatusOK, req.responseMIME, tts.Audio)
			result = &service.OpenAIForwardResult{
				RequestID: service.StableQwenAudioBillingRequestID(tts.TaskID), Model: req.model,
				BillingModel: req.model, UpstreamModel: tts.UpstreamModel, UpstreamEndpoint: req.upstream,
				Duration:   tts.Duration,
				AudioUsage: &service.AudioUsage{Mode: "tts", DurationOrUnits: float64(tts.BilledCharacters) / 1_000_000},
			}
		}
		h.recordQwenAudioUsage(c, apiKey, account, subscription, req, result)
		return
	}
}

func (h *OpenAIGatewayHandler) recordQwenAudioUsage(c *gin.Context, apiKey *service.APIKey, account *service.Account, subscription *service.UserSubscription, req qwenAudioRequest, result *service.OpenAIForwardResult) {
	if h == nil || c == nil || apiKey == nil || account == nil || result == nil || result.AudioUsage == nil || result.AudioUsage.DurationOrUnits <= 0 {
		return
	}
	userAgent := c.GetHeader("User-Agent")
	clientIP := ip.GetClientIP(c)
	sessionID := service.ExtractClientSessionID(c)
	quotaPlatform := service.QuotaPlatform(c.Request.Context(), apiKey)
	usageFields := clientRequestedUsageFields(c, service.ChannelMappingResult{}, req.model, result.UpstreamModel)
	h.submitMandatoryUsageRecordTask(c.Request.Context(), func(ctx context.Context) {
		if err := h.gatewayService.RecordUsage(ctx, &service.OpenAIRecordUsageInput{
			Result: result, APIKey: apiKey, User: apiKey.User, Account: account, Subscription: subscription,
			InboundEndpoint: req.inbound, UpstreamEndpoint: req.upstream, UserAgent: userAgent,
			IPAddress: clientIP, SessionID: sessionID, RequestPayloadHash: req.bodyHash,
			APIKeyService: h.apiKeyService, QuotaPlatform: quotaPlatform, ChannelUsageFields: usageFields,
		}); err != nil {
			logger.L().Error("qwen_audio.record_usage_failed",
				zap.Int64("api_key_id", apiKey.ID), zap.Int64("account_id", account.ID),
				zap.String("mode", req.audioMode), zap.Error(err))
		}
	})
}

func hashQwenAudioPayload(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
