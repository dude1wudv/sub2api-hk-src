//go:build unit

package handler

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

type uncertainCachedContentCreateRepo struct {
	*cachedBillingRepo
	committed, lookupUnavailable bool
}

func (r *uncertainCachedContentCreateRepo) Create(ctx context.Context, record *service.GeminiCachedContent) error {
	if r.committed {
		if err := r.cachedBillingRepo.Create(ctx, record); err != nil {
			return err
		}
	}
	return errors.New("connection lost while returning INSERT result")
}

func (r *uncertainCachedContentCreateRepo) GetForOwner(ctx context.Context, keyID int64, publicID string) (*service.GeminiCachedContent, error) {
	if r.lookupUnavailable {
		return nil, errors.New("database unavailable during readback")
	}
	return r.cachedBillingRepo.GetForOwner(ctx, keyID, publicID)
}

func TestGeminiCachedContentsCreateUncertainCommitNeverDeletesDurableCharge(t *testing.T) {
	for _, tc := range []struct {
		name                         string
		committed, lookupUnavailable bool
		status, deletes, charges     int
	}{
		{"committed and verified", true, false, http.StatusOK, 0, 1},
		{"confirmed absent", false, false, http.StatusServiceUnavailable, 1, 0},
		{"committed but readback unavailable", true, true, http.StatusServiceUnavailable, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := newCachedBillingEnv(t, 500, 900)
			env.upstream.createBody = cachedBillingCreateBody(time.Now().Add(100 * time.Hour))
			repo := &uncertainCachedContentCreateRepo{env.repo, tc.committed, tc.lookupUnavailable}
			svc := service.NewGeminiCachedContentService(repo, env.h.geminiCompatService, nil)
			svc.SetBillingGateway(env.h.gatewayService)
			env.h.geminiCachedContentService = svc
			response := env.create()
			require.Equal(t, tc.status, response.Code, response.Body.String())
			require.Equal(t, tc.deletes, env.upstream.methodCalls(http.MethodDelete))
			require.Equal(t, tc.charges, env.ledger.applied)
			if tc.lookupUnavailable {
				require.Len(t, env.repo.records, 1)
				repo.lookupUnavailable = false
				count, err := svc.RecoverPendingCharges(context.Background())
				require.NoError(t, err)
				require.Equal(t, 1, count)
				require.Equal(t, 1, env.ledger.applied)
				require.Equal(t, 0, env.upstream.methodCalls(http.MethodDelete))
			}
		})
	}
}
