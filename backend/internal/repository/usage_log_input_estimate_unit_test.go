//go:build unit

package repository

import (
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestUsageLogInputEstimationAllInsertPaths(t *testing.T) {
	log := &service.UsageLog{InputTokens: 5, CacheReadTokens: 95, InputTokensEstimated: true, CacheReadTokensEstimated: true, OutputTokensEstimated: true, UsageEstimationReason: "client_cancelled"}
	prepared := prepareUsageLogInsert(log)
	require.Len(t, prepared.args, len(usageLogInsertArgTypes))
	want := map[string]any{"input_tokens_estimated": true, "cache_read_tokens_estimated": true, "output_tokens_estimated": true, "usage_estimation_reason": "client_cancelled"}
	for i, column := range strings.Split(usageLogSelectColumns, ", ") {
		if value, ok := want[column]; ok {
			require.Equal(t, value, prepared.args[i-1], column)
		}
	}
	key := usageLogBatchKey("estimated", 1)
	query, _ := buildUsageLogBatchInsertQuery([]string{key}, map[string]usageLogInsertPrepared{key: prepared})
	for column := range want {
		require.Equal(t, 3, strings.Count(query, column), column+" must appear in CTE, INSERT and SELECT")
	}
	absent := prepareUsageLogInsert(&service.UsageLog{})
	require.Equal(t, []any{false, false, false, "", nil}, absent.args[len(absent.args)-5:])
}

func TestChatAuxiliaryUsageAllInsertPaths(t *testing.T) {
	quantity := int64(2)
	metadata := &service.ChatAuxiliaryUsage{PriceBasis: service.ChatAuxiliaryPriceBasis, Provider: "deepseek", Items: []service.ChatAuxiliaryUsageItem{{Kind: "ocr", Quantity: &quantity, Unit: "page", UnitPrice: "0.0015", PriceQuantity: 1, ChargedCost: "0.00300000", MeteringStatus: "confirmed", QuantitySource: "provider_usage"}}}
	prepared := prepareUsageLogInsert(&service.UsageLog{ChatAuxiliaryUsage: metadata})
	require.Len(t, prepared.args, len(usageLogInsertArgTypes))
	require.Equal(t, "jsonb", usageLogInsertArgTypes[len(usageLogInsertArgTypes)-1])
	require.Equal(t, nullChatAuxiliaryUsageJSON(metadata), prepared.args[len(prepared.args)-1])
	key := usageLogBatchKey("auxiliary", 1)
	batch, _ := buildUsageLogBatchInsertQuery([]string{key}, map[string]usageLogInsertPrepared{key: prepared})
	bestEffort, _ := buildUsageLogBestEffortInsertQuery([]usageLogInsertPrepared{prepared})
	for _, query := range []string{batch, bestEffort} {
		require.Equal(t, 3, strings.Count(query, "chat_auxiliary_usage"))
	}
	require.True(t, strings.HasSuffix(usageLogSelectColumns, ", chat_auxiliary_usage"))
}
