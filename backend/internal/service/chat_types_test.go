//go:build unit

package service

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func chatTestTree(id uuid.UUID) map[string]any {
	return map[string]any{
		"id": id.String(), "title": "完整分支", "rootMessageId": "u",
		"messages": []any{
			map[string]any{"id": "u", "role": "user", "content": "问题", "parentId": nil, "selectedChildId": "a", "files": []any{}},
			map[string]any{"id": "a", "role": "assistant", "parentId": "u", "status": "done", "blocks": []any{map[string]string{"type": "thinking", "thinking": "思考", "signature": "private-provider-signature"}, map[string]string{"type": "text", "text": "回答"}}},
			map[string]any{"id": "b", "role": "assistant", "parentId": "u", "status": "error", "blocks": []any{}, "error": map[string]string{"kind": "network", "message": "保留"}},
		},
	}
}

func TestChatSnapshotPreservesAllBranches(t *testing.T) {
	id := uuid.New()
	original := chatTestTree(id)
	raw, err := json.Marshal(original)
	require.NoError(t, err)
	snapshot, err := DecodeChatSnapshot(id, raw)
	require.NoError(t, err)
	require.JSONEq(t, string(raw), string(snapshot.Raw))
	require.Len(t, snapshot.Messages, 3)
	visible := snapshot.VisibleMessages()
	require.Len(t, visible, 2)
	require.Equal(t, "a", visible[1].ID)
	require.Len(t, visible[1].Blocks, 2)
}

func TestChatSnapshotRejectsBrokenTrees(t *testing.T) {
	cases := map[string]func(map[string]any){
		"duplicate":              func(s map[string]any) { s["messages"].([]any)[2].(map[string]any)["id"] = "a" },
		"missing-parent":         func(s map[string]any) { s["messages"].([]any)[2].(map[string]any)["parentId"] = "missing" },
		"hidden-cycle":           func(s map[string]any) { s["messages"].([]any)[2].(map[string]any)["parentId"] = "b" },
		"invalid-selection":      func(s map[string]any) { s["messages"].([]any)[1].(map[string]any)["selectedChildId"] = "b" },
		"system-role":            func(s map[string]any) { s["messages"].([]any)[0].(map[string]any)["role"] = "system" },
		"missing-root":           func(s map[string]any) { s["rootMessageId"] = "missing" },
		"nil-root-with-messages": func(s map[string]any) { s["rootMessageId"] = nil },
		"branch-without-root":    func(s map[string]any) { delete(s, "rootMessageId") },
		"attachment-limit": func(s map[string]any) {
			files := []any{}
			for range 51 {
				files = append(files, map[string]any{"id": uuid.NewString(), "size": 1})
			}
			s["messages"].([]any)[0].(map[string]any)["files"] = files
		},
		"invalid-file-id": func(s map[string]any) {
			s["messages"].([]any)[0].(map[string]any)["files"] = []any{map[string]any{"id": "../../secret", "size": 1}}
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			id := uuid.New()
			s := chatTestTree(id)
			mutate(s)
			raw, err := json.Marshal(s)
			require.NoError(t, err)
			_, err = DecodeChatSnapshot(id, raw)
			require.ErrorIs(t, err, ErrChatInvalid)
		})
	}
	_, err := DecodeChatSnapshot(uuid.New(), json.RawMessage(strings.Repeat("x", ChatMaxSessionBytes+1)))
	require.ErrorIs(t, err, ErrChatTooLarge)
}

func TestChatPreferencesStrictWhitelist(t *testing.T) {
	for _, bad := range []map[string]any{{"apiKey": "never-store"}, {"apiOrigin": "https://attacker.invalid"}, {"location": map[string]string{"region": "CN", "apiKey": "bad"}}, {"themePreference": "unknown"}} {
		raw, _ := json.Marshal(bad)
		_, err := ValidateChatPreferences(raw)
		require.ErrorIs(t, err, ErrChatInvalid)
	}
	for _, raw := range []string{"null", "[]", "{} {}"} {
		_, err := ValidateChatPreferences(json.RawMessage(raw))
		require.ErrorIs(t, err, ErrChatInvalid)
	}
	raw, _ := json.Marshal(map[string]any{"locale": "zh-CN", "themePreference": "dark", "search": true, "thinking": false})
	validated, err := ValidateChatPreferences(raw)
	require.NoError(t, err)
	require.JSONEq(t, string(raw), string(validated))
	_, err = ValidateChatPreferences(json.RawMessage(strings.Repeat("x", ChatMaxPreferenceBytes+1)))
	require.ErrorIs(t, err, ErrChatTooLarge)
}

