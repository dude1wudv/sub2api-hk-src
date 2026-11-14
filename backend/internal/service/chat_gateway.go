package service

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/google/uuid"
)

type ChatGatewayDispatch func(http.ResponseWriter, *http.Request)

func (s *ChatService) BindGatewayDispatch(dispatch ChatGatewayDispatch) error {
	if s == nil || dispatch == nil {
		return ErrChatUnavailable
	}
	s.dispatchMu.Lock()
	defer s.dispatchMu.Unlock()
	if s.dispatch != nil {
		return fmt.Errorf("chat gateway is already bound: %w", ErrChatUnavailable)
	}
	s.dispatch = dispatch
	s.startLifecycle()
	return nil
}

// Cancellation/deadlines survive, but JWT/session/auth/request values do not.
// The internal request is authenticated anew by APIKeyAuth.
type chatGatewayContext struct{ context.Context }

func (chatGatewayContext) Value(any) any { return nil }

func (s *ChatService) dispatchGateway(ctx context.Context, w http.ResponseWriter, key *APIKey, userID, groupID int64, sessionID uuid.UUID, clientIP, path string, body []byte, before func(string) error) (string, error) {
	method := http.MethodPost
	switch path {
	case "/v1/models":
		method = http.MethodGet
	case "/v1/chat/completions", "/v1/messages", "/v1/responses", "/v1/tools/search", "/v1/tools/fetch":
	default:
		if !isChatGeminiPath(path) {
			return "", ErrChatInvalid
		}
	}
	if !chatCredentialValid(key, userID, groupID) {
		return "", ErrChatCredentialUnavailable
	}
	ip := net.ParseIP(clientIP)
	if ip == nil || sessionID == uuid.Nil {
		return "", ErrChatInvalid
	}
	if s.maxRequestBodyBytes > 0 && int64(len(body)) > s.maxRequestBodyBytes {
		return "", ErrChatTooLarge
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	s.dispatchMu.RLock()
	dispatch := s.dispatch
	s.dispatchMu.RUnlock()
	if dispatch == nil {
		return "", ErrChatUnavailable
	}
	requestID := uuid.NewString()
	internal := context.WithValue(chatGatewayContext{ctx}, ctxkey.ClientRequestID, requestID)
	if native, ok := ctx.Value(chatNativeContextKey{}).(*chatNativeRequest); ok {
		internal = context.WithValue(internal, chatNativeContextKey{}, native)
	}
	request, err := http.NewRequestWithContext(internal, method, "http://chat.internal"+path, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	request.RemoteAddr = net.JoinHostPort(ip.String(), "0")
	request.Header.Set("Authorization", "Bearer "+key.Key)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "text/event-stream, application/json")
	request.Header.Set("User-Agent", "Sub2API-Web-Chat")
	request.Header.Set("X-Request-ID", requestID)
	request.Header.Set("session_id", sessionID.String())
	if path == "/v1/messages" {
		request.Header.Set("anthropic-version", "2023-06-01")
		if native, ok := internal.Value(chatNativeContextKey{}).(*chatNativeRequest); ok && len(native.files) > 0 {
			request.Header.Set("anthropic-beta", "files-api-2025-04-14")
		}
	}
	// Persist correlation before any potentially billable action. Body limits and
	// all live policy/billing middleware are still evaluated by the real router.
	if before != nil {
		if err := before(requestID); err != nil {
			return requestID, err
		}
	}
	dispatch(w, request)
	return requestID, ctx.Err()
}

func (s *ChatService) ReconnectPreset(ctx context.Context, userID int64, id uuid.UUID) error {
	preset, _, _, err := s.models.ResolvePreset(ctx, userID, id)
	if err != nil {
		return err
	}
	return s.credentials.Reconnect(ctx, userID, preset.GroupID)
}
