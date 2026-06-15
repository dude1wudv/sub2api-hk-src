# Anime Public Page Requirements

## Goal

Publish a no-login anime page at `https://fun.sunmmyapi.xyz/anime` for the Sub2API server.

## User Stories

- As a public visitor, I can open `/anime` without logging in.
- As a public visitor, I can browse this week's anime schedule and trending titles.
- As a public visitor, I can follow official or legal outbound watch/search links.
- As the operator, I can deploy the feature without packaging unrelated dirty worktree changes.

## Acceptance Criteria

- `/anime` returns HTTP 200 on `fun.sunmmyapi.xyz`.
- The page renders a meaningful first viewport on desktop and mobile.
- The page does not require authentication in frontend route guards or backend-mode route allowlists.
- The page uses AniList public metadata for schedules/trending and filters adult entries.
- The page may include only open-licensed sample playback or official outbound links.
- The app CSP permits the intended sample media without weakening unrelated frame or script policy.
- Existing `sub.sunmmyapi.xyz` health remains OK after deployment.

## Constraints

- Use `E:\AI-Platform-Projects\sub2api-jp-src` as the source of truth for this server.
- Build and deploy from a clean worktree containing only intended `/anime` and CSP changes.
- Do not reset Postgres or Redis volumes.
- Keep the app container bound to `127.0.0.1:8080` behind Caddy.
- Preserve existing Caddy streaming behavior and rewrites.
- Use `fun.sunmmyapi.xyz` as the public anime domain.
- Reference Animeko for source-adapter and aggregation ideas only.

## Non-Goals

- Do not proxy, scrape, download, seed, or redistribute third-party anime streams.
- Do not add account, billing, or admin changes.
- Do not include unrelated Kiro gateway/backend changes from the main worktree.
