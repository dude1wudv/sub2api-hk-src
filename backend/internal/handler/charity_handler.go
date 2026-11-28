package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// CharityHandler 处理 free.xjtu.shop 公益站公开接口（无需登录）。
type CharityHandler struct {
	charityService *service.CharityService
	cfg            *config.Config
}

// NewCharityHandler 创建公开公益站 handler。
func NewCharityHandler(charityService *service.CharityService, cfg *config.Config) *CharityHandler {
	return &CharityHandler{
		charityService: charityService,
		cfg:            cfg,
	}
}

// GetOverview GET /api/v1/charity/overview
// 返回今日全站共享总池进度、各启用公益分组（含今日公共 Key、独立子上限/共享总池、实时并发、模型与折前单价）及当前 IP 状态。
func (h *CharityHandler) GetOverview(c *gin.Context) {
	if h == nil || h.charityService == nil {
		response.ErrorFrom(c, service.ErrCharityDisabled)
		return
	}
	clientIP := ip.GetSecurityClientIP(c, h.cfg != nil && h.cfg.TrustForwardedIPForAPIKeyACL())
	overview, err := h.charityService.GetPublicOverview(c.Request.Context(), clientIP)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, overview)
}

type charityKeyLookupRequest struct {
	APIKey string `json:"api_key" binding:"required"`
}

// LookupKeyStatus POST /api/v1/charity/key-status
// 允许公益用户粘贴任意公益 Key 查询其分组、是否已轮换以及今日剩余额度。
func (h *CharityHandler) LookupKeyStatus(c *gin.Context) {
	if h == nil || h.charityService == nil {
		response.ErrorFrom(c, service.ErrCharityDisabled)
		return
	}
	var req charityKeyLookupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请输入有效的 API Key")
		return
	}
	result, err := h.charityService.LookupPublicKeyStatus(c.Request.Context(), req.APIKey)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, result)
}
