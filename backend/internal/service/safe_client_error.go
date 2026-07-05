package service

import (
	"net/http"
	"strings"
)

type ClientSafeError struct {
	Status  int
	ErrType string
	Code    string
	Message string
}

func (e ClientSafeError) MessageWithCode() string {
	if e.Code == "" {
		return e.Message
	}
	return e.Code + ": " + e.Message
}

func SafeUpstreamClientError(upstreamStatus int) ClientSafeError {
	switch upstreamStatus {
	case http.StatusBadRequest, http.StatusNotFound:
		return ClientSafeError{Status: http.StatusBadRequest, ErrType: "invalid_request_error", Code: "SUNM_BAD_REQUEST", Message: "Request could not be processed"}
	case http.StatusTooManyRequests:
		return ClientSafeError{Status: http.StatusTooManyRequests, ErrType: "rate_limit_error", Code: "SUNM_RATE_LIMITED", Message: "Service is busy. Please retry later"}
	case 529:
		return ClientSafeError{Status: http.StatusServiceUnavailable, ErrType: "server_error", Code: "SUNM_SERVICE_UNAVAILABLE", Message: "Service temporarily unavailable"}
	case http.StatusUnauthorized, http.StatusPaymentRequired, http.StatusForbidden,
		http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return ClientSafeError{Status: http.StatusBadGateway, ErrType: "server_error", Code: "SUNM_SERVICE_UNAVAILABLE", Message: "Service temporarily unavailable"}
	default:
		return ClientSafeError{Status: http.StatusBadGateway, ErrType: "server_error", Code: "SUNM_REQUEST_FAILED", Message: "Request failed"}
	}
}

func RedactUpstreamClientError(status int, errType, message string) (int, string, string) {
	if isOpenAIContextWindowError(message, nil) {
		return status, errType, message
	}
	lower := strings.ToLower(errType + " " + message)
	if strings.Contains(lower, "upstream") || strings.Contains(lower, "all available accounts exhausted") {
		safe := SafeUpstreamClientError(status)
		return safe.Status, safe.ErrType, safe.MessageWithCode()
	}
	return status, errType, message
}

func RedactUpstreamClientMessage(status int, message string) (int, string) {
	status, _, message = RedactUpstreamClientError(status, "", message)
	return status, message
}

func RedactUpstreamClientCode(status int, code, message string) (int, string, string) {
	lower := strings.ToLower(code + " " + message)
	if strings.Contains(lower, "upstream") || strings.Contains(lower, "all available accounts exhausted") {
		safe := SafeUpstreamClientError(status)
		return safe.Status, safe.Code, safe.MessageWithCode()
	}
	return status, code, message
}
