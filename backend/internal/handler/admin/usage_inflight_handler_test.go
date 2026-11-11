package admin

import (
	"github.com/Wei-Shaw/sub2api/internal/pkg/adminaccess"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInflightRejectsRestrictedAdministratorBeforeRead(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &UsageHandler{}
	for _, method := range []gin.HandlerFunc{h.Inflight, h.UserInflight} {
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		c.Request = req.WithContext(adminaccess.WithPrincipal(req.Context(), &adminaccess.Principal{KeyID: "scoped", Policy: adminaccess.Policy{}}))
		method(c)
		require.Equal(t, http.StatusForbidden, rec.Code)
	}
}
