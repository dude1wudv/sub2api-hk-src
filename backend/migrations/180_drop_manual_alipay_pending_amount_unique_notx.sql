-- Manual Alipay orders are identified by their order number, not by a unique
-- payment amount. Multiple pending orders may therefore share the same price.
DROP INDEX CONCURRENTLY IF EXISTS idx_payment_orders_manual_pending_pay_amount;