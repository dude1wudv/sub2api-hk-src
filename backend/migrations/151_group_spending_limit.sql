-- Add per-group cumulative spending limits.

ALTER TABLE groups
  ADD COLUMN IF NOT EXISTS spending_limit_usd DECIMAL(20,8),
  ADD COLUMN IF NOT EXISTS spending_used_usd DECIMAL(20,8) NOT NULL DEFAULT 0;

-- Initialize historical usage from recorded user-billed cost so newly configured
-- limits reflect existing group consumption.
UPDATE groups g
SET spending_used_usd = COALESCE(t.total, 0)
FROM (
  SELECT group_id, SUM(actual_cost) AS total
  FROM usage_logs
  WHERE group_id IS NOT NULL
  GROUP BY group_id
) t
WHERE g.id = t.group_id;
