//go:build unit

package service

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/mirasim"
	"github.com/stretchr/testify/require"
)

func TestMirasimModelSyncUsesSignedBodylessGetAndExtractsRelayCatalog(t *testing.T) {
	privateKey, err := mirasim.GenerateDeviceKey()
	require.NoError(t, err)
	privateKeyPEM, err := mirasim.MarshalPrivateKeyPEM(privateKey)
	require.NoError(t, err)
	const (
		accountID    = int64(928732)
		refreshToken = "mirasim-sync-refresh"
		accessToken  = "mirasim-sync-access"
		ticket       = "mirasim-sync-ticket"
	)

	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/auth/refresh" {
			http.Error(w, "unexpected auth request", http.StatusNotFound)
			return
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body["refresh_token"] != refreshToken {
			http.Error(w, "invalid refresh request", http.StatusBadRequest)
			return
		}
		_, _ = io.WriteString(w, `{"access_token":"`+accessToken+`","refresh_token":"`+refreshToken+`","expires_in":3600}`)
	}))
	defer authServer.Close()

	validSessionSignature := atomic.Bool{}
	relayServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/device/session" {
			http.Error(w, "unexpected relay request", http.StatusNotFound)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "cannot read session body", http.StatusBadRequest)
			return
		}
		var sessionRequest struct {
			PublicKey string `json:"publicKey"`
			DeviceID  string `json:"deviceId"`
		}
		if json.Unmarshal(body, &sessionRequest) != nil || r.Header.Get("Authorization") != "Bearer "+accessToken {
			http.Error(w, "invalid session request", http.StatusBadRequest)
			return
		}
		der, err := base64.StdEncoding.DecodeString(sessionRequest.PublicKey)
		if err != nil {
			http.Error(w, "invalid public key encoding", http.StatusBadRequest)
			return
		}
		parsedKey, err := x509.ParsePKIXPublicKey(der)
		publicKey, ok := parsedKey.(ed25519.PublicKey)
		if err != nil || !ok {
			http.Error(w, "invalid public key", http.StatusBadRequest)
			return
		}
		ts, err := strconv.ParseInt(r.Header.Get("x-mirasim-ts"), 10, 64)
		signature, decodeErr := base64.RawURLEncoding.DecodeString(r.Header.Get("x-mirasim-sig"))
		canonical := mirasim.CanonicalString(r.Method, r.URL.Path, ts, r.Header.Get("x-mirasim-nonce"), sessionRequest.DeviceID, r.Header.Get("x-mirasim-client"), accessToken, nil, body)
		if err == nil && decodeErr == nil && r.Header.Get("x-mirasim-device") == sessionRequest.DeviceID && ed25519.Verify(publicKey, []byte(canonical), signature) {
			validSessionSignature.Store(true)
		}
		_, _ = io.WriteString(w, `{"ticket":"`+ticket+`","expires_in":900}`)
	}))
	defer relayServer.Close()

	client, err := mirasim.NewClient(refreshToken, privateKey, relayServer.URL, authServer.URL, "", "", "")
	require.NoError(t, err)
	session := &mirasimSession{
		client:         client,
		initialToken:   refreshToken,
		currentToken:   refreshToken,
		persistedToken: refreshToken,
		keyHash:        sha256.Sum256([]byte(privateKeyPEM)),
	}
	mirasimSessions.Store(accountID, session)
	defer mirasimSessions.Delete(accountID)

	account := &Account{
		ID: accountID, Platform: PlatformMirasim, Type: AccountTypeAPIKey,
		Credentials: map[string]any{"refresh_token": refreshToken, "private_key": privateKeyPEM},
	}
	relayCatalogBody := `{"data":[{"id":"glm-5.3-flash"},{"id":"kimi-k3"}]}`
	upstream := &httpUpstreamRecorder{resp: &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(relayCatalogBody)),
	}}
	accountTest := NewAccountTestService(nil, nil, nil, nil, nil, upstream, &config.Config{}, nil)
	accountTest.SetOpenAIGatewayService(&OpenAIGatewayService{})

	catalog, err := accountTest.SyncUpstreamModelCatalog(context.Background(), account)
	require.NoError(t, err)
	require.Equal(t, []string{"glm-5.3-flash", "kimi-k3"}, catalog.Models)
	require.True(t, validSessionSignature.Load(), "device-session signature must verify")
	require.NotNil(t, upstream.lastReq)
	require.Equal(t, http.MethodGet, upstream.lastReq.Method)
	require.Equal(t, "/v1/models", upstream.lastReq.URL.Path)
	require.Nil(t, upstream.lastReq.Body, "model sync GET must have no request body")
	require.Equal(t, "Bearer "+ticket, upstream.lastReq.Header.Get("Authorization"))
	require.Equal(t, mirasim.DefaultClientVersion, upstream.lastReq.Header.Get("x-mirasim-client"))
	require.NotEmpty(t, upstream.lastReq.Header.Get("x-mirasim-enc"), "model request must include sealed signature headers")
	require.Empty(t, upstream.lastReq.Header.Get("x-api-key"), "Mirasim OAuth model sync must not use static API-key auth")
	require.NotContains(t, account.Credentials, "api_key")
}
