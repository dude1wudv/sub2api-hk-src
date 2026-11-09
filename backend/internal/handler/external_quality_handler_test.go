package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestExternalQualityRejectsUnavailableGroupBeforeAccess(t *testing.T) {
	for _, artwork := range []bool{false, true} {
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(http.MethodGet, "/?group_id=72", nil)
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 42})
		h := &ChannelMonitorV2Handler{apiKeyService: &channelMonitorV2GroupAuthorizerStub{groups: []service.Group{{ID: 237}}}}
		if artwork {
			h.ExternalQualityArtwork(c)
		} else {
			h.ExternalQuality(c)
		}
		require.Equal(t, http.StatusForbidden, recorder.Code)
	}
}

func TestExternalQualityAuthorizationIncludesMembershipAndAdmin(t *testing.T) {
	for _, admin := range []bool{false, true} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodGet, "/?group_id=72", nil)
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 42})
		if admin {
			c.Set(string(middleware.ContextKeyUserRole), service.RoleAdmin)
		}
		auth := &channelMonitorV2GroupAuthorizerStub{groups: []service.Group{{ID: 72}}}
		h := &ChannelMonitorV2Handler{apiKeyService: auth}
		group, ok := h.externalQualityGroup(c)
		require.True(t, ok)
		require.Equal(t, int64(72), group)
		if admin {
			require.Empty(t, auth.calls)
		} else {
			require.Equal(t, []int64{42}, auth.calls)
		}
	}
}

func TestExternalQualityImportRoleAndBodyLimit(t *testing.T) {
	for _, test := range []struct {
		name, body string
		admin      bool
		status     int
	}{
		{"ordinary", "{}", false, http.StatusForbidden},
		{"trailing", "{} {}", true, http.StatusBadRequest},
		{"oversized", `{"ignored":"` + strings.Repeat("x", int(service.ExternalQualityMaxImportBytes)) + `"}`, true, http.StatusRequestEntityTooLarge},
	} {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(recorder)
			c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(test.body))
			c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 42})
			if test.admin {
				c.Set(string(middleware.ContextKeyUserRole), service.RoleAdmin)
			}
			(&ChannelMonitorV2Handler{}).ImportExternalQuality(c)
			require.Equal(t, test.status, recorder.Code)
		})
	}
}
