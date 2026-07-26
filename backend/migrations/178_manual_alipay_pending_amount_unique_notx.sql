-- One live manual Alipay order may occupy each exact CNY amount.  The index
-- makes the random tail allocation safe across concurrent application nodes.
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS idx_payment_orders_manual_pending_pay_amount
ON payment_orders (pay_amount)
WHERE provider_key = 'alipay_manual' AND status = 'PENDING';