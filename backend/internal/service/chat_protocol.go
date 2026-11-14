package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"slices"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/google/uuid"
)

var ErrChatToolLimit = infraerrors.BadRequest("CHAT_TOOL_LIMIT", "已达到本轮工具调用上限，已保存部分内容")

type chatPreparedRequest struct {
	path    string
	body    map[string]any
	caps    ChatCapabilities
	options ChatRunOptions
	files   map[uuid.UUID]bool
}
type chatNativeContextKey struct{}
type chatNativeRequest struct {
	service    *ChatService
	userID     int64
	groupID    int64
	extensions bool
	files      map[uuid.UUID]bool
}

func chatHasProviderState(value any) bool {
	switch item := value.(type) {
	case map[string]any:
		for key, child := range item {
			if (key == "signature" || key == "encrypted_content") && chatString(child) != "" {
				return true
			}
			if chatHasProviderState(child) {
				return true
			}
		}
	case []any:
		for _, child := range item {
			if chatHasProviderState(child) {
				return true
			}
		}
	}
	return false
}

func chatAssistantBlocks(message ChatMessage, prefix, thinking bool) []any {
	var blocks []map[string]any
	unsigned := map[string]bool{}
	for _, raw := range message.Blocks {
		var block map[string]any
		if json.Unmarshal(raw, &block) != nil {
			continue
		}
		blocks = append(blocks, block)
		if block["type"] == "web_fetch_tool_result" && block["kind"] == "read_link" && chatObject(block["content"])["type"] == "web_fetch_result" && strings.TrimSpace(chatString(block["signature"])) == "" {
			unsigned[chatString(block["tool_use_id"])] = true
		}
	}
	out := []any{}
	var text map[string]any
	for _, block := range blocks {
		kind := chatString(block["type"])
		if kind == "text" {
			if block["isTemplate"] == true {
				continue
			}
			part := chatString(block["text"])
			if part == "" {
				continue
			}
			if text != nil {
				text["text"] = chatString(text["text"]) + part
			} else {
				if len(out) > 0 && chatObject(out[len(out)-1])["type"] == "text" {
					part = "\n\n" + part
				}
				text = map[string]any{"type": "text", "text": part}
				out = append(out, text)
			}
			continue
		}
		if kind == "thinking" && prefix && strings.TrimSpace(chatString(block["thinking"])) == "" {
			continue
		}
		text = nil
		if kind == "thinking" && prefix && thinking {
			out = append(out, map[string]any{"type": kind, "thinking": block["thinking"], "signature": chatString(block["signature"])})
		}
		if kind == "server_tool_use" && block["kind"] == "read_link" && !unsigned[chatString(block["id"])] {
			out = append(out, map[string]any{"type": kind, "id": block["id"], "name": block["name"], "input": block["input"], "kind": block["kind"]})
		}
		if kind == "web_fetch_tool_result" && block["kind"] == "read_link" && !unsigned[chatString(block["tool_use_id"])] {
			out = append(out, block)
		}
	}
	return out
}

func chatAssistantText(message ChatMessage) string {
	parts := []string{}
	for _, raw := range message.Blocks {
		var block map[string]any
		if json.Unmarshal(raw, &block) != nil {
			continue
		}
		switch block["type"] {
		case "text":
			if block["isTemplate"] != true {
				parts = append(parts, chatString(block["text"]))
			}
		case "web_fetch_tool_result":
			result := chatObject(block["content"])
			source := chatObject(chatObject(result["content"])["source"])
			if data := chatString(source["data"]); data != "" {
				parts = append(parts, "[Previous tool result]\n"+data)
			}
		case "web_search_tool_result":
			if results, ok := block["content"].([]any); ok {
				for _, item := range results {
					result := chatObject(item)
					if result["type"] == "web_search_result" {
						parts = append(parts, "[Previous search result] "+chatString(result["title"])+" "+chatString(result["url"])+"\n"+chatString(result["snippet"]))
					}
				}
			}
		}
	}
	return strings.Join(parts, "\n\n")
}

