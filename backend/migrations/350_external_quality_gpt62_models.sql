ALTER TABLE external_quality_history
    DROP CONSTRAINT IF EXISTS external_quality_history_model,
    ADD CONSTRAINT external_quality_history_model CHECK (
        model_id IN ('gpt-6-astra', 'gpt-6.1-sol', 'gpt-6.2-astra', 'gpt-6.2-sol')
    );
