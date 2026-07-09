package service

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/xai"
)

const (
	grokQuotaUpstreamTimeout = 20 * time.Second
	grokQuotaProbeInput      = "."
	// Known-working upstream model for rate-limit header probes.
	// Do not use GetMappedModel("grok"): unmapped aliases return "grok" and xAI 400s.
	grokQuotaDefaultModel  = "grok-4.5"
	grokBillingSnapshotKey = "grok_billing_snapshot"
)

type GrokQuotaProbeResult struct {
	Source          string               `json:"source"`
	Snapshot        *xai.QuotaSnapshot   `json:"snapshot,omitempty"`
	Billing         *xai.BillingSnapshot `json:"billing,omitempty"`
	StatusCode      int                  `json:"status_code,omitempty"`
	HeadersObserved bool                 `json:"headers_observed"`
	ResetSupported  bool                 `json:"reset_supported"`
	FetchedAt       int64                `json:"fetched_at"`
}

type GrokQuotaResetResult struct {
	Supported bool   `json:"supported"`
	Code      string `json:"code"`
	Message   string `json:"message"`
}

type GrokQuotaService struct {
	accountRepo   AccountRepository
	proxyRepo     ProxyRepository
	tokenProvider *GrokTokenProvider
	httpUpstream  HTTPUpstream
}

func NewGrokQuotaService(
	accountRepo AccountRepository,
	proxyRepo ProxyRepository,
	tokenProvider *GrokTokenProvider,
	httpUpstream HTTPUpstream,
) *GrokQuotaService {
	return &GrokQuotaService{
		accountRepo:   accountRepo,
		proxyRepo:     proxyRepo,
		tokenProvider: tokenProvider,
		httpUpstream:  httpUpstream,
	}
}

func (s *GrokQuotaService) ProbeUsage(ctx context.Context, accountID int64) (*GrokQuotaProbeResult, error) {
	account, token, proxyURL, err := s.prepareProbe(ctx, accountID)
	if err != nil {
		return nil, err
	}

	headerResult, err := s.probeRateLimitHeaders(ctx, account, token, proxyURL)
	if err != nil {
		return nil, err
	}

	billing := s.probeBilling(ctx, account, token, proxyURL)

	// Persist billing always. Only overwrite the short-window header snapshot when
	// the probe actually observed headers (or 429). A 400 model/body failure must
	// not clobber a good passive snapshot from live traffic.
	extraUpdates := map[string]any{
		grokBillingSnapshotKey: billing,
	}
	keepHeaderSnapshot := headerResult != nil && headerResult.Snapshot != nil &&
		(headerResult.HeadersObserved || headerResult.StatusCode == http.StatusTooManyRequests || headerResult.StatusCode < 400)
	if keepHeaderSnapshot {
		extraUpdates[grokQuotaSnapshotExtraKey] = headerResult.Snapshot
	}
	_ = s.accountRepo.UpdateExtra(ctx, account.ID, extraUpdates)

	result := &GrokQuotaProbeResult{
		Source:          "active_probe",
		Snapshot:        headerResult.Snapshot,
		Billing:         billing,
		StatusCode:      headerResult.StatusCode,
		HeadersObserved: headerResult.HeadersObserved,
		ResetSupported:  false,
		FetchedAt:       time.Now().Unix(),
	}
	if !keepHeaderSnapshot {
		// Prefer previously stored headers for the API response when probe body failed.
		if existing, _ := grokQuotaSnapshotFromExtra(account.Extra); existing != nil {
			result.Snapshot = existing
			result.HeadersObserved = existing.HasObservedHeaders()
		}
	}
	if headerResult.StatusCode == http.StatusTooManyRequests {
		return result, nil
	}
	// Soft-fail short-window header probe: still return/persist billing so weekly
	// quota is visible even when /responses rejects the probe model/body.
	if headerResult.StatusCode >= 400 {
		if billing != nil {
			return result, nil
		}
		return nil, headerResult.err
	}
	return result, nil
}

type grokHeaderProbeResult struct {
	Snapshot        *xai.QuotaSnapshot
	StatusCode      int
	HeadersObserved bool
	err             error
}

