//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/ent/paymentorder"
	"github.com/Wei-Shaw/sub2api/ent/usersubscription"
	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"
)

func TestPurchaseSubscriptionWithBalanceDeductsAndActivates(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	require.NoError(t, client.Schema.Create(ctx))

	account, err := client.User.Create().
		SetEmail("balance-buyer@example.com").
		SetUsername("balance-buyer").
		SetPasswordHash("hash").
		SetBalance(10).
		SetStatus(payment.EntityStatusActive).
		Save(ctx)
	require.NoError(t, err)

	planGroup, err := client.Group.Create().
		SetName("weekday-plan").
		SetSubscriptionType(domain.SubscriptionTypeSubscription).
		SetRateMultiplier(1).
		SetDailyLimitUsd(200).
		Save(ctx)
	require.NoError(t, err)

	plan, err := client.SubscriptionPlan.Create().
		SetGroupID(planGroup.ID).
		SetName("weekday-card").
		SetPrice(5).
		SetValidityDays(1).
		SetValidityUnit("day").
		SetPurchaseMode(SubscriptionPlanPurchaseModeBalance).
		SetForSale(true).
		SetSaleEndsAt(time.Now().Add(time.Hour)).
		Save(ctx)
	require.NoError(t, err)

	svc := NewPaymentService(client, payment.NewRegistry(), nil, nil, nil, nil, nil, nil, nil)
	before := time.Now()
	result, err := svc.PurchaseSubscriptionWithBalance(ctx, account.ID, plan.ID)
	require.NoError(t, err)
	require.Equal(t, 5.0, result.Balance)
	require.False(t, result.SubscriptionWasExtended)
	require.WithinDuration(t, before.Add(24*time.Hour), result.SubscriptionExpiresAt, 2*time.Second)

	account, err = client.User.Get(ctx, account.ID)
	require.NoError(t, err)
	require.Equal(t, 5.0, account.Balance)

	order, err := client.PaymentOrder.Query().Where(paymentorder.IDEQ(result.OrderID)).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, payment.OrderTypeBalance, order.PaymentType)
	require.Equal(t, payment.OrderTypeSubscription, order.OrderType)
	require.Equal(t, OrderStatusCompleted, order.Status)
	require.Equal(t, plan.ID, *order.PlanID)
	require.Equal(t, planGroup.ID, *order.SubscriptionGroupID)

	subscription, err := client.UserSubscription.Query().Where(
		usersubscription.UserIDEQ(account.ID),
		usersubscription.GroupIDEQ(planGroup.ID),
	).Only(ctx)
	require.NoError(t, err)
	require.Equal(t, SubscriptionStatusActive, subscription.Status)
	require.WithinDuration(t, result.SubscriptionExpiresAt, subscription.ExpiresAt, time.Second)

	renewal, err := svc.PurchaseSubscriptionWithBalance(ctx, account.ID, plan.ID)
	require.NoError(t, err)
	require.True(t, renewal.SubscriptionWasExtended)
	require.Equal(t, 0.0, renewal.Balance)
	require.True(t, renewal.SubscriptionExpiresAt.After(result.SubscriptionExpiresAt))

	_, err = svc.PurchaseSubscriptionWithBalance(ctx, account.ID, plan.ID)
	require.Error(t, err)
	count, err := client.PaymentOrder.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, count)
}

func TestPurchaseSubscriptionWithBalanceRejectsSecondLimitedPlanInSameGroup(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	require.NoError(t, client.Schema.Create(ctx))

	account, err := client.User.Create().
		SetEmail("one-purchase@example.com").
		SetUsername("one-purchase").
		SetPasswordHash("hash").
		SetBalance(20).
		SetStatus(payment.EntityStatusActive).
		Save(ctx)
	require.NoError(t, err)
	planGroup, err := client.Group.Create().
		SetName("one-purchase-group").
		SetSubscriptionType(domain.SubscriptionTypeSubscription).
		SetStatus(payment.EntityStatusActive).
		Save(ctx)
	require.NoError(t, err)

	createPlan := func(name string) int64 {
		plan, createErr := client.SubscriptionPlan.Create().
			SetGroupID(planGroup.ID).
			SetName(name).
			SetPrice(5).
			SetValidityDays(1).
			SetValidityUnit("day").
			SetPurchaseMode(SubscriptionPlanPurchaseModeBalance).
			SetOnePurchasePerUser(true).
			SetForSale(true).
			Save(ctx)
		require.NoError(t, createErr)
		return plan.ID
	}

	svc := NewPaymentService(client, payment.NewRegistry(), nil, nil, nil, nil, nil, nil, nil)
	_, err = svc.PurchaseSubscriptionWithBalance(ctx, account.ID, createPlan("first"))
	require.NoError(t, err)
	_, err = svc.PurchaseSubscriptionWithBalance(ctx, account.ID, createPlan("second"))
	require.ErrorContains(t, err, "only be purchased once")

	account, err = client.User.Get(ctx, account.ID)
	require.NoError(t, err)
	require.Equal(t, 15.0, account.Balance)
	orders, err := client.PaymentOrder.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, orders)
	claims, err := client.SubscriptionPurchaseClaim.Query().Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, claims)
}

func TestUpdatePlanCanClearSaleEndsAt(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	require.NoError(t, client.Schema.Create(ctx))

	plan, err := client.SubscriptionPlan.Create().
		SetGroupID(1).
		SetName("limited").
		SetPrice(5).
		SetValidityDays(1).
		SetPurchaseMode(SubscriptionPlanPurchaseModeBalance).
		SetForSale(true).
		SetSaleEndsAt(time.Now().Add(time.Hour)).
		Save(ctx)
	require.NoError(t, err)

	clearSaleEndsAt := true
	updated, err := NewPaymentConfigService(client, nil, nil).UpdatePlan(ctx, plan.ID, UpdatePlanRequest{
		ClearSaleEndsAt: &clearSaleEndsAt,
	})
	require.NoError(t, err)
	require.Nil(t, updated.SaleEndsAt)
}

func TestPurchaseSubscriptionWithBalanceRejectsExpiredPlan(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	require.NoError(t, client.Schema.Create(ctx))

	account, err := client.User.Create().
		SetEmail("expired-plan-buyer@example.com").
		SetUsername("expired-plan-buyer").
		SetPasswordHash("hash").
		SetBalance(10).
		Save(ctx)
	require.NoError(t, err)
	planGroup, err := client.Group.Create().
		SetName("expired-plan-group").
		SetSubscriptionType(domain.SubscriptionTypeSubscription).
		Save(ctx)
	require.NoError(t, err)
	plan, err := client.SubscriptionPlan.Create().
		SetGroupID(planGroup.ID).
		SetName("expired-plan").
		SetPrice(5).
		SetValidityDays(1).
		SetPurchaseMode(SubscriptionPlanPurchaseModeBalance).
		SetForSale(true).
		SetSaleEndsAt(time.Now().Add(-time.Second)).
		Save(ctx)
	require.NoError(t, err)

	svc := NewPaymentService(client, payment.NewRegistry(), nil, nil, nil, nil, nil, nil, nil)
	_, err = svc.PurchaseSubscriptionWithBalance(ctx, account.ID, plan.ID)
	require.Error(t, err)

	account, err = client.User.Get(ctx, account.ID)
	require.NoError(t, err)
	require.Equal(t, 10.0, account.Balance)
	count, err := client.PaymentOrder.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count)
}
