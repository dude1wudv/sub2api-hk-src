package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

func suiteResult(op *DeepSeekSuiteOperation, in *DeepSeekSuiteInput) *OpenAIForwardResult {
	kind := in.Kind
	switch kind {
	case "files":
		kind = "document_parse"
	case "tts_ws":
		kind = "tts"
	}
	usage, _ := chatAuxiliaryUsage(kind, nil, "unreported")
	if in.Conversation {
		usage = nil
		if bytes.Contains(in.Body, []byte(`"deepseek_web_search"`)) || bytes.Contains(in.Body, []byte(`"deepseek_read_url"`)) {
			usage = &ChatAuxiliaryUsage{PriceBasis: ChatAuxiliaryPriceBasis, Provider: "deepseek"}
			for _, tool := range []string{"search", "fetch", "image_search"} {
				item, _ := chatAuxiliaryUsage(tool, nil, "unreported")
				usage.Items = append(usage.Items, item.Items...)
			}
		}
	}
	model := "deepseek-" + in.Kind
	if in.Conversation {
		model = in.Model
	}
	return &OpenAIForwardResult{RequestID: "deepseek_suite:" + op.ID, Model: model,
		Stream: in.Stream, chatAuxiliaryOnly: !in.Conversation, ChatAuxiliaryUsage: usage}
}

func deepSeekSuiteAuxiliaryUsage(raw []byte) *ChatAuxiliaryUsage {
	usage := gjson.GetBytes(raw, "usage")
	if !usage.IsObject() || !usage.Get("server_tool_use").IsObject() {
		return nil
	}
	out := &ChatAuxiliaryUsage{PriceBasis: ChatAuxiliaryPriceBasis, Provider: "deepseek"}
	for _, meter := range []struct{ kind, field string }{{"search", "tool_search"}, {"fetch", "tool_fetch"}, {"image_search", "tool_image_search"}} {
		count := int64(0)
		value := usage.Get("server_tool_use." + meter.field)
		if value.Exists() {
			parsed, err := strconv.ParseInt(value.Raw, 10, 64)
			if err != nil || parsed < 0 {
				item, _ := chatAuxiliaryUsage(meter.kind, nil, "unreported")
				out.Items = append(out.Items, item.Items...)
				continue
			}
			count = parsed
		}
		item, _ := chatAuxiliaryUsage(meter.kind, &count, "confirmed")
		out.Items = append(out.Items, item.Items...)
	}
	return out
}

func observeDeepSeekConversation(raw []byte, kind string, result *OpenAIForwardResult, claude *ClaudeUsage) {
	if nested := gjson.GetBytes(raw, "response"); nested.IsObject() {
		raw = []byte(nested.Raw)
	}
	if kind == "messages" {
		if gjson.GetBytes(raw, "type").String() == "message" {
			*claude = *parseClaudeUsageFromResponseBody(raw)
		} else {
			parseSSEUsagePassthrough(string(raw), claude)
		}
		// OpenAI usage stores the inclusive prompt total; pricing later removes
		// read/write cache buckets exactly once.
		result.Usage.InputTokens = claude.InputTokens + claude.CacheReadInputTokens + claude.CacheCreationInputTokens
		result.Usage.OutputTokens = claude.OutputTokens
		result.Usage.CacheReadInputTokens = claude.CacheReadInputTokens
		result.Usage.CacheCreationInputTokens = claude.CacheCreationInputTokens
	} else if usage, ok := openAIUsageFromGJSON(gjson.GetBytes(raw, "usage")); ok {
		result.Usage = usage
	}
	typ := gjson.GetBytes(raw, "type").String()
	if typ != "message_start" {
		if auxiliary := deepSeekSuiteAuxiliaryUsage(raw); auxiliary != nil {
			result.ChatAuxiliaryUsage = auxiliary // cumulative snapshot, never sum
		}
	}
}

