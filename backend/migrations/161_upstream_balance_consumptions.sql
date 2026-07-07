CREATE TABLE IF NOT EXISTS upstream_balance_snapshots (
    account_id      BIGINT PRIMARY KEY,
    group_id        BIGINT NOT NULL DEFAULT 0,
    account_name    TEXT NOT NULL DEFAULT '',
    group_name      TEXT NOT NULL DEFAULT '',
    balance         DECIMAL(20, 10) NOT NULL DEFAULT 0,
    unit            VARCHAR(16) NOT NULL DEFAULT 'USD',
    observed_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS upstream_balance_consumptions (
    id               BIGSERIAL PRIMARY KEY,
    account_id       BIGINT NOT NULL,
    group_id         BIGINT NOT NULL DEFAULT 0,
    account_name     TEXT NOT NULL DEFAULT '',
    group_name       TEXT NOT NULL DEFAULT '',
    previous_balance DECIMAL(20, 10) NOT NULL,
    current_balance  DECIMAL(20, 10) NOT NULL,
    amount           DECIMAL(20, 10) NOT NULL,
    unit             VARCHAR(16) NOT NULL DEFAULT 'USD',
    consumed_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_upstream_balance_consumptions_consumed_at
    ON upstream_balance_consumptions(consumed_at);

CREATE INDEX IF NOT EXISTS idx_upstream_balance_consumptions_account_consumed
    ON upstream_balance_consumptions(account_id, consumed_at);
