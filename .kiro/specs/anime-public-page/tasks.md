# Anime Public Page Tasks

- [x] T1: Create Kiro-style spec files for the feature and deployment boundary.
  - Verify: spec files exist under `.kiro/specs/anime-public-page/`.
- [x] T2: Review Animeko source strategy and summarize safe reusable ideas.
  - Verify: compact worker summary names source modules and safe/non-safe boundaries.
- [x] T3: Review current `/anime` implementation against requirements.
  - Verify: controller review covered route auth, CSP, adult filtering, and source strategy.
- [x] T4: Apply any needed source-strategy or UI refinements in the clean worktree.
  - Verify: `git status --short` contains only intended files.
- [x] T5: Build and save Docker image from the clean worktree.
  - Verify: `docker image inspect` and local tarball exist.
- [x] T6: Upload and deploy image to `154.88.65.45`.
  - Verify: remote compose runs the new image and app health is OK.
- [ ] T7: Add/verify `fun.sunmmyapi.xyz` Caddy ingress.
  - Verify: Caddy config validates and source-origin `fun` route returns 200; public Cloudflare route is still 521 pending DNS/proxy fix.
- [ ] T8: Run final local/public QA and notify via Feishu.
  - Verify: rendered QA passes on deployed app; final Feishu link should wait until public `fun` route returns 200.
