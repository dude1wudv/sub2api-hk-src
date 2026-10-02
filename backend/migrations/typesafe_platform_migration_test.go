package migrations

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTypeSafePlatformMigration(t *testing.T) {
	content, err := FS.ReadFile("245_add_typesafe_platform.sql")
	require.NoError(t, err)

	sql := strings.Join(strings.Fields(string(content)), " ")
	require.Contains(t, sql, "DROP CONSTRAINT IF EXISTS user_platform_quotas_platform_check")
	require.Contains(t, sql, "DROP CONSTRAINT IF EXISTS composite_model_routes_target_platform_check")
	require.Contains(t, sql,
		"CHECK (platform IN ('anthropic', 'openai', 'gemini', 'antigravity', 'grok', 'kimi', 'zhipu', 'deepseek', 'minimax', 'stepfun', 'opencode_go', 'mirasim', 'typesafe'))")
	require.Contains(t, sql,
		"CHECK (target_platform IN ('anthropic', 'openai', 'gemini', 'antigravity', 'grok', 'kimi', 'zhipu', 'deepseek', 'minimax', 'stepfun', 'opencode_go', 'mirasim', 'typesafe'))")
}

func TestFinalPlatformChecksIncludeHKAndTypeSafeAfterAllMigrations(t *testing.T) {
	constraints := []struct {
		name  string
		field string
	}{
		{name: "user_platform_quotas_platform_check", field: "platform"},
		{name: "composite_model_routes_target_platform_check", field: "target_platform"},
	}

	entries, err := FS.ReadDir(".")
	require.NoError(t, err)
	for _, constraint := range constraints {
		checkPattern := regexp.MustCompile(`(?is)CHECK\s*\(\s*` + constraint.field + `\s+IN\s*\(([^)]*)\)\s*\)`)
		var latestCheck, latestMigration string
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
				continue
			}
			content, err := FS.ReadFile(entry.Name())
			require.NoError(t, err)
			if match := checkPattern.FindSubmatch(content); match != nil {
				latestCheck = string(match[1])
				latestMigration = entry.Name()
			}
		}

		require.Equal(t, "245_add_typesafe_platform.sql", latestMigration, "%s final migration", constraint.name)
		for _, platform := range []string{"stepfun", "mirasim", "typesafe"} {
			require.Contains(t, latestCheck, "'"+platform+"'", "%s final CHECK", constraint.name)
		}
	}
}