func (s *ChatService) chatUserContent(ctx context.Context, userID int64, message ChatMessage, prepared *chatPreparedRequest) (any, error) {
	transport := prepared.caps.Transport
	parts := []any{}
	for _, ref := range message.Files {
		if ref.Unavailable {
			return nil, ErrChatCapability
		}
		id, err := uuid.Parse(ref.ID)
		if err != nil {
			return nil, ErrChatInvalid
		}
		file, err := s.repo.GetAttachment(ctx, userID, id)
		if err != nil {
			return nil, err
		}
		if file.IsImage && !prepared.caps.ImageInput {
			return nil, ErrChatCapability
		}
		if !file.IsImage && !prepared.caps.DocumentInput {
			return nil, ErrChatCapability
		}
		if prepared.caps.NativeFiles && transport == "messages" {
			kind := "document"
			if file.IsImage {
				kind = "image"
			}
			prepared.files[id] = true
			parts = append(parts, map[string]any{"type": kind, "source": map[string]any{"type": "file", "file_id": "sub2api-chat:" + id.String()}})
			continue
		}
		if !file.IsImage {
			return nil, ErrChatCapability
		}
		reader, err := s.storage.OpenFile(userID, id)
		if err != nil {
			return nil, ErrChatNotFound
		}
		raw, err := io.ReadAll(io.LimitReader(reader, ChatMaxFileBytes+1))
		closeErr := reader.Close()
		if err != nil {
			return nil, err
		}
		if closeErr != nil {
			return nil, closeErr
		}
		if int64(len(raw)) != file.Size {
			return nil, ErrChatInvalid
		}
		encoded := base64.StdEncoding.EncodeToString(raw)
		url := "data:" + file.MIME + ";base64," + encoded
		switch transport {
		case "messages":
			parts = append(parts, map[string]any{"type": "image", "source": map[string]any{"type": "base64", "media_type": file.MIME, "data": encoded}})
		case "responses":
			parts = append(parts, map[string]any{"type": "input_image", "image_url": url})
		case "gemini":
			parts = append(parts, map[string]any{"inlineData": map[string]any{"mimeType": file.MIME, "data": encoded}})
		default:
			parts = append(parts, map[string]any{"type": "image_url", "image_url": map[string]any{"url": url}})
		}
	}
	if len(parts) == 0 {
		return message.Content, nil
	}
	if message.Content != "" {
		kind := "text"
		if transport == "responses" {
			kind = "input_text"
		}
		part := map[string]any{"text": message.Content}
		if transport != "gemini" {
			part["type"] = kind
		}
		parts = append(parts, part)
	}
	return parts, nil
}

