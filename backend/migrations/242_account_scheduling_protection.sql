ALTER TABLE groups ADD COLUMN IF NOT EXISTS slow_ttft_exempt_until timestamptz;
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS slow_ttft_until timestamptz;
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS slow_ttft_reason text NOT NULL DEFAULT '';
ALTER TABLE groups ADD COLUMN IF NOT EXISTS independent_scheduling boolean NOT NULL DEFAULT false;
ALTER TABLE groups ADD COLUMN IF NOT EXISTS scheduling_initialized boolean NOT NULL DEFAULT false;

-- Epoch and pause ownership apply equally to single edits, imports and bulk JSONB updates.
CREATE OR REPLACE FUNCTION account_slow_ttft_config_changed() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF TG_OP = 'INSERT' OR NEW.extra->'slow_ttft_protection' IS DISTINCT FROM OLD.extra->'slow_ttft_protection' THEN
    IF jsonb_typeof(NEW.extra->'slow_ttft_protection') = 'object' THEN
      NEW.extra := jsonb_set(NEW.extra, '{slow_ttft_protection,generation}', to_jsonb(md5(random()::text || clock_timestamp()::text)));
    END IF;
    IF COALESCE((NEW.extra->'slow_ttft_protection'->>'enabled')::boolean, false) = false THEN
      NEW.slow_ttft_until := NULL;
      NEW.slow_ttft_reason := '';
    END IF;
  END IF;
  RETURN NEW;
END $$;
CREATE TRIGGER account_slow_ttft_config_changed BEFORE INSERT OR UPDATE OF extra ON accounts
FOR EACH ROW EXECUTE FUNCTION account_slow_ttft_config_changed();
