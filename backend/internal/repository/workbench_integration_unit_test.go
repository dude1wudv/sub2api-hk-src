package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestWorkbenchRepositoryRejectsInvalidInputsWithoutDatabase(t *testing.T) {
	repo := &workbenchIntegrationRepository{}
	ctx := context.Background()

	_, err := repo.EnsureCredential(ctx, 0, service.WorkbenchHeliosGen, "sk-invalid", 1)
	require.ErrorIs(t, err, service.ErrWorkbenchUnavailable)
	_, err = repo.EnsureCredential(ctx, 1, "other", "sk-invalid", 1)
	require.ErrorIs(t, err, service.ErrWorkbenchUnavailable)
	_, err = repo.ConsumeGrant(ctx, "", time.Now().UTC())
	require.ErrorIs(t, err, service.ErrWorkbenchInvalidGrant)
	require.ErrorIs(t, repo.CreateGrant(ctx, nil), service.ErrWorkbenchUnavailable)
}

func TestWorkbenchRepositoryCreateGrantWithoutDatabaseFailsClosed(t *testing.T) {
	repo := &workbenchIntegrationRepository{}
	grant := &service.WorkbenchLaunchGrant{CodeHash: "hash", UserID: 1, APIKeyID: 1}
	require.ErrorIs(t, repo.CreateGrant(context.Background(), grant), service.ErrWorkbenchUnavailable)
}
