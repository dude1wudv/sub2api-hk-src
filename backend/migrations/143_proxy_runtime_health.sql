ALTER TABLE proxies ADD COLUMN IF NOT EXISTS cooldown_until TIMESTAMPTZ;
ALTER TABLE proxies ADD COLUMN IF NOT EXISTS cooldown_reason TEXT;
ALTER TABLE proxies ADD COLUMN IF NOT EXISTS failure_count INT NOT NULL DEFAULT 0;
ALTER TABLE proxies ADD COLUMN IF NOT EXISTS last_error_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_proxies_cooldown_until ON proxies(cooldown_until);
