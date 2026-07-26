package service

import (
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

func subscriptionBusinessNow() time.Time { return time.Now().In(subscriptionBusinessLocation) }

var subscriptionBusinessLocation = func() *time.Location {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	return location
}()

func validateFixedSubscriptionExpiry(now time.Time, fixedExpiresAt, existingExpiresAt *time.Time) error {
	if fixedExpiresAt == nil || fixedExpiresAt.After(now) {
		return nil
	}
	if existingExpiresAt != nil && existingExpiresAt.After(now) {
		return nil
	}
	return infraerrors.BadRequest("SUBSCRIPTION_FIXED_EXPIRY_PASSED", "subscription fixed expiry has passed")
}

// ResolveSubscriptionExpiry uses the fixed expiry when configured without shortening an active subscription.
func ResolveSubscriptionExpiry(now time.Time, validityDays int, fixedExpiresAt, existingExpiresAt *time.Time) time.Time {
	if fixedExpiresAt != nil {
		if existingExpiresAt != nil && existingExpiresAt.After(*fixedExpiresAt) {
			return *existingExpiresAt
		}
		return *fixedExpiresAt
	}
	base := now
	if existingExpiresAt != nil && existingExpiresAt.After(now) {
		base = *existingExpiresAt
	}
	return base.AddDate(0, 0, validityDays)
}
