//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSubscriptionAvailability(t *testing.T) {
	for _, tt := range []struct {
		name                                  string
		owned, expired, sale, external, ended bool
		want                                  SubscriptionAvailability
	}{
		{name: "empty"},
		{name: "sale only", sale: true, want: SubscriptionAvailability{HasSalePlans: true}},
		{name: "current without sale", owned: true, want: SubscriptionAvailability{HasActiveSubscriptions: true, HasSubscriptionHistory: true}},
		{name: "history without sale", owned: true, expired: true, want: SubscriptionAvailability{HasSubscriptionHistory: true}},
		{name: "external checkout retired", sale: true, external: true},
		{name: "sale ended", sale: true, ended: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			client := newPaymentOrderLifecycleTestClient(t)
			u := client.User.Create().SetEmail("availability@example.test").SetPasswordHash("fixture").SaveX(ctx)
			g := client.Group.Create().SetName("operator-pool").SetPlatform("openai").SetSubscriptionType("subscription").SaveX(ctx)
			if tt.sale {
				mode := "balance"
				if tt.external {
					mode = "external"
				}
				p := client.SubscriptionPlan.Create().SetName("plan").SetGroupID(g.ID).SetPrice(10).SetValidityDays(30).SetPurchaseMode(mode)
				if tt.ended {
					p.SetSaleEndsAt(time.Now().Add(-time.Hour))
				}
				p.SaveX(ctx)
			}
			if tt.owned {
				expires := time.Now().Add(time.Hour)
				if tt.expired {
					expires = time.Now().Add(-time.Hour)
				}
				client.UserSubscription.Create().SetUserID(u.ID).SetGroupID(g.ID).SetStartsAt(time.Now().Add(-2 * time.Hour)).SetExpiresAt(expires).SaveX(ctx)
			}
			got, err := (&SubscriptionService{entClient: client}).Availability(ctx, u.ID)
			require.NoError(t, err)
			require.Equal(t, tt.want, *got)
			other := client.User.Create().SetEmail("other@example.test").SetPasswordHash("fixture").SaveX(ctx)
			got, err = (&SubscriptionService{entClient: client}).Availability(ctx, other.ID)
			require.NoError(t, err)
			require.False(t, got.HasActiveSubscriptions)
			require.False(t, got.HasSubscriptionHistory)
		})
	}
}

func TestSubscriptionAvailabilityDatabaseFailureIsNotEmpty(t *testing.T) {
	client := newPaymentOrderLifecycleTestClient(t)
	require.NoError(t, client.Close())
	got, err := (&SubscriptionService{entClient: client}).Availability(context.Background(), 1)
	require.Error(t, err)
	require.Nil(t, got)
}
