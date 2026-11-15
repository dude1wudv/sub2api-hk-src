-- Retained day is inclusive. Preserve HK configured timezone and retention policy.
CREATE OR REPLACE FUNCTION invalidate_group_usage_rollup_state()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    affected_date DATE;
    published_before DATE;
    retained_date DATE;
    configured_timezone TEXT := current_setting('TimeZone');
BEGIN
    SELECT closed_before, (retained_from AT TIME ZONE configured_timezone)::date
    INTO published_before, retained_date
    FROM usage_group_rollup_state WHERE id = 1 FOR UPDATE;

    IF TG_OP = 'DELETE' THEN
        IF OLD.group_id IS NOT NULL THEN
            affected_date := (OLD.created_at AT TIME ZONE configured_timezone)::date;
        END IF;
    ELSE
        -- Either side may cross the retention boundary. Filter each side
        -- before taking MIN; an old row must not hide a retained new row.
        SELECT MIN(d) INTO affected_date
        FROM (VALUES
            (OLD.group_id, (OLD.created_at AT TIME ZONE configured_timezone)::date),
            (NEW.group_id, (NEW.created_at AT TIME ZONE configured_timezone)::date)
        ) AS changed(group_id, d)
        WHERE group_id IS NOT NULL AND d >= retained_date;
    END IF;

    IF published_before > affected_date AND affected_date >= retained_date THEN
        UPDATE usage_group_rollup_state
        SET closed_before = LEAST(closed_before, affected_date),
            updated_at = NOW()
        WHERE id = 1 AND affected_date >= (retained_from AT TIME ZONE configured_timezone)::date;
    END IF;

    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$;

CREATE OR REPLACE FUNCTION invalidate_group_usage_rollup_state_after_insert()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    affected_date DATE;
    published_before DATE;
    retained_date DATE;
    configured_timezone TEXT := current_setting('TimeZone');
BEGIN
    SELECT closed_before, (retained_from AT TIME ZONE configured_timezone)::date
    INTO published_before, retained_date
    FROM usage_group_rollup_state WHERE id = 1 FOR KEY SHARE;

    SELECT MIN((created_at AT TIME ZONE configured_timezone)::date)
    INTO affected_date FROM inserted_usage_logs
    WHERE group_id IS NOT NULL
      AND (created_at AT TIME ZONE configured_timezone)::date >= retained_date;
    IF affected_date IS NULL THEN
        RETURN NULL;
    END IF;

    IF published_before > affected_date AND affected_date >= retained_date THEN
        UPDATE usage_group_rollup_state
        SET closed_before = LEAST(closed_before, affected_date),
            updated_at = NOW()
        WHERE id = 1 AND affected_date >= (retained_from AT TIME ZONE configured_timezone)::date;
    END IF;

    RETURN NULL;
END;
$$;
