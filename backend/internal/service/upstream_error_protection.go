package service

import (
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/gin-gonic/gin"
)

const upstreamErrorProtectionContextKey = "upstream_error_protection_enabled"

// SetUpstreamErrorProtectionEnabled binds the route-level protection policy.
func SetUpstreamErrorProtectionEnabled(c *gin.Context, enabled bool) {
	if c != nil {
		c.Set(upstreamErrorProtectionContextKey, enabled)
	}
}

func IsUpstreamErrorProtectionEnabled(c *gin.Context) bool {
	if c == nil {
		return false
	}
	v, ok := c.Get(upstreamErrorProtectionContextKey)
	if !ok {
		return false
	}
	enabled, ok := v.(bool)
	return ok && enabled
}

// ProtectedUpstreamError maps provider status codes to the server's stable
// client-facing contract. It intentionally never includes provider text.
func ProtectedUpstreamError(c *gin.Context, upstreamStatus int) (int, string, string) {
	if upstreamStatus == http.StatusRequestEntityTooLarge {
		return http.StatusRequestEntityTooLarge, "invalid_request_error", "Request rejected by this server / 请求被本服务器拒绝"
	}
	switch upstreamStatus {
	case http.StatusBadRequest, http.StatusNotFound, http.StatusConflict, http.StatusUnprocessableEntity:
		return http.StatusBadRequest, "invalid_request_error", "Request rejected by this server / 请求被本服务器拒绝"
	case http.StatusUnauthorized, http.StatusForbidden:
		return http.StatusBadGateway, "api_error", "Server request failed / 本服务器请求失败"
	case http.StatusTooManyRequests:
		return http.StatusTooManyRequests, "rate_limit_error", "Request rate limited by this server, please retry later / 请求已被本服务器限流，请稍后重试"
	case http.StatusRequestTimeout, http.StatusGatewayTimeout:
		return http.StatusGatewayTimeout, "api_error", "Server request timed out / 本服务器请求超时"
	case http.StatusServiceUnavailable, 529:
		return http.StatusServiceUnavailable, "overloaded_error", "Service temporarily unavailable, please retry later / 服务暂时不可用，请稍后重试"
	default:
		return http.StatusBadGateway, "api_error", "Server request failed / 本服务器请求失败"
	}
}

// WriteProtectedUpstreamError writes a response that contains only local data.
func WriteProtectedUpstreamError(c *gin.Context, upstreamStatus int) {
	status, typ, msg := ProtectedUpstreamError(c, upstreamStatus)
	requestID := ""
	if c != nil && c.Request != nil {
		requestID, _ = c.Request.Context().Value(ctxkey.ClientRequestID).(string)
	}
	if strings.TrimSpace(requestID) == "" && c != nil && c.Request != nil {
		requestID = strings.TrimSpace(c.GetHeader("X-Client-Request-ID"))
	}
	if c != nil {
		c.Header("Content-Type", "application/json; charset=utf-8")
		c.Header("X-Client-Request-ID", requestID)
		c.JSON(status, gin.H{"error": gin.H{"type": typ, "message": msg, "request_id": requestID}})
	}
}
