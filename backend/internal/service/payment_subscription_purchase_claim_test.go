//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/payment"
	"github.com/stretchr/testify/require"
)

func TestReserveSubscriptionPurchaseClaimRejectsExistingClaimAndPaidHistory(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	require.NoError(t, client.Schema.Create(ctx))

	account, err := client.User.Create().
		SetEmail("claimed-purchase@example.com").
		SetUsername("claimed-purchase").
		SetPasswordHash("hash").
		Save(ctx)
	require.NoError(t, err)

	require.NoError(t, reserveSubscriptionPurchaseClaim(ctx, client, account.ID, 123))
	require.ErrorContains(t, reserveSubscriptionPurchaseClaim(ctx, client, account.ID, 123), "already in progress")

	other, err := client.User.Create().
		SetEmail("paid-purchase@example.com").
		SetUsername("paid-purchase").
		SetPasswordHash("hash").
		Save(ctx)
	require.NoError(t, err)
	_, err = client.PaymentOrder.Create().
		SetUserID(other.ID).
		SetUserEmail(other.Email).
		SetUserName(other.Username).
		SetAmount(5).
		SetPayAmount(5).
		SetFeeRate(0).
		SetRechargeCode("paid-claim-test").
		SetOutTradeNo("paid-claim-test-order").
		SetPaymentType("test").
		SetPaymentTradeNo("trade").
		SetOrderType(payment.OrderTypeSubscription).
		SetPlanID(99).
		SetSubscriptionGroupID(123).
		SetSubscriptionDays(1).
		SetStatus(OrderStatusCompleted).
		SetExpiresAt(time.Now()).
		SetPaidAt(time.Now()).
		SetClientIP("").
		SetSrcHost("").
		Save(ctx)
	require.NoError(t, err)

	require.ErrorContains(t, reserveSubscriptionPurchaseClaim(ctx, client, other.ID, 123), "only be purchased once")
}

func TestReleaseStaleSubscriptionPurchaseClaims(t *testing.T) {
	ctx := context.Background()
	client := newPaymentConfigServiceTestClient(t)
	require.NoError(t, client.Schema.Create(ctx))

	account, err := client.User.Create().
		SetEmail("stale-claim@example.com").
		SetUsername("stale-claim").
		SetPasswordHash("hash").
		Save(ctx)
	require.NoError(t, err)

	old := time.Now().Add(-10 * time.Minute)
	order, err := client.PaymentOrder.Create().
		SetUserID(account.ID).
		SetUserEmail(account.Email).
		SetUserName(account.Username).
		SetAmount(5).
		SetPayAmount(5).
		SetFeeRate(0).
		SetRechargeCode("claim-test").
		SetOutTradeNo("claim-test-order").
		SetPaymentType("test").
		SetPaymentTradeNo("").
		SetOrderType(payment.OrderTypeSubscription).
		SetPlanID(99).
		SetSubscriptionGroupID(123).
		SetSubscriptionDays(1).
		SetStatus(OrderStatusFailed).
		SetExpiresAt(old).
		SetClientIP("").
		SetSrcHost("").
		SetUpdatedAt(old).
		Save(ctx)
	require.NoError(t, err)

	_, err = client.SubscriptionPurchaseClaim.Create().
		SetUserID(account.ID).
		SetSubscriptionGroupID(123).
		SetPaymentOrderID(order.ID).
		SetStatus(subscriptionPurchaseClaimPending).
		Save(ctx)
	require.NoError(t, err)

	released, err := releaseStaleSubscriptionPurchaseClaims(ctx, client, time.Now().Add(-5*time.Minute))
	require.NoError(t, err)
	require.Equal(t, int64(1), released)
	count, err := client.SubscriptionPurchaseClaim.Query().Count(ctx)
	require.NoError(t, err)
	require.Zero(t, count)
}
