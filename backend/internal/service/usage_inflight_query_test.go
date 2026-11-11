package service

import (
	"context"
	"errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/stretchr/testify/require"
	"testing"
)

type liveInflightStub struct {
	rows []UsageInflightSnapshot
	err  error
}

func (*liveInflightStub) Save(context.Context, UsageInflightSnapshot) error { return nil }
func (*liveInflightStub) Delete(context.Context, int64, string) error       { return nil }
func (s *liveInflightStub) ListUser(context.Context, int64) ([]UsageInflightSnapshot, error) {
	return s.rows, s.err
}
func (s *liveInflightStub) ListAll(context.Context) ([]UsageInflightSnapshot, error) {
	return s.rows, s.err
}
func TestUsageInflightOwnershipPrivacyPaginationAndUnknown(t *testing.T) {
	rows := make([]UsageInflightSnapshot, 0, 80)
	for i := 0; i < 80; i++ {
		rows = append(rows, UsageInflightSnapshot{RequestID: string(rune('a' + i)), UserID: 8, APIKeyID: 7, Email: "private@example.test", AccountID: 6, AccountName: "upstream-secret", IPAddress: "127.0.0.1", UpstreamEndpoint: "private", GroupID: 0, Model: "model"})
	}
	rows = append(rows, UsageInflightSnapshot{RequestID: "other", UserID: 9})
	SetUsageInflightStore(&liveInflightStub{rows: rows})
	t.Cleanup(func() { SetUsageInflightStore(nil) })
	s := &UsageService{}
	p := s.ListInflight(context.Background(), 8, usagestats.UsageLogFilters{}, 1, 1000, false)
	require.True(t, p.RuntimeKnown)
	require.Equal(t, 80, p.Total)
	require.Len(t, p.Items, 10)
	for _, row := range p.Items {
		require.Equal(t, int64(8), row.UserID)
		require.Zero(t, row.AccountID)
		require.Empty(t, row.AccountName)
		require.Empty(t, row.Email)
		require.Empty(t, row.IPAddress)
		require.True(t, row.TokensEstimated)
	}
	admin := s.ListInflight(context.Background(), 0, usagestats.UsageLogFilters{}, 1, 1000, true)
	require.Len(t, admin.Items, 50)
	grouped, known := s.UserInflightGroupLoad(context.Background(), 8)
	require.True(t, known)
	require.Len(t, grouped, 1)
	require.Zero(t, grouped[0].GroupID)
	require.Equal(t, 80, grouped[0].Inflight)
	SetUsageInflightStore(&liveInflightStub{err: errors.New("Redis unavailable")})
	unknown := s.ListInflight(context.Background(), 8, usagestats.UsageLogFilters{}, 1, 10, false)
	require.False(t, unknown.RuntimeKnown)
	require.Empty(t, unknown.Items)
}
