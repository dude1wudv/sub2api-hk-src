package repository

import (
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	dbaccount "github.com/Wei-Shaw/sub2api/ent/account"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

func TestAccountListOrderSQL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		sortBy    string
		sortOrder string
		wantOrder string
	}{
		{
			name:      "platform and type ascending with stable ID tie break",
			sortBy:    "platform_type",
			sortOrder: "asc",
			wantOrder: `"accounts"."platform" ASC, "accounts"."type" ASC, "accounts"."id" ASC`,
		},
		{
			name:      "platform and type descending with stable ID tie break",
			sortBy:    "platform_type",
			sortOrder: "desc",
			wantOrder: `"accounts"."platform" DESC, "accounts"."type" DESC, "accounts"."id" DESC`,
		},
		{
			name:      "platform and type default to ascending",
			sortBy:    "platform_type",
			wantOrder: `"accounts"."platform" ASC, "accounts"."type" ASC, "accounts"."id" ASC`,
		},
		{
			name:      "invalid platform sort direction falls back to ascending",
			sortBy:    "platform_type",
			sortOrder: "sideways",
			wantOrder: `"accounts"."platform" ASC, "accounts"."type" ASC, "accounts"."id" ASC`,
		},
		{
			name:      "platform sort normalizes case and whitespace",
			sortBy:    " PLATFORM_TYPE ",
			sortOrder: " DESC ",
			wantOrder: `"accounts"."platform" DESC, "accounts"."type" DESC, "accounts"."id" DESC`,
		},
		{
			name:      "existing default remains name ascending",
			wantOrder: `"accounts"."name" ASC, "accounts"."id" ASC`,
		},
		{
			name:      "unknown sort field keeps name fallback",
			sortBy:    "unknown",
			sortOrder: "desc",
			wantOrder: `"accounts"."name" DESC, "accounts"."id" DESC`,
		},
		{
			name:      "existing ID descending is unchanged",
			sortBy:    "id",
			sortOrder: "desc",
			wantOrder: `"accounts"."id" DESC, "accounts"."id" DESC`,
		},
		{
			name:      "existing priority descending keeps stable ID tie break",
			sortBy:    "priority",
			sortOrder: "desc",
			wantOrder: `"accounts"."priority" DESC, "accounts"."id" DESC`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			params := pagination.PaginationParams{
				Page: 2, PageSize: 3, SortBy: tt.sortBy, SortOrder: tt.sortOrder,
			}
			selector := entsql.Dialect(dialect.Postgres).Select().From(entsql.Table(dbaccount.Table))
			selector.Select(selector.C(dbaccount.FieldID)).Limit(params.Limit()).Offset(params.Offset())
			for _, order := range accountListOrder(params) {
				order(selector)
			}

			query, args := selector.Query()
			require.NoError(t, selector.Err())
			require.Equal(t, `SELECT "accounts"."id" FROM "accounts" ORDER BY `+tt.wantOrder+` LIMIT 3 OFFSET 3`, query)
			require.Empty(t, args)
		})
	}
}
