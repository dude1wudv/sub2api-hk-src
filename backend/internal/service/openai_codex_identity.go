package service

import (
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
)

// codexUpstreamMinVersion is the minimum ChatGPT /backend-api/codex version header.
// Requests that send a lower version are rejected with 404 (issue #3901).
const codexUpstreamMinVersion = "0.144.0"

// enforceCodexIdentityHeaders finalizes OAuth (ChatGPT internal) outbound client
// identity headers. Upstream requires originator to match the User-Agent leading
// client name and be an official Codex identity; if version is present it must
// be >= 0.144.0. Pairing is derived from the final User-Agent; when no official
// identity can be derived, fall back to the default Codex CLI identity.
//
// Only requests that already carry originator are rewritten — the Anthropic
// messages compat bridge intentionally omits originator and must stay untouched.
// Call this after all User-Agent rewrites (custom UA / ForceCodexCLI / browser fallback).
func enforceCodexIdentityHeaders(h http.Header) {
	if h == nil || h.Get("originator") == "" {
		return
	}
	originator, pairedUA, ok := openai.PairCodexClientIdentity(h.Get("user-agent"))
	if !ok {
		originator, pairedUA = "codex_cli_rs", codexCLIUserAgent
	}
	h.Set("user-agent", pairedUA)
	h.Set("originator", originator)
	if v := strings.TrimSpace(h.Get("version")); v != "" && CompareVersions(v, codexUpstreamMinVersion) < 0 {
		h.Set("version", codexCLIVersion)
	}
}
