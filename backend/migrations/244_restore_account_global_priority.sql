-- Retain retired group-scheduling columns and association values for compatibility.
-- Rebuild cached candidates: older versions may have stored a group's priority
-- in the account priority projection. The scheduler now uses accounts.priority.
INSERT INTO scheduler_outbox (event_type) VALUES ('full_rebuild');
