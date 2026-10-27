# Build Note — DeepSeek native tools

| Rung | Feature | Acceptance check | Tier | Status |
|------|---------|------------------|------|--------|
| F0 | Local gateway → synthetic upstream → usage spine | go test -tags=unit ./internal/handler -run DeepSeekToolsSpine -count=1 | MUST | passed |
| F1 | Scoped native transport + scheduler isolation | go test -tags=unit ./internal/service -run DeepSeekTool -count=1 | MUST | passed |
| F2 | Exact per-call billing/reservation/idempotency + route guards | go test -tags=unit ./internal/service ./internal/handler ./internal/server/routes -run 'DeepSeekTool|ForcedUsageBillingRequestID' -count=1 | MUST | passed |
| F3 | Affected-package regressions, backend build, CI and assumption sweep | go test -tags=unit ./internal/service ./internal/handler ./internal/server/routes; make build; SILENT_ASSUMPTION_SWEEP.json | MUST | local passed; CI pending |

## Run record

- Base captured before implementation. Reuse existing OpenAI HTTP transport, scheduler, API key chain and web-search cost calculation. No new dependencies.
- F0: `go test -tags=unit ./internal/handler -run DeepSeekToolsSpine -count=1` exit 0, handler 1.736s; log `tmp/deepseek-tools-f0.log`. Entire call uses a local synthetic upstream.
- Existing route tests had two stale RegisterPaymentRoutes signatures; added the missing nil dependency only. Route compile check exit 0; log `tmp/deepseek-tools-routes-compile.log`.
- Transport check found non-canonical Accept override duplication; fix remains confined to native tools, with a wire/header regression assertion.
- Integration review found scheduler Redis metadata filters out unlisted Extra fields. Preserve the scoped tool key in that projection; a JSON round-trip admission test covers allowed/shared groups and malformed permissions.
- The same-family assumption sweep closed A-005/A-006/A-009/A-013 and clarified A-001 as production configuration; result is provisional rather than cross-family independent correctness proof.
- Native usage accounting skips chat 403 recovery and pins AccountStatsCost to the explicit per-call total, including zero for successful empty results.
- CI exposed stale migration fixtures after two upstream additions. The candidate count is 332; named historical baselines retain 286/320/322/327/328 by excluding the later scoped-admin-key and image-count-billing migrations. Migration SQL and checksum/convergence assertions remain unchanged. Earlier local integration commands exited 0; the final full CI run supplies actual database migration evidence.

## Deferred

- Production authorization received 2026-10-27; local validation is complete pending final build/CI evidence before fixed-SHA server build/switch and real endpoint acceptance.

## Blockers

- None.
