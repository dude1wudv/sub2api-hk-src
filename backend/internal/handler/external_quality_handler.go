package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// externalQualityGroup checks membership explicitly: scopeFilter alone only
// annotates a repository filter and does not authorize an individual group.
func (h *ChannelMonitorV2Handler) externalQualityGroup(c *gin.Context) (int64, bool) {
	groups, err := parseChannelMonitorV2GroupIDs(queryList(c, "group_id"))
	if err != nil || len(groups) != 1 {
		response.BadRequest(c, "exactly one group_id is required")
		return 0, false
	}
	groupID := groups[0]
	if !service.ExternalQualityGroupEnabled(groupID) {
		response.ErrorFrom(c, service.ErrExternalQualityNotFound)
		return 0, false
	}
	if channelMonitorV2IsAdmin(c) {
		return groupID, true
	}
	filter := service.ChannelMonitorV2Filter{GroupIDs: groups}
	if !h.scopeFilter(c, &filter, false) {
		return 0, false
	}
	for _, allowed := range filter.AllowedGroupIDs {
		if allowed == groupID {
			return groupID, true
		}
	}
	response.Error(c, http.StatusForbidden, "group access denied")
	return 0, false
}

func (h *ChannelMonitorV2Handler) ExternalQuality(c *gin.Context) {
	groupID, ok := h.externalQualityGroup(c)
	if !ok {
		return
	}
	report, err := h.service.ExternalQuality(c.Request.Context(), groupID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Header("Cache-Control", "private, no-store")
	response.Success(c, report)
}

func (h *ChannelMonitorV2Handler) ExternalQualityArtwork(c *gin.Context) {
	groupID, ok := h.externalQualityGroup(c)
	if !ok {
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid result ID")
		return
	}
	artwork, err := h.service.ExternalQualityArtwork(c.Request.Context(), groupID, id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Header("Cache-Control", "private, no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	response.Success(c, gin.H{"id": id, "group_id": groupID, "response_text": artwork})
}

func (h *ChannelMonitorV2Handler) ImportExternalQuality(c *gin.Context) {
	if !channelMonitorV2IsAdmin(c) {
		response.Error(c, http.StatusForbidden, "admin access required")
		return
	}
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "user not found in context")
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, service.ExternalQualityMaxImportBytes)
	decoder := json.NewDecoder(c.Request.Body)
	var input service.ExternalQualityImport
	err := decoder.Decode(&input)
	if err == nil {
		var trailing any
		if tailErr := decoder.Decode(&trailing); tailErr != io.EOF {
			if tailErr == nil {
				err = errors.New("trailing JSON")
			} else {
				err = tailErr
			}
		}
	}
	if err != nil {
		var oversized *http.MaxBytesError
		if errors.As(err, &oversized) {
			response.Error(c, http.StatusRequestEntityTooLarge, "import exceeds 4 MiB")
			return
		}
		response.BadRequest(c, "invalid quality history JSON")
		return
	}
	result, err := h.service.ImportExternalQuality(c.Request.Context(), input, subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Header("Cache-Control", "private, no-store")
	response.Success(c, result)
}
