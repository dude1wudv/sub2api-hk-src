package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSubscriptionPlanSaleWindowFiltersAndUnlistsExpiredPlans(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	require.NoError(t, client.Schema.Create(ctx))

	expired, err := client.SubscriptionPlan.Create().
		SetGroupID(1).
		SetName("expired").
		SetPrice(5).
		SetValidityDays(1).
		SetPurchaseMode(SubscriptionPlanPurchaseModeBalance).
		SetForSale(true).
		SetSaleEndsAt(time.Now().Add(-time.Minute)).
		Save(ctx)
	require.NoError(t, err)

	available, err := client.SubscriptionPlan.Create().
		SetGroupID(1).
		SetName("available").
		SetPrice(5).
		SetValidityDays(1).
		SetPurchaseMode(SubscriptionPlanPurchaseModeBalance).
		SetForSale(true).
		SetSaleEndsAt(time.Now().Add(time.Minute)).
		Save(ctx)
	require.NoError(t, err)

	configService := &PaymentConfigService{entClient: client}
	plans, err := configService.ListPlansForSale(ctx)
	require.NoError(t, err)
	require.Len(t, plans, 1)
	require.Equal(t, available.ID, plans[0].ID)

	saleWindowService := NewSubscriptionPlanSaleWindowService(client, time.Minute)
	saleWindowService.runOnce()

	expired, err = client.SubscriptionPlan.Get(ctx, expired.ID)
	require.NoError(t, err)
	require.False(t, expired.ForSale)

	available, err = client.SubscriptionPlan.Get(ctx, available.ID)
	require.NoError(t, err)
	require.True(t, available.ForSale)
}
