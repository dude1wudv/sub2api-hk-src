package service

import (
	"log/slog"
	"strings"
)

// ServiceTierBillingResolution describes how the billable service tier of one
// request was settled between the tier the client asked for and the tier the
// upstream reports having used.
type ServiceTierBillingResolution struct {
	Requested  string // tier carried by the request sent upstream ("" when none)
	Observed   string // tier declared by the upstream response ("" when none)
	Billing    string // tier used for billing and the usage log
	Downgraded bool   // authoritative upstream tier is cheaper than Requested
}

// ResolveBillingServiceTier picks the tier to bill. The upstream declaration is
// trusted only to lower the bill: a request served below the tier it asked for
// (OpenAI "priority" answered with service_tier "default", Anthropic speed=fast
// answered with usage.speed "standard") is billed at the cheaper tier. A response
// that claims a more expensive tier, an unknown one, or none at all leaves the
// requested tier in place, which is also the behaviour for upstreams that never
// declare a tier.
func ResolveBillingServiceTier(requested, observed string) ServiceTierBillingResolution {
	requested = normalizeBillingServiceTier(requested)
	observed = normalizeBillingServiceTier(observed)
	resolution := ServiceTierBillingResolution{Requested: requested, Observed: observed, Billing: requested}
	if observed == "" || observed == requested {
		return resolution
	}
	observedRank, known := serviceTierCostRank(observed)
	if !known {
		return resolution
	}
	requestedRank, _ := serviceTierCostRank(requested)
	if observedRank >= requestedRank {
		return resolution
	}
	resolution.Billing = observed
	resolution.Downgraded = true
	return resolution
}

// serviceTierCostRank orders tiers by their cost relative to the base rate, so a
// lower rank is always cheaper. Unknown tiers rank as the base rate and report
// known=false so callers can refuse to act on them.
func serviceTierCostRank(tier string) (rank int, known bool) {
	switch normalizeBillingServiceTier(tier) {
	case "flex":
		return 0, true
	case "", "default", "standard", "auto", "scale":
		return 1, true
	case "priority", "fast":
		return 2, true
	case "ultrafast":
		return 3, true
	default:
		return 1, false
	}
}

// ResolveOpenAIServiceTierBilling applies the response-tier contract for the
// selected credential. Public OpenAI API responses declare the actual tier and
// may lower billing. The private ChatGPT Codex endpoint does not: it commonly
// reports default for effective Fast turns, so OAuth-like credentials retain
// the final outbound tier while still exposing the observed value.
func ResolveOpenAIServiceTierBilling(account *Account, requested, observed string) ServiceTierBillingResolution {
	if account != nil && account.IsOpenAIOAuthLike() && codexOAuthResponseTierIsNonAuthoritative(observed) {
		return ServiceTierBillingResolution{
			Requested: normalizeBillingServiceTier(requested),
			Observed:  normalizeBillingServiceTier(observed),
			Billing:   normalizeBillingServiceTier(requested),
		}
	}
	return ResolveBillingServiceTier(requested, observed)
}

func codexOAuthResponseTierIsNonAuthoritative(observed string) bool {
	switch normalizeBillingServiceTier(observed) {
	case "default":
		return true
	default:
		return false
	}
}

// ApplyOpenAIServiceTierBillingResolution lowers result.ServiceTier only when
// the selected credential's upstream response tier is authoritative.
func ApplyOpenAIServiceTierBillingResolution(account *Account, result *OpenAIForwardResult) ServiceTierBillingResolution {
	return ApplyOpenAIGroupServiceTierBillingResolution(account, nil, result)
}

// ApplyOpenAIGroupServiceTierBillingResolution preserves the configured forced
// price only when the matching tier was actually sent. The response remains
// independently observable, including authoritative upstream downgrades.
func ApplyOpenAIGroupServiceTierBillingResolution(account *Account, group *Group, result *OpenAIForwardResult) ServiceTierBillingResolution {
	if result == nil {
		return ServiceTierBillingResolution{}
	}
	resolution := ResolveOpenAIServiceTierBilling(account, optionalStringValue(result.ServiceTier), result.UpstreamResponseServiceTier)
	if account != nil && account.IsOpenAI() && IsGroupContextValid(group) && groupSupportsOpenAIFast(group.Platform) {
		forced := ""
		if group.ForceOpenAIUltrafast {
			forced = OpenAIFastTierUltrafast
		} else if group.ForceOpenAIFast {
			forced = OpenAIFastTierPriority
		}
		if forced != "" && resolution.Requested == forced {
			resolution.Billing = resolution.Requested
		}
	}
	if resolution.Downgraded {
		billing := resolution.Billing
		result.ServiceTier = &billing
	}
	return resolution
}

// ApplyForwardServiceTierBillingResolution is the ForwardResult counterpart of
// ApplyOpenAIServiceTierBillingResolution.
func ApplyForwardServiceTierBillingResolution(result *ForwardResult) ServiceTierBillingResolution {
	if result == nil {
		return ServiceTierBillingResolution{}
	}
	resolution := ResolveBillingServiceTier(optionalStringValue(result.ServiceTier), result.UpstreamResponseServiceTier)
	if resolution.Downgraded {
		billing := resolution.Billing
		result.ServiceTier = &billing
	}
	return resolution
}

// applyServiceTierUsageMetadata keeps observed tier evidence separate from the
// billed tier. Missing upstream declarations remain unknown, including old rows.
func applyServiceTierUsageMetadata(usageLog *UsageLog, resolution ServiceTierBillingResolution) {
	usageLog.RequestedServiceTier = optionalTrimmedStringPtr(resolution.Requested)
	usageLog.UpstreamServiceTier = optionalTrimmedStringPtr(resolution.Observed)
	if resolution.Observed != "" {
		downgraded := resolution.Downgraded
		usageLog.ServiceTierDowngraded = &downgraded
	}
}

// logServiceTierBillingDowngrade records authoritative upstream downgrades,
// including forced groups that retain their configured billing tier.
func logServiceTierBillingDowngrade(component string, account *Account, requestID string, resolution ServiceTierBillingResolution) {
	if !resolution.Downgraded {
		return
	}
	attrs := []any{
		"component", component,
		"request_id", strings.TrimSpace(requestID),
		"requested_tier", resolution.Requested,
		"response_tier", resolution.Observed,
		"billed_tier", resolution.Billing,
	}
	if account != nil {
		attrs = append(attrs, "platform", account.Platform, "account_id", account.ID)
	}
	slog.Info("billing.service_tier_downgraded", attrs...)
}
