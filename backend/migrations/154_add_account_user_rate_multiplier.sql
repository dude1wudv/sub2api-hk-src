-- Add account-level user billing multiplier.
-- This stacks on top of the effective group/user-group multiplier for user billing.
ALTER TABLE accounts
    ADD COLUMN IF NOT EXISTS user_rate_multiplier DECIMAL(10,4) NOT NULL DEFAULT 1.0;

COMMENT ON COLUMN accounts.user_rate_multiplier IS '账号用户计费倍率；用户扣费倍率 = 有效分组倍率 * 此倍率';
