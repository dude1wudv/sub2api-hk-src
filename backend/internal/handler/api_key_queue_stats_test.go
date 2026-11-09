//go:build unit

package handler

import (
	"context"
	"fmt"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type queueStatsOwnershipRepo struct {
	service.APIKeyRepository
	ownerID int64
}

func (r *queueStatsOwnershipRepo) VerifyOwnership(_ context.Context, userID int64, ids []int64) ([]int64, error) {
	r.ownerID = userID
	return []int64{1}, nil
}
func TestAPIKeyQueueStatsIDsAndOwnership(t *testing.T) {
	ids, err := parseAPIKeyQueueStatsIDs("1, 2,1")
	require.NoError(t, err)
	require.Equal(t, []int64{1, 2}, ids)
	for _, raw := range []string{"-1", "0", "1,nope", "9223372036854775808"} {
		_, err := parseAPIKeyQueueStatsIDs(raw)
		require.Error(t, err)
	}
	var raw []string
	for i := 1; i <= 101; i++ {
		raw = append(raw, fmt.Sprint(i))
	}
	_, err = parseAPIKeyQueueStatsIDs(strings.Join(raw, ","))
	require.Error(t, err)
	repo := &queueStatsOwnershipRepo{}
	svc := service.NewAPIKeyService(repo, nil, nil, nil, nil, nil, nil)
	handler := NewAPIKeyHandler(svc)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/keys/concurrency?ids=1,2", nil)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 7})
	handler.GetConcurrencyQueue(c)
	require.Equal(t, int64(7), repo.ownerID)
	require.Equal(t, http.StatusNotFound, recorder.Code)
	require.NotContains(t, recorder.Body.String(), "current_concurrency")
}
func TestAPIKeyQueueHTTPErrors(t *testing.T) {
	for _, tc := range []struct {
		kind   service.APIKeyQueueErrorKind
		status int
		code   string
	}{
		{service.APIKeyQueueErrorFull, 429, apiKeyQueueFullCode},
		{service.APIKeyQueueErrorTimeout, 429, apiKeyQueueTimeoutCode},
		{service.APIKeyQueueErrorPolicyChanged, 503, ""},
		{service.APIKeyQueueErrorUnavailable, 503, ""},
	} {
		status, _, code, _ := concurrencyErrorResponse(&service.APIKeyQueueError{Kind: tc.kind}, "API key")
		require.Equal(t, tc.status, status)
		require.Equal(t, tc.code, code)
	}
}
