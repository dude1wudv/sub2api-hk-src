//go:build unit

package middleware

import (
	"context"
	"errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
	"testing"
)

func queueIdentityTestKey() *service.APIKey {
	groupID := int64(3)
	return &service.APIKey{ID: 1, UserID: 7, GroupID: &groupID, Status: service.StatusActive, ConcurrencyLimit: 2,
		User:  &service.User{ID: 7, Status: service.StatusActive},
		Group: &service.Group{ID: 3, Status: service.StatusActive, Platform: service.PlatformOpenAI},
	}
}
func TestAPIKeyAdmissionRevalidationRevocation(t *testing.T) {
	original := queueIdentityTestKey()
	latest := queueIdentityTestKey()
	revalidate := newAPIKeyAuthRevalidator(func(context.Context, string) (*service.APIKey, error) { return latest, nil }, "credential", "", original)
	limit, err := revalidate(context.Background())
	require.NoError(t, err)
	require.Equal(t, 2, limit)
	latest.Status = service.StatusAPIKeyDisabled
	_, err = revalidate(context.Background())
	var denied *service.APIKeyQueueAuthRejectedError
	require.True(t, errors.As(err, &denied))
}
func TestAPIKeyAdmissionIdentityKeepsOriginalRoutingOrder(t *testing.T) {
	original := queueIdentityTestKey()
	original.RoutingGroupIDs = []int64{3, 4}
	identity := captureAPIKeyQueueIdentity(original)
	original.RoutingGroupIDs[0] = 4
	original.GroupID = new(int64(4))
	latest := queueIdentityTestKey()
	latest.RoutingGroupIDs = []int64{3, 4}
	require.False(t, apiKeyQueueCoreIdentityChanged(identity, latest), "later request routing must not mutate captured raw key identity")
	latest.RoutingGroupIDs = []int64{4, 3}
	require.True(t, apiKeyQueueCoreIdentityChanged(identity, latest), "administrator routing changes require fresh admission")
}
