package service

import (
	"context"

	"github.com/Wei-Shaw/sub2api/ent/group"
	"github.com/Wei-Shaw/sub2api/ent/subscriptionplan"
	"github.com/Wei-Shaw/sub2api/ent/usersubscription"
)

type SubscriptionAvailability struct {
	HasSalePlans           bool `json:"has_sale_plans"`
	HasActiveSubscriptions bool `json:"has_active_subscriptions"`
	HasSubscriptionHistory bool `json:"has_subscription_history"`
}

// Availability only fetches existence flags; it does not hydrate subscription history.
func (s *SubscriptionService) Availability(ctx context.Context, userID int64) (*SubscriptionAvailability, error) {
	if s == nil || s.entClient == nil || userID <= 0 {
		return nil, ErrServiceUnavailable
	}
	now := subscriptionBusinessNow().UTC()
	active, err := s.entClient.UserSubscription.Query().Where(usersubscription.UserIDEQ(userID), usersubscription.StatusEQ(StatusActive), usersubscription.ExpiresAtGT(now)).Exist(ctx)
	if err != nil {
		return nil, err
	}
	history, err := s.entClient.UserSubscription.Query().Where(usersubscription.UserIDEQ(userID)).Exist(ctx)
	if err != nil {
		return nil, err
	}
	// Only balance checkout is currently supported. External payment gateways are retired.
	groupIDs, err := s.entClient.Group.Query().Where(group.StatusEQ(StatusActive), group.SubscriptionTypeEQ("subscription")).IDs(ctx)
	if err != nil {
		return nil, err
	}
	sale := false
	if len(groupIDs) > 0 {
		sale, err = s.entClient.SubscriptionPlan.Query().Where(
			subscriptionplan.GroupIDIn(groupIDs...), subscriptionplan.ForSaleEQ(true),
			subscriptionplan.PurchaseModeIn(SubscriptionPlanPurchaseModeBalance, SubscriptionPlanPurchaseModeBoth),
			subscriptionplan.Or(subscriptionplan.SaleEndsAtIsNil(), subscriptionplan.SaleEndsAtGT(now)),
			subscriptionplan.Or(subscriptionplan.FixedExpiresAtIsNil(), subscriptionplan.FixedExpiresAtGT(now)),
		).Exist(ctx)
		if err != nil {
			return nil, err
		}
	}
	return &SubscriptionAvailability{HasSalePlans: sale, HasActiveSubscriptions: active, HasSubscriptionHistory: history}, nil
}
