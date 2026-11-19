Changes to the managed-protocol contract must be checked against `../apps/heliosgen-sub2api/` on both ends.

## Git, PR, and Auto-Deploy Workflow
1. Create a feature branch from `main`/`master`. Never force push maintenance branches, never rewrite history, and never commit directly to `main`/`master`.
2. Read the target repository's `AGENTS.md`, `README`, and existing check commands first.
3. After editing, run the repository's existing tests, type checks, and builds; do not skip failing checks.
4. Open a PR clearly stating: what changed, why, what was verified, what was not verified, and what the risks are. Confirm CI passes.
5. Merge into `main`/`master`.

This repository is connected to a GitHub webhook: after merging into `main`/`master`, HK automatically pulls the pinned commit, builds, and deploys (preserving shared PostgreSQL, Redis, networks, and volumes; building locally and transferring an image is a fallback only).

If modifying the `chat-demo` submodule:
- First open and merge a PR in `dude1wudv/chat-demo`.
- Then update the submodule pointer in `dude1wudv/sub2api-hk-src` via PR and merge.
- Merging only `chat-demo` will NOT trigger HK deployment.