func TestChatPoliciesRejectWildcardAndProtocolMismatch(t *testing.T) {
	require.NoError(t, domain.ValidateChatModelPolicies(map[string]domain.ChatModelPolicy{"public-model": {Transport: "chat_completions"}}))
	require.Error(t, domain.ValidateChatModelPolicies(map[string]domain.ChatModelPolicy{"model-*": {Transport: "messages"}}))
	require.Error(t, domain.ValidateChatModelPolicies(map[string]domain.ChatModelPolicy{"model": {Transport: "responses", NativeFiles: true}}))
	require.Error(t, domain.ValidateChatModelPolicies(map[string]domain.ChatModelPolicy{"model": {Transport: "messages", DocumentInput: true}}))
}

func TestChatStoragePrivateAtomicAndBounded(t *testing.T) {
	dir := t.TempDir()
	s, err := NewChatStorage(dir)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, s.Close()) })
	id := uuid.New()
	saved, err := s.WriteFile(context.Background(), 7, id, strings.NewReader("hello"), 10)
	require.NoError(t, err)
	require.Equal(t, int64(5), saved.Size)
	require.Len(t, saved.SHA256, 64)
	f, err := s.OpenFile(7, id)
	require.NoError(t, err)
	contents, err := io.ReadAll(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	require.Equal(t, "hello", string(contents))
	_, err = s.OpenFile(8, id)
	require.Error(t, err)
	_, err = s.WriteFile(context.Background(), 7, id, strings.NewReader("overwrite"), 10)
	require.ErrorIs(t, err, os.ErrExist)
	_, err = s.WriteFile(context.Background(), 7, uuid.New(), strings.NewReader("too large"), 3)
	require.ErrorIs(t, err, ErrChatTooLarge)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = s.WriteFile(ctx, 7, uuid.New(), strings.NewReader("cancel"), 10)
	require.ErrorIs(t, err, context.Canceled)
	entries, err := os.ReadDir(filepath.Join(dir, "files", "7"))
	require.NoError(t, err)
	require.Len(t, entries, 1)
	if runtime.GOOS != "windows" {
		stat, err := os.Stat(filepath.Join(dir, filepath.FromSlash(saved.Key)))
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0600), stat.Mode().Perm())
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, filepath.FromSlash(saved.Key))+".tmp", []byte("interrupted"), 0600))
	require.NoError(t, s.RemoveFile(7, id))
	require.NoError(t, s.RemoveFile(7, id))
	entries, err = os.ReadDir(filepath.Join(dir, "files", "7"))
	require.NoError(t, err)
	require.Empty(t, entries)
}

func TestChatStorageRejectsSymlinkEscape(t *testing.T) {
	dir, outside := t.TempDir(), t.TempDir()
	s, err := NewChatStorage(dir)
	require.NoError(t, err)
	defer s.Close()
	if err = os.Symlink(outside, filepath.Join(dir, "files")); err != nil {
		t.Skipf("symlink privilege unavailable: %v", err)
	}
	_, err = s.WriteFile(context.Background(), 7, uuid.New(), strings.NewReader("private"), 10)
	require.Error(t, err)
	entries, err := os.ReadDir(outside)
	require.NoError(t, err)
	require.Empty(t, entries)
}

func TestChatSnapshotLegacyLinearHistory(t *testing.T) {
	id := uuid.New()
	tree := chatTestTree(id)
	delete(tree, "rootMessageId")
	messages := tree["messages"].([]any)[:2]
	for _, message := range messages {
		delete(message.(map[string]any), "parentId")
		delete(message.(map[string]any), "selectedChildId")
	}
	tree["messages"] = messages
	raw, err := json.Marshal(tree)
	require.NoError(t, err)
	snapshot, err := DecodeChatSnapshot(id, raw)
	require.NoError(t, err)
	require.Len(t, snapshot.VisibleMessages(), 2)
}
