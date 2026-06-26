# Upstream v0.1.139 Merge Notes

## Goal

Merge upstream `Wei-Shaw/sub2api` tag `v0.1.139` into the JP/HK workspace without regressing local production behavior.

Current local branch: `jp/daily-balance`.
Current upstream base: local tree already has `v0.1.138`; target tag is `v0.1.139`.

## Current Local State

- Local channel-monitor header fix is committed separately as `f9400869 fix(monitor): send default client fingerprint headers`.
- Branch is ahead of `origin/jp/daily-balance`.
- No `v0.1.139` merge has been started yet.

## Upstream v0.1.139 Themes To Import

Import these unless they directly conflict with a JP production invariant:

- Codex/OpenAI hardening:
  - `codex_cli_only` detection hardening.
  - unified engine fingerprint signals.
  - account-level app-server behavior.
  - GPT-5.5 Codex instructions and latest fallback.
  - Codex personal access token auth.
- OpenAI gateway fixes:
  - chat transport error failover.
  - 404 `model_not_found` when no account supports a model instead of generic 503.
  - passthrough function-call argument dedupe.
  - OpenAI response failed-event sanitization.
  - OpenAI image refusal passthrough as 400.
  - overloaded error-code detection.
- API compatibility fixes:
  - custom tool schema normalization.
  - Responses-to-Anthropic tool schema coverage.
  - Responses cache-input fix.
- Admin and account features:
  - `sub2api-admin` `SUB2API_JWT` fallback auth.
  - OpenAI weekly limit reset confirmation.
  - admin usage cache token breakdown.
- Grok/xAI support:
  - Grok subscription support.
  - Grok OAuth, quota fetch/probe, token refresh, UI quota cell.
- Billing/payment fixes:
  - prevent balance overdraft.
  - subscription order recharge multiplier.
  - plural subscription validity units.
  - payment provider cards still visible when `supported_types` is empty.
  - currency symbol display from order `currency`.
- UI/admin fixes:
  - Ops dashboard chart height fix.
  - Settings screen Codex policy controls.
  - new i18n strings for upstream features.
- Docs/assets/sponsors:
  - README updates and partner-logo churn are low risk; keep unless they conflict with local JP/HK runbook text.

## Local Changes That Must Survive

These are JP/HK production behavior and should win over upstream when there is a conflict:

- Live model routing:
  - OpenAI account testing uses `gpt-5.5`.
  - Do not reintroduce `gpt-5.4` or `gpt-5.4-mini` routing assumptions.
  - Keep compatibility redirects from stale `gpt-5.4` requests to `gpt-5.5` where present.
- Daily balance and billing:
  - 24h daily balance grant flow.
  - 1.5x fallback behavior.
  - user/account billing rate multipliers.
  - priority overflow and account multipliers.
  - prevent implicit fast-tier charging.
  - group spending-limit circuit breaker.
- Scheduling and failover:
  - sticky-session escape when a higher-priority or much-better candidate exists.
  - `UpstreamFailoverError` for temporary OpenAI passthrough blocking.
  - retry stream-capacity failover.
  - ignore terminal SSE events for first-token timing.
  - group scheduling priority.
- Gateway custom behavior:
  - local Kiro gateway account detection and concurrency cap.
  - local OpenAI/Gateway cache and recharge updates.
  - route attribution and SSE timing behavior used for live diagnostics.
- Channel monitor:
  - default Claude CLI headers for `x-api-key` checks.
  - default Codex CLI fingerprint headers for bearer OpenAI checks.
  - user extra headers still override defaults unless forbidden.
- Frontend/local product surface:
  - image2 workbench embed and navigation.
  - local cyber theme/login redesign.
  - high-density account/key/admin UI improvements.
  - model square and dashboard local changes.
- Deployment boundary:
  - HK live server is `154.88.65.45`.
  - app binds through Caddy; preserve Caddy streaming `flush_interval -1` assumptions in deployment docs/scripts.
  - do not reset DB/Redis volumes.
  - do not copy secret-bearing database rows or dumps into the workspace.

## Expected Conflict Files

`git merge-tree HEAD v0.1.139` reports conflicts in these files:

- `backend/cmd/server/wire_gen.go`
- `backend/internal/domain/constants.go`
- `backend/internal/handler/openai_gateway_handler.go`
- `backend/internal/repository/usage_billing_repo.go`
- `backend/internal/server/routes/gateway.go`
- `backend/internal/service/account.go`
- `backend/internal/service/admin_service.go`
- `backend/internal/service/billing_cache_service.go`
- `backend/internal/service/domain_constants.go`
- `backend/internal/service/openai_account_scheduler.go`
- `backend/internal/service/openai_gateway_service.go`
- `backend/internal/service/openai_gateway_service_test.go`
- `backend/internal/service/openai_images_responses.go`
- `frontend/src/i18n/locales/en.ts`
- `frontend/src/i18n/locales/zh.ts`

