package migrations

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// This manifest was taken from Git objects at the pre-merge HK HEAD, not from
// the merge worktree. Both bytes and the runner's TrimSpace checksum are frozen.
func TestHKMigrationHistoryRemainsImmutable(t *testing.T) {
	data, err := os.ReadFile("testdata/hk_head_6e78ddd9_sha256.json")
	require.NoError(t, err)
	var history struct {
		Head       string `json:"head"`
		Migrations map[string]struct {
			SHA256   string `json:"sha256"`
			Checksum string `json:"checksum"`
		} `json:"migrations"`
	}
	require.NoError(t, json.Unmarshal(data, &history))
	require.Equal(t, "6e78ddd9c4254ba6b4cc7ab00ace4b66020c4a1f", history.Head)
	require.Len(t, history.Migrations, 350)
	for name, expected := range history.Migrations {
		t.Run(name, func(t *testing.T) {
			content, err := FS.ReadFile(name)
			require.NoError(t, err)
			require.Equal(t, expected.SHA256, fmt.Sprintf("%x", sha256.Sum256(content)), "historical SQL bytes changed")
			require.Equal(t, expected.Checksum, fmt.Sprintf("%x", sha256.Sum256([]byte(strings.TrimSpace(string(content))))), "runner checksum changed")
		})
	}
}

func TestRollupReconciliationIsLastAndPreservesBoth251Files(t *testing.T) {
	names, err := fs.Glob(FS, "*.sql")
	require.NoError(t, err)
	sort.Strings(names)
	require.Len(t, names, 353, "HK 350 + two upstream 251 migrations + reconciliation 349")
	require.Equal(t, "349_group_usage_rollup_retention_nonblocking.sql", names[len(names)-1])
	require.Contains(t, names, "251_group_force_openai_ultrafast.sql")
	require.Contains(t, names, "251_group_usage_rollup_nonblocking_build.sql")
}

func TestMigration349UnifiesRetentionAndNonblockingInvalidation(t *testing.T) {
	content, err := FS.ReadFile("349_group_usage_rollup_retention_nonblocking.sql")
	require.NoError(t, err)
	sql := string(content)
	require.Contains(t, sql, "SET LOCAL lock_timeout")
	require.Contains(t, sql, "SET expired_before = GREATEST(expired_before,")
	require.Contains(t, sql, "(retained_from AT TIME ZONE timezone_name)::date::timestamp AT TIME ZONE timezone_name")
	for _, name := range []string{
		"apply_group_usage_rollup_invalidation",
		"invalidate_group_usage_rollups_at",
		"invalidate_group_usage_rollup_state",
		"invalidate_group_usage_rollup_state_after_insert",
	} {
		require.Contains(t, sql, "CREATE OR REPLACE FUNCTION "+name+"(")
	}
	require.Contains(t, sql, "building_until > affected_date")
	require.Contains(t, sql, "LEAST(build_dirty_from, affected_date)")
	require.Contains(t, sql, "OLD.created_at >= state_expired_before")
	require.Contains(t, sql, "NEW.created_at >= state_expired_before")
	require.Contains(t, sql, "(OLD.created_at AT TIME ZONE configured_timezone)::date >= state_retained_date")
	require.Contains(t, sql, "(NEW.created_at AT TIME ZONE configured_timezone)::date >= state_retained_date")
	require.Contains(t, sql, "(created_at AT TIME ZONE configured_timezone)::date >= state_retained_date")
	require.Contains(t, sql, "FOR KEY SHARE;")
	require.NotContains(t, sql, "FOR UPDATE;")
}
