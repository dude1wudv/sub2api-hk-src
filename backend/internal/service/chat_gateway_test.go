//go:build unit

package service

import (
	"context"
	"errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func chatGatewayTestKey() *APIKey {
	group := int64(11)
	return &APIKey{ID: 5, UserID: 7, GroupID: &group, Status: StatusActive, Key: "synthetic-private-key"}
}

func TestChatGatewayBindingPathsAndContextIsolation(t *testing.T) {
	s := &ChatService{}
	session := uuid.New()
	key := chatGatewayTestKey()
	recorder := httptest.NewRecorder()
	_, err := s.dispatchGateway(context.Background(), recorder, key, 7, 11, session, "192.0.2.8", "/v1/messages", nil, nil)
	require.ErrorIs(t, err, ErrChatUnavailable)
	require.ErrorIs(t, s.BindGatewayDispatch(nil), ErrChatUnavailable)
	var requests []*http.Request
	persisted := false
	dispatch := func(w http.ResponseWriter, r *http.Request) {
		require.True(t, persisted)
		requests = append(requests, r)
		w.WriteHeader(http.StatusOK)
	}
	require.NoError(t, s.BindGatewayDispatch(dispatch))
	require.ErrorIs(t, s.BindGatewayDispatch(dispatch), ErrChatUnavailable)
	type privateContextKey struct{}
	ctx := context.WithValue(context.Background(), privateContextKey{}, "jwt-user")
	ctx = context.WithValue(ctx, ctxkey.ClientRequestID, "outer-request")
	before := func(id string) error { _, e := uuid.Parse(id); require.NoError(t, e); persisted = true; return nil }
	a, err := s.dispatchGateway(ctx, recorder, key, 7, 11, session, "192.0.2.8", "/v1/messages", []byte("{}"), before)
	require.NoError(t, err)
	b, err := s.dispatchGateway(ctx, recorder, key, 7, 11, session, "192.0.2.8", "/v1/tools/search", []byte("{}"), before)
	require.NoError(t, err)
	require.NotEqual(t, a, b)
	require.Len(t, requests, 2)
	r := requests[0]
	require.Equal(t, "192.0.2.8:0", r.RemoteAddr)
	require.Equal(t, "chat.internal", r.Host)
	require.Equal(t, "Bearer synthetic-private-key", r.Header.Get("Authorization"))
	require.Empty(t, r.Header.Get("Cookie"))
	require.Empty(t, r.Header.Get("X-Forwarded-For"))
	require.Empty(t, r.URL.RawQuery)
	require.Nil(t, r.Context().Value(privateContextKey{}))
	require.Equal(t, a, r.Context().Value(ctxkey.ClientRequestID))
	require.Equal(t, session.String(), r.Header.Get("session_id"))
	for _, path := range []string{"https://evil.example/v1/messages", "/api/v1/admin/users", "/v1/messages?api_key=x", "/v1/messages/../models"} {
		_, err = s.dispatchGateway(ctx, recorder, key, 7, 11, session, "192.0.2.8", path, nil, nil)
		require.ErrorIs(t, err, ErrChatInvalid)
	}
	require.Len(t, requests, 2)
	failure := errors.New("could not save request ID")
	_, err = s.dispatchGateway(ctx, recorder, key, 7, 11, session, "192.0.2.8", "/v1/messages", nil, func(string) error { return failure })
	require.ErrorIs(t, err, failure)
	require.Len(t, requests, 2)
	s.maxRequestBodyBytes = 1
	_, err = s.dispatchGateway(ctx, recorder, key, 7, 11, session, "192.0.2.8", "/v1/messages", []byte("{}"), nil)
	require.ErrorIs(t, err, ErrChatTooLarge)
}

func TestChatGatewayCancellationReachesInternalRequest(t *testing.T) {
	s := &ChatService{}
	started := make(chan struct{})
	done := make(chan error, 1)
	require.NoError(t, s.BindGatewayDispatch(func(_ http.ResponseWriter, r *http.Request) { close(started); <-r.Context().Done() }))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		_, err := s.dispatchGateway(ctx, httptest.NewRecorder(), chatGatewayTestKey(), 7, 11, uuid.New(), "2001:db8::1", "/v1/messages", nil, nil)
		done <- err
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("dispatch did not start")
	}
	cancel()
	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(2 * time.Second):
		t.Fatal("dispatch did not stop")
	}
}
