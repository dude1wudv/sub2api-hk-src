package xai

import "net/http"

const (
	GrokCLIClientName           = "grok-shell"
	GrokCLIClientVersion        = "0.2.93"
	GrokCLIClientIdentifier     = "grok-pager"
	GrokCLIClientSurface        = "tui"
	GrokCLIAuthenticateResponse = "authenticate-response"
	GrokCLIResponsesUserAgent   = "grok-pager/0.2.93 grok-shell/0.2.93 (linux; x86_64)"
)

// SetGrokCLIResponsesHeaders applies the identity contract used by Grok CLI
// subscription requests. Authorization and content negotiation remain caller-owned.
func SetGrokCLIResponsesHeaders(header http.Header) {
	header.Set("X-Grok-Client-Name", GrokCLIClientName)
	header.Set("X-Grok-Client-Version", GrokCLIClientVersion)
	header.Set("X-Grok-Client-Identifier", GrokCLIClientIdentifier)
	header.Set("X-Grok-Client-Surface", GrokCLIClientSurface)
	header.Set("X-XAI-Token-Auth", TokenAuthCLI)
	header.Set("X-AuthenticateResponse", GrokCLIAuthenticateResponse)
	header.Set("User-Agent", GrokCLIResponsesUserAgent)
}
