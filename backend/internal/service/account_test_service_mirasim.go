package service

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func (s *AccountTestService) testMirasimAccountConnection(c *gin.Context, account *Account, modelID, prompt string) error {
	if s.openaiGatewayService == nil {
		return s.sendErrorAndEnd(c, "Mirasim gateway is unavailable")
	}
	model := strings.TrimSpace(modelID)
	if model == "" {
		model = "glm-5.3-flash"
	}
	model = account.GetMappedModel(model)
	if prompt == "" {
		prompt = "Hello"
	}
	path := "/v1/chat/completions"
	payload := map[string]any{"model": model, "stream": true, "messages": []any{map[string]any{"role": "user", "content": prompt}}}
	if strings.HasPrefix(model, "claude-") {
		path = "/v1/messages"
		payload["max_tokens"] = 32
	} else if strings.HasPrefix(model, "gpt-") {
		path = "/v1/responses"
		payload = map[string]any{"model": model, "stream": true, "input": prompt}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return s.sendErrorAndEnd(c, "Mirasim test request is invalid")
	}
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, "https://relay.mirasim.ai"+path, bytes.NewReader(body))
	if err != nil {
		return s.sendErrorAndEnd(c, "Mirasim test request could not be created")
	}
	proxyURL := ""
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	signed, err := s.openaiGatewayService.signMirasimRequest(req, proxyURL, account)
	if err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Mirasim signing failed: %v", err))
	}
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	s.sendEvent(c, TestEvent{Type: "test_start", Model: model})
	resp, err := s.httpUpstream.Do(signed, proxyURL, account.ID, account.Concurrency)
	if err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Mirasim request failed: %v", err))
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return s.sendErrorAndEnd(c, fmt.Sprintf("Mirasim returned HTTP %d: %s", resp.StatusCode, string(body)))
	}
	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	if contentType != "" && !strings.Contains(contentType, "text/event-stream") {
		return s.processMirasimTestJSON(c, resp.Body)
	}
	return s.processMirasimTestStream(c, resp.Body)
}

func (s *AccountTestService) processMirasimTestJSON(c *gin.Context, body io.Reader) error {
	data, err := io.ReadAll(io.LimitReader(body, 1<<20+1))
	if err != nil || len(data) > 1<<20 {
		return s.sendErrorAndEnd(c, "Mirasim response could not be read")
	}
	var response map[string]any
	if err := json.Unmarshal(data, &response); err != nil {
		return s.sendErrorAndEnd(c, "Mirasim returned an invalid response")
	}
	if detail, ok := response["error"].(map[string]any); ok {
		message, _ := detail["message"].(string)
		if message == "" {
			message = "Mirasim returned an error"
		}
		return s.sendErrorAndEnd(c, message)
	}
	var text strings.Builder
	if choices, ok := response["choices"].([]any); ok {
		for _, item := range choices {
			if choice, ok := item.(map[string]any); ok {
				if message, ok := choice["message"].(map[string]any); ok {
					if value, ok := message["content"].(string); ok {
						text.WriteString(value)
					}
				}
			}
		}
	}
	for _, field := range []string{"content", "output"} {
		if items, ok := response[field].([]any); ok {
			for _, item := range items {
				if part, ok := item.(map[string]any); ok {
					if value, ok := part["text"].(string); ok {
						text.WriteString(value)
					}
					if nested, ok := part["content"].([]any); ok {
						for _, nestedItem := range nested {
							if nestedPart, ok := nestedItem.(map[string]any); ok {
								if value, ok := nestedPart["text"].(string); ok {
									text.WriteString(value)
								}
							}
						}
					}
				}
			}
		}
	}
	if strings.TrimSpace(text.String()) == "" {
		return s.sendErrorAndEnd(c, "Mirasim response contained no text output")
	}
	s.sendEvent(c, TestEvent{Type: "content", Text: text.String()})
	s.sendEvent(c, TestEvent{Type: "test_complete", Success: true})
	return nil
}

// A relay can send message_start or keepalive frames before any model output.
// Only report success after text and a terminal event have both arrived.
func (s *AccountTestService) processMirasimTestStream(c *gin.Context, body io.Reader) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 16*1024), 1<<20)
	var dataLines []string
	eventName := ""
	seenText := false
	seenComplete := false
	processEvent := func() error {
		if len(dataLines) == 0 {
			eventName = ""
			return nil
		}
		raw := strings.Join(dataLines, "\n")
		dataLines = nil
		defer func() { eventName = "" }()
		if raw == "[DONE]" {
			seenComplete = true
			return nil
		}
		var data map[string]any
		if err := json.Unmarshal([]byte(raw), &data); err != nil {
			return fmt.Errorf("invalid Mirasim stream event: %w", err)
		}
		typ, _ := data["type"].(string)
		if typ == "" {
			typ = eventName
		}
		if typ == "error" || typ == "response.failed" || data["error"] != nil {
			message := "Mirasim returned a stream error"
			if detail, ok := data["error"].(map[string]any); ok {
				if value, ok := detail["message"].(string); ok && value != "" {
					message = value
				}
			} else if value, ok := data["error"].(string); ok && value != "" {
				message = value
			}
			return fmt.Errorf("%s", message)
		}
		text := ""
		switch typ {
		case "content_block_delta":
			if delta, ok := data["delta"].(map[string]any); ok {
				text, _ = delta["text"].(string)
			}
		case "content_block_start":
			if block, ok := data["content_block"].(map[string]any); ok {
				text, _ = block["text"].(string)
			}
		case "response.output_text.delta":
			text, _ = data["delta"].(string)
		case "response.output_text.done":
			// The delta events already carried this text; avoid showing it twice.
		case "response.completed", "response.done", "message_stop":
			seenComplete = true
		}
		if choices, ok := data["choices"].([]any); ok {
			for _, item := range choices {
				choice, ok := item.(map[string]any)
				if !ok {
					continue
				}
				if delta, ok := choice["delta"].(map[string]any); ok {
					if value, ok := delta["content"].(string); ok {
						text += value
					}
				}
				if message, ok := choice["message"].(map[string]any); ok {
					if value, ok := message["content"].(string); ok {
						text += value
					}
				}
				if choice["finish_reason"] != nil {
					seenComplete = true
				}
			}
		}
		if text != "" {
			s.sendEvent(c, TestEvent{Type: "content", Text: text})
			seenText = seenText || strings.TrimSpace(text) != ""
		}
		return nil
	}
	for scanner.Scan() {
		line := strings.TrimSuffix(scanner.Text(), "\r")
		switch {
		case line == "":
			if err := processEvent(); err != nil {
				return s.sendErrorAndEnd(c, err.Error())
			}
			if seenComplete {
				break
			}
		case strings.HasPrefix(line, "data:"):
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		case strings.HasPrefix(line, "event:"):
			eventName = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		}
		if seenComplete {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Mirasim stream failed: %v", err))
	}
	if !seenComplete && len(dataLines) > 0 {
		if err := processEvent(); err != nil {
			return s.sendErrorAndEnd(c, err.Error())
		}
	}
	if !seenText {
		return s.sendErrorAndEnd(c, "Mirasim stream completed without text output")
	}
	if !seenComplete {
		return s.sendErrorAndEnd(c, "Mirasim stream ended before completion")
	}
	s.sendEvent(c, TestEvent{Type: "test_complete", Success: true})
	return nil
}
