-- Track whether the recorded reasoning effort was explicit, default-injected, or model-suffix derived.
ALTER TABLE usage_logs
    ADD COLUMN IF NOT EXISTS reasoning_effort_source VARCHAR(20);

COMMENT ON COLUMN usage_logs.reasoning_effort_source IS
    'reasoning effort provenance: explicit, default, or model_suffix';
