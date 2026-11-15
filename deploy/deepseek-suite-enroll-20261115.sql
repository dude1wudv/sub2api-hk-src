-- Run on the server only, AFTER deploying the reviewed application commit.
-- This file is not a startup migration and must not be applied automatically.
BEGIN;
SET LOCAL lock_timeout = '10s';

DO $$
BEGIN
  IF (SELECT COUNT(*) FROM accounts WHERE id IN (1043,1181,1112,1182) AND deleted_at IS NULL
      AND type='apikey' AND name LIKE '%全家桶%' AND (extra IS NULL OR jsonb_typeof(extra)='object')
      AND ((id IN (1043,1181) AND platform='deepseek') OR (id IN (1112,1182) AND platform='openai'))
      AND ((id IN (1043,1112) AND credentials->>'base_url' ~ '^https://kedaya[.]ai(/|$)')
        OR (id IN (1181,1182) AND credentials->>'base_url' ~ '^https://vip[.]mdkj[.]lol(/|$)'))) <> 4 THEN
    RAISE EXCEPTION 'Enrollment identity/platform/label/provider preflight failed; inspect current configuration';
  END IF;
  IF EXISTS (
    SELECT 1 FROM (VALUES(1043,179),(1181,179),(1112,81),(1112,223),(1182,81),(1182,223)) AS required(account_id,group_id)
    WHERE NOT EXISTS (SELECT 1 FROM account_groups a WHERE a.account_id=required.account_id AND a.group_id=required.group_id)
  ) THEN
    RAISE EXCEPTION 'Enrollment group membership changed';
  END IF;
  IF (SELECT COUNT(*) FROM groups WHERE id IN (81,179,223) AND deleted_at IS NULL
      AND ((id=179 AND platform='deepseek') OR (id IN (81,223) AND platform='openai'))) <> 3 THEN
    RAISE EXCEPTION 'Enrollment group platform changed';
  END IF;
END $$;

-- Backup only the fields this rollout owns, without provider/API credentials.
CREATE TABLE IF NOT EXISTS hk_deepseek_suite_20261115_backup (
  entity TEXT NOT NULL,
  entity_id BIGINT NOT NULL,
  original JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY(entity,entity_id)
);
INSERT INTO hk_deepseek_suite_20261115_backup(entity,entity_id,original)
SELECT 'account',id,jsonb_build_object(
  'has_scope',COALESCE(extra,'{}'::jsonb) ? 'deepseek_tools_group_ids',
  'scope',extra->'deepseek_tools_group_ids',
  'has_capabilities',COALESCE(extra,'{}'::jsonb) ? 'deepseek_suite_capabilities',
  'capabilities',extra->'deepseek_suite_capabilities')
FROM accounts WHERE id IN (1043,1181,1112,1182)
ON CONFLICT DO NOTHING;
INSERT INTO hk_deepseek_suite_20261115_backup(entity,entity_id,original)
SELECT 'group',id,jsonb_build_object('model_allowlist',model_allowlist)
FROM groups WHERE id IN (81,179,223)
ON CONFLICT DO NOTHING;

UPDATE accounts SET extra=COALESCE(extra,'{}'::jsonb)||jsonb_build_object(
  'deepseek_tools_group_ids',CASE WHEN id IN (1043,1181) THEN '[179]'::jsonb ELSE '[81,223]'::jsonb END,
  'deepseek_suite_capabilities','["search","fetch","ocr","asr","tts","tts_ws","files","chat","responses","messages"]'::jsonb),
  updated_at=NOW()
WHERE id IN (1043,1181,1112,1182);

-- Preserve the allowlist's enabled state and existing models. Native tools and
-- Files are model-free; only OpenAI audio needs two discoverable model aliases.
UPDATE groups g SET model_allowlist=jsonb_set(COALESCE(g.model_allowlist,'{}'::jsonb),'{models}',
  (SELECT jsonb_agg(model ORDER BY model) FROM (
     SELECT DISTINCT jsonb_array_elements_text(CASE WHEN jsonb_typeof(g.model_allowlist->'models')='array' THEN g.model_allowlist->'models' ELSE '[]'::jsonb END||'["deepseek-asr","deepseek-tts"]'::jsonb) AS model
   ) m)),updated_at=NOW()
WHERE id IN (81,179,223);

-- Use the existing durable invalidation paths; never flush shared Redis.
INSERT INTO scheduler_outbox(event_type,account_id,payload)
SELECT 'account_changed',id,jsonb_build_object('group_ids',
  COALESCE((SELECT jsonb_agg(group_id) FROM account_groups WHERE account_id=a.id),'[]'::jsonb))
FROM accounts a WHERE id IN (1043,1181,1112,1182);
INSERT INTO scheduler_outbox(event_type,group_id)
SELECT 'group_changed',id FROM groups WHERE id IN (81,179,223);
SELECT enqueue_auth_cache_invalidation(key) FROM api_keys WHERE group_id IN (81,179,223) AND deleted_at IS NULL;
COMMIT;
