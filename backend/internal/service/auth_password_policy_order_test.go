//go:build unit

package service

import (
	"context"
	"strings"
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type passwordPolicyCodeCache struct {
	emailCacheStub
	deletes int
}

func (c *passwordPolicyCodeCache) DeleteVerificationCode(context.Context, string) error {
	c.deletes++
	c.data = nil
	return nil
}

func TestNewPasswordValidationPreservesEmailVerificationCode(t *testing.T) {
	ctx := context.Background()
	for _, flow := range []string{"registration", "oauth registration", "first email binding"} {
		t.Run(flow, func(t *testing.T) {
			for _, password := range []string{"secret-123", strings.Repeat("密", 25)} {
				cache := &passwordPolicyCodeCache{emailCacheStub: emailCacheStub{data: &VerificationCodeData{
					Code: "246810", ExpiresAt: time.Now().Add(time.Minute),
				}}}
				repo := &userRepoStub{user: &User{ID: 1, Email: "first" + LinuxDoConnectSyntheticEmailDomain}}
				svc := newAuthService(repo, map[string]string{SettingKeyRegistrationEnabled: "true", SettingKeyEmailVerifyEnabled: "true"}, cache, nil)
				var err error
				switch flow {
				case "registration":
					_, _, err = svc.RegisterWithVerification(ctx, "fresh@example.com", password, "246810", "", "", "")
				case "oauth registration":
					_, _, err = svc.RegisterOAuthEmailAccount(ctx, "fresh@example.com", password, "246810", "", "oidc")
				case "first email binding":
					_, err = svc.BindEmailIdentity(ctx, 1, "fresh@example.com", "246810", password)
				}
				require.Error(t, err)
				require.Contains(t, []string{"PASSWORD_TOO_SHORT", "PASSWORD_TOO_LONG"}, infraerrors.Reason(err))
				require.Zero(t, cache.deletes)
				require.NotNil(t, cache.data)
				require.Zero(t, cache.data.Attempts)
				require.Empty(t, repo.created)
				// The exact same code remains usable after correcting the password.
				require.NoError(t, svc.emailService.VerifyCode(ctx, "fresh@example.com", "246810"))
				require.Equal(t, 1, cache.deletes)
			}
		})
	}
}

func TestResetPasswordPolicyErrorPreservesTokenForCorrection(t *testing.T) {
	ctx := context.Background()
	cache := &resetTokenCacheStub{stored: &PasswordResetTokenData{Token: hashPasswordResetToken("reset-token")}}
	repo := &userRepoStub{user: &User{ID: 1, Email: "user@example.com", Status: StatusActive}}
	svc := newAuthService(repo, map[string]string{SettingKeyEmailVerifyEnabled: "true", SettingKeyPasswordResetEnabled: "true"}, cache, nil)
	for _, password := range []string{"secret-123", strings.Repeat("密", 25)} {
		err := svc.ResetPassword(ctx, "user@example.com", "reset-token", password)
		require.Error(t, err)
		require.Contains(t, []string{"PASSWORD_TOO_SHORT", "PASSWORD_TOO_LONG"}, infraerrors.Reason(err))
		require.Empty(t, cache.consumedHash)
		require.NotNil(t, cache.stored)
		require.Empty(t, repo.updated)
	}
	require.NoError(t, svc.ResetPassword(ctx, "user@example.com", "reset-token", "corrected-password"))
	require.Nil(t, cache.stored)
	require.Len(t, repo.updated, 1)
	require.True(t, svc.CheckPassword("corrected-password", repo.user.PasswordHash))
}

func TestBindExistingEmailKeepsLegacyPasswordAndCodeRetry(t *testing.T) {
	ctx := context.Background()
	cache := &passwordPolicyCodeCache{emailCacheStub: emailCacheStub{data: &VerificationCodeData{Code: "246810", ExpiresAt: time.Now().Add(time.Minute)}}}
	repo := &userRepoStub{user: &User{ID: 1, Email: "old@example.com", Status: StatusActive}}
	svc := newAuthService(repo, map[string]string{}, cache, nil)
	hash, err := svc.HashPassword("legacy-8")
	require.NoError(t, err)
	repo.user.PasswordHash = hash
	_, err = svc.BindEmailIdentity(ctx, 1, "new@example.com", "246810", "wrong-password")
	require.ErrorIs(t, err, ErrPasswordIncorrect)
	require.Zero(t, cache.deletes)
	require.Zero(t, cache.data.Attempts)
	updated, err := svc.BindEmailIdentity(ctx, 1, "new@example.com", "246810", "legacy-8")
	require.NoError(t, err)
	require.Equal(t, "new@example.com", updated.Email)
	require.True(t, svc.CheckPassword("legacy-8", updated.PasswordHash))
	require.Equal(t, 1, cache.deletes)
}
