-- External artifacts are independent reference data, never usage or routing signals.
CREATE TABLE IF NOT EXISTS external_quality_history (
    id BIGSERIAL PRIMARY KEY,
    source VARCHAR(24) NOT NULL CHECK (source = 'mdkj'),
    source_event_id UUID NOT NULL,
    remote_group_id BIGINT NOT NULL,
    group_id BIGINT NOT NULL,
    model_id VARCHAR(128) NOT NULL,
    response_model VARCHAR(128) NOT NULL DEFAULT '',
    status VARCHAR(24) NOT NULL CHECK (status = 'completed'),
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ NOT NULL CHECK (finished_at >= started_at),
    duration_ms BIGINT NOT NULL CHECK (duration_ms >= 0),
    artwork TEXT NOT NULL DEFAULT '',
    content_sha256 VARCHAR(64) NOT NULL,
    imported_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    imported_by BIGINT NOT NULL,
    CONSTRAINT external_quality_history_binding CHECK (
        (group_id = 72 AND remote_group_id = 62) OR
        (group_id = 237 AND remote_group_id = 170)
    ),
    CONSTRAINT external_quality_history_model CHECK (model_id IN ('gpt-6-astra', 'gpt-6.1-sol')),
    UNIQUE (source, group_id, source_event_id)
);
CREATE INDEX IF NOT EXISTS external_quality_history_group_model_time_idx
    ON external_quality_history (group_id, model_id, started_at DESC, id DESC);
COMMENT ON TABLE external_quality_history IS 'Imported generation artifacts; completed is not a quality verdict. Provenance is internal only.';
