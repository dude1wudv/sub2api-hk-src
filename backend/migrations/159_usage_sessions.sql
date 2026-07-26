CREATE TABLE IF NOT EXISTS usage_sessions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_index INTEGER NOT NULL CHECK (session_index > 0),
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, session_index)
);

CREATE TABLE IF NOT EXISTS usage_session_keys (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_key_hash TEXT NOT NULL,
    session_id BIGINT NOT NULL REFERENCES usage_sessions(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, session_key_hash)
);

ALTER TABLE usage_logs
    ADD COLUMN IF NOT EXISTS session_id BIGINT REFERENCES usage_sessions(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_usage_logs_session_id
    ON usage_logs(session_id)
    WHERE session_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_usage_session_keys_session_id
    ON usage_session_keys(session_id);
