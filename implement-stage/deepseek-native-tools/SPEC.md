# DeepSeek native tools

Base commit: `49109a1553417f45e24e63c9232b8ab79472710d`.

Target: expose authenticated `POST /v1/tools/search` and `/v1/tools/fetch` to fixed-group keys for group 223 through existing Kedaya account 1112. Preserve official JSON request/response contracts, existing chat routing and shared group 81 isolation.

Inputs: DeepSeek Search/Fetch JSON, gateway API key. Outputs: original upstream JSON/status, endpoint-attributed usage rows with `deepseek-search` / `deepseek-fetch` labels. Only HTTP 200 with search results/fetch text charges one call using group `web_search_price_per_call` and existing base user/group multiplier.

Success: `go test -tags=unit ./internal/service ./internal/handler ./internal/server/routes` and backend build, then fixed-commit CI. User explicitly authorized production deployment during implementation on 2026-10-27. Deploy only after local/CI checks pass; keep raw production evidence and secrets on HK.

No model protocol conversion, automatic retries, cross-group fallback, frontend settings UI, database migration, or historical billing rewrites.
