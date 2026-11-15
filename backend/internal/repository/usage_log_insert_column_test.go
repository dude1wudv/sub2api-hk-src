package repository

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Look up semantic columns so appended billing metadata cannot shift assertions.
func usageLogInsertArgIndex(t *testing.T, column string) int {
	t.Helper()
	index := slices.Index(strings.Split(usageLogSelectColumns, ", "), column)
	require.Positive(t, index, "column %q must follow the generated id", column)
	return index - 1
}
