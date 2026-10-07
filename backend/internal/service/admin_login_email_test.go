//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestAdminLoginEmailPolicy(t *testing.T) {
	for _, tt := range []struct {
		name, email, role, domain string
		allowed                   bool
	}{
		{"admin", "admin@sub.sunmmyapi.xyz", RoleAdmin, "sub.sunmmyapi.xyz", true},
		{"second admin", "admin2@sub.sunmmyapi.xyz", RoleAdmin, "sub.sunmmyapi.xyz", true},
		{"domain case", "admin@SUB.SUNMMYAPI.XYZ", RoleAdmin, "sub.sunmmyapi.xyz", true},
		{"other domain", "admin@example.com", RoleAdmin, "sub.sunmmyapi.xyz", false},
		{"suffix spoof", "admin@sub.sunmmyapi.xyz.evil.com", RoleAdmin, "sub.sunmmyapi.xyz", false},
		{"subdomain", "admin@evil.sub.sunmmyapi.xyz", RoleAdmin, "sub.sunmmyapi.xyz", false},
		{"multiple at", "admin@evil.com@sub.sunmmyapi.xyz", RoleAdmin, "sub.sunmmyapi.xyz", false},
		{"display address", "Admin <admin@sub.sunmmyapi.xyz>", RoleAdmin, "sub.sunmmyapi.xyz", false},
		{"missing local part", "@sub.sunmmyapi.xyz", RoleAdmin, "sub.sunmmyapi.xyz", false},
		{"ordinary user", "user@example.com", RoleUser, "sub.sunmmyapi.xyz", true},
		{"disabled", "admin@example.com", RoleAdmin, "", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := &AuthService{cfg: &config.Config{Security: config.SecurityConfig{AdminLoginEmailDomain: tt.domain}}}
			err := s.CheckAdminLoginEmail(&User{Email: tt.email, Role: tt.role})
			if tt.allowed {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, ErrInvalidCredentials)
			}
		})
	}
}

func TestAdminLoginEmailBlocksAuthentication(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Config{
		JWT:      config.JWTConfig{Secret: "test-secret", ExpireHour: 1},
		Security: config.SecurityConfig{AdminLoginEmailDomain: "sub.sunmmyapi.xyz"},
	}
	s := &AuthService{cfg: cfg, refreshTokenCache: &refreshTokenCacheStub{}}
	hash, err := s.HashPassword("test-password")
	require.NoError(t, err)
	for _, tt := range []struct {
		email, role string
		allowed     bool
	}{
		{"admin@example.com", RoleAdmin, false},
		{"admin@sub.sunmmyapi.xyz", RoleAdmin, true},
		{"admin2@sub.sunmmyapi.xyz", RoleAdmin, true},
		{"user@example.com", RoleUser, true},
	} {
		t.Run(tt.email, func(t *testing.T) {
			user := &User{ID: 1, Email: tt.email, Role: tt.role, Status: StatusActive, PasswordHash: hash}
			s.userRepo = &userRepoStub{user: user}
			token, loggedIn, loginErr := s.Login(ctx, user.Email, "test-password")
			access, accessErr := s.GenerateToken(ctx, user)
			pair, pairErr := s.GenerateTokenPair(ctx, user, "")
			if tt.allowed {
				require.NoError(t, loginErr)
				require.Equal(t, user, loggedIn)
				require.NotEmpty(t, token)
				require.NoError(t, accessErr)
				require.NotEmpty(t, access)
				require.NoError(t, pairErr)
				require.NotEmpty(t, pair.RefreshToken)
			} else {
				require.ErrorIs(t, loginErr, ErrInvalidCredentials)
				require.Nil(t, loggedIn)
				require.Empty(t, token)
				require.ErrorIs(t, accessErr, ErrInvalidCredentials)
				require.Empty(t, access)
				require.ErrorIs(t, pairErr, ErrInvalidCredentials)
				require.Nil(t, pair)
			}
		})
	}
}
