package admin

import (
	"context"
	"fmt"
	"strings"
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
		RefreshToken string `json:"refresh_token" binding:"required"`
		Provider     string `json:"provider" binding:"required"`
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
	client, err := mirasim.NewClient(input.RefreshToken, privateKey, "", "", "", "", "")
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
	groups, err := h.adminService.GetAllGroupsByPlatform(ctx, service.PlatformMirasim)
	if err != nil {
		response.ErrorFrom(c, fmt.Errorf("load Mirasim groups: %w", err))
		return
	}
	defaultGroupExists := false
	for _, group := range groups {
		if group.Name == "mirasim-default" {
			defaultGroupExists = true
			break
		}
	}
	if !defaultGroupExists {
		if _, err := h.adminService.CreateGroup(ctx, &service.CreateGroupInput{
			Name:             "mirasim-default",
			Platform:         service.PlatformMirasim,
			RateMultiplier:   1,
			SubscriptionType: service.SubscriptionTypeStandard,
		}); err != nil {
			response.ErrorFrom(c, fmt.Errorf("create Mirasim default group: %w", err))
			return
		}
	}
	account, err := h.adminService.CreateAccount(ctx, &service.CreateAccountInput{
		Name:        "Mirasim " + email,
		Platform:    service.PlatformMirasim,
		Type:        service.AccountTypeAPIKey,
		Concurrency: 5,
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
