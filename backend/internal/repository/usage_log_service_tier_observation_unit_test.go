//go:build unit

package repository

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestPrepareUsageLogInsertServiceTierObservation(t *testing.T) {
	requested, upstream, billed := "ultrafast", "default", "ultrafast"
	for _, downgraded := range []*bool{nil, usageTierObservationBool(true), usageTierObservationBool(false)} {
		log := &service.UsageLog{ServiceTier: &billed, RequestedServiceTier: &requested, UpstreamServiceTier: &upstream, ServiceTierDowngraded: downgraded}
		prepared := prepareUsageLogInsert(log)
		require.Len(t, prepared.args, len(usageLogInsertArgTypes))
		columns := strings.Split(usageLogSelectColumns, ", ")
		for i, name := range columns {
			switch name {
			case "requested_service_tier":
				require.Equal(t, sql.NullString{String: requested, Valid: true}, prepared.args[i-1])
			case "upstream_service_tier":
				require.Equal(t, sql.NullString{String: upstream, Valid: true}, prepared.args[i-1])
			case "service_tier_downgraded":
				want := sql.NullBool{}
				if downgraded != nil {
					want = sql.NullBool{Bool: *downgraded, Valid: true}
				}
				require.Equal(t, want, prepared.args[i-1])
			}
		}
		key := usageLogBatchKey("req-tier", 1)
		query, _ := buildUsageLogBatchInsertQuery([]string{key}, map[string]usageLogInsertPrepared{key: prepared})
		for _, name := range []string{"requested_service_tier", "upstream_service_tier", "service_tier_downgraded"} {
			require.Equal(t, 3, strings.Count(query, name), "CTE, INSERT and SELECT must all include %s", name)
		}
	}
	absent := prepareUsageLogInsert(&service.UsageLog{})
	require.Equal(t, []any{sql.NullString{}, sql.NullString{}, sql.NullBool{}}, absent.args[len(absent.args)-3:])
}

func usageTierObservationBool(value bool) *bool { return &value }
