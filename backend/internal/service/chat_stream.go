package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

var ErrChatStream = infraerrors.New(http.StatusBadGateway, "CHAT_STREAM_INTERRUPTED", "生成流异常或未完整结束，已保留可恢复内容")

type chatToolCall struct{ ID, Name, Arguments string }

type chatStreamDecoder struct {
	transport      string
	emit           func(map[string]any) error
	nextIndex      int
	parts          map[string]int
	open           map[int]bool
	tools          map[int]*chatToolCall
	responseTools  map[string]int
	responseOutput []any
	lastKind       string
	lastIndex      int
	done           bool
	finished       bool
	stopReason     string
}

func newChatStreamDecoder(transport string, base int, emit func(map[string]any) error) *chatStreamDecoder {
	return &chatStreamDecoder{transport: transport, emit: emit, nextIndex: base, parts: map[string]int{}, open: map[int]bool{}, tools: map[int]*chatToolCall{}, responseTools: map[string]int{}, lastIndex: -1, stopReason: "end_turn"}
}

func chatString(v any) string         { value, _ := v.(string); return value }
func chatObject(v any) map[string]any { value, _ := v.(map[string]any); return value }
func chatInteger(v any) int           { value, _ := parseSSEUsageInt(v); return value }

func (d *chatStreamDecoder) event(event map[string]any) error { return d.emit(event) }
func (d *chatStreamDecoder) startBlock(key string, block map[string]any) (int, error) {
	if index, ok := d.parts[key]; ok {
		return index, nil
	}
	index := d.nextIndex
	d.nextIndex++
	d.parts[key] = index
	d.open[index] = true
	return index, d.event(map[string]any{"type": "content_block_start", "index": index, "content_block": block})
}
func (d *chatStreamDecoder) closeBlocks() error {
	indexes := make([]int, 0, len(d.open))
	for index := range d.open {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)
	for _, index := range indexes {
		if err := d.event(map[string]any{"type": "content_block_stop", "index": index}); err != nil {
			return err
		}
		delete(d.open, index)
	}
	return nil
}
func (d *chatStreamDecoder) delta(index int, delta map[string]any) error {
	return d.event(map[string]any{"type": "content_block_delta", "index": index, "delta": delta})
}
func (d *chatStreamDecoder) text(kind, key, text string) error {
	if text == "" {
		return nil
	}
	block := map[string]any{"type": kind, kind: ""}
	index, err := d.startBlock(key, block)
	if err != nil {
		return err
	}
	return d.delta(index, map[string]any{"type": kind + "_delta", kind: text})
}
func (d *chatStreamDecoder) sequentialText(kind, text string) error {
	if text == "" {
		return nil
	}
	if kind != d.lastKind {
		if err := d.closeBlocks(); err != nil {
			return err
		}
		d.lastKind = kind
		d.lastIndex = d.nextIndex
	}
	return d.text(kind, fmt.Sprintf("cc:%d", d.lastIndex), text)
}
func chatDisplayUsage(usage map[string]any, completions bool) map[string]any {
	inputKey, outputKey, detailKey := "input_tokens", "output_tokens", "input_tokens_details"
	if completions {
		inputKey, outputKey, detailKey = "prompt_tokens", "completion_tokens", "prompt_tokens_details"
	}
	input, cached := chatInteger(usage[inputKey]), chatInteger(chatObject(usage[detailKey])["cached_tokens"])
	return map[string]any{"input_tokens": max(0, input-cached), "cache_read_input_tokens": max(0, cached), "cache_creation_input_tokens": 0, "output_tokens": max(0, chatInteger(usage[outputKey]))}
}
func (d *chatStreamDecoder) end(usage map[string]any) error {
	if err := d.closeBlocks(); err != nil {
		return err
	}
	event := map[string]any{"type": "message_delta", "delta": map[string]any{"stop_reason": d.stopReason, "stop_sequence": nil}}
	if usage != nil {
		event["usage"] = usage
	}
	d.finished = true
	return d.event(event)
}
func (d *chatStreamDecoder) calls() []chatToolCall {
	indexes := make([]int, 0, len(d.tools))
	for index := range d.tools {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)
	result := make([]chatToolCall, 0, len(indexes))
	for _, index := range indexes {
		result = append(result, *d.tools[index])
	}
	return result
}

func (d *chatStreamDecoder) accept(raw []byte) error {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("[DONE]")) {
		if d.transport != "chat_completions" || !d.finished {
			return ErrChatStream
		}
		d.done = true
		return nil
	}
	var event map[string]any
	if json.Unmarshal(raw, &event) != nil {
		return ErrChatStream
	}
	if event["error"] != nil || event["type"] == "error" {
		return ErrChatStream
	}
	if d.done {
		if event["type"] == "ping" {
			return nil
		}
		return ErrChatStream
	}
	switch d.transport {
	case "messages":
		return d.messages(event)
	case "chat_completions":
		return d.completions(event)
	case "responses":
		return d.responses(event)
	default:
		return ErrChatCapability
	}
}

