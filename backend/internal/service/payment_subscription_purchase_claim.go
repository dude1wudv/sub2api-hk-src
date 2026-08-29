package service

import (
	"context"
	"fmt"

	"entgo.io/ent/dialect"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/paymentorder"
	"github.com/Wei-Shaw/sub2api/ent/subscriptionpurchaseclaim"
	"github.com/Wei-Shaw/sub2api/ent/usersubscription"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	subscriptionPurchaseClaimPending   = "PENDING"
	subscriptionPurchaseClaimSucceeded = "SUCCEEDED"
)

func reserveSubscriptionPurchaseClaim(ctx context.Context, client *dbent.Client, userID, groupID int64) error {
	hasSubscription, err := client.UserSubscription.Query().Where(usersubscription.UserIDEQ(userID), usersubscription.GroupIDEQ(groupID)).Exist(ctx)
	if err != nil {
		return fmt.Errorf("check existing subscription: %w", err)
	}
	if hasSubscription {
		return infraerrors.Conflict("SUBSCRIPTION_ALREADY_PURCHASED", "this subscription group can only be purchased once")
	}
	purchased, err := client.PaymentOrder.Query().Where(paymentorder.UserIDEQ(userID), paymentorder.OrderTypeEQ(payment.OrderTypeSubscription), paymentorder.SubscriptionGroupIDEQ(groupID), paymentorder.PaidAtNotNil()).Exist(ctx)
	if err != nil {
		return fmt.Errorf("check subscription purchase history: %w", err)
	}
	if purchased {
		return infraerrors.Conflict("SUBSCRIPTION_ALREADY_PURCHASED", "this subscription group can only be purchased once")
	}
	query, args := `INSERT INTO subscription_purchase_claims (user_id, subscription_group_id, status, created_at, updated_at) VALUES (?, ?, 'PENDING', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP) ON CONFLICT (user_id, subscription_group_id) DO NOTHING RETURNING id`, []any{userID, groupID}
	if client.Driver().Dialect() == dialect.Postgres {
		query = `INSERT INTO subscription_purchase_claims (user_id, subscription_group_id, status, created_at, updated_at) VALUES ($1, $2, 'PENDING', NOW(), NOW()) ON CONFLICT (user_id, subscription_group_id) DO NOTHING RETURNING id`
	}
	rows, err := client.QueryContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("reserve subscription purchase: %w", err)
	}
	defer func() { _ = rows.Close() }()
	if rows.Next() {
		return rows.Scan(new(int64))
	}
	if err := rows.Err(); err != nil {
		return err
	}
	claim, err := client.SubscriptionPurchaseClaim.Query().Where(subscriptionpurchaseclaim.UserIDEQ(userID), subscriptionpurchaseclaim.SubscriptionGroupIDEQ(groupID)).Only(ctx)
	if err != nil {
		return fmt.Errorf("load subscription purchase reservation: %w", err)
	}
	if claim.Status == subscriptionPurchaseClaimSucceeded {
		return infraerrors.Conflict("SUBSCRIPTION_ALREADY_PURCHASED", "this subscription group can only be purchased once")
	}
	return infraerrors.Conflict("SUBSCRIPTION_PURCHASE_IN_PROGRESS", "a purchase for this subscription group is already in progress")
}

func bindSubscriptionPurchaseClaim(ctx context.Context, client *dbent.Client, userID, groupID, orderID int64) error {
	n, err := client.SubscriptionPurchaseClaim.Update().Where(subscriptionpurchaseclaim.UserIDEQ(userID), subscriptionpurchaseclaim.SubscriptionGroupIDEQ(groupID), subscriptionpurchaseclaim.PaymentOrderIDIsNil()).SetPaymentOrderID(orderID).Save(ctx)
	if err != nil {
		return fmt.Errorf("bind subscription purchase reservation: %w", err)
	}
	if n != 1 {
		return fmt.Errorf("bind subscription purchase reservation: expected one row, updated %d", n)
	}
	return nil
}

func completeSubscriptionPurchaseClaim(ctx context.Context, client *dbent.Client, orderID int64) error {
	_, err := client.SubscriptionPurchaseClaim.Update().Where(subscriptionpurchaseclaim.PaymentOrderIDEQ(orderID)).SetStatus(subscriptionPurchaseClaimSucceeded).Save(ctx)
	return err
}

// releaseSubscriptionPurchaseClaim allows a user to retry a one-time purchase
// when the order reached a terminal state before payment was confirmed.
// Succeeded claims are deliberately immutable.
func releaseSubscriptionPurchaseClaim(ctx context.Context, client *dbent.Client, orderID int64) error {
	_, err := client.SubscriptionPurchaseClaim.Delete().
		Where(
			subscriptionpurchaseclaim.PaymentOrderIDEQ(orderID),
			subscriptionpurchaseclaim.StatusEQ(subscriptionPurchaseClaimPending),
		).
		Exec(ctx)
	return err
}

// transitionPendingOrderAndReleaseClaim commits the unpaid terminal status and
// pending-claim release together. If either write fails, neither is visible.
func transitionPendingOrderAndReleaseClaim(ctx context.Context, client *dbent.Client, orderID int64, status string) (int, error) {
	tx, err := client.Tx(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	txClient := tx.Client()
	updated, err := txClient.PaymentOrder.Update().
		Where(paymentorder.IDEQ(orderID), paymentorder.StatusEQ(OrderStatusPending)).
		SetStatus(status).
		Save(ctx)
	if err != nil {
		return 0, err
	}
	if updated > 0 {
		if err := releaseSubscriptionPurchaseClaim(ctx, txClient, orderID); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return updated, nil
}
