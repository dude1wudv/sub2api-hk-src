ALTER TABLE usage_logs
    ADD COLUMN IF NOT EXISTS requested_service_tier VARCHAR(32),
    ADD COLUMN IF NOT EXISTS upstream_service_tier VARCHAR(32),
    ADD COLUMN IF NOT EXISTS service_tier_downgraded BOOLEAN;

COMMENT ON COLUMN usage_logs.requested_service_tier IS
    'Final service tier sent upstream after group policy; NULL for historical or unspecified requests';
COMMENT ON COLUMN usage_logs.upstream_service_tier IS
    'Service tier explicitly reported by upstream; NULL when absent or historical';
COMMENT ON COLUMN usage_logs.service_tier_downgraded IS
    'Upstream reported a lower tier than requested; independent of the billed service_tier, NULL when unknown';
