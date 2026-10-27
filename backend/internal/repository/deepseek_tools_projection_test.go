package repository

import (
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestDeepSeekToolsProjectionPreservesScopedAdmission(t *testing.T) {
	allowed, shared := int64(223), int64(81)
	for _, groups := range []any{[]int64{223}, []any{float64(223)}, nil, []any{223, "81"}} {
		account := service.Account{ID: 1112, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey,
			GroupIDs: []int64{223, 81}, Extra: map[string]any{"deepseek_tools_group_ids": groups, "unrelated_private_field": "drop"}}
		_, metadata, err := marshalSchedulerCacheAccount(account)
		require.NoError(t, err)
		var projected service.Account
		require.NoError(t, json.Unmarshal(metadata, &projected))
		require.Equal(t, account.DeepSeekToolsAllowedForGroup(&allowed), projected.DeepSeekToolsAllowedForGroup(&allowed))
		require.False(t, projected.DeepSeekToolsAllowedForGroup(&shared), "shared chat membership must never grant tools")
		require.NotContains(t, projected.Extra, "unrelated_private_field")
	}
}
