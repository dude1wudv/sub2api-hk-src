package service

import (
	"context"
	"encoding/base64"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

type workbenchRepositoryStub struct {
	mu          sync.Mutex
	credential  *WorkbenchCredential
	grant       *WorkbenchLaunchGrant
	resolution  *WorkbenchGrantResolution
	consumed    bool
	consumeHits int
}

func (r *workbenchRepositoryStub) EnsureCredential(context.Context, int64, string, string) (*WorkbenchCredential, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.credential == nil {
		r.credential = &WorkbenchCredential{UserID: 9, Workbench: WorkbenchHeliosGen, APIKeyID: 41}
	}
	return r.credential, nil
}

func (r *workbenchRepositoryStub) CreateGrant(_ context.Context, grant *WorkbenchLaunchGrant) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.grant = grant
	return nil
}

func (r *workbenchRepositoryStub) ConsumeGrant(context.Context, string, time.Time) (*WorkbenchGrantResolution, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.consumeHits++
	if r.consumed {
		return nil, nil
	}
	r.consumed = true
	return r.resolution, nil
}

func workbenchTestConfig() *config.Config {
	return &config.Config{HeliosWorkbench: config.HeliosWorkbenchConfig{
		Enabled: true, PublicURL: "https://canvas.sub.sunmmyapi.xyz",
		RedirectURI: "https://canvas.sub.sunmmyapi.xyz/bootstrap",
		APIBaseURL: "https://sub.sunmmyapi.xyz/v1", ClientID: "heliosgen-web",
		ClientSecret: strings.Repeat("s", 32), GrantTTLSeconds: 90,
	}, Default: config.DefaultConfig{APIKeyPrefix: "sk-"}}
}

func TestWorkbenchCreateLaunchUsesFragmentCodeAndStoresOnlyDigest(t *testing.T) {
	repo := &workbenchRepositoryStub{}
	svc := NewWorkbenchIntegrationService(repo, NewAPIKeyService(nil, nil, nil, nil, nil, nil, workbenchTestConfig()), workbenchTestConfig())
	launch, err := svc.CreateLaunch(context.Background(), 9)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(launch.URL)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Path != "/bootstrap" || !strings.HasPrefix(parsed.Fragment, "code=") {
		t.Fatalf("unexpected launch URL: %s", launch.URL)
	}
	code := strings.TrimPrefix(parsed.Fragment, "code=")
	decoded, err := base64.RawURLEncoding.DecodeString(code)
	if err != nil || len(decoded) != 32 {
		t.Fatalf("invalid one-time code")
	}
	if repo.grant == nil || repo.grant.CodeHash == code || len(repo.grant.CodeHash) != 64 {
		t.Fatalf("raw code was persisted: grant=%+v", repo.grant)
	}
	if launch.ExpiresIn != 90 {
		t.Fatalf("expires_in=%d, want 90", launch.ExpiresIn)
	}
}

func TestWorkbenchConsumeGrantConcurrentReplayOnlySucceedsOnce(t *testing.T) {
	cfg := workbenchTestConfig()
	repo := &workbenchRepositoryStub{}
	codeBytes := make([]byte, 32)
	for i := range codeBytes {
		codeBytes[i] = byte(i)
	}
	code := base64.RawURLEncoding.EncodeToString(codeBytes)
	repo.resolution = &WorkbenchGrantResolution{
		Grant: &WorkbenchLaunchGrant{UserID: 9, APIKeyID: 41, ClientID: cfg.HeliosWorkbench.ClientID, RedirectURI: cfg.HeliosWorkbench.RedirectURI},
		Credential: &WorkbenchCredential{UserID: 9, APIKeyID: 41, Workbench: WorkbenchHeliosGen},
		User:       &User{ID: 9, Status: StatusActive},
		APIKey:     &APIKey{ID: 41, UserID: 9, Status: StatusActive},
	}
	svc := NewWorkbenchIntegrationService(repo, nil, cfg)
	var wg sync.WaitGroup
	var mu sync.Mutex
	successes := 0
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := svc.ConsumeGrant(context.Background(), code); err == nil {
				mu.Lock()
				successes++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if successes != 1 {
		t.Fatalf("concurrent exchange successes=%d, want 1", successes)
	}
	if repo.consumeHits != 16 {
		t.Fatalf("consume hits=%d, want 16", repo.consumeHits)
	}
	if _, err := svc.ConsumeGrant(context.Background(), code); err == nil || !strings.Contains(err.Error(), "invalid_grant") {
		t.Fatalf("replay error=%v", err)
	}
}

func TestWorkbenchConstantTimeEqual(t *testing.T) {
	if !ConstantTimeEqual("client-secret", "different-secret") && ConstantTimeEqual("client-secret", "client-secret") {
		return
	}
	t.Fatal("constant-time equality returned an invalid result")
}
