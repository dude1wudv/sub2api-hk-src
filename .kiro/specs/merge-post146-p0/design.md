# merge-post146-p0 design

Use ordered cherry-picks from `upstream/main` onto `merge/upstream-post146-p0` (base `48124d12`), not a full merge.

Upstream `#3804` splits Forward into a separate file and drops the temporary Forward body-signal block when promoting detection to the handler. Local still uses monolithic `openai_gateway_service.go`, so after cherry-picking `2dd2be99` then `a56eb5b4`, delete the leftover `hasCompactionTriggerInInput` Forward call site and rely on path-based `isOpenAIResponsesCompactPath` once the handler has rewritten `/compact`.

Conflict policy for shared hot paths: keep JP behavior; add only upstream defect-fix logic.
