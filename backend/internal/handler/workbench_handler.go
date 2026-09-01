package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type WorkbenchHandler struct {
	service *service.WorkbenchIntegrationService
	cfg     *config.Config
}

func NewWorkbenchHandler(integration *service.WorkbenchIntegrationService, cfg *config.Config) *WorkbenchHandler {
	return &WorkbenchHandler{service: integration, cfg: cfg}
}

type workbenchLaunchRequest struct{}

type workbenchTokenRequest struct {
	Code string `json:"code"`
}

// Launch creates a short-lived one-time grant. The body is intentionally empty;
// client, redirect, and API key values are deployment-controlled.
func (h *WorkbenchHandler) Launch(c *gin.Context) {
	setWorkbenchNoStore(c)
	servermiddleware.SetAuditAction(c, "workbench.helios.launch")
	if err := decodeWorkbenchJSON(c, &workbenchLaunchRequest{}, true); err != nil {
		servermiddleware.SetAuditExtra(c, map[string]any{"result": "invalid_request"})
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	subject, ok := servermiddleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		servermiddleware.SetAuditExtra(c, map[string]any{"result": "unauthorized"})
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if h == nil || h.service == nil {
		servermiddleware.SetAuditExtra(c, map[string]any{"result": "unavailable"})
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "workbench_unavailable"})
		return
	}
	launch, err := h.service.CreateLaunch(c.Request.Context(), subject.UserID)
	if err != nil {
		servermiddleware.SetAuditExtra(c, map[string]any{"result": "error"})
		if !response.ErrorFrom(c, err) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "workbench_unavailable"})
		}
		return
	}
	servermiddleware.SetAuditExtra(c, map[string]any{"result": "issued"})
	c.JSON(http.StatusOK, launch)
}

// Token exchanges a one-time grant for the server-side Helios session payload.
// It is a confidential endpoint: browsers must not be able to call it via CORS.
func (h *WorkbenchHandler) Token(c *gin.Context) {
	setWorkbenchNoStore(c)
	servermiddleware.SetAuditAction(c, "workbench.helios.token")
	if h == nil || h.cfg == nil || !validWorkbenchBasic(c.Request, h.cfg) {
		servermiddleware.SetAuditExtra(c, map[string]any{"result": "invalid_client"})
		writeWorkbenchInvalidClient(c)
		return
	}
	var req workbenchTokenRequest
	if err := decodeWorkbenchJSON(c, &req, false); err != nil || req.Code == "" {
		servermiddleware.SetAuditExtra(c, map[string]any{"result": "invalid_grant"})
		writeWorkbenchInvalidGrant(c)
		return
	}
	resolution, err := h.service.ConsumeGrant(c.Request.Context(), req.Code)
	if err != nil || resolution == nil || resolution.APIKey == nil || resolution.User == nil {
		servermiddleware.SetAuditExtra(c, map[string]any{"result": "invalid_grant"})
		writeWorkbenchInvalidGrant(c)
		return
	}
	servermiddleware.SetAuditActor(c, resolution.User.ID, "")
	servermiddleware.SetAuditExtra(c, map[string]any{"result": "issued"})
	c.JSON(http.StatusOK, gin.H{
		"user_id":            strconv.FormatInt(resolution.User.ID, 10),
		"api_base_url":       h.cfg.HeliosWorkbench.APIBaseURL,
		"api_key":            resolution.APIKey.Key,
		"session_expires_in": service.WorkbenchSessionTTLSeconds,
	})
}

func validWorkbenchBasic(req *http.Request, cfg *config.Config) bool {
	if req == nil || cfg == nil {
		return false
	}
	clientID, clientSecret, ok := req.BasicAuth()
	return ok && service.ConstantTimeEqual(clientID, cfg.HeliosWorkbench.ClientID) &&
		service.ConstantTimeEqual(clientSecret, cfg.HeliosWorkbench.ClientSecret)
}

func decodeWorkbenchJSON(c *gin.Context, target any, allowEmpty bool) error {
	if c == nil || c.Request == nil || c.Request.Body == nil {
		if allowEmpty {
			return nil
		}
		return io.EOF
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		if allowEmpty && err == io.EOF {
			return nil
		}
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return errTrailingJSON
		}
		return err
	}
	return nil
}

var errTrailingJSON = &trailingJSONError{}

type trailingJSONError struct{}

func (*trailingJSONError) Error() string { return "multiple JSON values" }

func setWorkbenchNoStore(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
}

func writeWorkbenchInvalidClient(c *gin.Context) {
	c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_client"})
}

func writeWorkbenchInvalidGrant(c *gin.Context) {
	c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_grant"})
}
