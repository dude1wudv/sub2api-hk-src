-- Historical 159_usage_sessions.sql stored usage_logs.session_id as a BIGINT FK.
-- Upstream runtime persists the client-provided session identifier as text. Keep
-- old numeric references readable as their decimal text representation while
-- removing the obsolete runtime dependency on usage_sessions.
DROP INDEX IF EXISTS idx_usage_logs_session_id;
ALTER TABLE usage_logs DROP CONSTRAINT IF EXISTS usage_logs_session_id_fkey;
ALTER TABLE usage_logs
    ALTER COLUMN session_id TYPE VARCHAR(255)
    USING session_id::text;
CREATE INDEX IF NOT EXISTS idx_usage_logs_session_id
    ON usage_logs (session_id)
    WHERE session_id IS NOT NULL;