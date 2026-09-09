CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_logs_user_session_time
    ON usage_logs (user_id, session_id, created_at DESC, id DESC)
    WHERE session_id IS NOT NULL;
