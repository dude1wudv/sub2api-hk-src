package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/mirasim"
)

type mirasimSession struct {
	mu           sync.Mutex
	client       *mirasim.Client
	initialToken string
	currentToken string
	proxyURL     string
}

var mirasimSessions sync.Map // account ID -> *mirasimSession

func (s *OpenAIGatewayService) signMirasimRequest(request *http.Request, proxyURL string, account *Account) (*http.Request, error) {
	if request == nil || account == nil {
		return nil, errors.New("mirasim request or account is nil")
	}
	// The refresh token and device ticket must never be sent to a configurable base URL.
	if request.URL == nil || request.URL.Scheme != "https" || request.URL.Host != "relay.mirasim.ai" || request.URL.RawQuery != "" {
		return nil, errors.New("mirasim relay URL is not the official endpoint")
	}
	path := request.URL.EscapedPath()
	switch path {
	case "/v1/messages", "/v1/responses", "/v1/chat/completions":
	default:
		return nil, fmt.Errorf("unsupported mirasim endpoint: %s", path)
	}
	refreshToken := strings.TrimSpace(account.GetCredential("refresh_token"))
	privateKey := account.GetCredential("private_key")
	if refreshToken == "" || privateKey == "" {
		return nil, errors.New("mirasim OAuth credentials are missing")
	}
	loaded, ok := mirasimSessions.Load(account.ID)
	session, _ := loaded.(*mirasimSession)
	currentToken := ""
	if session != nil {
		session.mu.Lock()
		currentToken = session.currentToken
		session.mu.Unlock()
	}
	if !ok || session == nil || session.proxyURL != proxyURL || (refreshToken != session.initialToken && refreshToken != currentToken) {
		key, err := mirasim.ParsePrivateKeyPEM(privateKey)
		if err != nil {
			return nil, fmt.Errorf("invalid mirasim device key: %w", err)
		}
		client, err := mirasim.NewClient(refreshToken, key, "", "", "", "", proxyURL)
		if err != nil {
			return nil, err
		}
		session = &mirasimSession{client: client, initialToken: refreshToken, currentToken: refreshToken, proxyURL: proxyURL}
		client.OnRotate = func(newToken string) {
			session.mu.Lock()
			oldToken := session.currentToken
			session.currentToken = newToken
			session.mu.Unlock()
			s.persistMirasimRefreshToken(request.Context(), account.ID, oldToken, newToken)
		}
		mirasimSessions.Store(account.ID, session)
	}
	body, err := io.ReadAll(io.LimitReader(request.Body, 8<<20+1))
	if err != nil {
		return nil, err
	}
	if len(body) > 8<<20 {
		return nil, errors.New("mirasim request body too large")
	}
	body, err = normalizeMirasimBody(path, body)
	if err != nil {
		return nil, err
	}
	signed, err := session.client.SignedRequest(request.Context(), request.Method, path, body)
	if err != nil {
		return nil, err
	}
	for name, values := range request.Header {
		switch http.CanonicalHeaderKey(name) {
		case "Authorization", "X-Api-Key", "X-Mirasim-Client", "X-Mirasim-Enc", "Host":
			continue
		}
		for _, value := range values {
			signed.Header.Add(name, value)
		}
	}
	return signed, nil
}

func (s *OpenAIGatewayService) persistMirasimRefreshToken(ctx context.Context, accountID int64, oldToken, newToken string) {
	if s.accountRepo == nil {
		slog.Error("mirasim refresh token rotation cannot be persisted", "account_id", accountID)
		return
	}
	persistCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	account, err := s.accountRepo.GetByID(persistCtx, accountID)
	if err != nil {
		slog.Error("mirasim refresh token rotation lookup failed", "account_id", accountID, "error", err)
		return
	}
	if account.Platform != PlatformMirasim || account.GetCredential("refresh_token") != oldToken {
		// Reauthorization may have replaced this account while a request was in flight.
		return
	}
	credentials := shallowCopyMap(account.Credentials)
	credentials["refresh_token"] = newToken
	if err := persistAccountCredentials(persistCtx, s.accountRepo, account, credentials); err != nil {
		slog.Error("mirasim refresh token rotation save failed", "account_id", accountID, "error", err)
	}
}
