package handler

import (
	"context"
	"encoding/json"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type keyUsageDetailsRepo struct {
	service.UsageLogRepository
	calls         int
	userID, keyID int64
	start, end    time.Time
	zone          string
	durations     []time.Duration
}

func (r *keyUsageDetailsRepo) GetAPIKeyDailyUsageInTimezone(_ context.Context, user, key int64, start, end time.Time, zone string) ([]usagestats.APIKeyDailyUsagePoint, error) {
	r.durations = append(r.durations, end.Sub(start))
	r.calls++
	r.userID = user
	r.keyID = key
	r.start = start
	r.end = end
	r.zone = zone
	return []usagestats.APIKeyDailyUsagePoint{{Date: start.Format("2006-01-02"), Requests: 2, InputTokens: 4, OutputTokens: 6, TotalTokens: 10, Cost: 3, ActualCost: 1.5}}, nil
}
func keyUsageDetailsRouter(repo *keyUsageDetailsRepo, owner int64) *gin.Engine {
	gin.SetMode(gin.TestMode)
	keys := &dailyUsageAPIKeyRepoStub{keys: map[int64]*service.APIKey{7: {ID: 7, UserID: owner}}}
	h := NewUsageHandler(service.NewUsageService(repo, nil, nil, nil), service.NewAPIKeyService(keys, nil, nil, nil, nil, nil, nil), nil, nil)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 8})
		c.Next()
	})
	r.GET("/user/api-keys/:id/usage", h.GetMyAPIKeyUsage)
	return r
}
func TestKeyUsageDetailsOwnershipAndValidation(t *testing.T) {
	for _, tc := range []struct {
		name, url string
		owner     int64
		status    int
	}{{"owner", "/user/api-keys/7/usage", 9, 403}, {"unknown key", "/user/api-keys/9/usage", 8, 404}, {"invalid timezone", "/user/api-keys/7/usage?timezone=invalid-zone", 8, 400}, {"invalid days", "/user/api-keys/7/usage?days=91", 8, 400}, {"reverse", "/user/api-keys/7/usage?start_date=2026-11-11&end_date=2026-11-10", 8, 400}, {"overlong", "/user/api-keys/7/usage?start_date=2026-01-01&end_date=2026-05-01", 8, 400}} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &keyUsageDetailsRepo{}
			rec := httptest.NewRecorder()
			keyUsageDetailsRouter(repo, tc.owner).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.url, nil))
			require.Equal(t, tc.status, rec.Code)
			require.Zero(t, repo.calls)
		})
	}
}
func TestKeyUsageDetailsCalendarRangeAndProjection(t *testing.T) {
	repo := &keyUsageDetailsRepo{}
	rec := httptest.NewRecorder()
	keyUsageDetailsRouter(repo, 8).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/user/api-keys/7/usage?start_date=2026-03-08&end_date=2026-03-08&timezone=America/Denver", nil))
	require.Equal(t, 200, rec.Code)
	require.Equal(t, 23*time.Hour, repo.durations[0])
	require.Equal(t, int64(8), repo.userID)
	require.Equal(t, int64(7), repo.keyID)
	// The range has one calendar day across DST, not a fixed 24-hour subtraction.
	var body struct {
		Data struct {
			StartDate string                           `json:"start_date"`
			EndDate   string                           `json:"end_date"`
			Timezone  string                           `json:"timezone"`
			Range     usagestats.APIKeyDailyUsagePoint `json:"range"`
		}
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, "2026-03-08", body.Data.StartDate)
	require.Equal(t, body.Data.StartDate, body.Data.EndDate)
	require.Equal(t, "America/Denver", body.Data.Timezone)
	require.Equal(t, int64(2), body.Data.Range.Requests)
	require.Equal(t, 1.5, body.Data.Range.ActualCost)
	for _, secret := range []string{"account_cost", "upstream_endpoint", "ip_address", "request_id", "account_id"} {
		require.NotContains(t, rec.Body.String(), secret)
	}
}
