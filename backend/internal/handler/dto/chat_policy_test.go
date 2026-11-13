package dto

import (
	"encoding/json"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestChatPolicyAdminOnlyDTO(t *testing.T) {
	group := &service.Group{ID: 9, ChatModelPolicies: map[string]service.ChatModelPolicy{"exact-model": {Transport: "messages", NativeFiles: true}}}
	admin := GroupFromServiceAdmin(group)
	require.Equal(t, group.ChatModelPolicies, admin.ChatModelPolicies)
	encoded, err := json.Marshal(admin)
	require.NoError(t, err)
	require.Contains(t, string(encoded), "chat_model_policies")
	user, err := json.Marshal(GroupFromService(group))
	require.NoError(t, err)
	require.NotContains(t, string(user), "chat_model_policies")
	delete(admin.ChatModelPolicies, "exact-model")
	require.Len(t, group.ChatModelPolicies, 1)
}