func (d *chatStreamDecoder) messages(event map[string]any) error {
	kind := chatString(event["type"])
	if kind == "message_stop" {
		d.done = true
		return nil
	}
	switch kind {
	case "content_block_start", "content_block_delta", "content_block_stop":
		index := chatInteger(event["index"])
		if index < 0 || index > 50000 {
			return ErrChatStream
		}
		key := fmt.Sprintf("native:%d", index)
		if kind == "content_block_start" {
			block := chatObject(event["content_block"])
			if block["type"] == "tool_use" {
				if len(d.tools) >= 8 {
					return ErrChatCapability
				}
				toolIndex := len(d.tools)
				d.responseTools[key] = toolIndex
				d.tools[toolIndex] = &chatToolCall{ID: chatString(block["id"]), Name: chatString(block["name"])}
				if input := chatObject(block["input"]); len(input) > 0 {
					raw, _ := json.Marshal(input)
					d.tools[toolIndex].Arguments = string(raw)
				}
				return nil
			}
			_, err := d.startBlock(key, block)
			return err
		}
		if toolIndex, ok := d.responseTools[key]; ok {
			if kind == "content_block_delta" {
				delta := chatObject(event["delta"])
				if delta["type"] != "input_json_delta" {
					return ErrChatStream
				}
				d.tools[toolIndex].Arguments += chatString(delta["partial_json"])
				if len(d.tools[toolIndex].Arguments) > ChatMaxPreferenceBytes {
					return ErrChatTooLarge
				}
			}
			return nil
		}
		mapped, ok := d.parts[key]
		if !ok {
			return ErrChatStream
		}
		event["index"] = mapped
		if kind == "content_block_stop" {
			delete(d.open, mapped)
		}
	case "message_delta":
		d.stopReason = chatString(chatObject(event["delta"])["stop_reason"])
	}
	switch kind {
	case "message_start", "message_delta", "content_block_delta", "content_block_stop", "ping":
		return d.event(event)
	}
	return nil
}

func (d *chatStreamDecoder) completions(event map[string]any) error {
	if usage := chatObject(event["usage"]); usage != nil {
		if err := d.event(map[string]any{"type": "message_delta", "delta": map[string]any{}, "usage": chatDisplayUsage(usage, true)}); err != nil {
			return err
		}
	}
	choices, _ := event["choices"].([]any)
	if len(choices) == 0 {
		return nil
	}
	choice := chatObject(choices[0])
	delta := chatObject(choice["delta"])
	if err := d.sequentialText("thinking", chatString(delta["reasoning_content"])); err != nil {
		return err
	}
	if err := d.sequentialText("text", chatString(delta["content"])); err != nil {
		return err
	}
	if refusal := chatString(delta["refusal"]); refusal != "" {
		d.stopReason = "refusal"
		if err := d.sequentialText("text", refusal); err != nil {
			return err
		}
	}
	calls, _ := delta["tool_calls"].([]any)
	for _, raw := range calls {
		call := chatObject(raw)
		index := chatInteger(call["index"])
		if index < 0 || index >= 8 {
			return ErrChatCapability
		}
		current := d.tools[index]
		if current == nil {
			current = &chatToolCall{}
			d.tools[index] = current
		}
		if id := chatString(call["id"]); id != "" {
			current.ID = id
		}
		fn := chatObject(call["function"])
		current.Name += chatString(fn["name"])
		current.Arguments += chatString(fn["arguments"])
		if len(current.Arguments) > ChatMaxPreferenceBytes {
			return ErrChatTooLarge
		}
	}
	if reason := chatString(choice["finish_reason"]); reason != "" {
		switch reason {
		case "length":
			d.stopReason = "max_tokens"
		case "content_filter":
			d.stopReason = "refusal"
		case "tool_calls":
			d.stopReason = "tool_use"
		}
		return d.end(nil)
	}
	return nil
}

