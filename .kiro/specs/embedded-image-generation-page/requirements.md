# Embedded Image Generation Page Requirements

## Goal

Add a website-embeddable image generation page to Sub2API, similar to the provided screenshot: a focused public page where visitors can select an API key/model, enter a prompt or reference image, generate images through Sub2API image endpoints, and view/download the results without leaving the site.

## Resolved Decisions

- Use /image as the canonical public and embeddable route.
- Keep /image2 as a backwards-compatible alias during rollout.
- Keep the page public and embeddable by direct link or iframe; generation still requires a valid Sub2API API key.
- Store API keys in browser session storage only, never backend storage or local storage.
- Default the first implementation to gpt-image-2 while exposing a model selector for compatible image models.

## User Stories

- As a site visitor, I can open the image generation page from the website navigation without logging in.
- As a site visitor, I can paste or select a Sub2API API key and generate an image from text.
- As a site visitor, I can optionally upload a reference image and run image-to-image generation.
- As a site visitor, I can choose common parameters such as model, size, quality, output format, and count.
- As a site visitor, I can preview generated results, download them, and reuse a prompt after failures.
- As the operator, I can embed the page into another website with a compact layout and optional query parameters.

## Acceptance Criteria

- A public route renders the image generation page without authentication guard redirects.
- The first viewport matches the intended two-column workflow: controls on the left, generated result preview on the right.
- Text-to-image calls `POST /v1/images/generations` through the configured Sub2API base URL.
- Image-to-image calls `POST /v1/images/edits` with multipart form data when reference images are provided.
- API key input supports `Bearer` normalization and never appears in the URL after initial adoption.
- Query parameters can prefill non-secret settings such as API base URL, model, prompt, size, quality, and mode.
- Secret query parameters, if temporarily supported for embed compatibility, are moved to session storage and removed from the address bar immediately.
- The UI has clear loading, success, empty, and error states.
- Generated image extraction supports URL responses and base64 responses.
- The page works on desktop and mobile without horizontal overflow.
- Existing `/image2` behavior either continues to work or redirects/aliases to the new page intentionally.

## Constraints

- Source of truth is `E:\AI-Platform-Projects\sub2api-jp-src`.
- Keep implementation frontend-first unless a backend gap is found.
- Reuse existing image workbench helpers in `frontend/src/utils/imageWorkbench.ts` where possible.
- Do not store API keys, prompts, uploaded images, or generated images on the server in this phase.
- Do not weaken global CSP, auth guards, or billing/account behavior.
- Do not require new external services beyond the existing Sub2API image endpoints.
- Use pnpm for frontend validation.

## Non-Goals

- Do not build a full gallery, team workspace, or server-side asset library in the first phase.
- Do not add payment, wallet, or API-key management flows to this page.
- Do not proxy third-party image hosting separately unless required by current Sub2API responses.
- Do not redesign unrelated dashboard, admin, or anime pages.

