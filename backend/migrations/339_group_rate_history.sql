-- Launch is explicit: old groups and bulk imports remain unlaunched unless an
-- administrator supplies launched_at. Ordinary edits never restart the NEW badge.
ALTER TABLE groups ADD COLUMN IF NOT EXISTS launched_at TIMESTAMPTZ;
ALTER TABLE groups ADD COLUMN IF NOT EXISTS launch_rate NUMERIC(10,4);

CREATE TABLE IF NOT EXISTS group_rate_history (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    old_rate_multiplier NUMERIC(10,4) NOT NULL,
    new_rate_multiplier NUMERIC(10,4) NOT NULL,
    changed_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);
CREATE INDEX IF NOT EXISTS idx_group_rate_history_group_changed
    ON group_rate_history (group_id, changed_at DESC, id DESC);

CREATE OR REPLACE FUNCTION snapshot_group_launch_rate() RETURNS TRIGGER AS $$
BEGIN
    IF NEW.launched_at IS NULL THEN
        NEW.launch_rate := NULL;
    ELSIF TG_OP = 'INSERT' THEN
        NEW.launch_rate := NEW.rate_multiplier;
    ELSIF NEW.launched_at IS DISTINCT FROM OLD.launched_at THEN
        NEW.launch_rate := NEW.rate_multiplier;
    ELSE
        NEW.launch_rate := OLD.launch_rate;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_group_launch_rate ON groups;
CREATE TRIGGER trg_group_launch_rate
    BEFORE INSERT OR UPDATE ON groups
    FOR EACH ROW EXECUTE FUNCTION snapshot_group_launch_rate();

CREATE OR REPLACE FUNCTION record_group_rate_change() RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO group_rate_history (group_id, old_rate_multiplier, new_rate_multiplier)
    VALUES (NEW.id, OLD.rate_multiplier, NEW.rate_multiplier);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_group_rate_history ON groups;
CREATE TRIGGER trg_group_rate_history
    AFTER UPDATE ON groups
    FOR EACH ROW WHEN (OLD.rate_multiplier IS DISTINCT FROM NEW.rate_multiplier)
    EXECUTE FUNCTION record_group_rate_change();
