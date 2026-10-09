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

func TestFinalPlatformChecksAreApplicationOwnedAfterAllMigrations(t *testing.T) {
	entries, err := FS.ReadDir(".")
	require.NoError(t, err)
	operation := regexp.MustCompile(`(?i)(DROP|ADD)\s+CONSTRAINT\s+(?:IF\s+EXISTS\s+)?(user_platform_quotas_platform_check|composite_model_routes_target_platform_check)\b`)
	comments := regexp.MustCompile(`(?m)--[^\n]*`)
	replay := func(applied map[string]bool, state map[string]bool) {
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") || applied[entry.Name()] {
				continue
			}
			raw, readErr := FS.ReadFile(entry.Name())
			require.NoError(t, readErr)
			for _, match := range operation.FindAllSubmatch(comments.ReplaceAll(raw, nil), -1) {
				state[string(match[2])] = strings.EqualFold(string(match[1]), "ADD")
			}
		}
	}
	fresh := map[string]bool{}
	replay(nil, fresh)
	// Existing HK databases already ran every migration except upstream 242 and HK 246.
	priorFiles := map[string]bool{}
	for _, entry := range entries {
		if entry.Name() != "242_drop_platform_check_constraints.sql" && entry.Name() != "246_hk_platform_catalog_constraints.sql" {
			priorFiles[entry.Name()] = true
		}
	}
	upgraded := map[string]bool{"user_platform_quotas_platform_check": true, "composite_model_routes_target_platform_check": true}
	replay(priorFiles, upgraded)
	require.Equal(t, fresh, upgraded)
	require.Len(t, fresh, 2)
	for name, present := range fresh {
		require.False(t, present, name)
	}
}