func (s *ChatService) buildChatRequest(ctx context.Context, userID int64, snapshot *ChatSnapshot, preset *ChatModelPreset, model *ChatCatalogModel, options ChatRunOptions, operation string) (*chatPreparedRequest, error) {
	caps := model.Capabilities
	if caps.Reasoning && options.ReasoningEffort == "" {
		options.ReasoningEffort = preset.reasoningEffort(options.Thinking)
	}
	if caps.Reasoning && !slices.Contains(chatReasoningEfforts(preset.Model, caps), options.ReasoningEffort) {
		return nil, ErrChatCapability
	}
	prepared := &chatPreparedRequest{caps: caps, options: options, files: map[uuid.UUID]bool{}, body: map[string]any{"model": preset.Model, "stream": true}}
	messages := snapshot.VisibleMessages()
	if len(messages) < 2 {
		return nil, ErrChatInvalid
	}
	target, question := messages[len(messages)-1], messages[len(messages)-2]
	if strings.TrimSpace(question.Content) == "" && len(question.Files) == 0 {
		return nil, ErrChatInvalid
	}
	for _, message := range messages {
		if message.Role != "assistant" {
			continue
		}
		for _, raw := range message.Blocks {
			var value any
			if json.Unmarshal(raw, &value) != nil {
				return nil, ErrChatInvalid
			}
			if chatHasProviderState(value) {
				previous, err := s.repo.LastGeneration(ctx, userID, snapshot.ID, message.ID)
				if err != nil || previous == nil || previous.GroupID != preset.GroupID || previous.Model != preset.Model || (!caps.NativeDeepSeekExtensions && caps.Transport != "gemini") {
					return nil, ErrChatProviderContext
				}
				if caps.Transport == "gemini" {
					var recorded ChatRunOptions
					if json.Unmarshal(previous.Options, &recorded) != nil || recorded.Transport != "gemini" {
						return nil, ErrChatProviderContext
					}
				}
			}
		}
	}
	wire := []any{}
	appendUser := func(message ChatMessage) error {
		content, err := s.chatUserContent(ctx, userID, message, prepared)
		if err != nil {
			return err
		}
		wire = append(wire, map[string]any{"role": "user", "content": content})
		return nil
	}
	for i := 0; i+1 < len(messages)-2; i++ {
		user, assistant := messages[i], messages[i+1]
		if user.Role != "user" || assistant.Role != "assistant" || (assistant.Status != "done" && assistant.Status != "stopped") {
			continue
		}
		var content any = chatAssistantText(assistant)
		if caps.Transport == "gemini" {
			content = chatGeminiAssistantParts(assistant)
		} else if caps.NativeDeepSeekExtensions {
			blocks := chatAssistantBlocks(assistant, false, false)
			if len(blocks) == 0 {
				continue
			}
			content = blocks
		} else if content == "" {
			continue
		}
		if err := appendUser(user); err != nil {
			return nil, err
		}
		wire = append(wire, map[string]any{"role": "assistant", "content": content})
		i++
	}
	if caps.NativeDeepSeekExtensions && options.LatestReminder != "" {
		wire = append(wire, map[string]any{"role": "latest_reminder", "content": options.LatestReminder})
	}
	if err := appendUser(question); err != nil {
		return nil, err
	}
	if operation == "continue" {
		if caps.NativeDeepSeekExtensions {
			blocks := chatAssistantBlocks(target, true, options.Thinking || caps.Reasoning)
			if len(blocks) == 0 {
				return nil, ErrChatInvalid
			}
			wire = append(wire, map[string]any{"role": "assistant", "content": blocks})
		} else {
			prefix := chatAssistantText(target)
			if strings.TrimSpace(prefix) == "" {
				return nil, ErrChatInvalid
			}
			var content any = prefix
			if caps.Transport == "gemini" {
				content = chatGeminiAssistantParts(target)
			}
			wire = append(wire, map[string]any{"role": "assistant", "content": content}, map[string]any{"role": "user", "content": "Continue the preceding assistant response without repeating its existing text."})
		}
	}
	switch caps.Transport {
	case "chat_completions":
		prepared.path = "/v1/chat/completions"
		prepared.body["messages"] = wire
		prepared.body["stream_options"] = map[string]any{"include_usage": true}
		prepared.body["max_tokens"] = 8192
	case "responses":
		prepared.path = "/v1/responses"
		prepared.body["input"] = wire
		prepared.body["max_output_tokens"] = 8192
	case "messages":
		prepared.path = "/v1/messages"
		prepared.body["messages"] = wire
		prepared.body["max_tokens"] = 8192
	case "gemini":
		if !IsSafeGeminiModelPathSegment(preset.Model) {
			return nil, ErrChatInvalid
		}
		prepared.path = chatGeminiPrefix + preset.Model + chatGeminiSuffix
		prepared.body = map[string]any{"contents": chatGeminiContents(wire), "generationConfig": map[string]any{"maxOutputTokens": 8192}}
	default:
		return nil, ErrChatCapability
	}
	tools := []any{}
	if caps.NativeDeepSeekExtensions {
		thinking := "disabled"
		if caps.Reasoning {
			thinking = "enabled"
		}
		prepared.body["thinking"] = map[string]any{"type": thinking}
		prepared.body["safety_check"] = map[string]any{"level": "high"}
		prepared.body["title"] = map[string]any{"enabled": true}
		if options.Search && caps.NativeSearch {
			tools = append(tools, map[string]any{"type": "deepseek_web_search", "name": "deepseek_web_search"})
		}
		if caps.NativeReadURL {
			tools = append(tools, map[string]any{"type": "deepseek_read_url", "name": "deepseek_read_url"})
		}
	}
	applyChatReasoning(prepared, preset.Model)
	if options.Search && caps.Search && !caps.NativeSearch {
		tools = append(tools, chatToolDefinition(caps.Transport, "chat_search", "Search the web using this selected channel.", "query"))
	}
	if caps.ReadURL && !caps.NativeReadURL {
		tools = append(tools, chatToolDefinition(caps.Transport, "chat_fetch", "Read a public URL using this selected channel.", "url"))
	}
	if len(tools) > 0 {
		prepared.body["tools"] = tools
	}
	return prepared, nil
}

func chatToolDefinition(transport, name, description, argument string) map[string]any {
	schema := map[string]any{"type": "object", "properties": map[string]any{argument: map[string]any{"type": "string"}}, "required": []string{argument}, "additionalProperties": false}
	switch transport {
	case "messages":
		return map[string]any{"name": name, "description": description, "input_schema": schema}
	case "responses":
		return map[string]any{"type": "function", "name": name, "description": description, "parameters": schema}
	default:
		return map[string]any{"type": "function", "function": map[string]any{"name": name, "description": description, "parameters": schema}}
	}
}

