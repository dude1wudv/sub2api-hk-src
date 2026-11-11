package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/adminaccess"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/gin-gonic/gin"
	"strconv"
	"strings"
)

func (h *UsageHandler) Inflight(c *gin.Context) {
	if adminaccess.RestrictedFromContext(c.Request.Context()) != nil {
		response.Forbidden(c, "In-flight access unavailable to restricted administrator keys")
		return
	}
	var f usagestats.UsageLogFilters
	values := []struct {
		name   string
		target *int64
	}{{"user_id", &f.UserID}, {"api_key_id", &f.APIKeyID}, {"account_id", &f.AccountID}, {"group_id", &f.GroupID}}
	for _, v := range values {
		if raw := strings.TrimSpace(c.Query(v.name)); raw != "" {
			id, err := strconv.ParseInt(raw, 10, 64)
			if err != nil || id < 0 {
				response.BadRequest(c, "Invalid "+v.name)
				return
			}
			*v.target = id
		}
	}
	f.Model = strings.TrimSpace(c.Query("model"))
	f.RequestID = strings.TrimSpace(c.Query("request_id"))
	page, size := response.ParsePagination(c)
	response.Success(c, h.usageService.ListInflight(c.Request.Context(), f.UserID, f, page, size, true))
}

func (h *UsageHandler) UserInflight(c *gin.Context) {
	if adminaccess.RestrictedFromContext(c.Request.Context()) != nil {
		response.Forbidden(c, "User in-flight access unavailable to restricted administrator keys")
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid user ID")
		return
	}
	items, known := h.usageService.UserInflightGroupLoad(c.Request.Context(), id)
	response.Success(c, gin.H{"items": items, "runtime_known": known})
}
