# Mirasim OAuth accounts

Mirasim is a native Sub2API platform. Accounts use the Sub2API scheduler, API keys, usage logs, and group pricing. The internal account type is `apikey` for gateway compatibility, but authentication uses OAuth refresh tokens and device signatures; there is no static upstream API key to enter.

## Import and edit

1. Open **Admin → Accounts → Add account → Mirasim**.
2. Download the current **local callback helper** and run it on the browser's computer:
   `powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\mirasim-oauth-helper.ps1`
3. Click GitHub OAuth or Google OAuth. Keep the helper running.
4. After authorization returns, choose the account name, proxy, active Mirasim groups and concurrency, then click **Import account**. The selected proxy is used for token validation and subsequent requests. Leaving it empty means direct access.
5. Edit the account normally to change its name, proxy, concurrency, groups or model selection. Credentials remain on the server and are not required again.

The helper uses 127.0.0.1:8788 only to start authorization. Each flow has a separate loopback callback port and a ten-minute deadline because Mirasim replaces the client's OAuth state. Callbacks on shared port 8788 are rejected. Close an older helper before starting the new version.

The callback token is removed from the URL immediately and retained only in page memory until import or cancellation. Refreshing the page requires a new authorization. A failed import after upstream token rotation may also require reauthorization.

When no group is selected, import explicitly binds the account to an active `mirasim-default`. If it does not exist, an exclusive group with the verified-model allowlist is created. Review its prices and access before distributing API keys; an inactive default group must be re-enabled or a different group selected.

## Models and protocols

The conservative default catalog, verified through the deployed gateway on 2026-09-24, is:

- `glm-5.3-flash`
- `kimi-k3`

Account-specific availability may change. These model IDs supersede the historical September 23 catalog for this deployment; historical catalog results are not a guarantee of current access.

Sub2API selects the native upstream protocol by the mapped model: the two default models use Chat Completions; `claude-*` uses Messages; `gpt-*` uses Responses. Chat, Messages and Responses clients use the existing protocol converters where needed. Adding a model to the account does not prove upstream availability.

GPT's native upstream requires streaming. Direct non-streaming GPT Responses requests remain unsupported. Claude's relay-specific fingerprint and 200-byte system-prompt limit remain enforced, but oversized instructions now fail explicitly instead of being silently truncated. The two default models do not have this Claude-specific restriction.

## Credentials and pricing

Only official relay endpoints are accepted. Account clients serialize initialization and token refresh, rebuild when the proxy or device key changes, and persist rotated tokens before sending inference requests. Failed persistence is reported and retried while the application remains running. A restart after an unsaved rotation can still require reauthorization; multiple application replicas require a shared rotation lock before horizontal scaling.

Per-model group pricing already exists and should be used for Mirasim. Group倍率 multiplies the configured model unit prices; it is not a replacement for them. Keep the account and group model lists aligned. Do not infer upstream cost or margin from the account's default倍率.

Account connection tests require actual text plus a terminal response. Deployment acceptance should additionally exercise the intended API-key group and check usage/pricing records. Browser OAuth requires an interactive upstream sign-in and is separate from local helper tests.

On 2026-09-24 the relay returned 422 for `deepseek-v4.1-flash`. At the administrator's request it is disabled, with its group price retained. The account editor now supports **Sync upstream models** using the account's OAuth signer and proxy; no static API key is needed. Catalog sync lists availability and does not prove inference capacity or automatically change group pricing.
