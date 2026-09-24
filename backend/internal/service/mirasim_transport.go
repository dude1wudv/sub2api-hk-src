package service

import (
	"context"
	"crypto/sha256"
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
	mu             sync.Mutex
	client         *mirasim.Client
	initialToken   string
	currentToken   string
	proxyURL       string
	keyHash        [32]byte
	persistedToken string
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
	case "/v1/models":
		if request.Method != http.MethodGet {
			return nil, errors.New("Mirasim model catalog requires GET")
		}
	case "/v1/messages", "/v1/responses", "/v1/chat/completions":
		if request.Method != http.MethodPost {
			return nil, errors.New("Mirasim inference requires POST")
		}
	default:
		return nil, fmt.Errorf("unsupported mirasim endpoint: %s", path)
	}
	refreshToken := strings.TrimSpace(account.GetCredential("refresh_token"))
	privateKey := account.GetCredential("private_key")
	if refreshToken == "" || privateKey == "" {
		return nil, errors.New("mirasim OAuth credentials are missing")
	}
	loaded, _ := mirasimSessions.LoadOrStore(account.ID, &mirasimSession{})
	session := loaded.(*mirasimSession)
	// One client/rotation chain per account, including concurrent cold starts.
	session.mu.Lock()
	defer session.mu.Unlock()
	keyHash := sha256.Sum256([]byte(privateKey))
	accountToken := refreshToken
	sameAuthorization := session.client != nil && session.keyHash == keyHash &&
		(refreshToken == session.initialToken || refreshToken == session.currentToken || refreshToken == session.persistedToken)
	if sameAuthorization && session.proxyURL != proxyURL {
		// A proxy edit must not discard a rotation still waiting to be saved.
		if session.currentToken != session.persistedToken {
			if err := s.persistMirasimRefreshToken(request.Context(), account.ID, session.persistedToken, session.currentToken); err != nil {
				return nil, err
			}
			session.persistedToken = session.currentToken
		}
		refreshToken = session.currentToken
	}
	if session.client == nil || session.proxyURL != proxyURL || session.keyHash != keyHash || (refreshToken != session.initialToken && refreshToken != session.currentToken && refreshToken != session.persistedToken) {
		key, err := mirasim.ParsePrivateKeyPEM(privateKey)
		if err != nil {
			return nil, fmt.Errorf("invalid mirasim device key: %w", err)
		}
		client, err := mirasim.NewClient(refreshToken, key, "", "", "", "", proxyURL)
		if err != nil {
			return nil, err
		}
		session.client = client
		session.initialToken, session.currentToken, session.persistedToken = accountToken, refreshToken, refreshToken
		session.proxyURL, session.keyHash = proxyURL, keyHash
		client.OnRotate = func(newToken string) {
			session.currentToken = newToken
		}
	}
	var body []byte
	if path != "/v1/models" {
		if request.Body == nil {
			return nil, errors.New("Mirasim request body is required")
		}
		var err error
		body, err = io.ReadAll(io.LimitReader(request.Body, 8<<20+1))
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
	}
	// Retry a failed save before any subsequent refresh can advance the chain.
	if session.currentToken != session.persistedToken {
		if err := s.persistMirasimRefreshToken(request.Context(), account.ID, session.persistedToken, session.currentToken); err != nil {
			return nil, err
		}
		session.persistedToken = session.currentToken
	}
	signed, err := session.client.SignedRequest(request.Context(), request.Method, path, body)
	// Persist even if ticket minting fails after a successful token rotation.
	if session.currentToken != session.persistedToken {
		if persistErr := s.persistMirasimRefreshToken(request.Context(), account.ID, session.persistedToken, session.currentToken); persistErr != nil {
			return nil, persistErr
		}
		session.persistedToken = session.currentToken
	}
	if err != nil {
		return nil, err
	}
	for name, values := range request.Header {
		if strings.HasPrefix(strings.ToLower(name), "x-mirasim-") {
			continue
		}
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

func (s *OpenAIGatewayService) persistMirasimRefreshToken(ctx context.Context, accountID int64, oldToken, newToken string) error {
	if s.accountRepo == nil {
		slog.Error("mirasim refresh token rotation cannot be persisted", "account_id", accountID)
		return errors.New("Mirasim credential persistence is unavailable")
	}
	persistCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	account, err := s.accountRepo.GetByID(persistCtx, accountID)
	if err != nil {
		slog.Error("mirasim refresh token rotation lookup failed", "account_id", accountID, "error", err)
		return errors.New("Mirasim credential lookup failed")
	}
	if account.Platform != PlatformMirasim || account.GetCredential("refresh_token") != oldToken {
		// Reauthorization may have replaced this account while a request was in flight.
		return errors.New("Mirasim account was reauthorized; retry with current credentials")
	}
	credentials := shallowCopyMap(account.Credentials)
	credentials["refresh_token"] = newToken
	if err := persistAccountCredentials(persistCtx, s.accountRepo, account, credentials); err != nil {
		slog.Error("mirasim refresh token rotation save failed", "account_id", accountID, "error", err)
		return errors.New("Mirasim credential rotation could not be saved; retry later")
	}
	return nil
}
