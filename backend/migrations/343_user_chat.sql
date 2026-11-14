-- Add per-preset reasoning profiles without rewriting existing chat history.
ALTER TABLE chat_model_presets
    ADD COLUMN normal_reasoning_effort VARCHAR(6) NOT NULL DEFAULT 'low',
    ADD COLUMN thinking_reasoning_effort VARCHAR(6) NOT NULL DEFAULT 'high',
    ADD CONSTRAINT chat_presets_normal_reasoning_effort_check
        CHECK (normal_reasoning_effort IN ('low', 'medium', 'high')),
    ADD CONSTRAINT chat_presets_thinking_reasoning_effort_check
        CHECK (thinking_reasoning_effort IN ('low', 'medium', 'high'));
