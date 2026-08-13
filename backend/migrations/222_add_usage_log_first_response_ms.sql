ALTER TABLE usage_logs
    ADD COLUMN IF NOT EXISTS first_response_ms INTEGER;
