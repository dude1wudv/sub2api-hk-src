# Mirasim OAuth accounts

Mirasim is a native Sub2API platform. Its accounts appear in the admin account list, bind to `mirasim-default`, and use the usual Sub2API scheduler, API keys, usage logs, and billing. The first successful OAuth import creates the default group. Admins can edit group pricing and access before distributing keys.

## Authorize from Windows

1. Open `https://sub.sunmmyapi.xyz/admin/accounts` and sign in as an administrator.
2. Click **添加账号 → Mirasim** and download **本地回调助手** there. Run the downloaded script in PowerShell on the computer running the browser:

   ```powershell
   powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\mirasim-oauth-helper.ps1
   ```

   Keep that window open. The helper listens only on `127.0.0.1:8788` and does not store credentials.
3. In the same **添加账号 → Mirasim** dialog, click **GitHub OAuth** or **Google OAuth**. After signing in, the callback returns to the Sub2API admin page; the account and default group are created automatically. Refresh the original account tab if it does not update.

Mirasim only allows loopback OAuth callbacks. The helper sends the refresh token to the Sub2API page in the URL fragment, which is cleared before the admin API request. The server validates it with Mirasim, stores the rotated refresh token and device private key as account credentials, and signs requests to the official relay. Do not copy or share the callback URL.

## Routing

- Claude models use `/v1/messages`. GPT models use `/v1/responses` with streaming. Other Mirasim models may use `/v1/chat/completions`.
- On 2026-09-23, the live `/v1/models` catalog for the configured Mirasim account returned `claude-opus-5-5`, `deepseek-flash`, `deepseek-v4-flash`, `deepseek-v4-flash-vision-exp`, `glm-5.3-flash`, `gpt-6-luna`, `gpt-6-sol`, and `kimi-k3`. The built-in list mirrors that response; check the live catalog again if Mirasim changes it. `glm-5.3-flash` and `kimi-k3` both returned HTTP 200 and generated `OK` in live tests through the account's configured proxy. The upstream project leaves prices for GPT and several third-party models unset, so review Sub2API pricing before making the group available to users.
- Custom base URLs are rejected by the Mirasim signer. The OAuth refresh token and device ticket are sent only to official Mirasim endpoints.
- The account editor shows the official relay endpoints and supported models. Mirasim OAuth accounts have no static upstream API key; Sub2API signs requests with the stored OAuth credentials.
- To test routing, create or update a Sub2API API key to use `mirasim-default`, then send a streaming request for a model in the group. The account connection test selects `glm-5.3-flash` by default and only reports success after receiving generated text and a terminal stream event. The end-to-end Sub2API API-key route still needs its own acceptance check with a key for the intended group.
