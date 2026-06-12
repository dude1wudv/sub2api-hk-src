-- 每日余额（daily-balance）功能
-- 1) 新建 daily_balance_grants 表：带 24h 有效期的独立额度桶，仅在绑定的专属分组内消费。
-- 2) groups 增加两列：daily_balance_enabled（专属分组开关）、daily_fallback_multiplier（长期余额回退倍率）。
-- 所有变更幂等，老分组默认 daily_balance_enabled=false → 行为与现状完全一致。

CREATE TABLE IF NOT EXISTS daily_balance_grants (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT        NOT NULL,
    group_id    BIGINT        NOT NULL,
    amount      DECIMAL(20,8) NOT NULL DEFAULT 0,
    remaining   DECIMAL(20,8) NOT NULL DEFAULT 0,
    status      VARCHAR(20)   NOT NULL DEFAULT 'active',
    source      VARCHAR(20)   NOT NULL DEFAULT 'admin',
    source_ref  TEXT,
    granted_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ   NOT NULL,
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

-- 热查询：取某用户某专属分组的有效 Grant，按 expires_at 升序消耗
CREATE INDEX IF NOT EXISTS dailybalancegrant_user_id_group_id_status_expires_at
    ON daily_balance_grants (user_id, group_id, status, expires_at);
-- 过期扫描：按 status + expires_at 扫描 active 且已过期的行
CREATE INDEX IF NOT EXISTS dailybalancegrant_status_expires_at
    ON daily_balance_grants (status, expires_at);
CREATE INDEX IF NOT EXISTS dailybalancegrant_expires_at
    ON daily_balance_grants (expires_at);

ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS daily_balance_enabled BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS daily_fallback_multiplier DECIMAL(10,4) NOT NULL DEFAULT 1.5;
