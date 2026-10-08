//go:build unit

package middleware

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type smartRoutingMiddlewareUserRepo struct {
	service.UserRepository
	user *service.User
}

func (r *smartRoutingMiddlewareUserRepo) GetByID(context.Context, int64) (*service.User, error) {
	return r.user, nil
}

type smartRoutingMiddlewareGroupRepo struct {
	service.GroupRepository
	groups map[int64]*service.Group
}

func (r *smartRoutingMiddlewareGroupRepo) GetByID(_ context.Context, id int64) (*service.Group, error) {
	group, ok := r.groups[id]
	if !ok {
		return nil, service.ErrGroupNotFound
	}
	return group, nil
}

func newSmartRoutingMiddlewareKeyService(groups ...*service.Group) *service.APIKeyService {
	byID := make(map[int64]*service.Group, len(groups))
	groupIDs := make([]int64, 0, len(groups))
	for _, group := range groups {
		byID[group.ID] = group
		groupIDs = append(groupIDs, group.ID)
	}
	user := &service.User{ID: 7, Status: service.StatusActive, AllowedGroups: groupIDs}
	return service.NewAPIKeyService(
		&stubApiKeyRepo{},
		&smartRoutingMiddlewareUserRepo{user: user},
		&smartRoutingMiddlewareGroupRepo{groups: byID},
		nil, nil, nil, &config.Config{},
	)
}

func TestResolveSmartRoutingPreservesParsedRequestBodiesForSupportedClaudeEntries(t *testing.T) {
	gin.SetMode(gin.TestMode)
	group := &service.Group{ID: 119, Status: service.StatusActive, Platform: service.PlatformAnthropic}
	keys := newSmartRoutingMiddlewareKeyService(group)
	cases := []struct {
		name string
		path string
		body string
	}{
		{name: "messages", path: "/v1/messages", body: `{"model":"claude-sonnet-4-5","max_tokens":8,"messages":[{"role":"user","content":"hi"}]}`},
		{name: "chat", path: "/v1/chat/completions", body: `{"model":"claude-sonnet-4-5","messages":[{"role":"user","content":"hi"}]}`},
		{name: "responses", path: "/v1/responses", body: `{"model":"claude-sonnet-4-5","input":"hi"}`},
		{name: "count_tokens", path: "/v1/messages/count_tokens", body: `{"model":"claude-sonnet-4-5","messages":[{"role":"user","content":"hi"}]}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, tc.path, bytes.NewBufferString(tc.body))
			request.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = request
			key := &service.APIKey{ID: 14, UserID: 7, GroupID: &group.ID, Group: group, RoutingGroupIDs: []int64{group.ID}}

			_, routed := resolveSmartRoutingKey(ctx, keys, smartRoutingGateways{openai: &service.OpenAIGatewayService{}}, key, false)

			require.False(t, routed)
			require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
			require.Contains(t, recorder.Body.String(), "Claude routing is temporarily unavailable", "the valid Claude group must be selected after parsing each supported request")
			require.Equal(t, tc.body, string(mustReadMiddlewareRequestBody(t, ctx.Request)), "body parsing for each protocol must restore the exact client payload")
		})
	}
}

func TestResolveSmartRoutingMessagesValidationUsesAnthropicErrorEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	request := httptest.NewRequest(http.MethodPost, "/v1/messages", bytes.NewBufferString(`{"max_tokens":8,"messages":[]}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request
	key := &service.APIKey{UserID: 7, RoutingGroupIDs: []int64{1}}

	_, routed := resolveSmartRoutingKey(ctx, &service.APIKeyService{}, smartRoutingGateways{openai: &service.OpenAIGatewayService{}}, key, false)

	require.False(t, routed)
	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.JSONEq(t, `{"type":"error","error":{"type":"invalid_request_error","message":"A non-empty model is required for smart routing","code":"INVALID_REQUEST"}}`, recorder.Body.String())
}

func TestAbortSmartRoutingErrorUsesClaudeEnvelopeOnlyForMessagesPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cases := []struct {
		path string
		want string
	}{
		{path: "/v1/messages", want: `{"type":"error","error":{"type":"overloaded_error","message":"busy","code":"BUSY"}}`},
		{path: "/v1/messages/count_tokens", want: `{"type":"error","error":{"type":"overloaded_error","message":"busy","code":"BUSY"}}`},
		{path: "/v1/chat/completions", want: `{"code":"BUSY","message":"busy"}`},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodPost, tc.path, nil)
			abortSmartRoutingError(ctx, http.StatusServiceUnavailable, "BUSY", "busy")
			require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
			require.JSONEq(t, tc.want, recorder.Body.String())
		})
	}
}

