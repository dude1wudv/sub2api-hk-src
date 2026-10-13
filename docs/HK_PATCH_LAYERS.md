# HK Long-Term Patch Layers

This document is the ownership manifest for differences between this repository and `Wei-Shaw/sub2api`.
It classifies stable product requirements; it does not rewrite published Git history.

## Layer order

Apply and validate local behavior in this order after every upstream merge:

```text
upstream/main
  -> jp-business
  -> provider-adapters
  -> jp-product
  -> hk-deploy
```

Changes that are marked `upstream-owned` must not become a parallel local implementation.
Generated files are outputs, not standalone patches.

## 1. `jp-business`

Long-term local business rules:

- Manual Alipay static-QR provider, administrator confirmation, step-up authentication, and audit trail. The user-facing gateway checkout page is retired; payment callbacks, existing orders, and the admin order console stay.
- Balance-funded subscription purchases.
- Subscription purchase modes remain `external`, `balance`, and `both` in the backend. The user portal only offers `balance` and `both`.
- Sale windows and fixed subscription expiry snapshots.
- One-purchase-per-user claims.
  - A successful purchase permanently consumes the claim.
  - Provider creation failure, unpaid cancellation, and unpaid expiry release the pending claim so the user may retry.
- Historical payment and subscription migrations are immutable.
- Existing in-process `BackupService` remains supported; it is unrelated to the retired `datamanagementd` agent.

Primary paths:

- `backend/internal/payment/provider/alipay_manual.go`
- `backend/internal/service/payment_*.go`
- `backend/ent/schema/payment_order.go`
- `backend/ent/schema/subscription_plan.go`
- `backend/ent/schema/subscription_purchase_claim.go`
- `frontend/src/views/portal/PurchaseView.vue` (card redemption)
- `frontend/src/views/portal/SubscriptionsView.vue` (balance purchase only)

## 2. `provider-adapters`

Long-term local provider behavior:

- DeepSeek Chat Completions, Responses, and Anthropic protocol selection.
- StepFun (`payg` / `step_plan`) and Mirasim OAuth adapters are registered in the upstream platform catalog and Provider Profile. Keep one protocol resolver; Mirasim retains its signed transport and dedicated import/probe UI.
- Slow first-output protection remains layered around upstream profit admission. WebSocket turns must use the refreshed upstream billing snapshot and the HK post-slot protection check together.
- CN-provider adaptive endpoint selection and strict balance parsing.
- Unknown DeepSeek SKUs fail closed until an explicit reviewed price card is added.
- OpenAI WebSocket turn-level first-response/first-token timing and truncated-stream protection remain local only where upstream has no equivalent behavior.

Grok/xAI is **upstream-owned**. Do not add local Grok scheduling, free-tier gates, protocol forks, pricing rules, or UI semantics. When upstream changes Grok, adopt its implementation and tests.

Primary paths:

- `backend/internal/service/account_test_service.go`
- `backend/internal/service/cn_provider_*.go`
- `backend/internal/service/billing_service.go`
- `backend/internal/service/openai_ws_v2/`

## 3. `jp-product`

Long-term local product surfaces:

- Portal and ink workspace. New UI lives in fork-owned directories (`frontend/src/views/portal/`, `frontend/src/components/portal/`, `frontend/src/components/layout/UserShell.vue`, `frontend/src/components/layout/AdminShell.vue`, `frontend/src/styles/patrick*.css`). `frontend/src/hk/` only mounts shells, styles, and route adjustments.
- A upstream file rewritten by more than half moves to a fork-owned path. The old path is recorded in `hk/upstream-removed.json`. `frontend/src/views/admin/hk/GroupsView.vue` is the current example.
- Small edits to an upstream file stay as mount lines. New copy goes in a fork locale module or `copy(zh, en)`, not into upstream locale files.
- Self-serve gateway checkout is not linked from the user shell. Card redemption and balance-funded subscriptions stay. `/settings/profile` redirects to `/profile`.
- HK account operations summaries and upstream-balance widgets, kept as separable components/composables.
- Image2 workbench as a pinned submodule plus reproducible embedded build.
  - Sidebar links open `/image2/` in a new tab.
  - The legacy `/image2` route remains for bookmarks and redirects.
  - The submodule pointer and `backend/internal/web/image2-dist` must update atomically.

