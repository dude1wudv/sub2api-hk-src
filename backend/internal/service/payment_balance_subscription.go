package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
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
	if s == nil || s.entClient == nil {
		return nil, infraerrors.InternalServer("BALANCE_SUBSCRIPTION_UNAVAILABLE", "balance subscription purchase is unavailable")
	}

	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin balance subscription transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now()
	client := tx.Client()
	plan, err := client.SubscriptionPlan.Query().Where(
		subscriptionplan.IDEQ(planID),
		subscriptionplan.ForSaleEQ(true),
		subscriptionplan.PurchaseModeEQ(SubscriptionPlanPurchaseModeBalance),
		subscriptionplan.Or(
			subscriptionplan.SaleEndsAtIsNil(),
			subscriptionplan.SaleEndsAtGT(now),
		),
	).Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, infraerrors.NotFound("PLAN_NOT_AVAILABLE", "plan not found or not for sale")
		}
		return nil, fmt.Errorf("query balance subscription plan: %w", err)
	}

	planGroup, err := client.Group.Query().Where(
		group.IDEQ(plan.GroupID),
		group.StatusEQ(payment.EntityStatusActive),
		group.SubscriptionTypeEQ(domain.SubscriptionTypeSubscription),
	).Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, infraerrors.NotFound("GROUP_NOT_FOUND", "subscription group is no longer available")
		}
		return nil, fmt.Errorf("query subscription group: %w", err)
	}

	account, err := client.User.Query().Where(user.IDEQ(userID), user.StatusEQ(payment.EntityStatusActive)).Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, infraerrors.Forbidden("USER_INACTIVE", "user account is disabled")
		}
		return nil, fmt.Errorf("query purchasing user: %w", err)
	}
	if plan.OnePurchasePerUser {
		if err := reserveSubscriptionPurchaseClaim(ctx, client, userID, planGroup.ID); err != nil {
			return nil, err
		}
	}

	updated, err := client.User.Update().
		Where(user.IDEQ(userID), user.BalanceGTE(plan.Price)).
		AddBalance(-plan.Price).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("deduct subscription balance: %w", err)
	}
	if updated != 1 {
		return nil, infraerrors.BadRequest("INSUFFICIENT_BALANCE", "insufficient balance to purchase this subscription")
	}

	validityDays := psComputeValidityDays(plan.ValidityDays, plan.ValidityUnit)
	order, err := client.PaymentOrder.Create().
		SetUserID(account.ID).
		SetUserEmail(account.Email).
		SetUserName(account.Username).
		SetAmount(plan.Price).
		SetPayAmount(plan.Price).
		SetFeeRate(0).
		SetRechargeCode("").
		SetOutTradeNo("").
		SetPaymentType(payment.OrderTypeBalance).
		SetPaymentTradeNo("").
		SetOrderType(payment.OrderTypeSubscription).
		SetPlanID(plan.ID).
		SetSubscriptionGroupID(planGroup.ID).
		SetSubscriptionDays(validityDays).
		SetStatus(OrderStatusCompleted).
		SetExpiresAt(now).
		SetPaidAt(now).
		SetCompletedAt(now).
		SetClientIP("").
		SetSrcHost("").
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create balance subscription order: %w", err)
	}
	if plan.OnePurchasePerUser {
		if err := bindSubscriptionPurchaseClaim(ctx, client, userID, planGroup.ID, order.ID); err != nil {
			return nil, err
		}
	}

	note := fmt.Sprintf("balance subscription order %d", order.ID)
	subscription, extended, err := purchaseBalanceSubscriptionTerm(ctx, client, userID, planGroup.ID, validityDays, now, note)
	if err != nil {
		return nil, err
	}

	detail, _ := json.Marshal(map[string]any{
		"planID":         plan.ID,
		"groupID":        planGroup.ID,
		"price":          plan.Price,
		"validityDays":   validityDays,
		"subscriptionID": subscription.ID,
	})
	if _, err := client.PaymentAuditLog.Create().
		SetOrderID(strconv.FormatInt(order.ID, 10)).
		SetAction("BALANCE_SUBSCRIPTION_PURCHASED").
		SetDetail(string(detail)).
		SetOperator(fmt.Sprintf("user:%d", userID)).
		Save(ctx); err != nil {
		return nil, fmt.Errorf("write balance subscription audit: %w", err)
	}
	if plan.OnePurchasePerUser {
		if err := completeSubscriptionPurchaseClaim(ctx, client, order.ID); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit balance subscription transaction: %w", err)
	}

	s.invalidateBalanceSubscriptionCaches(userID, planGroup.ID)
	return &BalanceSubscriptionPurchaseResult{
		OrderID:                 order.ID,
		Balance:                 account.Balance - plan.Price,
		SubscriptionExpiresAt:   subscription.ExpiresAt,
		SubscriptionWasExtended: extended,
	}, nil
}