func TestSmartRoutingPreservesResponsesOperationsPayloadUntilCandidateResolution(t *testing.T) {
	gin.SetMode(gin.TestMode)
	group := &service.Group{ID: 120, Status: service.StatusActive, Platform: service.PlatformAnthropic}
	keys := newSmartRoutingMiddlewareKeyService(group)
	for _, body := range []string{
		`{"model":"claude-sonnet-4-5","previous_response_id":"resp_123","input":"hi"}`,
		`{"model":"claude-sonnet-4-5","input":[{"type":"compaction_trigger"}]}`,
	} {
		request := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewBufferString(body))
		request.Header.Set("Content-Type", "application/json")
		recorder := httptest.NewRecorder()
		ctx, _ := gin.CreateTestContext(recorder)
		ctx.Request = request
		key := &service.APIKey{ID: 16, UserID: 7, GroupID: &group.ID, Group: group, RoutingGroupIDs: []int64{group.ID}}

		_, routed := resolveSmartRoutingKey(ctx, keys, smartRoutingGateways{openai: &service.OpenAIGatewayService{}}, key, false)

		require.False(t, routed)
		require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
		require.Contains(t, recorder.Body.String(), "SMART_ROUTING_NO_AVAILABLE_GROUP", "the Claude candidate should be skipped for Responses-only operations")
		require.Equal(t, body, string(mustReadMiddlewareRequestBody(t, ctx.Request)), "unsupported operations must preserve the request while candidate filtering runs")
	}
}

func TestResolveSmartRoutingInspectionUsesConfiguredGroupOrder(t *testing.T) {
	gin.SetMode(gin.TestMode)
	openai := &service.Group{ID: 122, Status: service.StatusActive, Platform: service.PlatformOpenAI}
	claude := &service.Group{ID: 123, Status: service.StatusActive, Platform: service.PlatformAnthropic}
	keys := newSmartRoutingMiddlewareKeyService(openai, claude)
	key := &service.APIKey{ID: 17, UserID: 7, GroupID: &openai.ID, Group: openai, RoutingGroupIDs: []int64{openai.ID, claude.ID}}
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/usage", nil)

	selected, routed := resolveSmartRoutingKey(ctx, keys, smartRoutingGateways{openai: &service.OpenAIGatewayService{}}, key, false)

	require.True(t, routed)
	require.Equal(t, openai.ID, *selected.GroupID, "SmartRoutingKeys order must determine the first configured group")
	require.Equal(t, service.PlatformOpenAI, selected.Group.Platform)
}

func TestClaudeSmartRoutingSkipsResponsesWithOwnedStateOrCompaction(t *testing.T) {
	gin.SetMode(gin.TestMode)
	group := &service.Group{ID: 121, Status: service.StatusActive, Platform: service.PlatformAnthropic}
	keys := newSmartRoutingMiddlewareKeyService(group)
	cases := []struct {
		name string
		path string
		body string
	}{
		{
			name: "previous_response_id",
			path: "/v1/responses",
			body: `{"model":"claude-sonnet-4-5","previous_response_id":"resp_owned","input":"hi"}`,
		},
		{
			name: "compact_endpoint",
			path: "/v1/responses/compact",
			body: `{"model":"claude-sonnet-4-5","input":"hi"}`,
		},
		{
			name: "compaction_trigger",
			path: "/v1/responses",
			body: `{"model":"claude-sonnet-4-5","stream":false,"input":[{"type":"compaction_trigger"}]}`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, tc.path, bytes.NewBufferString(tc.body))
			request.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = request
			key := &service.APIKey{ID: 15, UserID: 7, GroupID: &group.ID, Group: group, RoutingGroupIDs: []int64{group.ID}}

			_, routed := resolveSmartRoutingKey(ctx, keys, smartRoutingGateways{openai: &service.OpenAIGatewayService{}}, key, false)

			require.False(t, routed)
			require.Equal(t, http.StatusServiceUnavailable, recorder.Code)
			require.Contains(t, recorder.Body.String(), "SMART_ROUTING_NO_AVAILABLE_GROUP", "Claude candidates must be skipped when response ownership or compaction requires the OpenAI Responses path")
			require.NotContains(t, recorder.Body.String(), "Claude routing is temporarily unavailable", "the resolver must not try the Claude gateway for these requests")
			require.Equal(t, tc.body, string(mustReadMiddlewareRequestBody(t, ctx.Request)))
		})
	}
}

func mustReadMiddlewareRequestBody(t *testing.T, request *http.Request) []byte {
	t.Helper()
	body, err := httputil.ReadRequestBodyWithPrealloc(request)
	require.NoError(t, err)
	return body
}
