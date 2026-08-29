//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/ent/subscriptionpurchaseclaim"
	"github.com/stretchr/testify/require"
)

func TestReleaseSubscriptionPurchaseClaimDeletesPendingClaim(t *testing.T) {
	ctx := context.Background()
	client := newPaymentOrderLifecycleTestClient(t)

	claim := client.SubscriptionPurchaseClaim.Create().
		SetUserID(101).
		SetSubscriptionGroupID(202).
		SetPaymentOrderID(303).
		SetStatus(subscriptionPurchaseClaimPending).
		SaveX(ctx)

	require.NoError(t, releaseSubscriptionPurchaseClaim(ctx, client, 303))
	require.False(t, client.SubscriptionPurchaseClaim.Query().Where(
		subscriptionpurchaseclaim.IDEQ(claim.ID),
	).ExistX(ctx))
}

func TestReleaseSubscriptionPurchaseClaimPreservesSucceededClaim(t *testing.T) {
	ctx := context.Background()
	client := newPaymentOrderLifecycleTestClient(t)

	claim := client.SubscriptionPurchaseClaim.Create().
		SetUserID(111).
		SetSubscriptionGroupID(222).
		SetPaymentOrderID(333).
		SetStatus(subscriptionPurchaseClaimSucceeded).
		SaveX(ctx)

	require.NoError(t, releaseSubscriptionPurchaseClaim(ctx, client, 333))
	require.True(t, client.SubscriptionPurchaseClaim.Query().Where(
		subscriptionpurchaseclaim.IDEQ(claim.ID),
	).ExistX(ctx))
}
