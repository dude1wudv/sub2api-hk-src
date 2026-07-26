-- Track whether cache token counts came from OpenAI or a local OAuth estimate.
ALTER TABLE usage_logs ADD COLUMN IF NOT EXISTS cache_usage_source VARCHAR(20) NOT NULL DEFAULT 'none';