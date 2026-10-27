package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// DeepSeekToolKind identifies a native, model-independent DeepSeek tool call.
type DeepSeekToolKind string

const (
	DeepSeekToolSearch DeepSeekToolKind = "search"
	DeepSeekToolFetch  DeepSeekToolKind = "fetch"
)

func (kind DeepSeekToolKind) Endpoint() string {
	switch kind {
	case DeepSeekToolSearch, DeepSeekToolFetch:
		return "/v1/tools/" + string(kind)
	default:
		return ""
	}
}

// Model is a usage-log label, never a model sent to the upstream service.
func (kind DeepSeekToolKind) Model() string {
	if kind.Endpoint() == "" {
		return ""
	}
	return "deepseek-" + string(kind)
}

// ValidateDeepSeekToolRequest validates only the documented fields. The original
// bytes are forwarded unchanged so unknown fields and search operators survive.
// The returned text is suitable for the existing request content audit.
func ValidateDeepSeekToolRequest(kind DeepSeekToolKind, body []byte) (string, error) {
	if kind.Endpoint() == "" {
		return "", errors.New("unsupported DeepSeek tool")
	}
	if !utf8.Valid(body) || !json.Valid(body) {
		return "", errors.New("request body must be a JSON object")
	}
	var request map[string]any
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&request); err != nil || request == nil {
		return "", errors.New("request body must be a JSON object")
	}
	if kind == DeepSeekToolFetch {
		rawURL, ok := request["url"].(string)
		if !ok || utf8.RuneCountInString(rawURL) > 8192 {
			return "", errors.New("'url' must be an HTTP/HTTPS URL of at most 8192 characters")
		}
		parsed, err := url.Parse(rawURL)
		if err != nil || parsed == nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Hostname() == "" || parsed.User != nil {
			return "", errors.New("'url' must be an HTTP/HTTPS URL with a host and no username or password")
		}
		return rawURL, nil
	}

	query, ok := request["query"].(string)
	if !ok || strings.TrimSpace(query) == "" || utf8.RuneCountInString(query) > 128 {
		return "", errors.New("'query' must contain 1 to 128 Unicode characters and cannot be blank")
	}
	if value, exists := request["count"]; exists {
		number, isNumber := value.(json.Number)
		if !isNumber {
			return "", errors.New("'count' must be an integer between 1 and 50")
		}
		count, err := number.Float64()
		if err != nil || count < 1 || count > 50 || math.Trunc(count) != count {
			return "", errors.New("'count' must be an integer between 1 and 50")
		}
	}
	auditText := query
	if value, exists := request["ori_question"]; exists && value != nil {
		question, isString := value.(string)
		if !isString || utf8.RuneCountInString(question) < 1 || utf8.RuneCountInString(question) > 128 {
			return "", errors.New("'ori_question' must contain 1 to 128 Unicode characters or be null")
		}
		auditText += "\n" + question
	}
	if value, exists := request["localization"]; exists {
		if _, isString := value.(string); !isString {
			return "", errors.New("'localization' must be a string")
		}
	}
	if value, exists := request["need_content"]; exists {
		if _, isBool := value.(bool); !isBool {
			return "", errors.New("'need_content' must be a boolean")
		}
	}
	return auditText, nil
}

