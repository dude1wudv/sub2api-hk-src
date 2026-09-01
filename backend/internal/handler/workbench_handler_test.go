package handler

import (
	"bytes"
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type handlerWorkbenchRepoStub struct {
	resolution *service.WorkbenchGrantResolution
}

func (r *handlerWorkbenchRepoStub) EnsureCredential(context.Context, int64, string, string, int64) (*service.WorkbenchCredential, error) {
	return &service.WorkbenchCredential{UserID: 9, APIKeyID: 41, Workbench: service.WorkbenchHeliosGen}, nil
}
func (r *handlerWorkbenchRepoStub) CreateGrant(context.Context, *service.WorkbenchLaunchGrant) error {
	return nil
}
func (r *handlerWorkbenchRepoStub) ConsumeGrant(context.Context, string, time.Time) (*service.WorkbenchGrantResolution, error) {
	return r.resolution, nil
}

func handlerWorkbenchConfig() *config.Config {
	return &config.Config{HeliosWorkbench: config.HeliosWorkbenchConfig{
		Enabled: true, ClientID: "heliosgen-web", ClientSecret: strings.Repeat("x", 32),
		RedirectURI: "https://canvas.sub.sunmmyapi.xyz/bootstrap", APIBaseURL: "https://sub.sunmmyapi.xyz/v1",
		GroupID: 7, GrantTTLSeconds: 90,
	}}
}

func validHandlerWorkbenchCode() string {
	return base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{7}, 32))
}

func newHandlerWorkbench(t *testing.T) (*WorkbenchHandler, *config.Config) {
	t.Helper()
	cfg := handlerWorkbenchConfig()
	repo := &handlerWorkbenchRepoStub{resolution: &service.WorkbenchGrantResolution{
		Grant:      &service.WorkbenchLaunchGrant{UserID: 9, APIKeyID: 41, ClientID: cfg.HeliosWorkbench.ClientID, RedirectURI: cfg.HeliosWorkbench.RedirectURI},
		Credential: &service.WorkbenchCredential{UserID: 9, APIKeyID: 41, Workbench: service.WorkbenchHeliosGen},
		User:       &service.User{ID: 9, Status: service.StatusActive},
		APIKey:     &service.APIKey{ID: 41, UserID: 9, Status: service.StatusActive, Key: "sk-server-only"},
	}}
	return NewWorkbenchHandler(service.NewWorkbenchIntegrationService(repo, nil, cfg), cfg), cfg
}

func TestWorkbenchTokenRequiresExactBasicAndNeverReturnsEnvelopeSecretsOnFailure(t *testing.T) {
	h, cfg := newHandlerWorkbench(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/token", h.Token)
	req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(`{"code":"bad"}`))
	req.SetBasicAuth(cfg.HeliosWorkbench.ClientID, "wrong")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusUnauthorized || strings.TrimSpace(recorder.Body.String()) != `{"error":"invalid_client"}` {
		t.Fatalf("invalid client response: %d %q", recorder.Code, recorder.Body.String())
	}
	if recorder.Header().Get("Cache-Control") != "no-store" || recorder.Header().Get("Pragma") != "no-cache" {
		t.Fatalf("missing no-store headers")
	}

	req = httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(`{"code":"`+validHandlerWorkbenchCode()+`","extra":"rejected"}`))
	req.SetBasicAuth(cfg.HeliosWorkbench.ClientID, cfg.HeliosWorkbench.ClientSecret)
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), "invalid_grant") {
		t.Fatalf("unknown token body response: %d %q", recorder.Code, recorder.Body.String())
	}
	if strings.Contains(recorder.Body.String(), "sk-server-only") {
		t.Fatal("failed token exchange exposed API key")
	}
}

func TestWorkbenchTokenReturnsKeyOnlyAfterConfidentialGrantExchange(t *testing.T) {
	h, cfg := newHandlerWorkbench(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/token", h.Token)
	req := httptest.NewRequest(http.MethodPost, "/token", strings.NewReader(`{"code":"`+validHandlerWorkbenchCode()+`"}`))
	req.SetBasicAuth(cfg.HeliosWorkbench.ClientID, cfg.HeliosWorkbench.ClientSecret)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"api_key":"sk-server-only"`) {
		t.Fatalf("token response: %d %q", recorder.Code, recorder.Body.String())
	}
	if recorder.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("token response is cacheable")
	}
}
