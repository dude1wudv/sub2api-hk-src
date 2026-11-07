package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestRetiredHKProvidersAreNotConfigurableOrSchedulable(t *testing.T) {
	for _, platform := range []string{PlatformStepFun, PlatformMirasim} {
		require.True(t, domain.IsRetiredPlatform(platform))
		require.False(t, domain.IsConcretePlatform(platform))
		require.False(t, IsMultiProtocolAPIKeyProvider(platform))
		require.False(t, SmartRoutingPlatformSupported(platform))
		require.NotContains(t, AllowedQuotaPlatforms, platform)
	}
}
