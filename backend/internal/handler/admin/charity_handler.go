package admin

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// CharityHandler 管理后台公益池配置、分组与公共 Key 轮换接口。
type CharityHandler struct {
	charityService *service.CharityService
}

// NewCharityHandler 创建管理端公益池 handler。
func NewCharityHandler(charityService *service.CharityService) *CharityHandler {
	return &CharityHandler{charityService: charityService}
}

// GetOverview GET /api/v1/admin/charity/overview
func (h *CharityHandler) GetOverview(c *gin.Context) {
	overview, err := h.charityService.GetAdminOverview(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, overview)
}

// UpdateSettings PUT /api/v1/admin/charity/settings
func (h *CharityHandler) UpdateSettings(c *gin.Context) {
	var req service.UpdateCharitySettingsInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的公益池配置参数: "+err.Error())
		return
	}
	updated, err := h.charityService.UpdateSettings(c.Request.Context(), &req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, updated)
}

// AddGroup POST /api/v1/admin/charity/groups
func (h *CharityHandler) AddGroup(c *gin.Context) {
	var req service.CreateCharityGroupInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的公益分组参数: "+err.Error())
		return
	}
	created, err := h.charityService.AddGroup(c.Request.Context(), &req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, created)
}

// UpdateGroup PUT /api/v1/admin/charity/groups/:id
func (h *CharityHandler) UpdateGroup(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "无效的公益分组 ID")
		return
	}
	var req service.UpdateCharityGroupInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的公益分组更新参数: "+err.Error())
		return
	}
	updated, err := h.charityService.UpdateGroup(c.Request.Context(), id, &req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, updated)
}

// RemoveGroup DELETE /api/v1/admin/charity/groups/:id
func (h *CharityHandler) RemoveGroup(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "无效的公益分组 ID")
		return
	}
	if err := h.charityService.RemoveGroup(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

// RotateGroupKey POST /api/v1/admin/charity/groups/:id/rotate-key
func (h *CharityHandler) RotateGroupKey(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "无效的公益分组 ID")
		return
	}
	rotated, err := h.charityService.RotateGroupKey(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, rotated)
}

// RotateAllKeys POST /api/v1/admin/charity/rotate-all
func (h *CharityHandler) RotateAllKeys(c *gin.Context) {
	groups, err := h.charityService.RotateAllKeys(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"groups": groups})
}
