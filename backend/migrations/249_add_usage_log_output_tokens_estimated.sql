ALTER TABLE usage_logs
    ADD COLUMN IF NOT EXISTS output_tokens_estimated BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN usage_logs.output_tokens_estimated IS
    'Output tokens estimated from received content because final upstream usage was missing; historical rows default to false';