func (s *GrokQuotaService) probeRateLimitHeaders(ctx context.Context, account *Account, token, proxyURL string) (*grokHeaderProbeResult, error) {
	body, err := buildGrokQuotaProbeBody(account)
	if err != nil {
		return nil, infraerrors.Newf(http.StatusBadRequest, "GROK_QUOTA_PROBE_BODY_ERROR", "failed to build probe body: %v", err)
	}
	targetURL, err := xai.BuildResponsesURL(account.GetGrokBaseURL())
	if err != nil {
		return nil, infraerrors.Newf(http.StatusBadRequest, "GROK_QUOTA_BASE_URL_INVALID", "invalid Grok base_url: %v", err)
	}

	callCtx, cancel := context.WithTimeout(ctx, grokQuotaUpstreamTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(callCtx, http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		return nil, infraerrors.Newf(http.StatusInternalServerError, "GROK_QUOTA_PROBE_REQUEST_BUILD_FAILED", "failed to build upstream request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "sub2api-grok-quota-probe/1.0")

	resp, err := s.httpUpstream.Do(req, proxyURL, account.ID, maxInt(account.Concurrency, 1))
	if err != nil {
		return nil, infraerrors.Newf(http.StatusBadGateway, "GROK_QUOTA_PROBE_REQUEST_FAILED", "upstream probe failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	snapshot := xai.ObserveQuotaHeaders(resp.Header, resp.StatusCode, "active_probe")
	out := &grokHeaderProbeResult{
		Snapshot:        snapshot,
		StatusCode:      resp.StatusCode,
		HeadersObserved: snapshot != nil && snapshot.HeadersObserved,
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return out, nil
	}
	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 240))
		bodyText := truncate(strings.TrimSpace(string(bodyBytes)), 240)
		slog.Warn("grok_quota_probe_failed", "account_id", account.ID, "status", resp.StatusCode, "body", bodyText)
		out.err = infraerrors.Newf(mapUpstreamStatus(resp.StatusCode), "GROK_QUOTA_PROBE_UPSTREAM_ERROR", "upstream returned %d: %s", resp.StatusCode, bodyText)
		return out, nil
	}
	return out, nil
}

func (s *GrokQuotaService) probeBilling(ctx context.Context, account *Account, token, proxyURL string) *xai.BillingSnapshot {
	if s == nil || s.httpUpstream == nil || account == nil {
		return xai.NewBillingErrorSnapshot(0, "billing_probe", "billing probe not configured")
	}

	callCtx, cancel := context.WithTimeout(ctx, grokQuotaUpstreamTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(callCtx, http.MethodGet, xai.BuildBillingURL(), nil)
	if err != nil {
		return xai.NewBillingErrorSnapshot(0, "billing_probe", err.Error())
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-XAI-Token-Auth", xai.TokenAuthCLI)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", xai.BillingUserAgent)

	resp, err := s.httpUpstream.Do(req, proxyURL, account.ID, maxInt(account.Concurrency, 1))
	if err != nil {
		return xai.NewBillingErrorSnapshot(0, "billing_probe", err.Error())
	}
	defer func() { _ = resp.Body.Close() }()
	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		// One refresh+retry aligned with openusage; if still failing, persist error state.
		if refreshed, refreshErr := s.refreshTokenForBilling(ctx, account); refreshErr == nil && refreshed != "" {
			retryCtx, retryCancel := context.WithTimeout(ctx, grokQuotaUpstreamTimeout)
			defer retryCancel()
			retryReq, reqErr := http.NewRequestWithContext(retryCtx, http.MethodGet, xai.BuildBillingURL(), nil)
			if reqErr == nil {
				retryReq.Header.Set("Authorization", "Bearer "+refreshed)
				retryReq.Header.Set("X-XAI-Token-Auth", xai.TokenAuthCLI)
				retryReq.Header.Set("Accept", "application/json")
				retryReq.Header.Set("User-Agent", xai.BillingUserAgent)
				if retryResp, retryDoErr := s.httpUpstream.Do(retryReq, proxyURL, account.ID, maxInt(account.Concurrency, 1)); retryDoErr == nil {
					defer func() { _ = retryResp.Body.Close() }()
					retryBody, _ := io.ReadAll(io.LimitReader(retryResp.Body, 1<<20))
					if retryResp.StatusCode < 400 {
						return xai.ParseBillingJSON(retryBody, retryResp.StatusCode, "billing_probe")
					}
					return xai.NewBillingErrorSnapshot(retryResp.StatusCode, "billing_probe", xai.FormatBillingProbeError(retryResp.StatusCode, string(retryBody)))
				}
			}
		}
		return xai.NewBillingErrorSnapshot(resp.StatusCode, "billing_probe", xai.FormatBillingProbeError(resp.StatusCode, string(bodyBytes)))
	}
	if resp.StatusCode >= 400 {
		return xai.NewBillingErrorSnapshot(resp.StatusCode, "billing_probe", xai.FormatBillingProbeError(resp.StatusCode, string(bodyBytes)))
	}
	return xai.ParseBillingJSON(bodyBytes, resp.StatusCode, "billing_probe")
}

func (s *GrokQuotaService) refreshTokenForBilling(ctx context.Context, account *Account) (string, error) {
	if s == nil || s.tokenProvider == nil || account == nil {
		return "", infraerrors.New(http.StatusInternalServerError, "GROK_QUOTA_NOT_CONFIGURED", "token provider missing")
	}
	// Force cache miss by clearing expires_at skew: GetAccessToken refreshes when near expiry.
	// Best-effort: call GetAccessToken again; provider refreshes if needed.
	return s.tokenProvider.GetAccessToken(ctx, account)
}

func (s *GrokQuotaService) ResetQuota(ctx context.Context, accountID int64) (*GrokQuotaResetResult, error) {
	if _, err := s.loadGrokOAuthAccount(ctx, accountID); err != nil {
		return nil, err
	}
	return nil, infraerrors.New(http.StatusNotImplemented, "GROK_QUOTA_RESET_UNSUPPORTED", "xAI does not expose a Grok subscription quota reset endpoint for OAuth accounts")
}

func (s *GrokQuotaService) prepareProbe(ctx context.Context, accountID int64) (*Account, string, string, error) {
	if s == nil || s.tokenProvider == nil || s.httpUpstream == nil {
		return nil, "", "", infraerrors.New(http.StatusInternalServerError, "GROK_QUOTA_NOT_CONFIGURED", "grok quota service is not configured")
	}
	account, err := s.loadGrokOAuthAccount(ctx, accountID)
	if err != nil {
		return nil, "", "", err
	}

	token, err := s.tokenProvider.GetAccessToken(ctx, account)
	if err != nil {
		return nil, "", "", infraerrors.Newf(http.StatusBadGateway, "GROK_QUOTA_TOKEN_UNAVAILABLE", "failed to acquire access token: %v", err)
	}
	if strings.TrimSpace(token) == "" {
		return nil, "", "", infraerrors.New(http.StatusBadGateway, "GROK_QUOTA_TOKEN_UNAVAILABLE", "access token is empty")
	}

	return account, token, s.resolveProxyURL(ctx, account), nil
}

func (s *GrokQuotaService) resolveProxyURL(ctx context.Context, account *Account) string {
	if account == nil || account.ProxyID == nil {
		return ""
	}
	switch {
	case account.Proxy != nil:
		return account.Proxy.URL()
	case s != nil && s.proxyRepo != nil:
		if proxy, err := s.proxyRepo.GetByID(ctx, *account.ProxyID); err == nil && proxy != nil {
			return proxy.URL()
		}
	}
	return ""
}

func (s *GrokQuotaService) loadGrokOAuthAccount(ctx context.Context, accountID int64) (*Account, error) {
	if s == nil || s.accountRepo == nil {
		return nil, infraerrors.New(http.StatusInternalServerError, "GROK_QUOTA_NOT_CONFIGURED", "grok quota service is not configured")
	}
	account, err := s.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return nil, infraerrors.Newf(http.StatusNotFound, "GROK_QUOTA_ACCOUNT_NOT_FOUND", "account not found: %v", err)
	}
	if account == nil {
		return nil, infraerrors.New(http.StatusNotFound, "GROK_QUOTA_ACCOUNT_NOT_FOUND", "account not found")
	}
	if account.Platform != PlatformGrok {
		return nil, infraerrors.New(http.StatusBadRequest, "GROK_QUOTA_INVALID_PLATFORM", "account is not a Grok account")
	}
	if account.Type != AccountTypeOAuth {
		return nil, infraerrors.New(http.StatusBadRequest, "GROK_QUOTA_INVALID_TYPE", "account is not an OAuth account")
	}
	return account, nil
}

func buildGrokQuotaProbeBody(account *Account) ([]byte, error) {
	_ = account // probe model is fixed; account only used for auth/proxy upstream.
	return json.Marshal(map[string]any{
		"model":             grokQuotaDefaultModel,
		"input":             grokQuotaProbeInput,
		"max_output_tokens": 1,
		"store":             false,
	})
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
