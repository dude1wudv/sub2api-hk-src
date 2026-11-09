package service

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
)

// ResolveSmartRoutingSubscription keeps the selected provider group and its
// subscription together when an uncommitted upstream attempt changes route.
func (s *APIKeyService) ResolveSmartRoutingSubscription(ctx context.Context, key *APIKey) (*UserSubscription, error) {
	if key == nil || key.Group == nil || !key.Group.IsSubscriptionType() {
		return nil, nil
	}
	return s.userSubRepo.GetActiveByUserIDAndGroupID(ctx, key.UserID, key.Group.ID)
}

func ContextWithSmartRoutingGroup(ctx context.Context, key *APIKey) context.Context {
	ctx = context.WithValue(ctx, ctxkey.Group, key.Group)
	ctx = context.WithValue(ctx, ctxkey.UserID, key.UserID)
	// A rejected attempt must not leave a released reservation attached to the
	// next route, especially when switching from balance to subscription mode.
	ctx = context.WithValue(ctx, inflightReservationCtxKey{}, (*InflightReservation)(nil))
	ctx = context.WithValue(ctx, openAIProfitControlGateCtxKey{}, (*openAIProfitControlGate)(nil))
	if _, ok := gatewayTokenRequestPricingAtFromContext(ctx); ok {
		ctx = context.WithValue(ctx, gatewayTokenRequestBillingGroupCtxKey{}, key.Group)
	}
	return WithSmartRouting(ctx)
}

func (s *OpenAIGatewayService) RebindSmartRoutingProfitContext(ctx context.Context, key *APIKey) context.Context {
	return s.withOpenAIProfitControlGate(ContextWithSmartRoutingGroup(ctx, key), key.GroupID)
}

func (s *GatewayService) RebindSmartRoutingProfitContext(ctx context.Context, key *APIKey) context.Context {
	return s.withGatewayProfitControlGate(ContextWithSmartRoutingGroup(ctx, key), key.GroupID)
}
