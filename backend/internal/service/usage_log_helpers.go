package service

import "strings"

func optionalTrimmedStringPtr(raw string) *string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func optionalStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func coalesceRequestedReasoningEffortWithSource(requested, forwarded, source *string) *string {
	if strings.EqualFold(optionalStringValue(source), ReasoningEffortSourceDefault) {
		return nil
	}
	if trimmed := optionalStringValue(requested); trimmed != "" {
		return &trimmed
	}
	if trimmed := optionalStringValue(forwarded); trimmed != "" {
		return &trimmed
	}
	return nil
}

func forwardResultBillingModel(requestedModel, upstreamModel string) string {
	if trimmed := strings.TrimSpace(requestedModel); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(upstreamModel)
}

func optionalInt64Ptr(v int64) *int64 {
	if v == 0 {
		return nil
	}
	return &v
}
