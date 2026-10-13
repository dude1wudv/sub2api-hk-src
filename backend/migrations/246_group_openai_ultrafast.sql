ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS force_openai_ultrafast BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS free_openai_ultrafast BOOLEAN NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN groups.force_openai_fast IS
    'Force OpenAI gateway requests to priority, overriding global tier policy unless this group forces ultrafast';
COMMENT ON COLUMN groups.force_openai_ultrafast IS
    'Force OpenAI gateway requests to ultrafast, overriding group Fast and global tier policy';
COMMENT ON COLUMN groups.free_openai_ultrafast IS
    'Bill Ultrafast requests in this OpenAI/Composite group to users at Standard price';
