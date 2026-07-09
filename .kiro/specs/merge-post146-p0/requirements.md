# merge-post146-p0 requirements

- Cherry-pick only confirmed post-v0.1.146 P0/P1 defect fixes missing from local `48124d12`.
- Include: Antigravity production forward default (`cbfeab96`), compact body-signal detection + handler promotion (`2dd2be99`, `a56eb5b4`), Grok text-only video routing (`3b209935`).
- After handler promotion, remove any stale Forward-path body-signal call site left by the monolithic `openai_gateway_service.go` layout.
- Do not include batch-image, pure-move refactors, force_priority, grok composer bridge, scheduler-score opt-in, sponsors, or deploy default changes.
- Preserve local HK/JP behavior: gpt-5.5 routing, sticky/priority/failover, usage/billing incentives, safe deploy defaults.