Primary paths:

- `frontend/src/hk/`
- `frontend/src/views/portal/`
- `frontend/src/components/layout/UserShell.vue`
- `frontend/src/components/layout/AdminShell.vue`
- `frontend/src/views/admin/hk/GroupsView.vue`
- `frontend/src/views/user/Image2RedirectView.vue`
- `hk/upstream-removed.json`
- `backend/internal/web/embed_on.go`
- `backend/internal/web/image2-dist/`
- `image2-playground`

## 4. `hk-deploy`

HK-only deployment policy:

- Application ports bind to loopback; Caddy is the public ingress.
- Docker build proxy and local base-image caching remain deployment concerns.
- Redis ACL and secure URL defaults remain deployment overlays.
- Migration CRLF normalization is a Windows/HK build guard, not a business feature.
- Release-branding cleanup is currently out of scope; preserve the existing HK release behavior.
- `datamanagementd` is retired. Do not reintroduce its socket, daemon, installer, routes, or UI API. Database backup/restore continues through the application `BackupService` and operator-owned server procedures.
- Binary archives and image tarballs must not be committed under `deploy_artifacts/`.

Primary paths:

- `deploy/`
- `Dockerfile`
- `.goreleaser*.yaml`
- root `Makefile`

## 5. `upstream-owned`

Use upstream implementations and retain only narrowly justified regression tests or presentation-only patches:

- Grok/xAI
- Passkey/WebAuthn
- Model Plaza
- Compact Home
- Security Audit/Qwen3Guard
- Kimi K3
- GLM-5.2
- Claude Sonnet 5 aliases
- Generic account bulk operations
- Generic SMTP and payment-selector fixes

## Generated outputs

Do not maintain these as independent features:

- `frontend/src/constants/platformCatalog.builtin.json` generated from the backend platform catalog and Provider Profiles (`UPDATE_PLATFORM_CATALOG=1` catalog test).
- `backend/ent/**` generated from `backend/ent/schema/**`
- `backend/cmd/server/wire_gen.go` generated by `go generate ./cmd/server`
- `backend/internal/web/image2-dist/**` generated from the pinned Image2 source
- version bumps, merge-reconciliation commits, and toolchain synchronization commits

## Upstream merge checklist

Run `pwsh -NoProfile -File hk/sync-upstream.ps1 -DryRun`, read `hk/reports/`, then run the script without `-DryRun`. It fetches through `http://127.0.0.1:7890`, disables rename detection, and does not commit or push. Proxy failure stops the sync.

1. Do not rewrite published migrations. HK migration 246 converges the platform constraints after immutable HK 245; verify both existing-HK upgrades and fresh installs. The report lists overlapping migration numbers.
2. Let the script delete conflicts for paths in `hk/upstream-removed.json`. Port the listed upstream commits into the replacement file instead of restoring the old path.
3. Rebase every `upstream-owned` area onto the upstream implementation.
4. Review local diffs by the four layers above; reject unclassified changes.
5. Regenerate Ent/Wire/Image2 outputs when their sources change. Do not hand-merge `backend/ent/**` or `wire_gen.go`.
6. Run money-path tests first: payment claims, subscriptions, DeepSeek pricing, and usage-log contracts.
7. Run `pnpm typecheck`, `pnpm lint:check`, `pnpm test:run`, and `pnpm build` in `frontend/`. The removed-path and backend-linked-route specs must pass.
8. Keep Git push, production deployment, and database operations as separate approved steps.
