package service

import (
	"context"
	"fmt"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/group"
	"github.com/Wei-Shaw/sub2api/ent/subscriptionplan"
	"github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/ent/usersubscription"
	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/Wei-Shaw/sub2api/internal/payment"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type BalanceSubscriptionPurchaseResult struct {
	OrderID                 int64     `json:"order_id"`
	Balance                 float64   `json:"balance"`
	SubscriptionExpiresAt   time.Time `json:"subscription_expires_at"`
	SubscriptionWasExtended bool      `json:"subscription_was_extended"`
}

func (s *PaymentService) PurchaseSubscriptionWithBalance(ctx context.Context, userID, planID int64) (*BalanceSubscriptionPurchaseResult, error) {
	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin balance subscription transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	client, now := tx.Client(), subscriptionBusinessNow()
	plan, err := client.SubscriptionPlan.Query().Where(subscriptionplan.IDEQ(planID), subscriptionplan.ForSaleEQ(true), subscriptionplan.Or(subscriptionplan.PurchaseModeEQ(SubscriptionPlanPurchaseModeBalance), subscriptionplan.PurchaseModeEQ(SubscriptionPlanPurchaseModeBoth)), subscriptionplan.Or(subscriptionplan.SaleEndsAtIsNil(), subscriptionplan.SaleEndsAtGT(now)), subscriptionplan.Or(subscriptionplan.FixedExpiresAtIsNil(), subscriptionplan.FixedExpiresAtGT(now))).Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, infraerrors.NotFound("PLAN_NOT_AVAILABLE", "plan not found or not for sale")
		}
		return nil, err
	}
	planGroup, err := client.Group.Query().Where(group.IDEQ(plan.GroupID), group.StatusEQ(payment.EntityStatusActive), group.SubscriptionTypeEQ(domain.SubscriptionTypeSubscription)).Only(ctx)
	if err != nil {
		return nil, infraerrors.NotFound("GROUP_NOT_FOUND", "subscription group is no longer available")
	}
	account, err := client.User.Query().Where(user.IDEQ(userID), user.StatusEQ(payment.EntityStatusActive)).Only(ctx)
	if err != nil {
		return nil, infraerrors.Forbidden("USER_INACTIVE", "user account is disabled")
	}
	if plan.OnePurchasePerUser {
		if err := reserveSubscriptionPurchaseClaim(ctx, client, userID, planGroup.ID); err != nil {
			return nil, err
		}
	}
	updated, err := client.User.Update().Where(user.IDEQ(userID), user.BalanceGTE(plan.Price)).AddBalance(-plan.Price).Save(ctx)
	if err != nil {
		return nil, err
	}
	if updated != 1 {
		return nil, infraerrors.BadRequest("INSUFFICIENT_BALANCE", "insufficient balance to purchase this subscription")
	}
	days := psComputeValidityDays(plan.ValidityDays, plan.ValidityUnit)
	sub, extended, err := purchaseBalanceSubscriptionTerm(ctx, client, userID, planGroup.ID, days, plan.FixedExpiresAt, now)
	if err != nil {
		return nil, err
	}
	order, err := client.PaymentOrder.Create().SetUserID(account.ID).SetUserEmail(account.Email).SetUserName(account.Username).SetAmount(plan.Price).SetPayAmount(plan.Price).SetRechargeCode("").SetOutTradeNo("").SetPaymentType(payment.OrderTypeBalance).SetPaymentTradeNo("").SetOrderType(payment.OrderTypeSubscription).SetPlanID(plan.ID).SetSubscriptionGroupID(planGroup.ID).SetSubscriptionDays(days).SetSubscriptionExpiresAt(sub.ExpiresAt).SetStatus(OrderStatusCompleted).SetExpiresAt(now).SetPaidAt(now).SetCompletedAt(now).SetClientIP("").SetSrcHost("").Save(ctx)
	if err != nil {
		return nil, err
	}
	if plan.OnePurchasePerUser {
		if err := bindSubscriptionPurchaseClaim(ctx, client, userID, planGroup.ID, order.ID); err != nil {
			return nil, err
		}
		if err := completeSubscriptionPurchaseClaim(ctx, client, order.ID); err != nil {
			return nil, err
		}
	}
	if _, err := client.PaymentAuditLog.Create().SetOrderID(fmt.Sprint(order.ID)).SetAction("BALANCE_SUBSCRIPTION_PURCHASED").SetDetail(fmt.Sprintf(`{"planID":%d,"groupID":%d}`, plan.ID, planGroup.ID)).SetOperator(fmt.Sprintf("user:%d", userID)).Save(ctx); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	s.invalidateBalanceSubscriptionCaches(userID, planGroup.ID)
	return &BalanceSubscriptionPurchaseResult{OrderID: order.ID, Balance: account.Balance - plan.Price, SubscriptionExpiresAt: sub.ExpiresAt, SubscriptionWasExtended: extended}, nil
}

func purchaseBalanceSubscriptionTerm(ctx context.Context, client *dbent.Client, userID, groupID int64, days int, fixedExpiresAt *time.Time, now time.Time) (*dbent.UserSubscription, bool, error) {
	existing, err := client.UserSubscription.Query().Where(usersubscription.UserIDEQ(userID), usersubscription.GroupIDEQ(groupID)).Only(ctx)
	if err != nil && !dbent.IsNotFound(err) {
		return nil, false, err
	}
	if dbent.IsNotFound(err) {
		expiresAt := ResolveSubscriptionExpiry(now, days, fixedExpiresAt, nil)
		sub, err := client.UserSubscription.Create().SetUserID(userID).SetGroupID(groupID).SetStartsAt(now).SetExpiresAt(expiresAt).SetStatus(SubscriptionStatusActive).SetAssignedAt(now).SetNotes("balance subscription purchase").Save(ctx)
		return sub, false, err
	}
	if err := validateFixedSubscriptionExpiry(now, fixedExpiresAt, &existing.ExpiresAt); err != nil {
		return nil, false, err
	}
	expiresAt := ResolveSubscriptionExpiry(now, days, fixedExpiresAt, &existing.ExpiresAt)
	update := client.UserSubscription.UpdateOneID(existing.ID).SetExpiresAt(expiresAt).SetStatus(SubscriptionStatusActive).SetNotes(balanceSubscriptionNotes(existing.Notes))
	if !existing.ExpiresAt.After(now) {
		update.SetStartsAt(now).SetDailyUsageUsd(0).SetWeeklyUsageUsd(0).SetMonthlyUsageUsd(0)
	}
	sub, err := update.Save(ctx)
	return sub, true, err
}

func balanceSubscriptionNotes(notes *string) string {
	if notes == nil || *notes == "" {
		return "balance subscription purchase"
	}
	return *notes + "\nbalance subscription purchase"
}

func (s *PaymentService) invalidateBalanceSubscriptionCaches(userID, groupID int64) {
	if s.subscriptionSvc != nil {
		_ = s.subscriptionSvc.invalidateSubscriptionCaches(userID, groupID)
	}
}
