# Mirasim OAuth accounts

Mirasim is a native Sub2API platform. Its accounts appear in the admin account list, bind to `mirasim-default`, and use the usual Sub2API scheduler, API keys, usage logs, and billing. The first successful OAuth import creates the default group. Admins can edit group pricing and access before distributing keys.

## Authorize from Windows

1. Open `https://sub.sunmmyapi.xyz/admin/accounts` and sign in as an administrator.
2. Download **回调助手** from the account page. Run the downloaded script in PowerShell:

   ```powershell
   powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\mirasim-oauth-helper.ps1
   ```

   Keep that window open. The helper listens only on `127.0.0.1:8788` and does not store credentials.
3. Click **Mirasim GitHub OAuth** or **Mirasim Google OAuth**. After signing in, the callback returns to the Sub2API admin page; the account and default group are created automatically. Refresh the original account tab if it does not update.

Mirasim only allows loopback OAuth callbacks. The helper sends the refresh token to the Sub2API page in the URL fragment, which is cleared before the admin API request. The server validates it with Mirasim, stores the rotated refresh token and device private key as account credentials, and signs requests to the official relay. Do not copy or share the callback URL.

## Routing

- Claude models use `/v1/messages`. GPT models use `/v1/responses` with streaming. Other Mirasim models may use `/v1/chat/completions`.
- The supported model catalog starts with the models from `mirasim2api`; adjust group allowlists and prices as needed. The upstream project leaves prices for GPT and several third-party models unset, so review Sub2API pricing before making the group available to users.
- Custom base URLs are rejected by the Mirasim signer. The OAuth refresh token and device ticket are sent only to official Mirasim endpoints.
- To test routing, create or update a Sub2API API key to use `mirasim-default`, then send a streaming request for a model in the group. OAuth login and live inference require a real Mirasim account and are left for the operator's manual acceptance test.
