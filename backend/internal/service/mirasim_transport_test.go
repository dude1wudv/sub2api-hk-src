package service

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestMirasimSignerRejectsNonOfficialRelayBeforeReadingCredentials(t *testing.T) {
	svc := &OpenAIGatewayService{}
	account := &Account{ID: 1, Platform: PlatformMirasim, Credentials: map[string]any{
		"refresh_token": "secret-refresh-token",
		"private_key":   "secret-private-key",
	}}
	for _, target := range []string{
		"https://relay.mirasim.ai.evil.example/v1/messages",
		"http://relay.mirasim.ai/v1/messages",
		"https://relay.mirasim.ai/v1/models",
		"https://relay.mirasim.ai/v1/messages?redirect=https://evil.example",
	} {
		req, err := http.NewRequest(http.MethodPost, target, strings.NewReader(`{"model":"claude-sonnet-5"}`))
		if err != nil {
			t.Fatal(err)
		}
		if _, err := svc.signMirasimRequest(req, "", account); err == nil || strings.Contains(err.Error(), "secret-") {
			t.Fatalf("target %s was not safely rejected: %v", target, err)
		}
	}
}

type mirasimCredentialsPersistenceRepo struct {
	AccountRepository
	account     Account
	updateCalls int
	failUpdate  bool
}

func (r *mirasimCredentialsPersistenceRepo) GetByID(context.Context, int64) (*Account, error) {
	copy := r.account
	copy.Credentials = shallowCopyMap(r.account.Credentials)
	return &copy, nil
}

func (r *mirasimCredentialsPersistenceRepo) UpdateCredentials(_ context.Context, _ int64, credentials map[string]any) error {
	r.updateCalls++
	if r.failUpdate {
		r.failUpdate = false
		return errors.New("temporary storage failure")
	}
	r.account.Credentials = shallowCopyMap(credentials)
	return nil
}

func TestMirasimRefreshTokenPersistenceCanRetryAfterSaveFailure(t *testing.T) {
	repo := &mirasimCredentialsPersistenceRepo{
		account: Account{ID: 71, Platform: PlatformMirasim, Credentials: map[string]any{
			"refresh_token": "refresh-old",
			"private_key":   "device-private-key",
		}},
		failUpdate: true,
	}
	svc := &OpenAIGatewayService{accountRepo: repo}
	if err := svc.persistMirasimRefreshToken(context.Background(), 71, "refresh-old", "refresh-new"); err == nil || !strings.Contains(err.Error(), "could not be saved") {
		t.Fatalf("expected save failure, got %v", err)
	}
	if got := repo.account.GetCredential("refresh_token"); got != "refresh-old" {
		t.Fatalf("failed update changed stored refresh token: %q", got)
	}
	if err := svc.persistMirasimRefreshToken(context.Background(), 71, "refresh-old", "refresh-new"); err != nil {
		t.Fatalf("retry failed: %v", err)
	}
	if got := repo.account.GetCredential("refresh_token"); got != "refresh-new" {
		t.Fatalf("retry did not persist rotated refresh token: %q", got)
	}
	if repo.updateCalls != 2 {
		t.Fatalf("update attempts = %d, want 2", repo.updateCalls)
	}
}
