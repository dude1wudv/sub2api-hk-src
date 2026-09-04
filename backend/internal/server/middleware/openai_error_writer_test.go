package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestOpenAIErrorWriterUsesOpenAIEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	OpenAIErrorWriter(c, http.StatusForbidden, "group required")

	require.Equal(t, http.StatusForbidden, w.Code)
	require.JSONEq(t, `{"error":{"type":"permission_error","message":"group required"}}`, w.Body.String())
}