func (d *chatStreamDecoder) responses(event map[string]any) error {
	kind := chatString(event["type"])
	key := fmt.Sprintf("%v:%v", event["item_id"], event["content_index"])
	switch kind {
	case "response.output_text.delta":
		return d.text("text", key, chatString(event["delta"]))
	case "response.reasoning_summary_text.delta", "response.reasoning_text.delta":
		return d.text("thinking", "reasoning:"+key, chatString(event["delta"]))
	case "response.refusal.delta":
		d.stopReason = "refusal"
		return d.text("text", key, chatString(event["delta"]))
	case "response.output_text.annotation.added":
		annotation := chatObject(event["annotation"])
		if annotation["type"] != "url_citation" {
			return nil
		}
		index, err := d.startBlock(key, map[string]any{"type": "text", "text": ""})
		if err != nil {
			return err
		}
		return d.delta(index, map[string]any{"type": "citations_delta", "citation": map[string]any{"type": "web_search_result_location", "url": annotation["url"], "title": annotation["title"], "cited_text": "", "encrypted_index": ""}})
	case "response.output_item.added":
		item := chatObject(event["item"])
		if item["type"] != "function_call" {
			return nil
		}
		index := len(d.tools)
		if index >= 8 {
			return ErrChatCapability
		}
		d.responseTools[chatString(item["id"])] = index
		d.tools[index] = &chatToolCall{ID: chatString(item["call_id"]), Name: chatString(item["name"]), Arguments: chatString(item["arguments"])}
	case "response.function_call_arguments.delta":
		index, ok := d.responseTools[chatString(event["item_id"])]
		if !ok {
			return ErrChatStream
		}
		d.tools[index].Arguments += chatString(event["delta"])
		if len(d.tools[index].Arguments) > ChatMaxPreferenceBytes {
			return ErrChatTooLarge
		}
	case "response.function_call_arguments.done":
		index, ok := d.responseTools[chatString(event["item_id"])]
		if !ok {
			return ErrChatStream
		}
		d.tools[index].Arguments = chatString(event["arguments"])
	case "response.failed", "response.cancelled":
		return ErrChatStream
	case "response.completed", "response.incomplete":
		response := chatObject(event["response"])
		if response["error"] != nil {
			return ErrChatStream
		}
		d.responseOutput, _ = response["output"].([]any)
		if kind == "response.incomplete" {
			d.stopReason = "max_tokens"
			if chatObject(response["incomplete_details"])["reason"] == "content_filter" {
				d.stopReason = "refusal"
			}
		}
		if len(d.tools) > 0 {
			d.stopReason = "tool_use"
		}
		if err := d.end(chatDisplayUsage(chatObject(response["usage"]), false)); err != nil {
			return err
		}
		d.done = true
	}
	return nil
}

// The gateway still owns authentication, scheduling, billing and cancellation.
// This writer only incrementally adapts its bounded response to chat events.
type chatGatewayWriter struct {
	mu         sync.Mutex
	header     http.Header
	status     int
	pending    []byte
	data       []byte
	decoder    *chatStreamDecoder
	err        error
	ctx        context.Context
	closed     chan bool
	stopNotify func() bool
}

func newChatGatewayWriter(ctx context.Context, decoder *chatStreamDecoder) *chatGatewayWriter {
	w := &chatGatewayWriter{header: http.Header{}, decoder: decoder, ctx: ctx, closed: make(chan bool, 1)}
	w.stopNotify = context.AfterFunc(ctx, func() { w.closed <- true })
	return w
}
func (w *chatGatewayWriter) Header() http.Header { return w.header }
func (w *chatGatewayWriter) WriteHeader(status int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.status == 0 {
		w.status = status
	}
}
func (w *chatGatewayWriter) Flush()                   {}
func (w *chatGatewayWriter) CloseNotify() <-chan bool { return w.closed }
func (w *chatGatewayWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.err != nil {
		return 0, w.err
	}
	if err := w.ctx.Err(); err != nil {
		return 0, err
	}
	if w.status == 0 {
		w.status = http.StatusOK
	}
	if w.status < 200 || w.status >= 300 {
		w.err = ErrChatStream
		return 0, w.err
	}
	if len(w.pending)+len(p) > ChatMaxSessionBytes {
		w.err = ErrChatTooLarge
		return 0, w.err
	}
	w.pending = append(w.pending, p...)
	for {
		end := bytes.IndexByte(w.pending, '\n')
		if end < 0 {
			break
		}
		line := bytes.TrimSuffix(w.pending[:end], []byte{'\r'})
		w.pending = w.pending[end+1:]
		if len(line) == 0 {
			if len(w.data) > 0 {
				w.err = w.decoder.accept(w.data)
				w.data = nil
				if w.err != nil {
					return 0, w.err
				}
			}
		} else if bytes.HasPrefix(line, []byte("data:")) {
			value := bytes.TrimPrefix(line[5:], []byte{' '})
			if len(w.data) > 0 {
				w.data = append(w.data, '\n')
			}
			if len(w.data)+len(value) > ChatMaxSessionBytes {
				w.err = ErrChatTooLarge
				return 0, w.err
			}
			w.data = append(w.data, value...)
		}
	}
	return len(p), nil
}
func (w *chatGatewayWriter) finish() error {
	w.stopNotify()
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.err != nil {
		return w.err
	}
	if !w.decoder.done {
		return ErrChatStream
	}
	if strings.TrimSpace(string(w.pending)) != "" || len(w.data) != 0 {
		return ErrChatStream
	}
	return nil
}
