package service

import (
	"context"
	"time"
)

type InflightScopeReservationRenewer interface {
	RenewInflightScope(context.Context, string, string, time.Duration) (bool, error)
}

// ReserveChatAuxiliary preserves fixed-charge admission for all billing modes.
// Lost reservations cancel the upstream operation instead of silently expiring.
func (s *BillingCacheService) ReserveChatAuxiliary(ctx context.Context, user *User, key *APIKey, group *Group, subscription *UserSubscription, amount float64) (context.Context, *InflightReservation, error) {
	res, err := s.ReserveFixedCharge(ctx, user, key, group, subscription, amount)
	if err != nil || res == nil {
		return ctx, res, err
	}
	balanceRenewer, balanceOK := s.cache.(InflightBalanceReservationRenewer)
	scopeRenewer, scopeOK := s.cache.(InflightScopeReservationRenewer)
	if (res.cache != nil && !balanceOK) || (len(res.scopes) > 0 && !scopeOK) {
		res.Release()
		return ctx, nil, ErrBillingServiceUnavailable
	}
	live, cancel := context.WithCancelCause(ctx)
	res.stopRenew = make(chan struct{})
	res.renewDone = make(chan struct{})
	go func() {
		defer close(res.renewDone)
		defer cancel(context.Canceled)
		interval := res.ttl / 3
		if interval < 10*time.Millisecond {
			interval = 10*time.Millisecond
		}
		timer := time.NewTicker(interval)
		defer timer.Stop()
		for {
			select {
			case <-res.stopRenew:
				return
			case <-live.Done():
				return
			case <-timer.C:
				renewCtx, stop := context.WithTimeout(live, inflightReservationRenewTimeout)
				alive := true
				var renewErr error
				if res.cache != nil {
					alive, renewErr = balanceRenewer.RenewInflightBalance(renewCtx, res.userID, res.requestID, res.ttl)
				}
				for _, scope := range res.scopes {
					if !alive || renewErr != nil {
						break
					}
					alive, renewErr = scopeRenewer.RenewInflightScope(renewCtx, scope, res.requestID, res.ttl)
				}
				stop()
				if !alive || renewErr != nil {
					cancel(ErrBillingServiceUnavailable)
					return
				}
			}
		}
	}()
	return WithInflightReservation(live, res), res, nil
}
