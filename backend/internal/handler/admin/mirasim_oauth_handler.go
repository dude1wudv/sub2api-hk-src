package admin

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/mirasim"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// CreateMirasimOAuth exchanges a loopback OAuth refresh token and stores one
// native Sub2API account. The browser never receives the device private key.
func (h *AccountHandler) CreateMirasimOAuth(c *gin.Context) {
	var input struct {
		RefreshToken string  `json:"refresh_token" binding:"required"`
		Provider     string  `json:"provider" binding:"required"`
		Name         string  `json:"name" binding:"max=100"`
		ProxyID      *int64  `json:"proxy_id" binding:"omitempty,gt=0"`
		GroupIDs     []int64 `json:"group_ids" binding:"max=100,dive,gt=0"`
		Concurrency  int     `json:"concurrency" binding:"omitempty,min=1,max=100"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "invalid Mirasim OAuth payload")
		return
	}
	input.RefreshToken = strings.TrimSpace(input.RefreshToken)
	if input.Provider != "github" && input.Provider != "google" {
		response.BadRequest(c, "unsupported Mirasim OAuth provider")
		return
	}
	if len(input.RefreshToken) > 16384 || len(strings.Split(input.RefreshToken, ".")) != 3 {
		response.BadRequest(c, "invalid Mirasim refresh token")
		return
	}
	claims := mirasim.JWTClaims(input.RefreshToken)
	if claims == nil || (claims["token_type"] != nil && claims["token_type"] != "refresh") {
		response.BadRequest(c, "invalid Mirasim refresh token claims")
		return
	}
	email, _ := claims["email"].(string)
	if email == "" {
		email, _ = claims["sub"].(string)
	}
	if email == "" || len(email) > 92 {
		response.BadRequest(c, "Mirasim account identity is missing")
		return
	}
	// Validate user-selected configuration before exchanging the rotating token.
	proxyURL := ""
	if input.ProxyID != nil {
		proxy, err := h.adminService.GetProxy(c.Request.Context(), *input.ProxyID)
		if err != nil || proxy == nil || !proxy.IsActive() || proxy.IsExpired(time.Now()) {
			response.BadRequest(c, "Mirasim proxy is unavailable")
			return
		}
		proxyURL = proxy.URL()
	}
	for _, id := range input.GroupIDs {
		group, err := h.adminService.GetGroup(c.Request.Context(), id)
		if err != nil || group == nil || group.Platform != service.PlatformMirasim || group.Status != service.StatusActive {
			response.BadRequest(c, "select an active Mirasim group")
			return
		}
	}
	if input.Concurrency == 0 {
		input.Concurrency = 5
	}
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		input.Name = "Mirasim " + email
	}
	if len(input.GroupIDs) == 0 {
		id, err := h.ensureMirasimDefaultGroup(c.Request.Context())
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}
		input.GroupIDs = []int64{id}
	}
	privateKey, err := mirasim.GenerateDeviceKey()
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	pem, err := mirasim.MarshalPrivateKeyPEM(privateKey)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	client, err := mirasim.NewClient(input.RefreshToken, privateKey, "", "", "", "", proxyURL)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	refreshToken := input.RefreshToken
	client.OnRotate = func(newToken string) { refreshToken = newToken }
	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()
	if _, err := client.AccessToken(ctx); err != nil {
		response.ErrorFrom(c, fmt.Errorf("Mirasim OAuth validation failed: %w", err))
		return
	}
	account, err := h.adminService.CreateAccount(ctx, &service.CreateAccountInput{
		Name:        input.Name,
		Platform:    service.PlatformMirasim,
		Type:        service.AccountTypeAPIKey,
		Concurrency: input.Concurrency,
		ProxyID:     input.ProxyID,
		GroupIDs:    input.GroupIDs,
		Credentials: map[string]any{
			"refresh_token":  refreshToken,
			"private_key":    pem,
			"oauth_provider": input.Provider,
			"api_protocol":   service.APIProtocolAdaptive,
			"base_url":       "https://relay.mirasim.ai/v1",
			"api_base_urls": map[string]any{
				"chat_completions": "https://relay.mirasim.ai/v1",
				"anthropic":        "https://relay.mirasim.ai",
				"responses":        "https://relay.mirasim.ai/v1",
			},
			"model_mapping": service.DefaultMirasimModelMapping(),
		},
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	response.Success(c, gin.H{"id": account.ID, "name": account.Name})
}

// Serialize first-import default-group creation in this application instance.
var mirasimDefaultGroupMu sync.Mutex

func (h *AccountHandler) ensureMirasimDefaultGroup(ctx context.Context) (int64, error) {
	mirasimDefaultGroupMu.Lock()
	defer mirasimDefaultGroupMu.Unlock()
	groups, err := h.adminService.GetAllGroupsByPlatform(ctx, service.PlatformMirasim)
	if err != nil {
		return 0, err
	}
	for _, group := range groups {
		if group.Name == "mirasim-default" {
			if group.Status != service.StatusActive {
				return 0, fmt.Errorf("Mirasim default group is inactive; select an active group")
			}
			return group.ID, nil
		}
	}
	group, err := h.adminService.CreateGroup(ctx, &service.CreateGroupInput{
		Name: "mirasim-default", Platform: service.PlatformMirasim,
		RateMultiplier: 1, IsExclusive: true,
		Description:      "Mirasim OAuth; review per-model pricing before granting access",
		SubscriptionType: service.SubscriptionTypeStandard,
		ModelAllowlist:   service.GroupModelAllowlist{Enabled: true, Models: service.DefaultMirasimModelIDs()},
	})
	if err != nil {
		return 0, err
	}
	return group.ID, nil
}
