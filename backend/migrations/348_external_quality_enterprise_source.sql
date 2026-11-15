-- Keep the original source IDs on retained history; new Plus imports use 142.
-- 170 remains valid only for existing records, with new imports gated by service validation.
ALTER TABLE external_quality_history
    DROP CONSTRAINT IF EXISTS external_quality_history_binding;
ALTER TABLE external_quality_history
    ADD CONSTRAINT external_quality_history_binding CHECK (
        (group_id = 72 AND remote_group_id = 62) OR
        (group_id = 237 AND remote_group_id IN (170, 142))
    );
