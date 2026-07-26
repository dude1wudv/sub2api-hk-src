-- Keep local OpenAI test accounts direct forever.
-- Any create/update/import/manual SQL write that names an account oto/oto2
-- must clear proxy_id before the row is stored.

CREATE OR REPLACE FUNCTION enforce_no_proxy_oto_accounts()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.name IS NOT NULL AND lower(btrim(NEW.name)) IN ('oto', 'oto2') THEN
        NEW.proxy_id := NULL;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_enforce_no_proxy_oto_accounts ON accounts;

CREATE TRIGGER trg_enforce_no_proxy_oto_accounts
BEFORE INSERT OR UPDATE OF name, proxy_id ON accounts
FOR EACH ROW
EXECUTE FUNCTION enforce_no_proxy_oto_accounts();

UPDATE accounts
SET proxy_id = NULL,
    updated_at = NOW()
WHERE deleted_at IS NULL
  AND lower(btrim(name)) IN ('oto', 'oto2')
  AND proxy_id IS NOT NULL;

INSERT INTO scheduler_outbox (event_type, payload)
VALUES ('full_rebuild', '{"reason":"no_proxy_oto_accounts"}'::jsonb);
