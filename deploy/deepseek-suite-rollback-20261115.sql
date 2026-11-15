-- Disable/restore enrollment only. Keep migrations, files, receipts and bills.
BEGIN;
SET LOCAL lock_timeout = '10s';
UPDATE accounts a SET extra=
  (COALESCE(a.extra,'{}'::jsonb)-'deepseek_tools_group_ids'-'deepseek_suite_capabilities'-'deepseek_suite_fallback_account_id')
  || CASE WHEN (b.original->>'has_scope')::boolean THEN jsonb_build_object('deepseek_tools_group_ids',b.original->'scope') ELSE '{}'::jsonb END
  || CASE WHEN (b.original->>'has_capabilities')::boolean THEN jsonb_build_object('deepseek_suite_capabilities',b.original->'capabilities') ELSE '{}'::jsonb END
  || CASE WHEN (b.original->>'has_fallback')::boolean THEN jsonb_build_object('deepseek_suite_fallback_account_id',b.original->'fallback') ELSE '{}'::jsonb END,
  updated_at=NOW()
FROM hk_deepseek_suite_20261115_backup b WHERE b.entity='account' AND b.entity_id=a.id;
UPDATE accounts a SET name=b.original->>'name',updated_at=NOW()
FROM hk_deepseek_suite_20261115_backup b
WHERE b.entity='account' AND b.entity_id=a.id AND a.id=882 AND b.original ? 'name';
DELETE FROM account_groups a USING hk_deepseek_suite_20261115_backup b
WHERE b.entity='official_membership' AND a.account_id=882 AND a.group_id=b.entity_id
  AND (b.original->>'existed')::boolean IS FALSE;
UPDATE groups g SET model_allowlist=b.original->'model_allowlist',updated_at=NOW()
FROM hk_deepseek_suite_20261115_backup b WHERE b.entity='group' AND b.entity_id=g.id;
INSERT INTO scheduler_outbox(event_type,account_id,payload)
SELECT 'account_changed',id,jsonb_build_object('group_ids',
  COALESCE((SELECT jsonb_agg(group_id) FROM account_groups WHERE account_id=a.id),'[]'::jsonb))
FROM accounts a WHERE id IN (882,1043,1181,1112,1182);
INSERT INTO scheduler_outbox(event_type,group_id)
SELECT 'group_changed',id FROM groups WHERE id IN (81,179,223);
SELECT enqueue_auth_cache_invalidation(key) FROM api_keys WHERE group_id IN (81,179,223) AND deleted_at IS NULL;
COMMIT;
