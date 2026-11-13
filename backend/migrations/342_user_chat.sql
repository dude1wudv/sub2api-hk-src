-- Private per-user chat. Existing billing tables, checksums and retention are untouched.
ALTER TABLE groups ADD COLUMN IF NOT EXISTS chat_model_policies JSONB;

CREATE TABLE chat_model_presets (
    id UUID PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_id BIGINT NOT NULL REFERENCES groups(id),
    model VARCHAR(256) NOT NULL CHECK (model <> ''),
    display_name VARCHAR(200) NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, group_id, model)
);
CREATE UNIQUE INDEX chat_presets_one_default ON chat_model_presets(user_id) WHERE is_default;
CREATE INDEX chat_presets_order ON chat_model_presets(user_id, sort_order);

-- Deliberately do not cascade key deletion: a lost/disabled binding must not
-- silently grant a replacement key or reset quota/rate limits.
CREATE TABLE chat_credentials (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_id BIGINT NOT NULL REFERENCES groups(id),
    api_key_id BIGINT NOT NULL UNIQUE REFERENCES api_keys(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, group_id)
);

CREATE TABLE chat_sessions (
    id UUID PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(500) NOT NULL DEFAULT '',
    title_manual BOOLEAN NOT NULL DEFAULT FALSE,
    snapshot JSONB NOT NULL CHECK (jsonb_typeof(snapshot) = 'object'),
    revision BIGINT NOT NULL DEFAULT 0 CHECK (revision >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, id)
);
CREATE INDEX chat_sessions_cursor ON chat_sessions(user_id, updated_at DESC, id DESC);

CREATE TABLE chat_generation_runs (
    id UUID PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_id UUID NOT NULL,
    assistant_message_id VARCHAR(200) NOT NULL,
    preset_id UUID NOT NULL,
    group_id BIGINT NOT NULL,
    model VARCHAR(256) NOT NULL,
    options JSONB NOT NULL DEFAULT '{}',
    request_id UUID NOT NULL,
    operation VARCHAR(20) NOT NULL CHECK (operation IN ('send','regenerate','continue')),
    status VARCHAR(20) NOT NULL DEFAULT 'running' CHECK (status IN ('running','done','stopped','error')),
    owner_instance_id UUID NOT NULL,
    lease_expires_at TIMESTAMPTZ NOT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    heartbeat_at TIMESTAMPTZ NOT NULL,
    ended_at TIMESTAMPTZ,
    gateway_request_ids JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, request_id),
    FOREIGN KEY (user_id, session_id) REFERENCES chat_sessions(user_id,id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX chat_runs_one_active ON chat_generation_runs(session_id) WHERE status = 'running';
CREATE INDEX chat_runs_leases ON chat_generation_runs(status, lease_expires_at);

CREATE TABLE chat_imports (
    id UUID PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','committed','failed')),
    expires_at TIMESTAMPTZ NOT NULL,
    history_revision BIGINT NOT NULL CHECK (history_revision >= 0),
    manifest JSONB NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(manifest) = 'object'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id,id)
);
CREATE INDEX chat_imports_expiry ON chat_imports(status, expires_at);

CREATE TABLE chat_attachments (
    id UUID PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    storage_key VARCHAR(200) NOT NULL UNIQUE,
    original_name VARCHAR(255) NOT NULL,
    mime_type VARCHAR(128) NOT NULL DEFAULT '',
    byte_size BIGINT NOT NULL CHECK (byte_size >= 0 AND byte_size <= 67108864),
    sha256 VARCHAR(64) NOT NULL DEFAULT '',
    purpose VARCHAR(32) NOT NULL CHECK (purpose IN ('image','rich_layout_recognition','user_data')),
    status VARCHAR(20) NOT NULL DEFAULT 'uploading' CHECK (status IN ('uploading','ready','staged','orphan','deleting')),
    import_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id,id),
    CHECK (purpose <> 'user_data' OR byte_size <= 33554432),
    FOREIGN KEY (user_id,import_id) REFERENCES chat_imports(user_id,id)
);
CREATE INDEX chat_attachments_owner_status ON chat_attachments(user_id,status);
CREATE INDEX chat_attachments_cleanup ON chat_attachments(status,updated_at);

-- Composite FKs enforce both owners even if a future caller misses a predicate.
CREATE TABLE chat_session_attachments (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_id UUID NOT NULL,
    attachment_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (session_id,attachment_id),
    FOREIGN KEY (user_id,session_id) REFERENCES chat_sessions(user_id,id) ON DELETE CASCADE,
    FOREIGN KEY (user_id,attachment_id) REFERENCES chat_attachments(user_id,id)
);
CREATE INDEX chat_session_attachments_refs ON chat_session_attachments(attachment_id);

CREATE TABLE chat_attachment_uploads (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    attachment_id UUID NOT NULL,
    account_id BIGINT NOT NULL REFERENCES accounts(id),
    remote_file_id VARCHAR(512) NOT NULL DEFAULT '',
    status VARCHAR(20) NOT NULL DEFAULT 'uploading' CHECK (status IN ('uploading','ready','invalid','deleting','error')),
    cleanup_attempts INTEGER NOT NULL DEFAULT 0 CHECK (cleanup_attempts >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (attachment_id,account_id),
    FOREIGN KEY (user_id,attachment_id) REFERENCES chat_attachments(user_id,id) ON DELETE CASCADE
);
CREATE INDEX chat_attachment_uploads_owner_status ON chat_attachment_uploads(user_id,status);

CREATE TABLE chat_preferences (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    preferences JSONB NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(preferences) = 'object'),
    history_revision BIGINT NOT NULL DEFAULT 0 CHECK (history_revision >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
