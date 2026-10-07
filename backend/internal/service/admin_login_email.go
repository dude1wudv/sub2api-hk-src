package service

import (
	"net/mail"
	"strings"
)

// CheckAdminLoginEmail applies the deployment's administrator email policy.
// Regular users keep their existing login behavior. Use the same generic error
// as a bad password so this check does not disclose administrator identities.
func (s *AuthService) CheckAdminLoginEmail(user *User) error {
	if user == nil {
		return ErrInvalidCredentials
	}
	if !user.IsAdmin() || s == nil || s.cfg == nil {
		return nil
	}
	domain := strings.TrimSpace(s.cfg.Security.AdminLoginEmailDomain)
	if domain == "" {
		return nil
	}
	address, err := mail.ParseAddress(user.Email)
	local, actualDomain, found := strings.Cut(user.Email, "@")
	if err != nil || address.Address != user.Email || !found || local == "" || !strings.EqualFold(actualDomain, domain) {
		return ErrInvalidCredentials
	}
	return nil
}