func purchaseBalanceSubscriptionTerm(ctx context.Context, client *dbent.Client, userID, groupID int64, validityDays int, now time.Time, note string) (*dbent.UserSubscription, bool, error) {
	existing, err := client.UserSubscription.Query().Where(
		usersubscription.UserIDEQ(userID),
		usersubscription.GroupIDEQ(groupID),
	).Only(ctx)
	if err != nil && !dbent.IsNotFound(err) {
		return nil, false, fmt.Errorf("query existing subscription: %w", err)
	}

	if dbent.IsNotFound(err) {
		expiresAt := now.AddDate(0, 0, validityDays)
		subscription, createErr := client.UserSubscription.Create().
			SetUserID(userID).
			SetGroupID(groupID).
			SetStartsAt(now).
			SetExpiresAt(expiresAt).
			SetStatus(SubscriptionStatusActive).
			SetAssignedAt(now).
			SetNotes(note).
			Save(ctx)
		if createErr != nil {
			return nil, false, fmt.Errorf("create balance subscription: %w", createErr)
		}
		return subscription, false, nil
	}

	if existing.ExpiresAt.After(now) {
		expiresAt := existing.ExpiresAt.AddDate(0, 0, validityDays)
		existingNotes := ""
		if existing.Notes != nil {
			existingNotes = *existing.Notes
		}
		update := client.UserSubscription.UpdateOneID(existing.ID).
			SetExpiresAt(expiresAt).
			SetNotes(appendSubscriptionNotes(existingNotes, note))
		if existing.Status != SubscriptionStatusActive {
			update.SetStatus(SubscriptionStatusActive)
		}
		subscription, updateErr := update.Save(ctx)
		if updateErr != nil {
			return nil, false, fmt.Errorf("extend balance subscription: %w", updateErr)
		}
		return subscription, true, nil
	}

	windowStart := startOfDay(now)
	expiresAt := now.AddDate(0, 0, validityDays)
	existingNotes := ""
	if existing.Notes != nil {
		existingNotes = *existing.Notes
	}
	subscription, updateErr := client.UserSubscription.UpdateOneID(existing.ID).
		SetStartsAt(now).
		SetExpiresAt(expiresAt).
		SetStatus(SubscriptionStatusActive).
		SetDailyWindowStart(windowStart).
		SetWeeklyWindowStart(windowStart).
		SetMonthlyWindowStart(windowStart).
		SetDailyUsageUsd(0).
		SetWeeklyUsageUsd(0).
		SetMonthlyUsageUsd(0).
		SetNotes(appendSubscriptionNotes(existingNotes, note)).
		Save(ctx)
	if updateErr != nil {
		return nil, false, fmt.Errorf("renew balance subscription: %w", updateErr)
	}
	return subscription, true, nil
}

func (s *PaymentService) invalidateBalanceSubscriptionCaches(userID, groupID int64) {
	if s.subscriptionSvc == nil {
		return
	}
	if err := s.subscriptionSvc.invalidateSubscriptionCaches(userID, groupID); err != nil {
		slog.Warn("invalidate balance subscription cache failed", "userID", userID, "groupID", groupID, "error", err)
	}
	if s.subscriptionSvc.billingCacheService == nil {
		return
	}
	if err := s.subscriptionSvc.billingCacheService.InvalidateUserBalance(context.Background(), userID); err != nil {
		slog.Warn("invalidate balance cache failed", "userID", userID, "error", err)
	}
}
