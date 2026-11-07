package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRetiredProviderModelSyncStopsBeforeUpstreamAccess(t *testing.T) {
	svc := &AccountTestService{}
	_, err := svc.SyncUpstreamModelCatalog(context.Background(), &Account{Platform: PlatformMirasim})

	var syncErr *UpstreamModelSyncError
	require.ErrorAs(t, err, &syncErr)
	require.Equal(t, UpstreamModelSyncErrorConfiguration, syncErr.Kind)
	require.Contains(t, syncErr.SafeMessage(), "retired")
}
