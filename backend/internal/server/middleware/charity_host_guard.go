package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

var globalCharityAdmissionService atomic.Pointer[service.CharityService]

// SetCharityAdmissionService 注册公益池准入服务（与 SetIngressRejectRecorder 同模式）。
func SetCharityAdmissionService(svc *service.CharityService) {
	globalCharityAdmissionService.Store(svc)
}

func getCharityAdmissionService() *service.CharityService {
	return globalCharityAdmissionService.Load()
}

// IsCharityHostRequest 判断当前请求是否命中公益池独立域名（如 free.xjtu.shop）。
func IsCharityHostRequest(c *gin.Context) bool {
	if c == nil || c.Request == nil {
		return false
	}
	v, ok := c.Request.Context().Value(ctxkey.CharityHost).(bool)
	return ok && v
}

// CharityHostGuard 负责公益独立域名（free.xjtu.shop）的路由隔离与独立前端托管：
//   - 放行健康检查、公开公益 API 和支持的推理路径；
//   - 首页、文档与静态资源交由独立前端托管；
//   - 其余路径返回 404。
func CharityHostGuard(charityService *service.CharityService, serveCharityFrontend gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		isCharityHost := service.NormalizeCharityHost(c.Request.Host) == service.CharityDefaultHost
		if !isCharityHost && charityService != nil {
			isCharityHost = charityService.IsCharityHost(c.Request.Context(), c.Request.Host)
		}
		if !isCharityHost {
			c.Next()
			return
		}

		ctx := context.WithValue(c.Request.Context(), ctxkey.CharityHost, true)
		c.Request = c.Request.WithContext(ctx)

		path := c.Request.URL.Path
		switch path {
		case "/health", "/healthz", "/api/v1/charity/overview", "/api/v1/charity/key-status":
			c.Next()
			return
		}
		if isCharityGatewayRequest(c.Request) {
			c.Next()
			return
		}
		if (c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead) && serveCharityFrontend != nil &&
			(path == "/" || path == "/index.html" || path == "/docs" || strings.HasPrefix(path, "/docs/") || path == "/favicon.svg" || path == "/favicon.ico" || strings.HasPrefix(path, "/assets/")) {
			serveCharityFrontend(c)
			c.Abort()
			return
		}

		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"code":    "NOT_FOUND",
				"message": "Not found",
			},
		})
	}
}

func isCharityModelDiscovery(req *http.Request) bool {
	if req.Method != http.MethodGet && req.Method != http.MethodHead {
		return false
	}
	switch strings.TrimRight(req.URL.Path, "/") {
	case "/models", "/v1/models", "/v1beta/models", "/antigravity/models", "/antigravity/v1/models", "/antigravity/v1beta/models", "/backend-api/codex/models":
		return true
	}
	for _, prefix := range []string{"/models/", "/v1/models/", "/v1beta/models/", "/antigravity/v1beta/models/"} {
		if model, ok := strings.CutPrefix(req.URL.Path, prefix); ok {
			return model != "" && !strings.ContainsAny(model, "/:")
		}
	}
	return false
}

func isCharityGatewayRequest(req *http.Request) bool {
	if isCharityModelDiscovery(req) {
		return true
	}
	if req.Method != http.MethodPost {
		return false
	}
	path := strings.TrimRight(req.URL.Path, "/")
	switch path {
	case "/chat/completions", "/v1/chat/completions", "/responses", "/v1/responses", "/backend-api/codex/responses",
		"/v1/messages", "/messages/count_tokens", "/v1/messages/count_tokens", "/anthropic/v1/messages", "/anthropic/v1/messages/count_tokens",
		"/antigravity/v1/messages", "/antigravity/v1/messages/count_tokens", "/embeddings", "/v1/embeddings",
		"/images/generations", "/v1/images/generations", "/images/edits", "/v1/images/edits":
		return true
	}
	for _, prefix := range []string{"/v1beta/models/", "/antigravity/v1beta/models/"} {
		if rest, ok := strings.CutPrefix(path, prefix); ok {
			model, action, found := strings.Cut(rest, ":")
			return found && model != "" && !strings.Contains(model, "/") && (action == "generateContent" || action == "streamGenerateContent" || action == "countTokens")
		}
	}
	return false
}

// checkCharityAPIKeyAdmission 在网关 API Key 鉴权中间件中处理公益 Key 准入与公益域名边界。
// 返回 (res, aborted)：
//   - aborted=true 表示已直接写出错误响应并中止请求；
//   - res != nil && res.IsCharity 表示已通过公益准入，调用方应跳过余额/订阅检查并在请求结束时调用 res.Release()。
func checkCharityAPIKeyAdmission(
	c *gin.Context,
	apiKey *service.APIKey,
	cfg *config.Config,
	skipBilling bool,
	googleStyle bool,
) (*service.CharityAdmissionResult, bool) {
	svc := getCharityAdmissionService()
	onCharityHost := IsCharityHostRequest(c)

	if svc == nil {
		if apiKey != nil && strings.HasPrefix(apiKey.Key, service.CharityKeyPrefix) {
			abortCharityError(c, service.ErrBillingServiceUnavailable, googleStyle)
			return nil, true
		}
		if onCharityHost {
			abortCharityError(c, service.ErrCharityHostKeyNotAllowed, googleStyle)
			return nil, true
		}
		return nil, false
	}

	clientIP := ip.GetSecurityClientIP(c, cfg != nil && cfg.TrustForwardedIPForAPIKeyACL())
	supported := isCharityGatewayRequest(c.Request)
	res, err := svc.AdmitRequest(c.Request.Context(), apiKey, clientIP, !supported || skipBilling || isCharityModelDiscovery(c.Request))
	if err != nil {
		abortCharityError(c, err, googleStyle)
		return nil, true
	}
	if res == nil || !res.IsCharity {
		if onCharityHost {
			abortCharityError(c, service.ErrCharityHostKeyNotAllowed, googleStyle)
			return nil, true
		}
		return nil, false
	}
	if !supported {
		if res.Release != nil {
			res.Release()
		}
		abortCharityError(c, infraerrors.NotFound("CHARITY_ENDPOINT_UNAVAILABLE", "该接口暂未对公益 Key 开放"), googleStyle)
		return nil, true
	}

	if res.Meta != nil {
		meta := *res.Meta
		meta.APIKeyID = apiKey.ID
		meta.UserID = apiKey.UserID
		ctx := context.WithValue(c.Request.Context(), ctxkey.CharityBilling, &meta)
		c.Request = c.Request.WithContext(ctx)
	}
	return res, false
}

func abortCharityError(c *gin.Context, err error, googleStyle bool) {
	status := http.StatusTooManyRequests
	code := "CHARITY_LIMIT_EXCEEDED"
	msg := err.Error()

	var appErr *infraerrors.ApplicationError
	if errors.As(err, &appErr) {
		if appErr.Code > 0 {
			status = int(appErr.Code)
		}
		if appErr.Reason != "" {
			code = appErr.Reason
		}
		if appErr.Message != "" {
			msg = appErr.Message
		}
	}

	if googleStyle {
		abortWithGoogleError(c, status, msg)
		return
	}
	if status == http.StatusTooManyRequests && isOpenAICompatibleAPIKeyRequest(c) {
		abortWithOpenAIQuotaError(c, status, msg)
		return
	}
	AbortWithError(c, status, code, msg)
}
