package admin

import (
	"context"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"strconv"
)

type groupSchedulingAdmin interface {
	GetGroupScheduling(context.Context, int64) (*service.GroupScheduling, error)
	SaveGroupScheduling(context.Context, int64, *service.GroupScheduling) (*service.GroupScheduling, error)
}

func (h *GroupHandler) GetScheduling(c *gin.Context) {
	id, e := strconv.ParseInt(c.Param("id"), 10, 64)
	if e != nil || id <= 0 {
		response.BadRequest(c, "invalid group ID")
		return
	}
	s, ok := h.adminService.(groupSchedulingAdmin)
	if !ok {
		response.InternalError(c, "scheduling unavailable")
		return
	}
	out, e := s.GetGroupScheduling(c.Request.Context(), id)
	if e != nil {
		response.ErrorFrom(c, e)
		return
	}
	response.Success(c, out)
}
func (h *GroupHandler) SaveScheduling(c *gin.Context) {
	id, e := strconv.ParseInt(c.Param("id"), 10, 64)
	if e != nil || id <= 0 {
		response.BadRequest(c, "invalid group ID")
		return
	}
	var in service.GroupScheduling
	if c.ShouldBindJSON(&in) != nil || in.Version == "" {
		response.BadRequest(c, "invalid scheduling settings")
		return
	}
	s, ok := h.adminService.(groupSchedulingAdmin)
	if !ok {
		response.InternalError(c, "scheduling unavailable")
		return
	}
	out, e := s.SaveGroupScheduling(c.Request.Context(), id, &in)
	if e != nil {
		response.ErrorFrom(c, e)
		return
	}
	response.Success(c, out)
}
func (h *AccountHandler) ClearSlowTTFT(c *gin.Context) {
	id, e := strconv.ParseInt(c.Param("id"), 10, 64)
	if e != nil || id <= 0 {
		response.BadRequest(c, "invalid account ID")
		return
	}
	s, ok := h.adminService.(interface {
		ClearSlowTTFTPause(context.Context, int64) error
	})
	if !ok {
		response.InternalError(c, "protection unavailable")
		return
	}
	if e = s.ClearSlowTTFTPause(c.Request.Context(), id); e != nil {
		response.ErrorFrom(c, e)
		return
	}
	response.Success(c, gin.H{"cleared": true})
}