// ForwardDeepSeekTool sends one native JSON request. Errors are deliberately
// independent of chat-account health and never cause automatic replay/failover.
// A non-nil result means a structurally valid HTTP 200 response; its billable
// call count is zero when Search/Fetch returned no usable result.
func (s *OpenAIGatewayService) ForwardDeepSeekTool(ctx context.Context, c *gin.Context, account *Account, groupID int64, kind DeepSeekToolKind, body []byte) (*OpenAIForwardResult, error) {
	if s == nil || c == nil || account == nil {
		return nil, errors.New("service, context, and account are required")
	}
	identity, exists := c.Get("api_key")
	apiKey, authenticated := identity.(*APIKey)
	if !exists || !authenticated || apiKey == nil || apiKey.GroupID == nil || *apiKey.GroupID != groupID {
		err := errors.New("authenticated API key group does not match the DeepSeek tool request")
		writeDeepSeekToolError(c, http.StatusForbidden, "permission_error", err.Error())
		return nil, err
	}
	if !account.DeepSeekToolsAllowedForGroup(&groupID) {
		err := errors.New("DeepSeek tools are not enabled for this account and group")
		writeDeepSeekToolError(c, http.StatusForbidden, "permission_error", err.Error())
		return nil, err
	}
	if _, err := ValidateDeepSeekToolRequest(kind, body); err != nil {
		writeDeepSeekToolError(c, http.StatusBadRequest, "invalid_request_error", err.Error())
		return nil, err
	}
	// Account slots may wait in a queue. Recheck the current scope and actual
	// group membership after that wait so a revoked permission cannot use an old
	// scheduler snapshot. Keep the admitted credentials/account for usage parity.
	current := account
	if s.accountRepo != nil {
		latest, err := s.accountRepo.GetByID(ctx, account.ID)
		if err != nil || latest == nil || latest.ID != account.ID {
			writeDeepSeekToolError(c, http.StatusServiceUnavailable, "api_error", "Unable to verify current upstream account")
			return nil, errors.New("unable to verify current DeepSeek tools account")
		}
		current = latest
	}
	if !current.DeepSeekToolsAllowedForGroup(&groupID) || !current.IsSchedulable() || !s.openAIAccountMatchesSchedulingGroup(current, &groupID) {
		err := errors.New("DeepSeek tools account permission or availability changed")
		writeDeepSeekToolError(c, http.StatusForbidden, "permission_error", err.Error())
		return nil, err
	}
	base, err := s.validateUpstreamBaseURL(account.GetOpenAIBaseURL())
	if err != nil {
		writeDeepSeekToolError(c, http.StatusBadGateway, "upstream_error", "Invalid upstream configuration")
		return nil, err
	}
	token, _, err := s.GetAccessToken(ctx, account)
	if err != nil {
		writeDeepSeekToolError(c, http.StatusBadGateway, "upstream_error", "Invalid upstream credentials")
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, buildOpenAIEndpointURL(base, kind.Endpoint()), bytes.NewReader(body))
	if err != nil {
		writeDeepSeekToolError(c, http.StatusBadGateway, "upstream_error", "Invalid upstream configuration")
		return nil, err
	}
	request = request.WithContext(WithHTTPUpstreamProfile(request.Context(), HTTPUpstreamProfileOpenAI))
	headers, err := s.buildOpenAIAuthenticationHeaders(ctx, account, token)
	if err != nil {
		writeDeepSeekToolError(c, http.StatusBadGateway, "upstream_error", "Invalid upstream credentials")
		return nil, err
	}
	request.Header = headers
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	account.ApplyHeaderOverrides(request.Header)
	// This is a unary JSON API even when a chat account overrides Accept to SSE.
	deleteHeaderAllForms(request.Header, "accept")
	request.Header.Set("Accept", "application/json")
	SetActualOpenAIUpstreamEndpoint(c, kind.Endpoint())
	requestID := "deepseek_tools:" + uuid.NewString()
	started := time.Now()
	response, err := s.doOpenAIUpstream(request, resolveAccountProxyURL(account), account)
	SetOpsLatencyMs(c, OpsUpstreamLatencyMsKey, time.Since(started).Milliseconds())
	if err != nil {
		return nil, deepSeekToolTransportError(c, err)
	}
	if response == nil || response.Body == nil {
		err := errors.New("upstream returned no response body")
		writeDeepSeekToolError(c, http.StatusBadGateway, "upstream_error", "Malformed upstream response")
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()
	responseBody, err := ReadUpstreamResponseBody(response.Body, s.cfg, c, nil)
	if err != nil {
		return nil, deepSeekToolTransportError(c, err)
	}
	var calls int
	if response.StatusCode == http.StatusOK {
		calls, err = deepSeekToolBillableCalls(kind, responseBody)
		if err != nil {
			setOpsUpstreamError(c, http.StatusBadGateway, "malformed DeepSeek tool response", "")
			writeDeepSeekToolError(c, http.StatusBadGateway, "upstream_error", "Malformed upstream response")
			return nil, err
		}
	}
	writeOpenAIPassthroughResponseHeaders(c.Writer.Header(), response.Header, s.responseHeaderFilter)
	contentType := response.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/json"
	}
	c.Data(response.StatusCode, contentType, responseBody)
	if response.StatusCode != http.StatusOK {
		if response.StatusCode >= http.StatusBadRequest {
			setOpsUpstreamError(c, response.StatusCode, "DeepSeek tool request failed", "")
		}
		return nil, nil
	}
	return &OpenAIForwardResult{
		RequestID:         requestID,
		Model:             kind.Model(),
		DeepSeekToolKind:  kind,
		BillableToolCalls: calls,
		UpstreamEndpoint:  kind.Endpoint(),
		UpstreamHeaders:   response.Header,
		ResponseHeaders:   response.Header,
		Duration:          time.Since(started),
	}, nil
}

