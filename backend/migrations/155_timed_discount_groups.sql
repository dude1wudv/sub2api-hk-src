-- 155_timed_discount_groups.sql
-- Add optional daily time window for timed discount groups.
ALTER TABLE groups
  ADD COLUMN IF NOT EXISTS timed_discount_enabled BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS timed_discount_start_minute INTEGER NOT NULL DEFAULT 30,
  ADD COLUMN IF NOT EXISTS timed_discount_end_minute INTEGER NOT NULL DEFAULT 450;

ALTER TABLE groups
  DROP CONSTRAINT IF EXISTS groups_timed_discount_start_minute_range,
  DROP CONSTRAINT IF EXISTS groups_timed_discount_end_minute_range,
  DROP CONSTRAINT IF EXISTS groups_timed_discount_window_nonempty;

ALTER TABLE groups
  ADD CONSTRAINT groups_timed_discount_start_minute_range
    CHECK (timed_discount_start_minute >= 0 AND timed_discount_start_minute < 1440),
  ADD CONSTRAINT groups_timed_discount_end_minute_range
    CHECK (timed_discount_end_minute >= 0 AND timed_discount_end_minute < 1440),
  ADD CONSTRAINT groups_timed_discount_window_nonempty
    CHECK (timed_discount_start_minute <> timed_discount_end_minute);