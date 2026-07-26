-- Add an opt-in one-time purchase policy and serialize purchases per user/group.
ALTER TABLE subscription_plans
    ADD COLUMN IF NOT EXISTS one_purchase_per_user BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE IF NOT EXISTS subscription_purchase_claims (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    subscription_group_id BIGINT NOT NULL,
    payment_order_id BIGINT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT subscription_purchase_claims_user_group_key
        UNIQUE (user_id, subscription_group_id),
    CONSTRAINT subscription_purchase_claims_order_key
        UNIQUE (payment_order_id)
);

CREATE INDEX IF NOT EXISTS subscription_purchase_claims_status
    ON subscription_purchase_claims (status);

CREATE INDEX IF NOT EXISTS payment_orders_subscription_purchase_history
    ON payment_orders (user_id, subscription_group_id, paid_at)
    WHERE order_type = 'subscription' AND paid_at IS NOT NULL;