func deepSeekToolBillableCalls(kind DeepSeekToolKind, body []byte) (int, error) {
	var response map[string]json.RawMessage
	if !utf8.Valid(body) || json.Unmarshal(body, &response) != nil || response == nil {
		return 0, errors.New("DeepSeek tool response must be a JSON object")
	}
	if kind == DeepSeekToolFetch {
		var content string
		rawContent, exists := response["content"]
		if !exists || string(rawContent) == "null" || json.Unmarshal(rawContent, &content) != nil {
			return 0, errors.New("DeepSeek fetch response content must be a string")
		}
		if title, exists := response["title"]; exists {
			var value string
			if string(title) == "null" || json.Unmarshal(title, &value) != nil {
				return 0, errors.New("DeepSeek fetch response title must be a string")
			}
		}
		if strings.TrimSpace(content) != "" {
			return 1, nil
		}
		return 0, nil
	}
	var results []map[string]json.RawMessage
	rawResults, exists := response["results"]
	if !exists || string(rawResults) == "null" || json.Unmarshal(rawResults, &results) != nil {
		return 0, errors.New("DeepSeek search response results must be an array of objects")
	}
	for _, result := range results {
		if result == nil {
			return 0, errors.New("DeepSeek search response results must be an array of objects")
		}
		for _, field := range []string{"title", "url", "snippet", "summary", "content", "site_name", "site_icon", "date"} {
			value, exists := result[field]
			if !exists {
				continue
			}
			if string(value) == "null" && (field == "site_name" || field == "site_icon" || field == "date") {
				continue
			}
			var text string
			if string(value) == "null" || json.Unmarshal(value, &text) != nil {
				return 0, fmt.Errorf("DeepSeek search response %s must be a string", field)
			}
		}
		for _, field := range []string{"rerank_score", "authority"} {
			if value, exists := result[field]; exists {
				var number float64
				if string(value) == "null" || json.Unmarshal(value, &number) != nil || (field == "authority" && math.Trunc(number) != number) {
					return 0, fmt.Errorf("DeepSeek search response %s must be a number", field)
				}
				if field == "rerank_score" && (number < 0 || number > 3) {
					return 0, errors.New("DeepSeek search response rerank_score is out of range")
				}
				if field == "authority" && (number < -1 || number > 4) {
					return 0, errors.New("DeepSeek search response authority is out of range")
				}
			}
		}
	}
	if len(results) > 0 {
		return 1, nil
	}
	return 0, nil
}

func deepSeekToolTransportError(c *gin.Context, err error) error {
	if errors.Is(err, context.Canceled) {
		return err
	}
	status := http.StatusBadGateway
	message := "Upstream request failed"
	var networkError net.Error
	if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &networkError) && networkError.Timeout()) {
		status = http.StatusGatewayTimeout
		message = "Upstream request timed out"
	}
	setOpsUpstreamError(c, status, message, "")
	writeDeepSeekToolError(c, status, "upstream_error", message)
	return err
}

func writeDeepSeekToolError(c *gin.Context, status int, kind, message string) {
	c.JSON(status, gin.H{"error": gin.H{"message": message, "type": kind, "param": nil, "code": kind}})
}
