package service

import (
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
		model = "claude-sonnet-5"
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
	// A successful HTTP handshake alone is insufficient for SSE: read the first
	// actual data event so a delayed relay error is visible in the account test.
	buffer := make([]byte, 4096)
	n, readErr := resp.Body.Read(buffer)
	if readErr != nil && readErr != io.EOF {
		return s.sendErrorAndEnd(c, fmt.Sprintf("Mirasim stream failed: %v", readErr))
	}
	if n == 0 || bytes.Contains(buffer[:n], []byte(`"type":"error"`)) {
		return s.sendErrorAndEnd(c, "Mirasim stream did not return a usable response")
	}
	s.sendEvent(c, TestEvent{Type: "test_complete", Success: true})
	return nil
}
