package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/ent/paymentorder"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/shopspring/decimal"
)

// ConfirmManualAlipayPayment transitions a manually verified static-code order through normal fulfillment.
func (s *PaymentService) ConfirmManualAlipayPayment(ctx context.Context, orderID int64, receivedAmount, note, operator string) error {
	order, err := s.entClient.PaymentOrder.Get(ctx, orderID)
	if err != nil {
		return infraerrors.NotFound("NOT_FOUND", "order not found")
	}
	if order.ProviderKey == nil || *order.ProviderKey != payment.TypeAlipayManual {
		return infraerrors.BadRequest("INVALID_PAYMENT_METHOD", "order is not a manual Alipay payment")
	}
	if order.Status == OrderStatusCompleted || order.Status == OrderStatusPaid || order.Status == OrderStatusRecharging {
		return nil
	}
	if order.Status != OrderStatusPending {
		return infraerrors.BadRequest("INVALID_STATUS", "manual payment order is no longer pending")
	}
	now := subscriptionBusinessNow()
	if !order.ExpiresAt.After(now) {
		return infraerrors.BadRequest("ORDER_EXPIRED", "manual payment order has expired")
	}
	paid, err := decimal.NewFromString(strings.TrimSpace(receivedAmount))
	if err != nil || paid.LessThanOrEqual(decimal.Zero) || paid.Exponent() < -2 {
		return infraerrors.BadRequest("INVALID_AMOUNT", "received amount must be a positive CNY amount with at most two decimal places")
	}
	expected := decimal.NewFromFloat(order.PayAmount).Round(2)
	if !paid.Equal(expected) {
		return infraerrors.BadRequest("PAYMENT_AMOUNT_MISMATCH", fmt.Sprintf("received amount must exactly equal %s", expected.StringFixed(2)))
	}
	updated, err := s.entClient.PaymentOrder.Update().Where(paymentorder.IDEQ(order.ID), paymentorder.StatusEQ(OrderStatusPending), paymentorder.ExpiresAtGT(now)).SetStatus(OrderStatusPaid).SetPaidAt(now).SetPaymentTradeNo("manual:" + operator).ClearFailedAt().ClearFailedReason().Save(ctx)
	if err != nil {
		return fmt.Errorf("mark manual payment paid: %w", err)
	}
	if updated == 0 {
		return s.ConfirmManualAlipayPayment(ctx, orderID, receivedAmount, note, operator)
	}
	s.writeAuditLog(ctx, order.ID, "MANUAL_PAYMENT_CONFIRMED", operator, map[string]any{"receivedAmount": paid.StringFixed(2), "note": note, "confirmedAt": now.UTC().Format(time.RFC3339)})
	return s.executeFulfillment(ctx, order.ID)
}
