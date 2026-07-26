-- Support plans payable by balance and external checkout, plus immutable fixed-expiry order snapshots.
ALTER TABLE subscription_plans
    ADD COLUMN IF NOT EXISTS fixed_expires_at TIMESTAMPTZ;

ALTER TABLE subscription_plans
    DROP CONSTRAINT IF EXISTS subscription_plans_purchase_mode_check;

ALTER TABLE subscription_plans
    ADD CONSTRAINT subscription_plans_purchase_mode_check
    CHECK (purchase_mode IN ('external', 'balance', 'both'));

ALTER TABLE payment_orders
    ADD COLUMN IF NOT EXISTS subscription_expires_at TIMESTAMPTZ;
