package admin

import (
	"context"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

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
