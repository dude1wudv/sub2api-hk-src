-- Add balance-purchasable subscription plan metadata and a server-enforced sale cutoff.
ALTER TABLE subscription_plans
    ADD COLUMN IF NOT EXISTS purchase_mode VARCHAR(20) NOT NULL DEFAULT 'external';

ALTER TABLE subscription_plans
    ADD COLUMN IF NOT EXISTS sale_ends_at TIMESTAMPTZ;

ALTER TABLE subscription_plans
    DROP CONSTRAINT IF EXISTS subscription_plans_purchase_mode_check;

ALTER TABLE subscription_plans
    ADD CONSTRAINT subscription_plans_purchase_mode_check
    CHECK (purchase_mode IN ('external', 'balance'));

CREATE INDEX IF NOT EXISTS idx_subscription_plans_active_sale_window
    ON subscription_plans (sale_ends_at)
    WHERE for_sale = TRUE AND sale_ends_at IS NOT NULL;