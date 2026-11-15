package service

import (
	"net/url"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/urljoin"
)

// buildAnthropicMessagesURL appends the Messages endpoint to an already validated
// base URL. Cloudflare's account REST endpoint rejects the beta query parameter;
// all other upstreams, including Cloudflare's native gateway, keep it.
func buildAnthropicMessagesURL(baseURL string) string {
	targetURL := urljoin.APIPath(baseURL, "/v1/messages")
	parsed, err := url.Parse(targetURL)
	if err == nil && strings.EqualFold(parsed.Scheme, "https") && parsed.User == nil &&
		parsed.RawQuery == "" && !parsed.ForceQuery && parsed.Fragment == "" && parsed.RawPath == "" {
		host := strings.ToLower(parsed.Host)
		parts := strings.Split(parsed.Path, "/")
		if (host == "api.cloudflare.com" || host == "api.cloudflare.com:443") &&
			len(parts) == 8 && parts[0] == "" && parts[1] == "client" && parts[2] == "v4" &&
			parts[3] == "accounts" && parts[4] != "" && parts[4] != "." && parts[4] != ".." &&
			parts[5] == "ai" && parts[6] == "v1" && parts[7] == "messages" {
			return targetURL
		}
	}
	return urljoin.WithQueryParam(targetURL, "beta", "true")
}

func buildAnthropicCountTokensURL(base string) string {
	return urljoin.WithQueryParam(urljoin.APIPath(base, "/v1/messages/count_tokens"), "beta", "true")
}
