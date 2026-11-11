ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS anthropic_cache_ttl_mode VARCHAR(10) NOT NULL DEFAULT 'inherit';
ALTER TABLE api_keys ADD CONSTRAINT api_keys_anthropic_cache_ttl_mode_check CHECK (anthropic_cache_ttl_mode IN ('inherit', 'off', '5m', '1h'));