func (s *ChatService) executeChatRequest(ctx context.Context, key *APIKey, run *ChatRunRecord, clientIP string, prepared *chatPreparedRequest, state *chatReplyAccumulator, emit func(map[string]any) error) error {
	before := func(requestID string) error { return s.repo.AddGenerationRequest(ctx, run, requestID) }
	callsExecuted := 0
	for round := range 4 {
		preset, model, _, err := s.models.ResolvePreset(ctx, run.UserID, run.PresetID)
		if err != nil {
			return err
		}
		if preset.GroupID != run.GroupID || preset.Model != run.Model || model.Capabilities.Transport != prepared.caps.Transport {
			return ErrChatProviderContext
		}
		if (prepared.options.Search && !model.Capabilities.Search) || (prepared.caps.NativeDeepSeekExtensions && !model.Capabilities.NativeDeepSeekExtensions) || (len(prepared.files) > 0 && !model.Capabilities.NativeFiles) {
			return ErrChatCapability
		}
		raw, err := json.Marshal(prepared.body)
		if err != nil {
			return err
		}
		requestCtx := ctx
		if prepared.caps.NativeDeepSeekExtensions || len(prepared.files) > 0 {
			requestCtx = context.WithValue(ctx, chatNativeContextKey{}, &chatNativeRequest{service: s, userID: run.UserID, groupID: run.GroupID, extensions: prepared.caps.NativeDeepSeekExtensions, files: prepared.files})
		}
		base := len(state.blocks)
		decoder := newChatStreamDecoder(prepared.caps.Transport, base, emit)
		writer := newChatGatewayWriter(requestCtx, decoder)
		_, dispatchErr := s.dispatchGateway(requestCtx, writer, key, run.UserID, run.GroupID, run.SessionID, clientIP, prepared.path, raw, before)
		streamErr := writer.finish()
		if dispatchErr != nil {
			return dispatchErr
		}
		if streamErr != nil {
			return streamErr
		}
		calls := decoder.calls()
		if len(calls) == 0 {
			return nil
		}
		if round == 3 || callsExecuted+len(calls) > 8 {
			return ErrChatToolLimit
		}
		results := make([]any, 0, len(calls))
		for _, call := range calls {
			result, err := s.executeChatTool(ctx, key, run, clientIP, call, model.Capabilities, prepared.options, emit, len(state.blocks), before)
			if err != nil {
				return err
			}
			results = append(results, result)
			callsExecuted++
		}
		appendChatToolRound(prepared, decoder, calls, results, state.blocks[base:])
	}
	return ErrChatToolLimit
}

type chatToolWriter struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (w *chatToolWriter) Header() http.Header { return w.header }
func (w *chatToolWriter) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
}
func (w *chatToolWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = 200
	}
	if w.body.Len()+len(p) > ChatMaxSessionBytes {
		return 0, ErrChatTooLarge
	}
	return w.body.Write(p)
}

func (s *ChatService) executeChatTool(ctx context.Context, key *APIKey, run *ChatRunRecord, clientIP string, call chatToolCall, caps ChatCapabilities, options ChatRunOptions, emit func(map[string]any) error, index int, before func(string) error) (any, error) {
	if call.ID == "" || len(call.ID) > 512 {
		return nil, ErrChatStream
	}
	var arguments map[string]any
	if json.Unmarshal([]byte(call.Arguments), &arguments) != nil || len(arguments) != 1 {
		return nil, ErrChatInvalid
	}
	toolPath, kind, name := "", "", ""
	input := map[string]any{}
	switch call.Name {
	case "chat_search":
		if !options.Search || !caps.Search || caps.NativeSearch {
			return nil, ErrChatCapability
		}
		toolPath = "/v1/tools/search"
		kind = "search"
		name = "deepseek_web_search"
		query := chatString(arguments["query"])
		input = map[string]any{"action": "search", "query": query, "queries": []string{query}}
	case "chat_fetch":
		if !caps.ReadURL || caps.NativeReadURL {
			return nil, ErrChatCapability
		}
		toolPath = "/v1/tools/fetch"
		kind = "read_link"
		name = "deepseek_read_url"
		input = map[string]any{"action": "read_link", "url": chatString(arguments["url"])}
	default:
		return nil, ErrChatCapability
	}
	raw, err := json.Marshal(arguments)
	if err != nil {
		return nil, err
	}
	toolKind := DeepSeekToolSearch
	if kind == "read_link" {
		toolKind = DeepSeekToolFetch
	}
	if _, err = ValidateDeepSeekToolRequest(toolKind, raw); err != nil {
		return nil, ErrChatInvalid
	}
	use := map[string]any{"type": "server_tool_use", "id": call.ID, "name": name, "input": input, "kind": kind, "caller": map[string]any{"type": "direct"}}
	if err = emit(map[string]any{"type": "content_block_start", "index": index, "content_block": use}); err != nil {
		return nil, err
	}
	if err = emit(map[string]any{"type": "content_block_stop", "index": index}); err != nil {
		return nil, err
	}
	writer := &chatToolWriter{header: http.Header{}}
	if _, err = s.dispatchGateway(ctx, writer, key, run.UserID, run.GroupID, run.SessionID, clientIP, toolPath, raw, before); err != nil {
		return nil, err
	}
	if writer.status != http.StatusOK {
		return nil, ErrChatStream
	}
	var result map[string]any
	if json.Unmarshal(writer.body.Bytes(), &result) != nil || result["error"] != nil {
		return nil, ErrChatStream
	}
	block := map[string]any{"tool_use_id": call.ID, "kind": kind}
	if kind == "search" {
		entries, ok := result["results"].([]any)
		if !ok {
			return nil, ErrChatStream
		}
		content := []any{}
		for _, item := range entries {
			entry := chatObject(item)
			snippet := chatString(entry["snippet"])
			if snippet == "" {
				snippet = chatString(entry["summary"])
			}
			if snippet == "" {
				snippet = chatString(entry["content"])
			}
			content = append(content, map[string]any{"type": "web_search_result", "title": chatString(entry["title"]), "url": chatString(entry["url"]), "snippet": snippet, "site_icon": entry["site_icon"], "site_name": entry["site_name"], "encrypted_content": "", "page_age": entry["date"]})
		}
		block["type"], block["content"] = "web_search_tool_result", content
	} else {
		content, ok := result["content"].(string)
		if !ok {
			return nil, ErrChatStream
		}
		block["type"] = "web_fetch_tool_result"
		block["content"] = map[string]any{"type": "web_fetch_result", "url": arguments["url"], "published_at": nil, "content": map[string]any{"type": "document", "source": map[string]any{"type": "text", "media_type": "text/plain", "data": content}, "title": result["title"]}}
	}
	if err = emit(map[string]any{"type": "content_block_start", "index": index + 1, "content_block": block}); err != nil {
		return nil, err
	}
	if err = emit(map[string]any{"type": "content_block_stop", "index": index + 1}); err != nil {
		return nil, err
	}
	return result, nil
}

