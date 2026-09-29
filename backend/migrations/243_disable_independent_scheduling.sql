-- Group-specific priorities are now consumed by the regular scheduler.
-- Keep the column for rolling-upgrade compatibility, but retire the persisted strict mode.
UPDATE groups
SET independent_scheduling = FALSE
WHERE independent_scheduling = TRUE;
