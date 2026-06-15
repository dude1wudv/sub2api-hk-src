# Embedded Image Generation Page Tasks

- [x] T1: Confirm route and embed decisions with the operator.
  - Depends: none
  - Read: `.kiro/specs/embedded-image-generation-page/requirements.md`, `.kiro/specs/embedded-image-generation-page/design.md`
  - Write: `.kiro/specs/embedded-image-generation-page/requirements.md`, `.kiro/specs/embedded-image-generation-page/design.md`
  - Verify: resolved decisions are recorded in the spec.

- [x] T2: Audit current `/image2` implementation for reuse gaps.
  - Depends: T1
  - Read: `frontend/src/views/user/ImageWorkbenchView.vue`, `frontend/src/utils/imageWorkbench.ts`, `frontend/src/router/index.ts`
  - Write: none
  - Verify: handoff summarizes reusable code, missing controls, route impact, and privacy risks.

- [x] T3: Add canonical public route and navigation entry.
  - Depends: T2
  - Read: `frontend/src/router/index.ts`, frontend navigation/layout components
  - Write: `frontend/src/router/index.ts`, relevant navigation/layout files
  - Verify: route has `requiresAuth: false` and `/image2` compatibility is intentional.

- [x] T4: Refactor image workbench state only if needed.
  - Depends: T2
  - Read: `frontend/src/views/user/ImageWorkbenchView.vue`, `frontend/src/utils/imageWorkbench.ts`
  - Write: `frontend/src/composables/useImageGenerationWorkbench.ts`, `frontend/src/utils/imageWorkbench.ts`, `frontend/src/views/user/ImageWorkbenchView.vue`
  - Verify: existing text/image generation behavior remains covered by focused tests or manual QA.

- [x] T5: Implement polished embeddable image generation UI.
  - Depends: T3, T4
  - Read: `frontend/src/views/user/ImageWorkbenchView.vue`, existing public/user view style patterns
  - Write: `frontend/src/views/public/ImageGenerationView.vue` or `frontend/src/views/user/ImageWorkbenchView.vue`
  - Verify: desktop, mobile, empty, loading, error, and generated-result states render correctly.

- [x] T6: Extend safe embed query-parameter handling.
  - Depends: T5
  - Read: image generation view/composable, `frontend/src/utils/imageWorkbench.ts`
  - Write: image generation view/composable, `frontend/src/utils/imageWorkbench.ts`
  - Verify: non-secret params prefill correctly; `api_key` is removed from URL immediately if supported.

- [x] T7: Add focused frontend tests.
  - Depends: T5, T6
  - Read: `frontend/src/utils/__tests__/imageWorkbench.spec.ts`, `frontend/src/router/__tests__`
  - Write: focused utility/router/component specs
  - Verify: `corepack pnpm --dir frontend exec vitest run src/utils/__tests__/imageWorkbench.spec.ts src/router/__tests__/image-route.spec.ts` passes.

- [x] T8: Run frontend validation and build.
  - Depends: T7
  - Read: `frontend/package.json`
  - Write: none
  - Verify: focused Vitest, `corepack pnpm --dir frontend run typecheck`, and `corepack pnpm --dir frontend run build` pass.

- [x] T9: Prepare deploy handoff.
  - Depends: T8
  - Read: `.kiro/specs/embedded-image-generation-page/requirements.md`, `.kiro/specs/embedded-image-generation-page/design.md`, `.kiro/specs/embedded-image-generation-page/tasks.md`
  - Write: deployment notes or update task status only
  - Verify: final handoff lists changed files, validation results, route URL, embed URL example, and rollback note.



