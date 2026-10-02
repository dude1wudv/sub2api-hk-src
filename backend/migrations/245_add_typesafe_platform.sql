-- Add TypeSafe (Jev System One) as a first-class platform.
--
-- 1. user_platform_quotas.platform CHECK
-- 2. composite_model_routes.target_platform CHECK
--
-- TypeSafe 不是对话模型，不进入渠道监控 provider，因此 channel_monitors /
-- channel_monitor_request_templates 的约束保持不变。
--
-- Runs after both the upstream and HK platform migrations.
-- Keep every existing concrete platform so upgrades preserve HK rows and
-- fresh installs converge to the same constraints.

ALTER TABLE user_platform_quotas
    DROP CONSTRAINT IF EXISTS user_platform_quotas_platform_check;

ALTER TABLE user_platform_quotas
    ADD CONSTRAINT user_platform_quotas_platform_check
    CHECK (platform IN ('anthropic', 'openai', 'gemini', 'antigravity', 'grok',
                        'kimi', 'zhipu', 'deepseek', 'minimax', 'stepfun',
                        'opencode_go', 'mirasim', 'typesafe'));

ALTER TABLE composite_model_routes
    DROP CONSTRAINT IF EXISTS composite_model_routes_target_platform_check;

ALTER TABLE composite_model_routes
    ADD CONSTRAINT composite_model_routes_target_platform_check
    CHECK (target_platform IN ('anthropic', 'openai', 'gemini', 'antigravity', 'grok',
                               'kimi', 'zhipu', 'deepseek', 'minimax', 'stepfun',
                               'opencode_go', 'mirasim', 'typesafe'));
