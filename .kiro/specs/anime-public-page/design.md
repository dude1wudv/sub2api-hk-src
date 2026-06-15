# Anime Public Page Design

## Architecture

The page is a public Vue route, `frontend/src/views/public/AnimeHubView.vue`, registered in `frontend/src/router/index.ts` as `/anime`.

It runs entirely client-side:

- Fetch weekly airing schedules and seasonal trending titles from the AniList GraphQL API.
- Normalize titles, schedule times, scores, popularity, artwork, and genre labels.
- Filter adult entries from the API response and suppress adult-oriented genre tags.
- Extract AniList `externalLinks` with `type: STREAMING`, keep only official platform allowlist hosts, and render them before search fallbacks.
- Render schedule tabs, featured title details, official outbound links, and an open sample trailer.

The backend change is limited to the default CSP string in `backend/internal/config/config.go`, adding `media-src 'self' https:` so the open sample media can load.

## Animeko Reference Boundary

Animeko is useful as a reference for how an anime client thinks about sources: adapters, multiple providers, metadata aggregation, playback state, and user-facing source selection.

For this public Sub2API page, the safe subset is metadata aggregation and official outbound linking. The page must not reuse source extraction logic that obtains direct third-party episode video URLs unless the source is clearly official and authorized for embedding.

The implemented source strategy mirrors Animeko at a small scale:

- A static official streaming provider allowlist behaves like a provider registry.
- AniList metadata is the schedule and link provider.
- Official streaming links have priority over fallback search links.
- Any unknown host, disabled link, non-streaming link, non-HTTP(S) value, torrent, cache, or hidden video extraction path is excluded.

## Deployment Design

Deployment uses a clean detached worktree under `E:\AI-Platform-Projects\tmp\anime-deploy-worktree`:

- Build frontend and backend image from the clean worktree.
- Save the Docker image to a tarball.
- Upload the tarball to `/opt/sub2api-deploy/`.
- Load the image on `154.88.65.45`.
- Update only the `sub2api` service image in `/opt/sub2api/docker-compose.yml`.
- Add `fun.sunmmyapi.xyz` to Caddy's existing Sub2API site block or an equivalent reverse proxy block.
- Recreate only the `sub2api` container.

## Verification

- Local build: `corepack pnpm --dir frontend run build`.
- Container build: `docker build` from the clean worktree.
- Remote health: `curl http://127.0.0.1:8080/health`.
- Public health: `https://sub.sunmmyapi.xyz/health`.
- Public page: `https://fun.sunmmyapi.xyz/anime` returns 200 and contains `Anime Weekly`.
- CSP header includes `media-src`.
- Desktop and mobile render checks show no blank page, framework overlay, or horizontal overflow.
