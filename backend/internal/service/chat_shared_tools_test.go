//go:build unit

package service

import (
	"context"
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestChatSharedToolsAcrossTransports(t *testing.T) {
	for _, tc := range []struct{ transport, platform string }{{"chat_completions", PlatformOpenAI}, {"responses", PlatformOpenAI}, {"messages", PlatformAnthropic}, {"gemini", PlatformGemini}} {
		t.Run(tc.transport, func(t *testing.T) {
			s, repo, request := chatGenerationFixture(t)
			groups := s.models.groups.(*chatGroupsStub)
			groups.groups[0].Platform = tc.platform
			groups.groups[0].ChatModelPolicies = map[string]ChatModelPolicy{"shared-model": {Transport: tc.transport}}
			candidates := s.models.accounts.(*chatAccountsStub)
			candidates.accounts[0].Platform = tc.platform
			shared := newDeepSeekToolsAccount()
			shared.Credentials["base_url"] = "https://api.deepseek.com"
			shared.GroupIDs = []int64{99}
			shared.Extra = nil
			s.auxiliary = NewChatAuxiliaryService(&config.Config{Chat: config.ChatConfig{AuxiliaryAccountID: shared.ID}}, &deepSeekToolsAccountRepo{latest: shared})
			s.models.auxiliary = s.auxiliary
			yes := true
			caps, _, ok, err := s.models.modelCapabilities(t.Context(), &groups.groups[0], "shared-model", ModelMetadata{MetadataMatch: "exact", ToolCall: &yes}, ChatModelPolicy{Transport: tc.transport}, candidates.accounts)
			require.NoError(t, err)
			require.True(t, ok)
			require.True(t, caps.Search)
			key, err := s.credentials.Ensure(t.Context(), 7, 11)
			require.NoError(t, err)
			calls := 0
			s.dispatch = func(w http.ResponseWriter, r *http.Request) {
				calls++
				require.Equal(t, "/internal/chat/auxiliary/search", r.URL.Path)
				require.True(t, HasChatAuxiliaryScope(r.Context()))
				scope := r.Context().Value(chatAuxiliaryContextKey{}).(*chatAuxiliaryScope)
				require.Equal(t, int64(11), scope.sourceGroupID)
				require.Equal(t, shared.ID, scope.configuredAccountID)
				require.Equal(t, "Bearer "+key.Key, r.Header.Get("Authorization"))
				_, _ = w.Write([]byte(`{"results":[{"title":"Synthetic","url":"https://example.test","snippet":"data"}]}`))
			}
			run := &ChatRunRecord{UserID: 7, GroupID: 11, PresetID: request.PresetID, SessionID: repo.record.ID}
			call := chatToolCall{ID: "call-1", Name: "chat_search", Arguments: `{"query":"data"}`}
			emit := func(map[string]any) error { return nil }
			_, err = s.executeChatTool(t.Context(), key, run, "192.0.2.1", call, caps, ChatRunOptions{Search: true}, emit, 0, nil)
			require.NoError(t, err)
			require.Equal(t, 1, calls)
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			_, err = s.executeChatTool(ctx, key, run, "192.0.2.1", call, caps, ChatRunOptions{Search: true}, emit, 0, nil)
			require.Error(t, err)
			require.Equal(t, 1, calls)
			run.UserID = 8
			_, err = s.executeChatTool(t.Context(), key, run, "192.0.2.1", call, caps, ChatRunOptions{Search: true}, emit, 0, nil)
			require.Error(t, err)
			require.Equal(t, 1, calls)
			shared.Schedulable = false
			run.UserID = 7
			_, err = s.executeChatTool(t.Context(), key, run, "192.0.2.1", call, caps, ChatRunOptions{Search: true}, emit, 0, nil)
			require.Error(t, err)
			require.Equal(t, 1, calls)
		})
	}
}
