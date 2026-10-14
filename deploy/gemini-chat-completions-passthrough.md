# Gemini accounts with a Chat Completions upstream

Some third-party Gemini relays expose `/v1beta/models` but lose function tools
in that protocol while their `/v1/chat/completions` endpoint supports them.
For these accounts, Chat Completions clients can use the existing raw Chat
pipeline without converting requests and responses through Gemini.

Enable the boolean account extra field:

```json
{
  "gemini_chat_completions_passthrough": true
}
```

This option applies only to `platform=gemini`, `type=apikey` accounts with an
explicit nonempty `credentials.base_url`. The existing `api_key` is used as a
Bearer credential. Base URL validation, configured proxy, header overrides,
account scheduling, model mapping, upstream error handling, stream truncation
checks, and billing remain in the existing gateway pipeline. No new secrets or
database migrations are required.

Only incoming Chat Completions requests use this option. Gemini native,
Anthropic Messages, cache APIs, and other account types retain their existing
routes. Accounts without the boolean opt-in also retain their existing routes.
Clients requiring this compatibility path should use an OpenAI-compatible
adapter with the gateway base URL ending in `/v1`.

The upstream endpoint recorded for these requests is `/v1/chat/completions`.
Cached prompt tokens are separated from uncached input before passing usage to
the Gemini billing path, avoiding double charging. Function definitions,
assistant tool calls, tool results, and streamed argument fragments are passed
through by the existing raw Chat implementation.

Set the extra field to `false` or remove it to restore the prior Chat-to-Gemini
conversion. Use the administrative account update mechanism so scheduler
caches are refreshed. Any operational database update must preserve all other
account fields and be accompanied by the normal cache refresh/restart.