## Conflict Resolution Rules

### Wire and routes

- Prefer preserving both dependency injections.
- After resolving `wire.go` / `wire_gen.go`, run the smallest compile check first.
- If generated code is inconsistent, run the repo's existing generation command rather than hand-editing generated output repeatedly.

### Domain constants

- Keep local JP/Gateway constants and add upstream Grok/Codex constants.
- Do not remove Kiro gateway constants or local group/platform names.

### OpenAI gateway and scheduler

- Preserve local account-selection semantics first:
  - priority-aware scheduling,
  - sticky escape,
  - runtime block fast path,
  - stream-capacity failover,
  - first-token timing.
- Add upstream behavior around:
  - codex fingerprint detection,
  - PAT auth,
  - model availability / 404 handling,
  - function-argument dedupe,
  - transport failover.
- Avoid duplicating equivalent retry/failover code. Prefer one shared path if both sides solve the same failure.

### Billing/cache/repository

- Keep local daily-balance grant and rate-multiplier math.
- Add upstream cache token-breakdown fields and payment fixes.
- Carefully check SQL column order, `sqlmock` expectations, and stats structs.
- Any balance-affecting path must have a targeted test before handoff.

### Account/admin services

- Keep local account group, Kiro, proxy, quota, and multiplier fields.
- Add upstream Grok OAuth/quota and Codex app-server behavior.
- Watch for interface additions; update all stubs/mocks in `backend/internal`.

### OpenAI images

- Preserve local incomplete/image response handling.
- Add upstream content-refusal passthrough and failed-event sanitize behavior.

### Frontend i18n

- Merge keys, not layout opinions.
- Preserve local Chinese/English strings for existing JP UI where the same key was customized.
- Add upstream keys for Grok, Codex policy, payment, usage, and quota screens.

## Verification Plan

Run progressively; do not jump directly to full `make test` while conflicts are still fresh.

1. Conflict-resolution sanity:
   - `git diff --check`
   - `rg "<<<<<<<|=======|>>>>>>>" backend frontend`
2. Backend compile hotspots:
   - `go test ./internal/domain ./internal/config ./internal/pkg/openai ./internal/pkg/apicompat -count=1`
   - `go test ./internal/repository -run 'Usage|Billing|Stats|Token|Grok' -count=1`
   - `go test ./internal/service -run 'OpenAI|Gateway|Scheduler|Billing|Daily|Grok|Codex|Image|Token|Quota' -count=1`
   - `go test ./internal/handler -run 'OpenAI|Gateway|NoAccount|Grok|Payment|Codex' -count=1`
3. Frontend targeted checks:
   - `pnpm --dir frontend run typecheck`
   - `pnpm --dir frontend exec vitest run frontend/src/views/admin/__tests__/SettingsView.spec.ts`
   - `pnpm --dir frontend exec vitest run frontend/src/components/account/__tests__/EditAccountModal.spec.ts`
   - `pnpm --dir frontend exec vitest run frontend/src/components/payment/__tests__/currency.spec.ts frontend/src/components/payment/__tests__/PaymentQRDialog.spec.ts`
4. Wider checks before deploy:
   - `go test -tags=unit ./...`
   - `pnpm --dir frontend run lint:check`
   - `pnpm --dir frontend run test:run` if targeted checks reveal no broad breakage.
5. Deploy gate:
   - Build image locally.
   - Deploy only after local tests pass or failures are explicitly accepted.
   - Verify HK direct `/health`, Caddy `/health`, and one OpenAI/GPT-5.5 route probe.

Known current blocker before merge: `go test ./internal/service` fails at compile time with `undefined: rateLimitAccountRepoStub` in `openai_gateway_service_test.go`. Fix this separately or account for it during the merge, otherwise it will obscure new failures.

## Suggested Execution Order

1. Create merge branch:
   - `git switch -c merge/upstream-v0.1.139`
2. Start merge:
   - `git merge --no-ff v0.1.139`
3. Resolve backend infrastructure conflicts:
   - constants,
   - wire,
   - routes.
4. Resolve OpenAI/Gateway conflicts:
   - scheduler,
   - gateway service,
   - handler,
   - images.
5. Resolve billing/account/admin conflicts.
6. Resolve frontend i18n conflicts.
7. Run targeted verification.
8. Commit merge.
9. Only then consider push/deploy.

## Do Not Do In This Merge

- Do not redesign the UI.
- Do not move ops files into `sub2api-jp-src`.
- Do not remove local migrations or ent-generated daily balance files.
- Do not normalize away JP/HK deployment docs.
- Do not make live-server changes during merge resolution.
- Do not introduce new dependencies unless upstream already added them and the lockfiles require it.

