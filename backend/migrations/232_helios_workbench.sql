-- HeliosGen hosted workbench credentials and one-time launch grants.
-- Grant codes are never persisted; only lowercase SHA-256 hex digests are stored.
CREATE TABLE IF NOT EXISTS workbench_credentials (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workbench   VARCHAR(64) NOT NULL,
    api_key_id  BIGINT NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT workbench_credentials_workbench_check CHECK (workbench IN ('heliosgen'))
);

CREATE UNIQUE INDEX IF NOT EXISTS workbench_credentials_user_workbench_key
    ON workbench_credentials (user_id, workbench);
CREATE UNIQUE INDEX IF NOT EXISTS workbench_credentials_api_key_id_key
    ON workbench_credentials (api_key_id);

CREATE TABLE IF NOT EXISTS workbench_launch_grants (
    id           BIGSERIAL PRIMARY KEY,
    code_hash    VARCHAR(64) NOT NULL,
    user_id      BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    api_key_id   BIGINT NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    client_id    VARCHAR(128) NOT NULL,
    redirect_uri VARCHAR(2048) NOT NULL,
    expires_at   TIMESTAMPTZ NOT NULL,
    consumed_at  TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT workbench_launch_grants_code_hash_check CHECK (code_hash ~ '^[0-9a-f]{64}$')
);

CREATE UNIQUE INDEX IF NOT EXISTS workbench_launch_grants_code_hash_key
    ON workbench_launch_grants (code_hash);
CREATE INDEX IF NOT EXISTS workbench_launch_grants_user_id_idx
    ON workbench_launch_grants (user_id);
CREATE INDEX IF NOT EXISTS workbench_launch_grants_api_key_id_idx
    ON workbench_launch_grants (api_key_id);
CREATE INDEX IF NOT EXISTS workbench_launch_grants_expires_at_idx
    ON workbench_launch_grants (expires_at);