func appendChatToolRound(prepared *chatPreparedRequest, decoder *chatStreamDecoder, calls []chatToolCall, results []any, blocks []any) {
	text, thinking := strings.Builder{}, strings.Builder{}
	for _, item := range blocks {
		block := chatObject(item)
		if block["type"] == "text" {
			text.WriteString(chatString(block["text"]))
		}
		if block["type"] == "thinking" {
			thinking.WriteString(chatString(block["thinking"]))
		}
	}
	switch prepared.caps.Transport {
	case "chat_completions":
		messages, _ := prepared.body["messages"].([]any)
		wireCalls := []any{}
		for _, call := range calls {
			wireCalls = append(wireCalls, map[string]any{"id": call.ID, "type": "function", "function": map[string]any{"name": call.Name, "arguments": call.Arguments}})
		}
		assistant := map[string]any{"role": "assistant", "content": text.String(), "tool_calls": wireCalls}
		if thinking.Len() > 0 {
			assistant["reasoning_content"] = thinking.String()
		}
		messages = append(messages, assistant)
		for i, call := range calls {
			raw, _ := json.Marshal(results[i])
			messages = append(messages, map[string]any{"role": "tool", "tool_call_id": call.ID, "content": string(raw)})
		}
		prepared.body["messages"] = messages
	case "messages":
		messages, _ := prepared.body["messages"].([]any)
		content := []any{}
		if text.Len() > 0 {
			content = append(content, map[string]any{"type": "text", "text": text.String()})
		}
		outputs := []any{}
		for i, call := range calls {
			var arguments any
			_ = json.Unmarshal([]byte(call.Arguments), &arguments)
			content = append(content, map[string]any{"type": "tool_use", "id": call.ID, "name": call.Name, "input": arguments})
			raw, _ := json.Marshal(results[i])
			outputs = append(outputs, map[string]any{"type": "tool_result", "tool_use_id": call.ID, "content": string(raw)})
		}
		prepared.body["messages"] = append(messages, map[string]any{"role": "assistant", "content": content}, map[string]any{"role": "user", "content": outputs})
	case "responses":
		input, _ := prepared.body["input"].([]any)
		if len(decoder.responseOutput) > 0 {
			input = append(input, decoder.responseOutput...)
		} else {
			for _, call := range calls {
				input = append(input, map[string]any{"type": "function_call", "call_id": call.ID, "name": call.Name, "arguments": call.Arguments})
			}
		}
		for i, call := range calls {
			raw, _ := json.Marshal(results[i])
			input = append(input, map[string]any{"type": "function_call_output", "call_id": call.ID, "output": string(raw)})
		}
		prepared.body["input"] = input
	}
}
