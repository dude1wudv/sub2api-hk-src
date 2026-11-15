CREATE TABLE deepseek_suite_operations (
    id UUID PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    api_key_id BIGINT NOT NULL REFERENCES api_keys(id),
    group_id BIGINT NOT NULL REFERENCES groups(id),
    account_id BIGINT NOT NULL REFERENCES accounts(id),
    kind TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'started',
    remote_id TEXT NOT NULL DEFAULT '',
    upstream_name TEXT NOT NULL DEFAULT '',
    filename TEXT NOT NULL DEFAULT '',
    purpose TEXT NOT NULL DEFAULT '',
    bytes BIGINT NOT NULL DEFAULT 0,
    metadata JSONB,
    receipt JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (status IN ('started', 'unknown', 'awaiting_meter', 'settling', 'settled', 'failed', 'deleted'))
);
CREATE INDEX deepseek_suite_files_owner ON deepseek_suite_operations (user_id, group_id, created_at DESC, id)
    WHERE kind = 'files' AND remote_id <> '' AND status NOT IN ('failed', 'deleted');
CREATE INDEX deepseek_suite_pending ON deepseek_suite_operations (updated_at, id)
    WHERE status IN ('started', 'unknown', 'awaiting_meter', 'settling');
CREATE UNIQUE INDEX deepseek_suite_remote_file ON deepseek_suite_operations (account_id, remote_id)
    WHERE kind = 'files' AND remote_id <> '';
