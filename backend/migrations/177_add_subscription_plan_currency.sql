-- Add an optional ISO-4217 display currency to subscription plans.
ALTER TABLE subscription_plans
    ADD COLUMN IF NOT EXISTS currency VARCHAR(3) NOT NULL DEFAULT '';