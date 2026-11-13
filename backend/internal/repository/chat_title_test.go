//go:build integration

package repository

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestChatSessionUnicodeTitle(t *testing.T) {
	tx := testEntTx(t)
	ctx := dbent.NewTxContext(context.Background(), tx)
	client := tx.Client()
	owner := mustCreateUser(t, client, &service.User{})
	repo := NewChatRepository(integrationEntClient)
	original := chatIntegrationSnapshot(t, uuid.New())
	var tree map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(original.Raw, &tree))
	title := strings.Repeat("中", 500)
	encoded, err := json.Marshal(title)
	require.NoError(t, err)
	tree["title"] = encoded
	raw, err := json.Marshal(tree)
	require.NoError(t, err)
	snapshot, err := service.DecodeChatSnapshot(original.ID, raw)
	require.NoError(t, err)
	saved, err := repo.SaveSession(ctx, owner.ID, snapshot, 0)
	require.NoError(t, err)
	require.Equal(t, title, saved.Title)
	renamed, err := repo.RenameSession(ctx, owner.ID, snapshot.ID, strings.Repeat("文", 500), saved.Revision)
	require.NoError(t, err)
	require.Equal(t, strings.Repeat("文", 500), renamed.Title)
}
