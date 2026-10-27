package service

import (
	"context"
	"encoding/json"
	"math"
)

const deepSeekToolsGroupIDsExtraKey = "deepseek_tools_group_ids"

// DeepSeekToolsAllowedForGroup keeps native tools separate from a shared
// account's chat membership. Missing or malformed permission data denies tools.
func (a *Account) DeepSeekToolsAllowedForGroup(groupID *int64) bool {
	if groupID == nil || *groupID <= 0 || !a.deepSeekToolsEnabled() {
		return false
	}
	groups, _ := deepSeekToolsGroupIDs(a.Extra[deepSeekToolsGroupIDsExtraKey])
	for _, id := range groups {
		if id == *groupID {
			return true
		}
	}
	return false
}

func (a *Account) deepSeekToolsEnabled() bool {
	if a == nil || a.Type != AccountTypeAPIKey ||
		(a.Platform != PlatformOpenAI && a.Platform != PlatformDeepseek) {
		return false
	}
	_, valid := deepSeekToolsGroupIDs(a.Extra[deepSeekToolsGroupIDsExtraKey])
	return valid
}

func deepSeekToolsGroupIDs(value any) ([]int64, bool) {
	var groups []int64
	switch values := value.(type) {
	case []int64:
		groups = values
	case []int:
		groups = make([]int64, len(values))
		for i, id := range values {
			groups[i] = int64(id)
		}
	case []any:
		groups = make([]int64, len(values))
		for i, value := range values {
			switch id := value.(type) {
			case int:
				groups[i] = int64(id)
			case int64:
				groups[i] = id
			case float64:
				if math.IsNaN(id) || math.IsInf(id, 0) || id <= 0 || id >= math.Exp2(63) || id != math.Trunc(id) {
					return nil, false
				}
				groups[i] = int64(id)
			case json.Number:
				parsed, err := id.Int64()
				if err != nil {
					return nil, false
				}
				groups[i] = parsed
			default:
				return nil, false
			}
		}
	default:
		return nil, false
	}
	if len(groups) == 0 {
		return nil, false
	}
	for _, id := range groups {
		if id <= 0 {
			return nil, false
		}
	}
	return groups, true
}

type deepSeekToolsSchedulingGroupContextKey struct{}

func withDeepSeekToolsSchedulingGroup(ctx context.Context, groupID *int64, capability OpenAIEndpointCapability) context.Context {
	if capability != OpenAIEndpointCapabilityDeepSeekTools {
		return ctx
	}
	return context.WithValue(ctx, deepSeekToolsSchedulingGroupContextKey{}, derefGroupID(groupID))
}

func deepSeekToolsSchedulingGroup(ctx context.Context) *int64 {
	groupID, ok := ctx.Value(deepSeekToolsSchedulingGroupContextKey{}).(int64)
	if !ok || groupID <= 0 {
		return nil
	}
	return &groupID
}
