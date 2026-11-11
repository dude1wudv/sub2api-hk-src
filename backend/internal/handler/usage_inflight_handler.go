package handler

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/gin-gonic/gin"
)

func (h *UsageHandler) Inflight(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	filters, ok := h.parseUserUsageFilters(c, false)
	if !ok {
		return
	}
	page, size := response.ParsePagination(c)
	f := filters.Filters
	f.StartTime = nil
	f.EndTime = nil
	response.Success(c, h.usageService.ListInflight(c.Request.Context(), subject.UserID, f, page, size, false))
}
