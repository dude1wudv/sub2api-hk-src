//go:build unit

package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestChatAuditOmitsBodyWithoutConsumingIt(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &auditCaptureRepository{}
	audit := service.NewAuditLogService(repo, nil)
	audit.Start()
	router := gin.New()
	router.Use(func(c *gin.Context) { c.Set(string(ContextKeyUser), AuthSubject{UserID: 77}); c.Next() })
	router.Use(gin.HandlerFunc(NewAuditLogMiddleware(audit)))
	router.PUT("/api/v1/chat/sessions/:id", func(c *gin.Context) {
		raw, err := io.ReadAll(c.Request.Body)
		require.NoError(t, err)
		require.Equal(t, "private-chat-canary", string(raw))
		c.Status(http.StatusOK)
	})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPut, "/api/v1/chat/sessions/example", strings.NewReader("private-chat-canary")))
	audit.Stop()
	require.Equal(t, http.StatusOK, recorder.Code)
	repo.mu.Lock()
	defer repo.mu.Unlock()
	require.Len(t, repo.logs, 1)
	require.Equal(t, "<private chat body omitted>", repo.logs[0].RequestBody)
	require.NotContains(t, repo.logs[0].RequestBody, "canary")
}
