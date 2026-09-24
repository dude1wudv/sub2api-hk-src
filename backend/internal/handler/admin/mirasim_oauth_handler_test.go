package admin

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type mirasimOAuthOptionsAdminService struct {
	service.AdminService
	proxyCalls int
	groupCalls int
	proxy      *service.Proxy
	group      *service.Group
}

func (s *mirasimOAuthOptionsAdminService) GetProxy(context.Context, int64) (*service.Proxy, error) {
	s.proxyCalls++
	return s.proxy, nil
}

func (s *mirasimOAuthOptionsAdminService) GetGroup(context.Context, int64) (*service.Group, error) {
	s.groupCalls++
	return s.group, nil
}

func validMirasimRefreshToken() string {
	encode := func(value string) string { return base64.RawURLEncoding.EncodeToString([]byte(value)) }
	return encode(`{"alg":"none"}`) + "." + encode(`{"email":"mirasim@example.com","token_type":"refresh"}`) + ".signature"
}

func invokeMirasimOAuthHandler(body string, adminService service.AdminService) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/admin/accounts/mirasim/oauth", strings.NewReader(body))
	(&AccountHandler{adminService: adminService}).CreateMirasimOAuth(ctx)
	return recorder
}

func TestCreateMirasimOAuthRejectsInvalidImportOptionsBeforeExchange(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tooManyGroups := make([]int64, 101)
	for i := range tooManyGroups {
		tooManyGroups[i] = int64(i + 1)
	}
	tests := []struct {
		name string
		data map[string]any
	}{
		{name: "zero proxy id", data: map[string]any{"proxy_id": 0}},
		{name: "negative group id", data: map[string]any{"group_ids": []int64{-1}}},
		{name: "too many groups", data: map[string]any{"group_ids": tooManyGroups}},
		{name: "negative concurrency", data: map[string]any{"concurrency": -1}},
		{name: "concurrency above maximum", data: map[string]any{"concurrency": 101}},
		{name: "name too long", data: map[string]any{"name": strings.Repeat("a", 101)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := map[string]any{"refresh_token": validMirasimRefreshToken(), "provider": "github"}
			for key, value := range tt.data {
				input[key] = value
			}
			body, err := json.Marshal(input)
			if err != nil {
				t.Fatal(err)
			}
			recorder := invokeMirasimOAuthHandler(string(body), nil)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
			}
			if !strings.Contains(recorder.Body.String(), "invalid Mirasim OAuth payload") {
				t.Fatalf("request failed for an unexpected reason: %s", recorder.Body.String())
			}
		})
	}
}

func TestCreateMirasimOAuthValidatesProxyAndGroupBeforeTokenExchange(t *testing.T) {
	adminService := &mirasimOAuthOptionsAdminService{
		proxy: &service.Proxy{ID: 12, Protocol: "http", Host: "127.0.0.1", Port: 8080, Status: service.StatusActive},
		group: &service.Group{ID: 34, Platform: service.PlatformOpenAI, Status: service.StatusActive},
	}
	body, err := json.Marshal(map[string]any{
		"refresh_token": validMirasimRefreshToken(),
		"provider":      "github",
		"proxy_id":      12,
		"group_ids":     []int64{34},
		"concurrency":   8,
	})
	if err != nil {
		t.Fatal(err)
	}
	recorder := invokeMirasimOAuthHandler(string(body), adminService)
	if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), "select an active Mirasim group") {
		t.Fatalf("expected group validation before exchange, status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if adminService.proxyCalls != 1 || adminService.groupCalls != 1 {
		t.Fatalf("expected proxy and group lookups once each, got proxy=%d group=%d", adminService.proxyCalls, adminService.groupCalls)
	}
}
