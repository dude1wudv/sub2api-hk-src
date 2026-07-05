//go:build unit

package service

import (
	"strings"
	"testing"
)

func TestOpenAIOAuthUsageSummaryQueryUsesResetBaseline(t *testing.T) {
	query := openAIOAuthUsageSummaryQuery()

	for _, want := range []string{
		settingKeyOpenAIOAuthUsageBaselineAccountIDs,
		settingKeyOpenAIOAuthUsageCountAfter,
		"a.id IN (SELECT id FROM baseline_account_ids)",
		"a.created_at >= (SELECT count_after FROM oauth_usage_baseline)",
		"COUNT(DISTINCT tu.account_id)::bigint AS accounts_with_usage",
	} {
		if !strings.Contains(query, want) {
			t.Fatalf("query missing %q:\n%s", want, query)
		}
	}
}
