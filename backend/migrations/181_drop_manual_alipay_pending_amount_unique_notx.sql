-- Clean up a temporary index that may remain after an interrupted historical migration.
DROP INDEX CONCURRENTLY IF EXISTS idx_payment_orders_manual_pending_pay_amount;