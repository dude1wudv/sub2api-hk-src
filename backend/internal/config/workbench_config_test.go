package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHeliosWorkbenchConfigValidation(t *testing.T) {
	resetViperWithJWTSecret(t)
	base, err := Load()
	require.NoError(t, err)
	base.HeliosWorkbench = HeliosWorkbenchConfig{
		Enabled:         true,
		PublicURL:       "https://canvas.sub.sunmmyapi.xyz",
		RedirectURI:     "https://canvas.sub.sunmmyapi.xyz/bootstrap",
		APIBaseURL:      "https://sub.sunmmyapi.xyz/v1",
		ClientID:        "heliosgen-web",
		GroupID:         7,
		ClientSecret:    strings.Repeat("s", 32),
		GrantTTLSeconds: 90,
	}

	tests := []struct {
		name   string
		mutate func(*HeliosWorkbenchConfig)
		want   string
	}{
		{name: "valid production contract"},
		{name: "valid main-site prefix", mutate: func(cfg *HeliosWorkbenchConfig) {
			cfg.PublicURL = "https://patrickapi.microedulab.com/cancas"
			cfg.RedirectURI = cfg.PublicURL + "/bootstrap"
		}},
		{name: "reject encoded separator", mutate: func(cfg *HeliosWorkbenchConfig) {
			cfg.PublicURL = "https://patrickapi.microedulab.com/cancas%2fother"
			cfg.RedirectURI = "https://patrickapi.microedulab.com/cancas/other/bootstrap"
		}, want: "invalid path prefix"},
		{name: "reject path traversal", mutate: func(cfg *HeliosWorkbenchConfig) {
			cfg.PublicURL = "https://patrickapi.microedulab.com/cancas/../other"
			cfg.RedirectURI = cfg.PublicURL + "/bootstrap"
		}, want: "invalid path prefix"},
		{name: "reject bootstrap outside prefix", mutate: func(cfg *HeliosWorkbenchConfig) {
			cfg.PublicURL = "https://patrickapi.microedulab.com/cancas"
			cfg.RedirectURI = "https://patrickapi.microedulab.com/bootstrap"
		}, want: "bootstrap path under"},
		{
			name: "public URL must use HTTPS",
			mutate: func(cfg *HeliosWorkbenchConfig) {
				cfg.PublicURL = "http://canvas.sub.sunmmyapi.xyz"
			},
			want: "public_url must use HTTPS",
		},
		{
			name: "redirect URL must use HTTPS",
			mutate: func(cfg *HeliosWorkbenchConfig) {
				cfg.RedirectURI = "http://canvas.sub.sunmmyapi.xyz/bootstrap"
			},
			want: "redirect_uri must use HTTPS",
		},
		{
			name: "redirect URL must stay on public origin",
			mutate: func(cfg *HeliosWorkbenchConfig) {
				cfg.RedirectURI = "https://other.example/bootstrap"
			},
			want: "redirect_uri must be the bootstrap path under the public URL",
		},
		{
			name: "redirect URL must use bootstrap path",
			mutate: func(cfg *HeliosWorkbenchConfig) {
				cfg.RedirectURI = "https://canvas.sub.sunmmyapi.xyz/other"
			},
			want: "redirect_uri must be the bootstrap path under the public URL",
		},
		{
			name: "API base must use HTTPS",
			mutate: func(cfg *HeliosWorkbenchConfig) {
				cfg.APIBaseURL = "http://sub.sunmmyapi.xyz/api"
			},
			want: "api_base_url must use HTTPS",
		},
		{
			name: "API base must end in v1",
			mutate: func(cfg *HeliosWorkbenchConfig) {
				cfg.APIBaseURL = "https://sub.sunmmyapi.xyz/api"
			},
			want: "api_base_url path must end with /v1",
		},
		{
			name: "client id is fixed",
			mutate: func(cfg *HeliosWorkbenchConfig) {
				cfg.ClientID = "another-client"
			},
			want: "client_id must be heliosgen-web",
		},
		{
			name: "group id must be positive",
			mutate: func(cfg *HeliosWorkbenchConfig) {
				cfg.GroupID = 0
			},
			want: "group_id must be positive",
		},
		{
			name: "client secret has minimum strength",
			mutate: func(cfg *HeliosWorkbenchConfig) {
				cfg.ClientSecret = strings.Repeat("s", 31)
			},
			want: "client_secret must be at least 32 bytes",
		},
		{
			name: "grant TTL is fixed",
			mutate: func(cfg *HeliosWorkbenchConfig) {
				cfg.GrantTTLSeconds = 91
			},
			want: "grant_ttl_seconds must be 90",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := *base
			if tt.mutate != nil {
				tt.mutate(&cfg.HeliosWorkbench)
			}
			err := cfg.Validate()
			if tt.want == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.want)
		})
	}
}
