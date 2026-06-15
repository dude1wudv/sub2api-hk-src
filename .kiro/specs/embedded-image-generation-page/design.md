# Embedded Image Generation Page Design

## Current State

Sub2API already has a public `/image2` route backed by `frontend/src/views/user/ImageWorkbenchView.vue`. The view supports text-to-image, image-to-image, browser-only API key handling, draft persistence, request tracing headers, and result extraction via `frontend/src/utils/imageWorkbench.ts`.

The backend already contains OpenAI-compatible image endpoints such as `/v1/images/generations` and image intent/rate-limit handling. No backend change is expected for the first planning target unless validation finds missing route/CSP support.

## Target UX

Create a polished website-embeddable page with the structure shown in the screenshot:

- Left panel: API key/source selector, mode tabs, model selector, prompt/reference controls, size, count, quality, format, and generate button.
- Right panel: result card with empty state, progress state, generated image grid, image metadata, download/open actions, and retry affordance.
- Compact embed mode: optional chrome-light layout suitable for iframe usage, controlled by query parameter such as `?embed=1`.
- Mobile layout: controls stack above results, with sticky generate action only if it does not obscure inputs.

## Route Strategy

Recommended route plan:

- Keep `/image2` as a backwards-compatible alias during rollout.
- Add a clearer public route such as `/image` or `/tools/image` for website navigation and embeds.
- Use `requiresAuth: false` route metadata.
- Add or update router title tests if title behavior is covered nearby.

The final route name should be chosen before implementation. If this page is meant to replace `/image2`, `/image2` should redirect to the canonical route after QA.

## Component Strategy

Prefer an incremental refactor rather than duplicating the current workbench:

- Extract reusable image generation state/actions into a composable, e.g. `frontend/src/composables/useImageGenerationWorkbench.ts`, only if the existing view becomes too large or the embed page needs a second component.
- Keep request helpers in `frontend/src/utils/imageWorkbench.ts` and extend them for model, count, and output format only when the API payload supports those fields.
- Move the polished UI into either the existing `ImageWorkbenchView.vue` or a new public view such as `frontend/src/views/public/ImageGenerationView.vue`.
- Avoid introducing a UI library dependency; follow existing Vue/Tailwind-style classes used in the frontend.

## Data Flow

1. Restore non-secret draft settings from `localStorage`.
2. Restore API key from `sessionStorage` only.
3. Adopt safe query parameters for embedding.
4. Normalize API base URL with `normalizeImageApiBase`.
5. Build request headers with `buildBearerHeaders` and `buildClientTraceHeaders`.
6. Send JSON payload to `/v1/images/generations` for text mode.
7. Send multipart payload to `/v1/images/edits` for reference-image mode.
8. Extract generated images from URL or base64 response using `extractImageResults`.
9. Render results locally without server persistence.

## Embed Contract

Supported query parameters should be explicit and documented in the UI or route README:

- `embed=1`: compact layout for iframe.
- `api_base` or `base_url`: prefill Sub2API base URL.
- `model`: prefill image model.
- `prompt`: prefill prompt text.
- `size`: prefill output size.
- `quality`: prefill output quality.
- `mode`: `text` or `image`.

Avoid accepting secrets in URL long term. If compatibility requires `api_key` initially, remove it from the URL immediately after storing it in `sessionStorage`, matching the current `/image2` behavior.

## Safety And Privacy

- API keys stay in browser memory/session storage and are never written to backend storage by this page.
- Reference images are sent only to the configured Sub2API image endpoint during generation.
- Generated images are displayed from response data only; no automatic upload or gallery persistence.
- Error messages should be useful but should not echo full keys, tokens, or raw response bodies with secrets.
- The page should not bypass existing server-side wallet, rate limit, or risk-control enforcement.

## Verification Strategy

- Unit tests for helper changes in `frontend/src/utils/__tests__/imageWorkbench.spec.ts`.
- Route/title test if a new public route or alias is added.
- Component test for key UX states if the project already has adjacent Vue test patterns suitable for this view.
- Manual browser QA for desktop, mobile, and `?embed=1` layouts.
- Targeted frontend validation: `pnpm --dir frontend run typecheck` and focused Vitest specs.
- Full frontend build before deploy: `pnpm --dir frontend run build`.

## Deployment Boundary

This phase should be deployable as a frontend-only image update if backend validation confirms image endpoints and CSP are already sufficient. Preserve existing live server invariants: no DB/Redis resets, app remains behind Caddy, and deployment should not disturb Kiro gateway or anime public page configuration.
