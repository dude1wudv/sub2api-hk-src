-- Disable/restore enrollment only. Keep migrations, files, receipts and bills.
BEGIN;
SET LOCAL lock_timeout = '10s';
UPDATE accounts a SET extra=
  (COALESCE(a.extra,'{}'::jsonb)-'deepseek_tools_group_ids'-'deepseek_suite_capabilities')
  || CASE WHEN (b.original->>'has_scope')::boolean THEN jsonb_build_object('deepseek_tools_group_ids',b.original->'scope') ELSE '{}'::jsonb END
  || CASE WHEN (b.original->>'has_capabilities')::boolean THEN jsonb_build_object('deepseek_suite_capabilities',b.original->'capabilities') ELSE '{}'::jsonb END,
  updated_at=NOW()
FROM hk_deepseek_suite_20261115_backup b WHERE b.entity='account' AND b.entity_id=a.id;
UPDATE groups g SET model_allowlist=b.original->'model_allowlist',updated_at=NOW()
FROM hk_deepseek_suite_20261115_backup b WHERE b.entity='group' AND b.entity_id=g.id;
INSERT INTO scheduler_outbox(event_type,account_id,payload)
SELECT 'account_changed',id,jsonb_build_object('group_ids',
  COALESCE((SELECT jsonb_agg(group_id) FROM account_groups WHERE account_id=a.id),'[]'::jsonb))
FROM accounts a WHERE id IN (1043,1181,1112,1182);
INSERT INTO scheduler_outbox(event_type,group_id)
SELECT 'group_changed',id FROM groups WHERE id IN (81,179,223);
SELECT enqueue_auth_cache_invalidation(key) FROM api_keys WHERE group_id IN (81,179,223) AND deleted_at IS NULL;
COMMIT;
