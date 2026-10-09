-- Converge fresh installs and existing HK databases on application-owned
-- platform validation. Upstream 242 drops these constraints, but immutable HK
-- migration 245 recreates them on a fresh install. Existing HK databases have
-- already applied 245 when the newly added upstream 242 runs.
ALTER TABLE user_platform_quotas
    DROP CONSTRAINT IF EXISTS user_platform_quotas_platform_check;

ALTER TABLE composite_model_routes
    DROP CONSTRAINT IF EXISTS composite_model_routes_target_platform_check;