func (s *OpenAIGatewayService) ForwardDeepSeekSuite(ctx context.Context, c *gin.Context, key *APIKey, account *Account, in *DeepSeekSuiteInput, op *DeepSeekSuiteOperation) (result *OpenAIForwardResult, payload []byte, err error) {
	requestStarted := false
	defer func() {
		if !requestStarted {
			op.Status = "failed"
		}
		if result != nil {
			result.RequestID = "deepseek_suite:" + op.ID
			if in.Conversation {
				result.Model = in.Model
				result.BillingModel = gjson.GetBytes(in.Body, "model").String()
				result.UpstreamModel = result.BillingModel
				if result.ChatAuxiliaryUsage == nil {
					result.ChatAuxiliaryUsage = suiteResult(op, in).ChatAuxiliaryUsage
				}
			}
		}
	}()
	account, err = s.DeepSeekSuiteAccount(ctx, key, account, in.Kind)
	if err != nil {
		return nil, nil, err
	}
	if in.Conversation {
		mapped := resolveMappedUpstreamModel(account, in.Body, "")
		model := strings.ToLower(mapped)
		if !strings.HasPrefix(model, "deepseek-flash") && !strings.HasPrefix(model, "deepseek-v4.1-flash") {
			return nil, nil, errors.New("full-suite conversation extensions require DeepSeek V4.1 Flash")
		}
		in.Body, err = sjson.SetBytes(in.Body, "model", mapped)
		if err != nil {
			return nil, nil, err
		}
		if in.Kind == "responses" {
			in.Body = normalizeDeepSeekResponsesRequestBody(account, in.Body)
		}
		if in.Kind == "chat" && in.Stream {
			in.Body, err = ensureOpenAIChatStreamUsage(in.Body)
			if err != nil {
				return nil, nil, err
			}
		}
	}
	var nativeCapture *incompleteForwardCapture
	if in.Conversation {
		if in.Kind == "messages" {
			nativeCapture = newIncompleteForwardCapture(ctx, c, in.Body)
			defer func() { nativeCapture.finishOpenAI(&result, &err) }()
		} else {
			var finish func(**OpenAIForwardResult, *error)
			ctx, finish = beginOpenAIIncompleteAttempt(ctx, c, in.Body)
			defer finish(&result, &err)
		}
	}
	target, err := s.deepSeekSuiteURL(account, in.Kind, "")
	if err != nil {
		return nil, nil, err
	}
	if in.File != nil && in.Kind == "files" {
		op.Filename, op.Bytes, op.Purpose = in.File.Filename, in.File.Size, in.Fields["purpose"]
		extension := path.Ext(in.File.Filename)
		if len(extension) > 16 {
			extension = ""
		}
		op.UpstreamName = op.ID + extension
		repo, _ := s.DeepSeekSuiteRepository()
		if err := repo.SaveDeepSeekOperation(ctx, op); err != nil {
			return nil, nil, err
		}
	}
	body, contentType, err := in.upstreamBody(op)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = body.Close() }()
	request, err := http.NewRequestWithContext(WithHTTPUpstreamRedirectsDisabled(ctx), http.MethodPost, target, body)
	if err != nil {
		return nil, nil, err
	}
	request = request.WithContext(WithHTTPUpstreamProfile(request.Context(), HTTPUpstreamProfileOpenAI))
	account.ApplyHeaderOverrides(request.Header)
	deleteHeaderAllForms(request.Header, "authorization")
	deleteHeaderAllForms(request.Header, "x-api-key")
	deleteHeaderAllForms(request.Header, "content-type")
	deleteHeaderAllForms(request.Header, "accept")
	request.Header.Set("Content-Type", contentType)
	request.Header.Set("Authorization", "Bearer "+account.GetCredential("api_key"))
	request.Header.Set("Accept", "application/json")
	if in.Stream {
		request.Header.Set("Accept", "text/event-stream")
	}
	if in.Kind == "tts" {
		request.Header.Set("Accept", "audio/*")
	}
	if in.Kind == "messages" {
		request.Header.Set("anthropic-version", "2023-06-01")
		if bytes.Contains(in.Body, []byte(`"file_id"`)) {
			request.Header.Set("anthropic-beta", "files-api-2025-04-14")
		}
	}
	SetActualOpenAIUpstreamEndpoint(c, request.URL.Path)
	started := time.Now()
	requestStarted = true
	var response *http.Response
	if nativeCapture != nil {
		response, err = nativeCapture.Do(request, func(r *http.Request) (*http.Response, error) {
			return s.doOpenAIUpstream(r, resolveAccountProxyURL(account), account)
		})
	} else {
		response, err = s.doOpenAIUpstream(request, resolveAccountProxyURL(account), account)
	}
	if err != nil {
		return nil, nil, err
	}
	if response == nil || response.Body == nil {
		return nil, nil, ErrChatStream
	}
	defer func() { _ = response.Body.Close() }()
	result = suiteResult(op, in)
	result.UpstreamEndpoint = request.URL.Path
	result.UpstreamHeaders, result.ResponseHeaders = response.Header, response.Header
	if in.Conversation {
		result.BillingModel = gjson.GetBytes(in.Body, "model").String()
		result.UpstreamModel = result.BillingModel
	}
	defer func() { result.Duration = time.Since(started) }()
	if response.StatusCode >= 400 {
		payload, err = readChatBounded(response.Body, 2<<20)
		if in.Conversation {
			observeDeepSeekConversation(payload, in.Kind, result, &ClaudeUsage{})
		}
		if err == nil && (response.StatusCode < 500 || response.StatusCode == http.StatusNotImplemented) && !deepSeekSuiteHasMeterOrResource(payload) {
			op.Status = "failed"
		}
		writeOpenAIPassthroughResponseHeaders(c.Writer.Header(), response.Header, s.responseHeaderFilter)
		c.Data(response.StatusCode, "application/json", payload)
		return result, nil, err
	}
	writeOpenAIPassthroughResponseHeaders(c.Writer.Header(), response.Header, s.responseHeaderFilter)
	if in.Conversation && strings.Contains(response.Header.Get("Content-Type"), "text/event-stream") {
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		scanner := s.newUpstreamSSEScanner(response.Body)
		claude := &ClaudeUsage{}
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "data:") {
				raw := []byte(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
				observeDeepSeekConversation(raw, in.Kind, result, claude)
			}
			if _, err = fmt.Fprintln(c.Writer, line); err != nil {
				return result, nil, err
			}
			if line == "" {
				c.Writer.Flush()
			}
		}
		return result, nil, scanner.Err()
	}
	if in.Kind == "tts" {
		contentType := response.Header.Get("Content-Type")
		if !strings.HasPrefix(contentType, "audio/") && contentType != "application/octet-stream" {
			return result, nil, errors.New("upstream did not return audio")
		}
		c.Header("Content-Type", contentType)
		const maxAudioBytes = 24000*2*30*60 + 4096
		count, copyErr := io.Copy(c.Writer, io.LimitReader(response.Body, maxAudioBytes+1))
		if count > maxAudioBytes {
			copyErr = ErrChatTooLarge
		}
		status := "incomplete"
		framed := response.ContentLength >= 0 || response.ProtoMajor >= 2
		for _, encoding := range response.TransferEncoding {
			framed = framed || encoding == "chunked"
		}
		if copyErr == nil && count > 0 && framed && (response.ContentLength < 0 || count == response.ContentLength) {
			status = "confirmed"
		}
		if status != "confirmed" && copyErr == nil {
			copyErr = ErrChatStream
		}
		result.ChatAuxiliaryUsage, _ = chatAuxiliaryUsage("tts", &in.TTSCharacters, status)
		return result, nil, copyErr
	}
	payload, err = readChatBounded(response.Body, 32<<20)
	if err != nil || !gjson.ValidBytes(payload) {
		return result, nil, errors.New("invalid upstream response")
	}
	if gjson.GetBytes(payload, "error").Exists() {
		if in.Conversation {
			observeDeepSeekConversation(payload, in.Kind, result, &ClaudeUsage{})
		}
		c.Data(response.StatusCode, "application/json", payload)
		return result, nil, errors.New("upstream operation failed")
	}
	if in.Conversation {
		observeDeepSeekConversation(payload, in.Kind, result, &ClaudeUsage{})
		return result, payload, nil
	}
	var quantity *int64
	meter := in.Kind
	switch in.Kind {
	case "search", "fetch":
		calls, meterErr := deepSeekToolBillableCalls(DeepSeekToolKind(in.Kind), payload)
		if meterErr != nil {
			return result, nil, meterErr
		}
		count := int64(calls)
		quantity = &count
	case "asr":
		duration := gjson.GetBytes(payload, "duration_ms")
		if !duration.Exists() {
			return result, payload, nil
		}
		milliseconds, meterErr := strconv.ParseInt(duration.Raw, 10, 64)
		if duration.Type != gjson.Number || meterErr != nil {
			return result, nil, errors.New("invalid upstream ASR duration")
		}
		seconds, meterErr := chatAuxiliaryASRSeconds(milliseconds)
		if meterErr != nil {
			return result, nil, meterErr
		}
		quantity = &seconds
	case "ocr", "files":
		for _, path := range []string{"usage.pages", "usage.page_count", "page_count", "pages_count"} {
			value := gjson.GetBytes(payload, path)
			if value.Type == gjson.Number {
				pages, parseErr := strconv.ParseInt(value.Raw, 10, 64)
				if parseErr == nil && pages >= 0 && pages <= 1000 {
					quantity = &pages
					break
				}
			}
		}
		if in.Kind == "files" {
			meter = "document_parse"
			op.RemoteID = gjson.GetBytes(payload, "id").String()
			if !chatValidRemoteID(op.RemoteID) {
				return result, nil, errors.New("upstream returned no file ID")
			}
			if in.Fields["purpose"] == "user_data" {
				zero := int64(0)
				quantity = &zero
			}
			payload, err = suiteFileMetadata(payload, op, in.AnthropicFiles)
			if err != nil {
				return result, nil, err
			}
			op.Metadata = json.RawMessage(payload)
		}
	}
	status := "unreported"
	if quantity != nil {
		status = "confirmed"
	}
	result.ChatAuxiliaryUsage, _ = chatAuxiliaryUsage(meter, quantity, status)
	return result, payload, nil
}

func deepSeekSuiteHasMeterOrResource(body []byte) bool {
	for _, field := range []string{"usage", "response.usage", "page_count", "pages_count", "duration", "duration_ms",
		"num_chars", "id", "file_id", "server_tool_calls"} {
		if gjson.GetBytes(body, field).Exists() {
			return true
		}
	}
	return false
}
