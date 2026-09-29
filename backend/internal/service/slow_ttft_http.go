package service

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
)

// Only inference services receive this decorator. Probe/balance/test services
// continue to use the original transport. The payload and response are untouched.
type slowTTFTUpstream struct {
	HTTPUpstream
	protection *RateLimitService
}

func WithSlowTTFTUpstream(upstream HTTPUpstream, p *RateLimitService) HTTPUpstream {
	if upstream == nil || p == nil {
		return upstream
	}
	if _, ok := upstream.(*slowTTFTUpstream); ok {
		return upstream
	}
	return &slowTTFTUpstream{upstream, p}
}
func (s *slowTTFTUpstream) Do(req *http.Request, proxy string, id int64, cap int) (*http.Response, error) {
	return s.do(req, id, func() (*http.Response, error) { return s.HTTPUpstream.Do(req, proxy, id, cap) })
}
func (s *slowTTFTUpstream) DoWithTLS(req *http.Request, proxy string, id int64, cap int, p *tlsfingerprint.Profile) (*http.Response, error) {
	return s.do(req, id, func() (*http.Response, error) { return s.HTTPUpstream.DoWithTLS(req, proxy, id, cap, p) })
}
func slowTTFTStreamingRequest(req *http.Request) bool {
	if req == nil || req.URL == nil || req.Method != http.MethodPost {
		return false
	}
	path := strings.ToLower(req.URL.Path)
	for _, v := range []string{"/images", "/videos", "/audio", "/embeddings", "counttokens", "count_tokens", "/batches", "/compact"} {
		if strings.Contains(path, v) {
			return false
		}
	}
	stream := strings.Contains(path, "streamgeneratecontent") || strings.Contains(path, "invoke-with-response-stream")
	if req.GetBody == nil {
		return stream
	}
	body, err := req.GetBody()
	if err != nil {
		return false
	}
	defer body.Close()
	var payload map[string]json.RawMessage
	if json.NewDecoder(io.LimitReader(body, 32<<20)).Decode(&payload) != nil {
		return false
	}
	if bytes.Equal(bytes.TrimSpace(payload["stream"]), []byte("true")) {
		stream = true
	}
	// Native image output and Responses image-generation tools are not dialogue.
	if bytes.Contains(payload["tools"], []byte("image_generation")) || bytes.Contains(payload["generationConfig"], []byte("IMAGE")) {
		return false
	}
	return stream
}
func (s *slowTTFTUpstream) do(req *http.Request, id int64, call func() (*http.Response, error)) (*http.Response, error) {
	if !slowTTFTStreamingRequest(req) || s.protection.accountRepo == nil {
		return call()
	}
	a, err := s.protection.accountRepo.GetByID(req.Context(), id)
	if err != nil || a == nil {
		return call()
	}
	o := s.protection.BeginSlowTTFT(req.Context(), a)
	resp, err := call()
	if o == nil {
		return resp, err
	}
	if err != nil || resp == nil || resp.StatusCode < 200 || resp.StatusCode >= 300 || resp.Body == nil {
		o.Close()
		return resp, err
	}
	resp.Body = &slowTTFTBody{ReadCloser: resp.Body, observer: o, binary: strings.Contains(req.URL.Path, "invoke-with-response-stream")}
	return resp, nil
}

type slowTTFTBody struct {
	io.ReadCloser
	observer *SlowTTFTObserver
	buffer   []byte
	finished atomic.Bool
	binary   bool
}

func (b *slowTTFTBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if !b.binary && !b.finished.Load() && n > 0 {
		b.buffer = append(b.buffer, p[:n]...)
		for {
			i := bytes.IndexByte(b.buffer, '\n')
			if i < 0 {
				break
			}
			line := bytes.TrimSpace(b.buffer[:i])
			b.buffer = b.buffer[i+1:]
			if bytes.HasPrefix(line, []byte("data:")) {
				line = bytes.TrimSpace(line[5:])
			}
			if SlowTTFTMeaningfulOutput(line) {
				b.finished.Store(true)
				b.observer.FirstOutput()
				b.buffer = nil
				break
			}
		}
		// Bound memory for malformed/unframed streams. Never alter forwarded bytes.
		if len(b.buffer) > 4<<20 {
			b.buffer = nil
		}
	}
	if err != nil {
		b.observer.Close()
	}
	return n, err
}
func (b *slowTTFTBody) Close() error { b.observer.Close(); return b.ReadCloser.Close() }

// A protocol-independent semantic predicate shared with WS and Bedrock hooks.
// Metadata, heartbeat, role-only and image events deliberately do not count.
func SlowTTFTMeaningfulOutput(data []byte) bool {
	var m map[string]json.RawMessage
	if json.Unmarshal(data, &m) != nil {
		return false
	}
	var typ string
	_ = json.Unmarshal(m["type"], &typ)
	nonempty := func(v json.RawMessage) bool { var s string; return json.Unmarshal(v, &s) == nil && s != "" }
	switch typ {
	case "response.output_text.delta", "response.function_call_arguments.delta":
		return nonempty(m["delta"])
	case "content_block_delta":
		var d map[string]json.RawMessage
		_ = json.Unmarshal(m["delta"], &d)
		return nonempty(d["text"]) || nonempty(d["partial_json"])
	case "content_block_start":
		var d map[string]json.RawMessage
		_ = json.Unmarshal(m["content_block"], &d)
		return nonempty(d["text"]) || (string(d["type"]) == `"tool_use"` && nonempty(d["name"]))
	case "response.output_text.done", "response.function_call_arguments.done":
		return nonempty(m["text"]) || nonempty(m["arguments"])
	case "response.output_item.added":
		var item map[string]json.RawMessage
		_ = json.Unmarshal(m["item"], &item)
		return string(item["type"]) == `"function_call"` && nonempty(item["name"])
	}
	var choices []struct {
		Delta struct {
			Content      string `json:"content"`
			FunctionCall struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			} `json:"function_call"`
			ToolCalls []struct {
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"delta"`
	}
	if json.Unmarshal(m["choices"], &choices) == nil {
		for _, c := range choices {
			if c.Delta.Content != "" || c.Delta.FunctionCall.Name != "" || c.Delta.FunctionCall.Arguments != "" {
				return true
			}
			for _, t := range c.Delta.ToolCalls {
				if t.Function.Name != "" || t.Function.Arguments != "" {
					return true
				}
			}
		}
	}
	var candidates []struct {
		Content struct {
			Parts []struct {
				Text         string          `json:"text"`
				Thought      bool            `json:"thought"`
				FunctionCall json.RawMessage `json:"functionCall"`
			} `json:"parts"`
		} `json:"content"`
	}
	if json.Unmarshal(m["candidates"], &candidates) == nil {
		for _, c := range candidates {
			for _, p := range c.Content.Parts {
				if !p.Thought && p.Text != "" || len(p.FunctionCall) > 0 && string(p.FunctionCall) != "null" {
					return true
				}
			}
		}
	}
	// Antigravity wraps the Gemini response in an envelope.
	if len(m["response"]) > 0 && typ == "" {
		return SlowTTFTMeaningfulOutput(m["response"])
	}
	return false
}

func slowTTFTImageTools(tools any) bool {
	data, _ := json.Marshal(tools)
	return bytes.Contains(data, []byte(`"image_generation"`))
}

func (b *slowTTFTBody) ObserveOutput(data []byte) {
	if SlowTTFTMeaningfulOutput(data) {
		b.observer.FirstOutput()
	}
}
