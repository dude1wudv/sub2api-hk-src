package dto

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestGroupRateHintsRequireAuthorizationForUserAndNestedMappers(t *testing.T) {
	now := time.Now().UTC()
	rate := 1.5
	change := &service.GroupRateChange{OldRate: 2, NewRate: rate, ChangedAt: now}
	for _, authorized := range []bool{false, true} {
		name := "unverified raw repository group"
		if authorized {
			name = "available group checked with no personal override"
		}
		t.Run(name, func(t *testing.T) {
			group := &service.Group{
				ID: 7, Name: "published", RateMultiplier: 1, LaunchedAt: &now,
				LaunchRate: &rate, IsNew: true, RecentRateChange: change,
				RateHintsAuthorized: authorized,
			}
			for mapper, output := range map[string]*Group{
				"direct user":         GroupFromService(group),
				"shallow user":        GroupFromServiceShallow(group),
				"API key nested":      APIKeyFromService(&service.APIKey{ID: 1, UserID: 2, Group: group}).Group,
				"subscription nested": UserSubscriptionFromService(&service.UserSubscription{ID: 3, UserID: 2, Group: group}).Group,
			} {
				t.Run(mapper, func(t *testing.T) {
					require.NotNil(t, output)
					require.Equal(t, authorized, output.LaunchRate != nil)
					require.Equal(t, authorized, output.RecentRateChange != nil)
					require.True(t, output.IsNew)
					require.Equal(t, &now, output.LaunchedAt)
					fields := marshalToMap(t, output)
					if !authorized {
						require.Nil(t, fields["launch_rate"])
						require.Nil(t, fields["recent_rate_change"])
					}
					require.NotContains(t, fields, "RateHintsAuthorized")
				})
			}
			// User redaction must not mutate the source or strip admin's stored data.
			require.Equal(t, &rate, group.LaunchRate)
			require.Equal(t, change, group.RecentRateChange)
			admin := GroupFromServiceAdmin(group)
			require.Equal(t, &rate, admin.LaunchRate)
			require.Equal(t, change, admin.RecentRateChange)
			serialized, err := json.Marshal(group)
			require.NoError(t, err)
			require.NotContains(t, string(serialized), "RateHintsAuthorized", "authorization must not survive service-model serialization/cache round trips")
		})
	}
}
