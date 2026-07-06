-- Token incentive plan: fixed 5-day token usage reward cycles.
CREATE TABLE IF NOT EXISTS token_incentive_claims (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    threshold_tokens BIGINT NOT NULL,
    reward_balance DECIMAL(20,8) NOT NULL,
    usage_tokens_at_claim BIGINT NOT NULL DEFAULT 0,
    claimed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, period_start, threshold_tokens)
);

CREATE INDEX IF NOT EXISTS idx_token_incentive_claims_user_period
    ON token_incentive_claims(user_id, period_start);

INSERT INTO settings (key, value, updated_at)
VALUES (
    'token_incentive_launch_at',
    to_char(NOW() AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
    NOW()
)
ON CONFLICT (key) DO NOTHING;