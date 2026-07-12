package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/xai"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func (s *OpenAIGatewayService) forwardGrokResponses(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
	originalModel string,
	reqStream bool,
	startTime time.Time,
) (*OpenAIForwardResult, error) {
	if account.Type != AccountTypeOAuth {
		return nil, fmt.Errorf("grok account type %s is not supported by subscription forwarding", account.Type)
	}

	upstreamModel := account.GetMappedModel(originalModel)
	if strings.TrimSpace(upstreamModel) == "" {
		upstreamModel = "grok-4.3"
	}
	patchedBody, err := patchGrokResponsesBody(body, upstreamModel)
	if err != nil {
		if errors.Is(err, ErrGrokImageGenerationUnsupported) {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{
				"type":    "invalid_request_error",
				"message": err.Error(),
				"code":    "client_compat",
			}})
		}
		return nil, err
	}

	token, _, err := s.GetAccessToken(ctx, account)
	if err != nil {
		return nil, err
	}

	upstreamCtx, releaseUpstreamCtx := detachUpstreamContext(ctx)
	defer releaseUpstreamCtx()
	upstreamReq, err := buildGrokResponsesRequest(upstreamCtx, c, account, patchedBody, token)
	if err != nil {
		return nil, err
	}

	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}

	upstreamStart := time.Now()
	resp, err := s.httpUpstream.Do(upstreamReq, proxyURL, account.ID, account.Concurrency)
	SetOpsLatencyMs(c, OpsUpstreamLatencyMsKey, time.Since(upstreamStart).Milliseconds())
	if err != nil {
		return nil, s.handleOpenAIUpstreamTransportError(ctx, c, account, err, false)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		respBody := s.readUpstreamErrorBody(resp)
		resp.Body = io.NopCloser(bytes.NewReader(respBody))
		s.updateGrokUsageSnapshot(ctx, account.ID, xai.ParseQuotaHeaders(resp.Header, resp.StatusCode))
		upstreamMsg := sanitizeUpstreamErrorMessage(extractUpstreamErrorMessage(respBody))
		if upstreamMsg == "" {
			upstreamMsg = fmt.Sprintf("xAI upstream returned status %d", resp.StatusCode)
		}
		appendOpsUpstreamError(c, OpsUpstreamErrorEvent{
			Platform:           account.Platform,
			AccountID:          account.ID,
			AccountName:        account.Name,
			UpstreamStatusCode: resp.StatusCode,
			UpstreamRequestID:  firstNonEmpty(resp.Header.Get("x-request-id"), resp.Header.Get("xai-request-id")),
			Kind:               grokUpstreamErrorKind(resp.StatusCode, upstreamMsg),
			Message:            upstreamMsg,
		})
		s.handleGrokAccountUpstreamError(ctx, account, resp.StatusCode, resp.Header, respBody)
		if s.shouldFailoverUpstreamError(resp.StatusCode) {
			return nil, &UpstreamFailoverError{
				StatusCode:             resp.StatusCode,
				ResponseBody:           respBody,
				RetryableOnSameAccount: account.IsPoolMode() && account.IsPoolModeRetryableStatus(resp.StatusCode),
			}
		}
		return s.handleErrorResponse(ctx, resp, c, account, patchedBody, upstreamModel)
	}

	quotaSnapshot := xai.ParseQuotaHeaders(resp.Header, resp.StatusCode)
	s.updateGrokUsageSnapshot(ctx, account.ID, quotaSnapshot)
	s.maybeClearGrokTempUnschedulable(ctx, account, quotaSnapshot)

	var usage *OpenAIUsage
	var firstTokenMs *int
	responseID := ""
	if reqStream {
		streamResult, err := s.handleStreamingResponse(ctx, resp, c, account, startTime, originalModel, upstreamModel, upstreamStart)
		if err != nil {
			return nil, err
		}
		usage = streamResult.usage
		firstTokenMs = streamResult.firstTokenMs
		responseID = strings.TrimSpace(streamResult.responseID)
	} else {
		nonStreamResult, err := s.handleNonStreamingResponse(ctx, resp, c, account, originalModel, upstreamModel)
		if err != nil {
			return nil, err
		}
		usage = nonStreamResult.usage
		responseID = strings.TrimSpace(nonStreamResult.responseID)
	}

	if usage == nil {
		usage = &OpenAIUsage{}
	}
	s.bindHTTPResponseAccount(ctx, c, account, responseID)
	reasoningEffort := extractOpenAIReasoningEffortFromBody(patchedBody, originalModel)
	return &OpenAIForwardResult{
		RequestID:       firstNonEmpty(resp.Header.Get("x-request-id"), resp.Header.Get("xai-request-id")),
		ResponseID:      responseID,
		Usage:           *usage,
		Model:           originalModel,
		UpstreamModel:   upstreamModel,
		ReasoningEffort: reasoningEffort,
		Stream:          reqStream,
		OpenAIWSMode:    false,
		ResponseHeaders: resp.Header.Clone(),
		Duration:        time.Since(startTime),
		FirstTokenMs:    firstTokenMs,
	}, nil
}

