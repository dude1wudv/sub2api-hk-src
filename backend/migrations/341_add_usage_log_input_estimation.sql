-- Add provenance only. Never recalculate, backfill fees, or update historical bills.
ALTER TABLE usage_logs
    ADD COLUMN IF NOT EXISTS input_tokens_estimated BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS cache_read_tokens_estimated BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS usage_estimation_reason VARCHAR(64) NOT NULL DEFAULT '';

COMMENT ON COLUMN usage_logs.input_tokens_estimated IS 'Input tokens estimated from the dispatched request; historical rows default to false';
COMMENT ON COLUMN usage_logs.cache_read_tokens_estimated IS 'Estimated input discount, not upstream cache hits; historical rows default to false';
COMMENT ON COLUMN usage_logs.usage_estimation_reason IS 'Reason for incomplete usage estimation; historical rows remain empty';
