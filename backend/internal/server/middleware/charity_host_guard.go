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
//   - 放行 /health、/api/v1/charity/* 与推理网关协议路径；
//   - 拦截所有主站管理/登录/用户/聊天/支付接口返回 404；
//   - 其余浏览器 GET/HEAD 路径全部交由独立 charity-web 前端，不落入主站 SPA。
func CharityHostGuard(charityService *service.CharityService, serveCharityFrontend gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		isCharityHost := strings.EqualFold(strings.TrimSpace(c.GetHeader("X-Sub2API-Surface")), "charity")
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
		if isCharityAllowedBackendPath(path) {
			c.Next()
			return
		}
		if isCharityBlockedMainSitePath(path) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{
				"error": gin.H{
					"code":    "NOT_FOUND",
					"message": "Not found",
				},
			})
			return
		}

		// 公益域名下其余 GET/HEAD 页面与静态资源由独立公益前端托管。
		if (c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead) && serveCharityFrontend != nil {
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

func isCharityAllowedBackendPath(path string) bool {
	if path == "/health" || path == "/healthz" {
		return true
	}
	allowedPrefixes := []string{
		"/api/v1/charity",
		"/v1/",
		"/v1beta/",
		"/responses",
		"/messages",
		"/chat/completions",
		"/models",
		"/openai/",
		"/antigravity/",
		"/gemini/",
		"/backend-api/",
		"/images/",
		"/videos/",
		"/audio/",
		"/embeddings",
	}
	for _, prefix := range allowedPrefixes {
		if path == strings.TrimSuffix(prefix, "/") || strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func isCharityBlockedMainSitePath(path string) bool {
	blockedPrefixes := []string{
		"/api/",
		"/internal/",
		"/oauth/",
		"/admin",
		"/login",
		"/register",
		"/chat",
		"/image2",
		"/dashboard",
	}
	for _, prefix := range blockedPrefixes {
		if path == strings.TrimSuffix(prefix, "/") || strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func isCharityReadOnlyEndpoint(c *gin.Context) bool {
	if c == nil || c.Request == nil {
		return false
	}
	if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
		return false
	}
	p := strings.TrimRight(c.Request.URL.Path, "/")
	return p == "/v1/models" || p == "/models" || p == "/v1beta/models" || p == "/openai/v1/models"
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
		if onCharityHost && apiKey != nil && !strings.HasPrefix(apiKey.Key, service.CharityKeyPrefix) {
			abortCharityError(c, service.ErrCharityHostKeyNotAllowed, googleStyle)
			return nil, true
		}
		return nil, false
	}

	clientIP := ip.GetSecurityClientIP(c, cfg != nil && cfg.TrustForwardedIPForAPIKeyACL())
	res, err := svc.AdmitRequest(c.Request.Context(), apiKey, clientIP, skipBilling || isCharityReadOnlyEndpoint(c))
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

	if res.Meta != nil {
		ctx := context.WithValue(c.Request.Context(), ctxkey.CharityBilling, res.Meta)
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
