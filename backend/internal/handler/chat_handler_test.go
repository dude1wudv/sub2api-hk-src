//go:build unit

package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestChatHandlerRejectsUnauthenticated(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &ChatHandler{}
	for _, fn := range []gin.HandlerFunc{h.ListSessions, h.GetSession, h.SaveSession, h.RenameSession, h.GetPreferences, h.SavePreferences, h.Catalog, h.ListPresets, h.CreatePreset, h.UpdatePreset, h.OrderPresets, h.DeletePreset, h.ReconnectPreset, h.Generate, h.StopGeneration, h.DeleteSessions, h.UploadAttachment, h.DownloadAttachment, h.BeginImport, h.StageImportSession, h.CommitImport, h.AbortImport} {
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/chat/sessions", nil)
		fn(c)
		require.Equal(t, http.StatusUnauthorized, recorder.Code)
		require.Contains(t, recorder.Header().Get("Cache-Control"), "no-store")
	}
}

func TestChatHandlerRejectsInvalidObjectBeforeService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/chat/sessions/bad-id", nil)
	c.Params = gin.Params{{Key: "id", Value: "bad-id"}}
	c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{UserID: 7})
	(&ChatHandler{}).GetSession(c)
	require.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestChatHandlerJSONBoundary(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		body  string
		limit int64
		want  error
	}{{"{} {}", 64, service.ErrChatInvalid}, {"[1]", 64, service.ErrChatInvalid}, {strings.Repeat(" ", 65) + "{}", 64, service.ErrChatTooLarge}} {
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/chat/preferences", strings.NewReader(tc.body))
		var value struct{}
		require.ErrorIs(t, decodeChatJSON(c, &value, tc.limit), tc.want)
	}
}

func TestChatHandlerRevisionConflictResponse(t *testing.T) {
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	chatError(c, service.ErrChatRevisionConflict)
	require.Equal(t, http.StatusConflict, recorder.Code)
	require.Contains(t, recorder.Body.String(), "CHAT_REVISION_CONFLICT")
}
