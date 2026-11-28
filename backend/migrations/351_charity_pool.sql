-- 公益池（free.xjtu.shop）配置、分组绑定与按日折前用量账本
-- 只增不改：不修改任何存量表结构与历史账单记录。

CREATE TABLE IF NOT EXISTS charity_pool_settings (
    id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    daily_budget_usd NUMERIC(20, 8) NOT NULL DEFAULT 100.00000000 CHECK (daily_budget_usd >= 0),
    max_concurrency INT NOT NULL DEFAULT 20 CHECK (max_concurrency >= 0),
    per_ip_daily_limit_usd NUMERIC(20, 8) NOT NULL DEFAULT 0.00000000 CHECK (per_ip_daily_limit_usd >= 0),
    per_ip_max_concurrency INT NOT NULL DEFAULT 0 CHECK (per_ip_max_concurrency >= 0),
    per_ip_rpm_limit INT NOT NULL DEFAULT 0 CHECK (per_ip_rpm_limit >= 0),
    auto_rotate_daily BOOLEAN NOT NULL DEFAULT TRUE,
    rotation_timezone VARCHAR(64) NOT NULL DEFAULT 'Asia/Shanghai',
    site_title VARCHAR(128) NOT NULL DEFAULT 'XJTU Free AI',
    site_notice TEXT NOT NULL DEFAULT '',
    host_allowlist TEXT[] NOT NULL DEFAULT ARRAY['free.xjtu.shop'],
    system_user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO charity_pool_settings (id)
VALUES (1)
ON CONFLICT (id) DO NOTHING;

CREATE TABLE IF NOT EXISTS charity_groups (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL UNIQUE REFERENCES groups(id) ON DELETE CASCADE,
    display_name VARCHAR(128) NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    daily_limit_usd NUMERIC(20, 8) CHECK (daily_limit_usd IS NULL OR daily_limit_usd >= 0),
    max_concurrency INT NOT NULL DEFAULT 0 CHECK (max_concurrency >= 0),
    sort_order INT NOT NULL DEFAULT 0,
    active_api_key_id BIGINT REFERENCES api_keys(id) ON DELETE SET NULL,
    last_rotated_date VARCHAR(10) NOT NULL DEFAULT '',
    last_rotated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_charity_groups_active_api_key
    ON charity_groups (active_api_key_id)
    WHERE active_api_key_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_charity_groups_sort_order
    ON charity_groups (sort_order, id);

CREATE TABLE IF NOT EXISTS charity_daily_group_usage (
    usage_date DATE NOT NULL,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    total_cost_usd NUMERIC(20, 8) NOT NULL DEFAULT 0.00000000,
    request_count BIGINT NOT NULL DEFAULT 0,
    total_tokens BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (usage_date, group_id)
);

CREATE INDEX IF NOT EXISTS idx_charity_daily_group_usage_date
    ON charity_daily_group_usage (usage_date DESC);

CREATE TABLE IF NOT EXISTS charity_daily_ip_usage (
    usage_date DATE NOT NULL,
    client_ip VARCHAR(128) NOT NULL,
    total_cost_usd NUMERIC(20, 8) NOT NULL DEFAULT 0.00000000,
    request_count BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (usage_date, client_ip)
);

CREATE INDEX IF NOT EXISTS idx_charity_daily_ip_usage_date
    ON charity_daily_ip_usage (usage_date DESC);
