# Assumption Ledger — DeepSeek native tools
<!-- ASK mode: never -->

| ID | Under-determined by the request | Chosen | Class | Source |
|----|--------------------------------|--------|-------|--------|
| A-001 | Upstream | Production deployment config enables native tools only on existing Kedaya account 1112 for group 223; code keeps the capability generic and fail-closed on the explicit per-account group allowlist | interface | user; sweep |
| A-002 | Price | Existing group web_search_price_per_call, currently USD 0.013, times existing user/group base multiplier | semantic | user |
| A-003 | Permission scope | Account extra.deepseek_tools_group_ids=[223]; all scheduler paths and forward recheck scope; shared group 81 denied | interface | user |
| A-004 | Billable success | HTTP 200 plus nonempty Search results or nonblank Fetch content; successful empty response logs per_request zero cost | semantic | user |
| A-005 | Request identity | One deepseek_tools:<uuid> per invocation, forced over client IDs; mandatory helper prevents queue overflow drops and the existing billing path remains idempotent, while DB/process settlement failures retain the existing best-effort/no-persistent-retry boundary | semantic | user; sweep |
| A-006 | Failure policy | No automatic retries/fallback; preserve upstream errors, 504 transport timeout and 502 malformed response; native tool usage does not reset the chat-specific 403 health counter | interface | user; sweep |
| A-007 | Compatibility | Preserve unknown JSON fields; local validation of known official fields; no model injected upstream | interface | user |
| A-008 | Initial support surface | Only specified /v1 routes, fixed-group OpenAI/DeepSeek keys with explicit scoped account permission | interface | default |
| A-009 | Logging and accounting | Existing nonempty model column uses display labels only; explicit tool result branch bypasses all token/model pricing and pins AccountStatsCost to the calculated per-call total (zero for empty results), including empty results | semantic | user; sweep |
| A-010 | Tool pricing defaults | Reuse existing web-search helper and its existing nil/default semantics; group 223 has explicit 0.013; no new global price | semantic | default |
| A-011 | Safety/audit | Query/original question and Fetch URL normalized to existing text audit input; fetch URL forwarded only, never locally fetched | interface | default |
| A-012 | Profit comparison | Like existing native Qwen audio, tool requests suppress model/token profit comparison because they have no model; admission and settlement use the explicit per-call price | semantic | default |
| A-013 | Queued request snapshot | After queue admission, recheck the latest account for scope, group membership, and schedulable state; retain the admitted snapshot for base URL, credentials, proxy, and header overrides so billing and transport use one consistent credential snapshot | interface | sweep |
