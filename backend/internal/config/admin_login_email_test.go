package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadAdminLoginEmailDomain(t *testing.T) {
	for _, domain := range []string{"", "sub.sunmmyapi.xyz"} {
		t.Run("domain="+domain, func(t *testing.T) {
			resetViperWithJWTSecret(t)
			t.Setenv("SECURITY_ADMIN_LOGIN_EMAIL_DOMAIN", domain)
			cfg, err := Load()
			require.NoError(t, err)
			require.Equal(t, domain, cfg.Security.AdminLoginEmailDomain)
		})
	}
}
