-- Converge both paths: existing HK 345 -> upstream 251 -> 349, and fresh replay 251 -> 345 -> 349.
-- Historical SQL/checksums remain unchanged. Keep HK's inclusive retained day and the
-- exact expired_before timestamp, plus upstream's nonblocking build dirty range.
SET LOCAL lock_timeout = '5s';

-- Old HK only recorded an inclusive retained day. Seed the exact floor at that
-- day's start (never the first record's timestamp); do not exclude earlier rows
-- within the retained day. Fresh replay starts at the epoch and is unchanged.
UPDATE usage_group_rollup_state
SET expired_before = GREATEST(expired_before,
        (retained_from AT TIME ZONE timezone_name)::date::timestamp AT TIME ZONE timezone_name)
WHERE id = 1;

-- 按受影响日期后退发布水位，并在重建进行中记录脏日期；水位与脏日期都已不晚于该日期时不写状态行。
-- 调用方必须已对状态行持有 FOR KEY SHARE 或更强的行锁：重建的登记与发布持有 FOR UPDATE，
-- 由此保证每次源记录变更要么先于登记提交、被重建快照看到，要么看到登记并记入脏日期。
CREATE OR REPLACE FUNCTION apply_group_usage_rollup_invalidation(affected_date DATE)
RETURNS VOID
LANGUAGE plpgsql
AS $$
BEGIN
    UPDATE usage_group_rollup_state
    SET closed_before = LEAST(closed_before, affected_date),
        build_dirty_from = CASE
            WHEN building_until > affected_date THEN LEAST(build_dirty_from, affected_date)
            ELSE build_dirty_from
        END,
        updated_at = NOW()
    WHERE id = 1
        AND (
            closed_before > affected_date
            OR (
                building_until > affected_date
                AND (build_dirty_from IS NULL OR build_dirty_from > affected_date)
            )
        );
END;
$$;

-- 供应用侧调用：使 affected_at 在 bucket_timezone 中的日期及之后的日桶失效；
-- 早于 expired_before 的部分不属于任何可读日桶，失效从 expired_before 起算。
CREATE OR REPLACE FUNCTION invalidate_group_usage_rollups_at(affected_at TIMESTAMPTZ, bucket_timezone TEXT)
RETURNS VOID
LANGUAGE plpgsql
AS $$
DECLARE
    state_expired_before TIMESTAMPTZ;
    state_retained_date DATE;
BEGIN
    SELECT expired_before, (retained_from AT TIME ZONE bucket_timezone)::date
    INTO state_expired_before, state_retained_date
    FROM usage_group_rollup_state
    WHERE id = 1
    FOR KEY SHARE;

    IF NOT FOUND OR affected_at IS NULL THEN
        RETURN;
    END IF;

    PERFORM apply_group_usage_rollup_invalidation(
        GREATEST((GREATEST(affected_at, state_expired_before) AT TIME ZONE bucket_timezone)::date, state_retained_date)
    );
END;
$$;

-- DELETE/UPDATE 行级触发器。FOR KEY SHARE 与 usage_logs 写入及其他失效互相兼容，只与重建的登记、发布串行。
CREATE OR REPLACE FUNCTION invalidate_group_usage_rollup_state()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    configured_timezone TEXT := current_setting('TimeZone');
    old_date DATE;
    new_date DATE;
    affected_date DATE;
    state_closed_before DATE;
    state_building_until DATE;
    state_build_dirty_from DATE;
    state_expired_before TIMESTAMPTZ;
    state_retained_date DATE;
BEGIN
    SELECT closed_before, building_until, build_dirty_from, expired_before,
           (retained_from AT TIME ZONE configured_timezone)::date
    INTO state_closed_before, state_building_until, state_build_dirty_from, state_expired_before, state_retained_date
    FROM usage_group_rollup_state
    WHERE id = 1
    FOR KEY SHARE;

    IF FOUND THEN
        IF OLD.group_id IS NOT NULL AND OLD.created_at >= state_expired_before
            AND (OLD.created_at AT TIME ZONE configured_timezone)::date >= state_retained_date THEN
            old_date := (OLD.created_at AT TIME ZONE configured_timezone)::date;
        END IF;
        IF TG_OP = 'UPDATE' THEN
            IF NEW.group_id IS NOT NULL AND NEW.created_at >= state_expired_before
                AND (NEW.created_at AT TIME ZONE configured_timezone)::date >= state_retained_date THEN
                new_date := (NEW.created_at AT TIME ZONE configured_timezone)::date;
            END IF;
        END IF;

        affected_date := LEAST(old_date, new_date);
        IF state_closed_before > affected_date
            OR (
                state_building_until > affected_date
                AND (state_build_dirty_from IS NULL OR state_build_dirty_from > affected_date)
            ) THEN
            PERFORM apply_group_usage_rollup_invalidation(affected_date);
        END IF;
    END IF;

    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$;

-- INSERT 语句级触发器：每个批量 INSERT 只读一次状态行；当天写入既不后退水位也不记脏。
CREATE OR REPLACE FUNCTION invalidate_group_usage_rollup_state_after_insert()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    configured_timezone TEXT := current_setting('TimeZone');
    affected_date DATE;
    state_closed_before DATE;
    state_building_until DATE;
    state_build_dirty_from DATE;
    state_expired_before TIMESTAMPTZ;
    state_retained_date DATE;
BEGIN
    IF NOT EXISTS (SELECT 1 FROM inserted_usage_logs WHERE group_id IS NOT NULL) THEN
        RETURN NULL;
    END IF;

    SELECT closed_before, building_until, build_dirty_from, expired_before,
           (retained_from AT TIME ZONE configured_timezone)::date
    INTO state_closed_before, state_building_until, state_build_dirty_from, state_expired_before, state_retained_date
    FROM usage_group_rollup_state
    WHERE id = 1
    FOR KEY SHARE;

    IF NOT FOUND THEN
        RETURN NULL;
    END IF;

    SELECT MIN((created_at AT TIME ZONE configured_timezone)::date)
    INTO affected_date
    FROM inserted_usage_logs
    WHERE group_id IS NOT NULL
        AND created_at >= state_expired_before
        AND (created_at AT TIME ZONE configured_timezone)::date >= state_retained_date;

    IF state_closed_before > affected_date
        OR (
            state_building_until > affected_date
            AND (state_build_dirty_from IS NULL OR state_build_dirty_from > affected_date)
        ) THEN
        PERFORM apply_group_usage_rollup_invalidation(affected_date);
    END IF;

    RETURN NULL;
END;
$$;
