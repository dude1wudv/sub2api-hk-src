package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"time"
)

import (
	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	WorkbenchHeliosGen = "heliosgen"
	WorkbenchKeyName   = "HeliosGen Workbench"
	WorkbenchSessionTTLSeconds = 86400
)

var (
	ErrWorkbenchKeyDisabled = infraerrors.Conflict("workbench_key_disabled", "workbench key is disabled")
	ErrWorkbenchInvalidGrant = infraerrors.BadRequest("invalid_grant", "invalid grant")
	ErrWorkbenchInvalidClient = infraerrors.Unauthorized("invalid_client", "invalid client")
	ErrWorkbenchUnavailable = infraerrors.ServiceUnavailable("WORKBENCH_UNAVAILABLE", "workbench integration is unavailable")
)

// WorkbenchCredential is the service representation of a dedicated workbench key binding.
type WorkbenchCredential struct {
	ID        int64
	UserID    int64
	Workbench string
	APIKeyID  int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

// WorkbenchLaunchGrant contains only persisted grant metadata. Raw codes never
// appear in this type; callers receive a one-time code only at issuance.
type WorkbenchLaunchGrant struct {
	ID          int64
	CodeHash    string
	UserID      int64
	APIKeyID    int64
	ClientID    string
	RedirectURI string
	ExpiresAt   time.Time
	ConsumedAt  *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// WorkbenchGrantResolution is returned after atomic consumption. The
// repository commits consumption before returning, even when the referenced
// user/key/binding is no longer valid, so invalid grants cannot be retried.
type WorkbenchGrantResolution struct {
	Grant      *WorkbenchLaunchGrant
	Credential *WorkbenchCredential
	User       *User
	APIKey     *APIKey
}

type WorkbenchIntegrationRepository interface {
	EnsureCredential(ctx context.Context, userID int64, workbench, apiKey string) (*WorkbenchCredential, error)
	CreateGrant(ctx context.Context, grant *WorkbenchLaunchGrant) error
	ConsumeGrant(ctx context.Context, codeHash string, now time.Time) (*WorkbenchGrantResolution, error)
}

type WorkbenchLaunch struct {
	URL       string `json:"launch_url"`
	ExpiresIn int    `json:"expires_in"`
}

type WorkbenchIntegrationService struct {
	repo         WorkbenchIntegrationRepository
	apiKeyService *APIKeyService
	cfg          *config.Config
}

func NewWorkbenchIntegrationService(repo WorkbenchIntegrationRepository, apiKeyService *APIKeyService, cfg *config.Config) *WorkbenchIntegrationService {
	return &WorkbenchIntegrationService{repo: repo, apiKeyService: apiKeyService, cfg: cfg}
}

func (s *WorkbenchIntegrationService) enabled() bool {
	return s != nil && s.repo != nil && s.cfg != nil && s.cfg.HeliosWorkbench.Enabled
}

func (s *WorkbenchIntegrationService) EnsureCredential(ctx context.Context, userID int64) (*WorkbenchCredential, error) {
	if !s.enabled() || userID <= 0 || s.apiKeyService == nil {
		return nil, ErrWorkbenchUnavailable
	}
	// The repository serializes the binding row under the user row lock. The
	for range 8 {
		key, err := s.apiKeyService.GenerateKey()
		if err != nil {
			return nil, fmt.Errorf("generate workbench api key: %w", err)
		}
		credential, err := s.repo.EnsureCredential(ctx, userID, WorkbenchHeliosGen, key)
		if err == nil {
			return credential, nil
		}
		if errors.Is(err, ErrAPIKeyExists) {
			continue
		}
		return nil, err
	}
	return nil, fmt.Errorf("generate workbench api key: %w", ErrAPIKeyExists)
}

func (s *WorkbenchIntegrationService) CreateLaunch(ctx context.Context, userID int64) (*WorkbenchLaunch, error) {
	if !s.enabled() {
		return nil, ErrWorkbenchUnavailable
	}
	credential, err := s.EnsureCredential(ctx, userID)
	if err != nil {
		return nil, err
	}
	raw := make([]byte, 32)
	if _, err := cryptoRandomRead(raw); err != nil {
		return nil, fmt.Errorf("generate workbench grant: %w", err)
	}
	code := base64.RawURLEncoding.EncodeToString(raw)
	hash := sha256.Sum256([]byte(code))
	now := time.Now().UTC()
	grant := &WorkbenchLaunchGrant{
		CodeHash:    fmt.Sprintf("%x", hash[:]),
		UserID:      userID,
		APIKeyID:    credential.APIKeyID,
		ClientID:    s.cfg.HeliosWorkbench.ClientID,
		RedirectURI: s.cfg.HeliosWorkbench.RedirectURI,
		ExpiresAt:   now.Add(time.Duration(s.cfg.HeliosWorkbench.GrantTTLSeconds) * time.Second),
	}
	if err := s.repo.CreateGrant(ctx, grant); err != nil {
		return nil, err
	}
	redirect, err := url.Parse(grant.RedirectURI)
	if err != nil {
		return nil, fmt.Errorf("build workbench launch URL: %w", err)
	}
	redirect.Fragment = "code=" + code
	return &WorkbenchLaunch{URL: redirect.String(), ExpiresIn: s.cfg.HeliosWorkbench.GrantTTLSeconds}, nil
}

func (s *WorkbenchIntegrationService) ConsumeGrant(ctx context.Context, code string) (*WorkbenchGrantResolution, error) {
	if !s.enabled() || !validWorkbenchCode(code) {
		return nil, ErrWorkbenchInvalidGrant
	}
	hash := sha256.Sum256([]byte(code))
	resolution, err := s.repo.ConsumeGrant(ctx, fmt.Sprintf("%x", hash[:]), time.Now().UTC())
	if err != nil {
		return nil, err
	}
	if resolution == nil || resolution.Grant == nil || resolution.Credential == nil ||
		resolution.User == nil || resolution.APIKey == nil {
		return nil, ErrWorkbenchInvalidGrant
	}
	grant := resolution.Grant
	if grant.ClientID != s.cfg.HeliosWorkbench.ClientID || grant.RedirectURI != s.cfg.HeliosWorkbench.RedirectURI ||
		grant.UserID != resolution.User.ID || grant.APIKeyID != resolution.APIKey.ID ||
		resolution.Credential.UserID != grant.UserID || resolution.Credential.Workbench != WorkbenchHeliosGen ||
		resolution.Credential.APIKeyID != grant.APIKeyID || resolution.APIKey.UserID != grant.UserID ||
		!resolution.User.IsActive() || resolution.User.DeletedAt != nil || !resolution.APIKey.IsActive() || resolution.APIKey.IsExpired() {
		return nil, ErrWorkbenchInvalidGrant
	}
	return resolution, nil
}

func validWorkbenchCode(code string) bool {
	if len(code) == 0 || len(code) > 256 {
		return false
	}
	decoded, err := base64.RawURLEncoding.DecodeString(code)
	if err != nil || len(decoded) != 32 {
		return false
	}
	canonical := base64.RawURLEncoding.EncodeToString(decoded)
	return subtle.ConstantTimeCompare([]byte(canonical), []byte(code)) == 1
}

// cryptoRandomRead is a variable for focused tests without changing production
// randomness semantics.
var cryptoRandomRead = func(dst []byte) (int, error) {
	return rand.Read(dst)
}
// ConstantTimeEqual compares credentials without making a length-dependent
// secret comparison.
func ConstantTimeEqual(a, b string) bool {
	ha := sha256.Sum256([]byte(a))
	hb := sha256.Sum256([]byte(b))
	return subtle.ConstantTimeCompare(ha[:], hb[:]) == 1
}
