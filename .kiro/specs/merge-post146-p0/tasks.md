# merge-post146-p0 tasks

1. Create `merge/upstream-post146-p0` from `48124d12`.
2. Cherry-pick `cbfeab96` (Antigravity prod forward default).
3. Cherry-pick `2dd2be99` then `a56eb5b4` (compact body-signal).
4. Remove stale Forward body-signal call site in monolithic `openai_gateway_service.go`.
5. Cherry-pick `3b209935` (Grok text-only video routing).
6. Verify: `go build` service/handler; `go test -tags=unit ./internal/handler/ -run Compact|BodySignal|Grok`.
7. Reject batch-image, pure-move refactors, force_priority, composer bridge, scheduler-score opt-in.
