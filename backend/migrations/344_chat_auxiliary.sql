-- Server-owned auxiliary metering. No historical bills or defaults are rewritten.
ALTER TABLE usage_logs ADD COLUMN chat_auxiliary_usage JSONB;
ALTER TABLE chat_model_presets DROP CONSTRAINT chat_presets_normal_reasoning_effort_check;
ALTER TABLE chat_model_presets ADD CONSTRAINT chat_presets_normal_reasoning_effort_check
    CHECK (normal_reasoning_effort IN ('none','low','medium','high'));

ALTER TABLE chat_attachment_uploads ADD COLUMN lease_token UUID;
ALTER TABLE chat_attachment_uploads ADD COLUMN lease_expires_at TIMESTAMPTZ;

-- No cascading delete: a known billing receipt must survive body/attachment removal
-- until its original command has been applied idempotently.
CREATE TABLE chat_attachment_parses (
    id UUID PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    attachment_id UUID NOT NULL,
    account_id BIGINT NOT NULL REFERENCES accounts(id),
    purpose VARCHAR(32) NOT NULL DEFAULT 'text_recognition' CHECK (purpose = 'text_recognition'),
    status VARCHAR(20) NOT NULL DEFAULT 'processing' CHECK (status IN ('processing','ready','failed','unknown')),
    result JSONB,
    page_count INTEGER CHECK (page_count >= 0 AND page_count <= 1000),
    operation_id UUID NOT NULL,
    request_id VARCHAR(128) NOT NULL DEFAULT '',
    billing_receipt JSONB,
    billing_applied BOOLEAN NOT NULL DEFAULT FALSE,
    reserved_bytes BIGINT NOT NULL DEFAULT 0 CHECK (reserved_bytes >= 0 AND reserved_bytes <= 8388608),
    result_bytes BIGINT NOT NULL DEFAULT 0 CHECK (result_bytes >= 0 AND result_bytes <= 8388608),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (attachment_id,account_id,purpose),
    FOREIGN KEY (user_id,attachment_id) REFERENCES chat_attachments(user_id,id),
    CHECK (reserved_bytes = 0 OR result_bytes = 0),
    CHECK (result IS NULL OR octet_length(result::text) <= 8388608),
    CHECK (billing_receipt IS NULL OR octet_length(billing_receipt::text) <= 65536),
    CHECK (NOT billing_applied OR billing_receipt IS NOT NULL)
);
CREATE INDEX chat_attachment_parses_owner_status ON chat_attachment_parses(user_id,status);
CREATE INDEX chat_attachment_parses_pending_receipt ON chat_attachment_parses(updated_at)
    WHERE billing_receipt IS NOT NULL AND NOT billing_applied;