func patchGrokResponsesBody(body []byte, upstreamModel string) ([]byte, error) {
	if !json.Valid(body) {
		return nil, fmt.Errorf("invalid json request body")
	}
	out, err := sjson.SetBytes(body, "model", upstreamModel)
	if err != nil {
		return nil, err
	}
	for _, unsupportedField := range []string{"prompt_cache_retention", "safety_identifier"} {
		if gjson.GetBytes(out, unsupportedField).Exists() {
			out, err = sjson.DeleteBytes(out, unsupportedField)
			if err != nil {
				return nil, err
			}
		}
	}
	if strings.EqualFold(upstreamModel, "grok-4.5") {
		for _, unsupportedField := range []string{"presence_penalty", "presencePenalty", "frequency_penalty", "frequencyPenalty", "stop"} {
			if gjson.GetBytes(out, unsupportedField).Exists() {
				out, err = sjson.DeleteBytes(out, unsupportedField)
				if err != nil {
					return nil, err
				}
			}
		}
	}
	out, err = sanitizeGrokResponsesUnsupportedFields(out)
	if err != nil {
		return nil, err
	}
	// Upgrade deprecated Live Search (search_parameters / live_search) to Agent Tools
	// on the Responses path before tool whitelist filtering.
	out, err = sanitizeGrokDeprecatedLiveSearch(out, grokLiveSearchSanitizeOptions{UpgradeToAgentTools: true})
	if err != nil {
		return nil, err
	}
	out, err = sanitizeGrokResponsesTools(out)
	if err != nil {
		return nil, err
	}
	// xAI Grok only accepts low|medium|high; clamp xhigh/max before upstream.
	out, err = clampGrokReasoningEffort(out)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// clampGrokReasoningEffort maps unsupported effort levels onto Grok's
// low|medium|high set. Client/Codex "xhigh"/"max" become "high".
func clampGrokReasoningEffort(body []byte) ([]byte, error) {
	out := body
	for _, path := range []string{"reasoning.effort", "reasoning_effort"} {
		raw := strings.TrimSpace(gjson.GetBytes(out, path).String())
		if raw == "" {
			continue
		}
		clamped := clampGrokEffortLevel(raw)
		if clamped == "" || strings.EqualFold(clamped, raw) {
			continue
		}
		updated, err := sjson.SetBytes(out, path, clamped)
		if err != nil {
			return nil, err
		}
		out = updated
	}
	return out, nil
}

func clampGrokEffortLevel(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	value = strings.NewReplacer("-", "", "_", "", " ", "").Replace(value)
	switch value {
	case "low", "medium", "high":
		return value
	case "xhigh", "extrahigh", "max":
		return "high"
	case "none", "minimal":
		return "low"
	default:
		return ""
	}
}

// grokLiveSearchSanitizeOptions controls how deprecated xAI Live Search fields
// are handled. Chat Completions only strips them (Agent Tools are Responses-only);
// Responses may upgrade an active search intent into web_search / x_search tools.
type grokLiveSearchSanitizeOptions struct {
	UpgradeToAgentTools bool
}

var grokDeprecatedLiveSearchFields = map[string]struct{}{
	"search_parameters": {},
	"live_search":       {},
}

func sanitizeGrokDeprecatedLiveSearch(body []byte, opts grokLiveSearchSanitizeOptions) ([]byte, error) {
	if !bytes.Contains(body, []byte(`"search_parameters"`)) && !bytes.Contains(body, []byte(`"live_search"`)) {
		return body, nil
	}

	wantUpgrade := opts.UpgradeToAgentTools && grokLiveSearchIntentActive(body)
	alreadyHasSearchTool := grokBodyHasSearchAgentTool(body)

	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	changed := deleteJSONFields(payload, grokDeprecatedLiveSearchFields)

	if wantUpgrade && !alreadyHasSearchTool {
		if root, ok := payload.(map[string]any); ok {
			tools, _ := root["tools"].([]any)
			if tools == nil {
				tools = make([]any, 0, 2)
			}
			if !grokToolsSliceHasSearchAgentTool(tools) {
				tools = append(tools,
					map[string]any{"type": "web_search"},
					map[string]any{"type": "x_search"},
				)
				root["tools"] = tools
				changed = true
			}
		}
	}

	if !changed {
		return body, nil
	}
	return json.Marshal(payload)
}

func grokLiveSearchIntentActive(body []byte) bool {
	if live := gjson.GetBytes(body, "live_search"); live.Exists() {
		switch live.Type {
		case gjson.True:
			return true
		case gjson.String:
			switch strings.ToLower(strings.TrimSpace(live.String())) {
			case "true", "1", "on", "auto", "enabled":
				return true
			}
		case gjson.Number:
			return live.Num != 0
		}
	}

	mode := strings.ToLower(strings.TrimSpace(gjson.GetBytes(body, "search_parameters.mode").String()))
	switch mode {
	case "on", "auto":
		return true
	case "off", "":
		// Empty mode with a present search_parameters object historically meant
		// "enable search with defaults" on some clients; treat presence of a
		// non-off mode-less object as inactive only when mode is explicitly off
		// or the field is absent. If search_parameters exists without mode, do
		// not upgrade — strip only.
		return false
	default:
		// Unknown non-off modes (e.g. "enabled") still imply search intent.
		return mode != "off"
	}
}

func grokBodyHasSearchAgentTool(body []byte) bool {
	tools := gjson.GetBytes(body, "tools")
	if !tools.Exists() || !tools.IsArray() {
		return false
	}
	for _, tool := range tools.Array() {
		switch strings.TrimSpace(tool.Get("type").String()) {
		case "web_search", "x_search":
			return true
		}
	}
	return false
}

func grokToolsSliceHasSearchAgentTool(tools []any) bool {
	for _, item := range tools {
		tool, ok := item.(map[string]any)
		if !ok {
			continue
		}
		toolType, _ := tool["type"].(string)
		switch strings.TrimSpace(toolType) {
		case "web_search", "x_search":
			return true
		}
	}
	return false
}

var grokResponsesUnsupportedRecursiveFields = map[string]struct{}{
	"external_web_access": {},
}

func sanitizeGrokResponsesUnsupportedFields(body []byte) ([]byte, error) {
	if !bytes.Contains(body, []byte(`"external_web_access"`)) {
		return body, nil
	}

	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	if !deleteJSONFields(payload, grokResponsesUnsupportedRecursiveFields) {
		return body, nil
	}
	return json.Marshal(payload)
}

func deleteJSONFields(value any, fields map[string]struct{}) bool {
	switch typed := value.(type) {
	case map[string]any:
		changed := false
		for field := range fields {
			if _, ok := typed[field]; ok {
				delete(typed, field)
				changed = true
			}
		}
		for _, child := range typed {
			if deleteJSONFields(child, fields) {
				changed = true
			}
		}
		return changed
	case []any:
		changed := false
		for _, child := range typed {
			if deleteJSONFields(child, fields) {
				changed = true
			}
		}
		return changed
	default:
		return false
	}
}

var grokResponsesSupportedToolTypes = map[string]struct{}{
	"code_execution":     {},
	"code_interpreter":   {},
	"collections_search": {},
	"file_search":        {},
	"function":           {},
	"mcp":                {},
	"shell":              {},
	"web_search":         {},
	"x_search":           {},
}

func grokResponsesHasImageGenerationTool(body []byte) bool {
	tools := gjson.GetBytes(body, "tools")
	if !tools.Exists() || !tools.IsArray() {
		return false
	}
	for _, tool := range tools.Array() {
		toolType := strings.TrimSpace(tool.Get("type").String())
		if toolType == "image_generation" || toolType == "image_generation_call" {
			return true
		}
		name := strings.TrimSpace(tool.Get("name").String())
		if name == "" {
			name = strings.TrimSpace(tool.Get("function.name").String())
		}
		if name == "image_gen" || name == "image_generation" {
			return true
		}
	}
	choiceType := strings.TrimSpace(gjson.GetBytes(body, "tool_choice.type").String())
	return choiceType == "image_generation"
}

// ErrGrokImageGenerationUnsupported is returned when a Responses-shaped Grok
// request asks for image_generation. Grok images go through /v1/images/* (see
// grok_media.go); silently stripping the tool hid the mismatch from clients.
var ErrGrokImageGenerationUnsupported = fmt.Errorf(
	"image_generation is not supported on Grok chat/responses; use /v1/images/generations or /v1/images/edits",
)

func sanitizeGrokResponsesTools(body []byte) ([]byte, error) {
	if grokResponsesHasImageGenerationTool(body) {
		return nil, ErrGrokImageGenerationUnsupported
	}

	tools := gjson.GetBytes(body, "tools")
	if !tools.Exists() || !tools.IsArray() {
		return body, nil
	}

	rawTools := tools.Array()
	filteredTools := make([]json.RawMessage, 0, len(rawTools))
	for _, tool := range rawTools {
		toolType := strings.TrimSpace(tool.Get("type").String())
		if _, ok := grokResponsesSupportedToolTypes[toolType]; ok {
			filteredTools = append(filteredTools, json.RawMessage(tool.Raw))
		}
	}

	var err error
	if len(filteredTools) != len(rawTools) {
		if len(filteredTools) == 0 {
			body, err = sjson.DeleteBytes(body, "tools")
		} else {
			var encoded []byte
			encoded, err = json.Marshal(filteredTools)
			if err != nil {
				return nil, err
			}
			body, err = sjson.SetRawBytes(body, "tools", encoded)
		}
		if err != nil {
			return nil, err
		}
	}

	toolChoice := gjson.GetBytes(body, "tool_choice")
	if !toolChoice.Exists() {
		return body, nil
	}
	if shouldDropGrokToolChoice(toolChoice, filteredTools) {
		body, err = sjson.DeleteBytes(body, "tool_choice")
		if err != nil {
			return nil, err
		}
	}
	return body, nil
}

func shouldDropGrokToolChoice(toolChoice gjson.Result, tools []json.RawMessage) bool {
	if len(tools) == 0 {
		return true
	}
	if !toolChoice.IsObject() {
		return false
	}
	choiceType := strings.TrimSpace(toolChoice.Get("type").String())
	if choiceType == "" {
		return false
	}
	if _, ok := grokResponsesSupportedToolTypes[choiceType]; !ok {
		return true
	}
	if choiceType == "function" {
		choiceName := strings.TrimSpace(toolChoice.Get("name").String())
		if choiceName == "" {
			choiceName = strings.TrimSpace(toolChoice.Get("function.name").String())
		}
		if choiceName == "" {
			return false
		}
		for _, tool := range tools {
			var item struct {
				Type     string `json:"type"`
				Name     string `json:"name"`
				Function struct {
					Name string `json:"name"`
				} `json:"function"`
			}
			if err := json.Unmarshal(tool, &item); err != nil {
				continue
			}
			name := strings.TrimSpace(item.Name)
			if name == "" {
				name = strings.TrimSpace(item.Function.Name)
			}
			if strings.TrimSpace(item.Type) == "function" && name == choiceName {
				return false
			}
		}
		return true
	}
	return false
}

func buildGrokResponsesRequest(ctx context.Context, c *gin.Context, account *Account, body []byte, token string) (*http.Request, error) {
	targetURL, err := xai.BuildResponsesURL(account.GetGrokBaseURL())
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	xai.SetGrokCLIResponsesHeaders(req.Header)
	if c != nil {
		if v := c.GetHeader("OpenAI-Beta"); strings.TrimSpace(v) != "" {
			req.Header.Set("OpenAI-Beta", v)
		}
		// Prefer an explicit session_id header; otherwise derive from prompt_cache_key
		// so xAI can keep the same conversation/cache affinity across turns.
		sessionSignal := strings.TrimSpace(c.GetHeader("session_id"))
		if sessionSignal == "" {
			sessionSignal = strings.TrimSpace(gjson.GetBytes(body, "prompt_cache_key").String())
		}
		if sessionSignal != "" {
			apiKeyID := getAPIKeyIDFromContext(c)
			req.Header.Set("session_id", generateSessionUUID(isolateOpenAISessionID(apiKeyID, sessionSignal)))
		}
	}
	return req, nil
}

func (s *OpenAIGatewayService) updateGrokUsageSnapshot(ctx context.Context, accountID int64, snapshot *xai.QuotaSnapshot) {
	if s == nil || s.accountRepo == nil || accountID <= 0 || snapshot == nil {
		return
	}
	if s.codexSnapshotThrottle != nil && !s.codexSnapshotThrottle.Allow(accountID, time.Now()) {
		return
	}
	_ = s.accountRepo.UpdateExtra(ctx, accountID, map[string]any{
		grokQuotaSnapshotExtraKey: snapshot,
	})
}

func grokUpstreamErrorKind(statusCode int, upstreamMsg string) string {
	if isGrokDeprecatedLiveSearchError(statusCode, upstreamMsg) {
		return "client_compat"
	}
	return "failover"
}

func isGrokDeprecatedLiveSearchError(statusCode int, upstreamMsg string) bool {
	if statusCode != http.StatusGone {
		return false
	}
	msg := strings.ToLower(strings.TrimSpace(upstreamMsg))
	return strings.Contains(msg, "live search is deprecated") ||
		strings.Contains(msg, "agent tools api") ||
		strings.Contains(msg, "search_parameters")
}

func (s *OpenAIGatewayService) handleGrokAccountUpstreamError(ctx context.Context, account *Account, statusCode int, headers http.Header, responseBody []byte) {
	if s == nil || account == nil {
		return
	}
	// 410 Gone (e.g. deprecated Live Search) is a request-shape / client-compat
	// issue, not an account fault — never temp-unschedulable or failover on it.
	// shouldFailoverUpstreamError also excludes 410.
	switch statusCode {
	case http.StatusUnauthorized:
		s.tempUnscheduleGrok(ctx, account, 10*time.Minute, statusCode, "grok oauth token unauthorized", nil)
	case http.StatusForbidden:
		s.tempUnscheduleGrok(ctx, account, 30*time.Minute, statusCode, "grok entitlement or subscription tier denied", nil)
	case http.StatusTooManyRequests:
		snapshot := xai.ParseQuotaHeaders(headers, statusCode)
		cooldown := grokRateLimitCooldown(snapshot)
		s.tempUnscheduleGrok(ctx, account, cooldown, statusCode, "grok rate limited", snapshot)
	default:
		if statusCode >= 500 {
			s.tempUnscheduleGrok(ctx, account, 2*time.Minute, statusCode, "grok upstream temporary error", nil)
		}
	}
	_ = responseBody
}

const grokTempUnschedMaxCooldown = 30 * time.Minute

func grokRateLimitCooldown(snapshot *xai.QuotaSnapshot) time.Duration {
	cooldown := 2 * time.Minute
	now := time.Now()
	if snapshot != nil && snapshot.RetryAfterSeconds != nil && *snapshot.RetryAfterSeconds > 0 {
		cooldown = time.Duration(*snapshot.RetryAfterSeconds) * time.Second
	} else if snapshot != nil {
		var earliest *time.Time
		for _, window := range []*xai.QuotaWindow{snapshot.Requests, snapshot.Tokens} {
			if window == nil || window.ResetUnix == nil || *window.ResetUnix <= 0 {
				continue
			}
			resetAt := time.Unix(*window.ResetUnix, 0)
			if resetAt.After(now) && (earliest == nil || resetAt.Before(*earliest)) {
				earliest = &resetAt
			}
		}
		if earliest != nil {
			cooldown = earliest.Sub(now)
		}
	}
	if cooldown < time.Second {
		cooldown = time.Second
	}
	if cooldown > grokTempUnschedMaxCooldown {
		cooldown = grokTempUnschedMaxCooldown
	}
	return cooldown
}

func (s *OpenAIGatewayService) tempUnscheduleGrok(ctx context.Context, account *Account, cooldown time.Duration, statusCode int, reason string, snapshot *xai.QuotaSnapshot) {
	if s == nil || account == nil {
		return
	}
	now := time.Now()
	until := now.Add(cooldown)
	if account.TempUnschedulableUntil != nil && account.TempUnschedulableUntil.After(until) {
		until = *account.TempUnschedulableUntil
	}

	state := &TempUnschedState{
		UntilUnix:       until.Unix(),
		TriggeredAtUnix: now.Unix(),
		StatusCode:      statusCode,
		MatchedKeyword:  reason,
		ErrorMessage:    reason,
	}
	if snapshot != nil && snapshot.RetryAfterSeconds != nil {
		state.ErrorMessage = fmt.Sprintf("%s (retry-after=%ds)", reason, *snapshot.RetryAfterSeconds)
	}
	reasonPayload := reason
	if raw, err := json.Marshal(state); err == nil {
		reasonPayload = string(raw)
	}

	s.BlockAccountScheduling(account, until, reason)
	if s.accountRepo != nil {
		stateCtx, cancel := openAIAccountStateContext(ctx)
		defer cancel()
		_ = s.accountRepo.SetTempUnschedulable(stateCtx, account.ID, until, reasonPayload)
	}
	if s.rateLimitService != nil && s.rateLimitService.tempUnschedCache != nil {
		stateCtx, cancel := openAIAccountStateContext(ctx)
		defer cancel()
		_ = s.rateLimitService.tempUnschedCache.SetTempUnsched(stateCtx, account.ID, state)
	}
	account.TempUnschedulableUntil = &until
	account.TempUnschedulableReason = reasonPayload
}

func (s *OpenAIGatewayService) maybeClearGrokTempUnschedulable(ctx context.Context, account *Account, snapshot *xai.QuotaSnapshot) {
	if s == nil || account == nil || account.ID <= 0 {
		return
	}
	now := time.Now()
	hasActiveTemp := account.TempUnschedulableUntil != nil && account.TempUnschedulableUntil.After(now)
	runtimeBlocked := s.isOpenAIAccountRuntimeBlocked(account)
	if !hasActiveTemp && !runtimeBlocked {
		return
	}

	// Clear when cooldown already expired, or upstream headers show recovery.
	expired := account.TempUnschedulableUntil != nil && !account.TempUnschedulableUntil.After(now)
	recoveredByHeaders := false
	if snapshot != nil {
		retryActive := grokQuotaRetryAfterActive(snapshot, now)
		hasRemaining := false
		for _, window := range []*xai.QuotaWindow{snapshot.Requests, snapshot.Tokens} {
			if window != nil && window.Remaining != nil && *window.Remaining > 0 {
				hasRemaining = true
				break
			}
		}
		recoveredByHeaders = !retryActive && (hasRemaining || snapshot.StatusCode > 0 && snapshot.StatusCode < 400)
	}
	if !expired && !recoveredByHeaders {
		return
	}

	s.ClearAccountSchedulingBlock(account.ID)
	if s.accountRepo != nil {
		stateCtx, cancel := openAIAccountStateContext(ctx)
		defer cancel()
		_ = s.accountRepo.ClearTempUnschedulable(stateCtx, account.ID)
	}
	if s.rateLimitService != nil && s.rateLimitService.tempUnschedCache != nil {
		stateCtx, cancel := openAIAccountStateContext(ctx)
		defer cancel()
		_ = s.rateLimitService.tempUnschedCache.DeleteTempUnsched(stateCtx, account.ID)
	}
	account.TempUnschedulableUntil = nil
	account.TempUnschedulableReason = ""
}
