package service

import (
	"context"
	"fmt"
	"time"

	"entgo.io/ent/dialect"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/paymentorder"
	"github.com/Wei-Shaw/sub2api/ent/subscriptionpurchaseclaim"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	subscriptionPurchaseClaimPending   = "PENDING"
	subscriptionPurchaseClaimSucceeded = "SUCCEEDED"
)

func reserveSubscriptionPurchaseClaim(ctx context.Context, client *dbent.Client, userID, groupID int64) error {
	purchased, err := client.PaymentOrder.Query().Where(
		paymentorder.UserIDEQ(userID),
		paymentorder.OrderTypeEQ(payment.OrderTypeSubscription),
		paymentorder.SubscriptionGroupIDEQ(groupID),
		paymentorder.PaidAtNotNil(),
	).Exist(ctx)
	if err != nil {
		return fmt.Errorf("check subscription purchase history: %w", err)
	}
	if purchased {
		return infraerrors.Conflict("SUBSCRIPTION_ALREADY_PURCHASED", "this subscription group can only be purchased once")
	}

	query := `INSERT INTO subscription_purchase_claims
		(user_id, subscription_group_id, status, created_at, updated_at)
		VALUES (?, ?, 'PENDING', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (user_id, subscription_group_id) DO NOTHING
		RETURNING id`
	args := []any{userID, groupID}
	if client.Driver().Dialect() == dialect.Postgres {
		query = `INSERT INTO subscription_purchase_claims
			(user_id, subscription_group_id, status, created_at, updated_at)
			VALUES ($1, $2, 'PENDING', NOW(), NOW())
			ON CONFLICT (user_id, subscription_group_id) DO NOTHING
			RETURNING id`
	}
	rows, err := client.QueryContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("reserve subscription purchase: %w", err)
	}
	defer func() { _ = rows.Close() }()
	if rows.Next() {
		var claimID int64
		if err := rows.Scan(&claimID); err != nil {
			return fmt.Errorf("read subscription purchase reservation: %w", err)
		}
		return nil
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("reserve subscription purchase: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close subscription purchase reservation result: %w", err)
	}

	claim, err := client.SubscriptionPurchaseClaim.Query().Where(
		subscriptionpurchaseclaim.UserIDEQ(userID),
		subscriptionpurchaseclaim.SubscriptionGroupIDEQ(groupID),
	).Only(ctx)
	if err != nil {
		return fmt.Errorf("load subscription purchase reservation: %w", err)
	}
	if claim.Status == subscriptionPurchaseClaimSucceeded {
		return infraerrors.Conflict("SUBSCRIPTION_ALREADY_PURCHASED", "this subscription group can only be purchased once")
	}
	return infraerrors.Conflict("SUBSCRIPTION_PURCHASE_IN_PROGRESS", "a purchase for this subscription group is already in progress")
}

func releaseStaleSubscriptionPurchaseClaims(ctx context.Context, client *dbent.Client, cutoff time.Time) (int64, error) {
	query := `DELETE FROM subscription_purchase_claims
		WHERE status = ?
		AND payment_order_id IN (
			SELECT id FROM payment_orders
			WHERE status IN (?, ?, ?)
			AND paid_at IS NULL
			AND updated_at <= ?
		)`
	args := []any{
		subscriptionPurchaseClaimPending,
		OrderStatusCancelled,
		OrderStatusExpired,
		OrderStatusFailed,
		cutoff,
	}
	if client.Driver().Dialect() == dialect.Postgres {
		query = `DELETE FROM subscription_purchase_claims
			WHERE status = $1
			AND payment_order_id IN (
				SELECT id FROM payment_orders
				WHERE status IN ($2, $3, $4)
				AND paid_at IS NULL
				AND updated_at <= $5
			)`
	}
	result, err := client.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("release stale subscription purchase reservations: %w", err)
	}
	released, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("count released subscription purchase reservations: %w", err)
	}
	return released, nil
}

func bindSubscriptionPurchaseClaim(ctx context.Context, client *dbent.Client, userID, groupID, orderID int64) error {
	updated, err := client.SubscriptionPurchaseClaim.Update().Where(
		subscriptionpurchaseclaim.UserIDEQ(userID),
		subscriptionpurchaseclaim.SubscriptionGroupIDEQ(groupID),
		subscriptionpurchaseclaim.PaymentOrderIDIsNil(),
	).SetPaymentOrderID(orderID).Save(ctx)
	if err != nil {
		return fmt.Errorf("bind subscription purchase reservation: %w", err)
	}
	if updated != 1 {
		return fmt.Errorf("bind subscription purchase reservation: expected one row, updated %d", updated)
	}
	return nil
}

func completeSubscriptionPurchaseClaim(ctx context.Context, client *dbent.Client, orderID int64) error {
	_, err := client.SubscriptionPurchaseClaim.Update().Where(
		subscriptionpurchaseclaim.PaymentOrderIDEQ(orderID),
	).SetStatus(subscriptionPurchaseClaimSucceeded).Save(ctx)
	if err != nil {
		return fmt.Errorf("complete subscription purchase reservation: %w", err)
	}
	return nil
}
