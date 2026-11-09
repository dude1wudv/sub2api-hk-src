package handler

import (
	"context"
	"errors"
	"strings"

	middleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// Adapted from kiss-kedaya/sub2api AlongKeyRoutes. Keep the local API/storage
// contract, but choose again from quality-ordered eligible groups on each
// retryable, uncommitted attempt. Existing forwarders decide whether replay is
// safe; this helper never replays a response or resets their retry budgets.
type smartRouteGateway interface {
	inflightReservationEstimator
	ResolveChannelMappingAndRestrict(context.Context, *int64, string) (service.ChannelMappingResult, bool)
	RebindSmartRoutingProfitContext(context.Context, *service.APIKey) context.Context
	SmartRoutingModels(context.Context, *service.APIKey) ([]string, error)
}

type smartRouteBinding struct {
	Key          **service.APIKey
	Subscription **service.UserSubscription
	Mapping      *service.ChannelMappingResult
	InflightDone *func()
	Model        string
	FixedModel   string
	Body         []byte
	Accept       func(*service.APIKey) bool
	ResolveModel func(*service.APIKey, service.ChannelMappingResult) string
}

type smartRouteSelect func(context.Context, *service.APIKey, string) (*service.AccountSelectionResult, service.OpenAIAccountScheduleDecision, error)

func selectAcrossSmartRoutes(c *gin.Context, keys *service.APIKeyService, billing *service.BillingCacheService, gateway smartRouteGateway, binding smartRouteBinding, selectAccount smartRouteSelect) (*service.AccountSelectionResult, service.OpenAIAccountScheduleDecision, error) {
	current := *binding.Key
	if len(current.RoutingGroupIDs) == 0 || keys == nil {
		model := binding.Model
		if binding.Mapping != nil && binding.Mapping.Mapped {
			model = binding.Mapping.MappedModel
		}
		if binding.FixedModel != "" {
			model = binding.FixedModel
		}
		return selectAccount(c.Request.Context(), current, model)
	}
	candidates, err := keys.SmartRoutingKeys(c.Request.Context(), current)
	if err != nil {
		return nil, service.OpenAIAccountScheduleDecision{}, err
	}
	// Billing eligibility increments RPM. Charge admission once per group, not
	// once per account retry; the handler already checked the original group.
	const checkedKey = "smart_route_checked_billing_groups"
	value, _ := c.Get(checkedKey)
	checked, _ := value.(map[int64]bool)
	if checked == nil {
		checked = map[int64]bool{}
		if current.GroupID != nil {
			checked[*current.GroupID] = true
		}
		c.Set(checkedKey, checked)
	}
	var lastErr error
	for _, candidate := range candidates {
		if err := c.Request.Context().Err(); err != nil {
			return nil, service.OpenAIAccountScheduleDecision{}, err
		}
		if binding.Accept != nil && !binding.Accept(candidate) {
			continue
		}
		if !candidate.Group.ModelAllowlist.Allows(binding.Model) {
			continue
		}
		ctx := gateway.RebindSmartRoutingProfitContext(c.Request.Context(), candidate)
		mapping, restricted := gateway.ResolveChannelMappingAndRestrict(ctx, candidate.GroupID, binding.Model)
		if restricted {
			continue
		}
		model := binding.Model
		if mapping.Mapped {
			model = mapping.MappedModel
		}
		if binding.ResolveModel != nil {
			model = binding.ResolveModel(candidate, mapping)
		}
		models, catalogErr := gateway.SmartRoutingModels(ctx, candidate)
		if catalogErr != nil {
			return nil, service.OpenAIAccountScheduleDecision{}, catalogErr
		}
		claimed := false
		for _, id := range models {
			if strings.EqualFold(id, model) {
				claimed = true
				break
			}
		}
		if !claimed {
			continue
		}
		subscription, subErr := keys.ResolveSmartRoutingSubscription(ctx, candidate)
		if subErr != nil {
			lastErr = subErr
			continue
		}
		selection, decision, selectErr := selectAccount(ctx, candidate, model)
		if selectErr != nil {
			lastErr = selectErr
			if errors.Is(selectErr, service.ErrNoAvailableAccounts) || errors.Is(selectErr, service.ErrNoAvailableCompactAccounts) {
				continue
			}
			return nil, decision, selectErr
		}
		if selection == nil || selection.Account == nil {
			continue
		}
		release := func() {
			if selection.ReleaseFunc != nil {
				selection.ReleaseFunc()
			}
		}
		if !checked[*candidate.GroupID] && billing != nil {
			if err := billing.CheckBillingEligibility(ctx, candidate.User, candidate, candidate.Group, subscription, service.QuotaPlatform(ctx, candidate)); err != nil {
				release()
				lastErr = err
				continue
			}
			checked[*candidate.GroupID] = true
		}
		changed := current.GroupID == nil || *current.GroupID != *candidate.GroupID
		if changed {
			// No forward has started for this selection; release the previous
			// reservation before repricing the request against the chosen group.
			if binding.InflightDone != nil && *binding.InflightDone != nil {
				(*binding.InflightDone)()
				*binding.InflightDone = inflightNoop
			}
			if billing != nil && binding.InflightDone != nil {
				reservedCtx, done, err := reserveInflightBalanceCtx(ctx, billing, gateway, candidate, subscription, tokenInflightEstimate(binding.Model, binding.Body))
				if err != nil {
					release()
					return nil, decision, err
				}
				ctx = reservedCtx
				*binding.InflightDone = done
			}
		} else {
			// Preserve the live reservation already acquired by the handler.
			if existing := service.InflightReservationFromContext(c.Request.Context()); existing != nil {
				ctx = service.WithInflightReservation(ctx, existing)
			}
		}
		if binding.Mapping != nil {
			*binding.Mapping = mapping
		}
		if binding.Subscription != nil {
			*binding.Subscription = subscription
		}
		*binding.Key = candidate
		c.Set(string(middleware.ContextKeyAPIKey), candidate)
		c.Set(string(middleware.ContextKeySubscription), subscription)
		c.Request = c.Request.WithContext(ctx)
		return selection, decision, nil
	}
	if lastErr == nil {
		lastErr = service.ErrNoAvailableAccounts
	}
	return nil, service.OpenAIAccountScheduleDecision{}, lastErr
}
