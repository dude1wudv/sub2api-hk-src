// All records are invented. There are no upstream credentials or production calls.
import { buildTrafficReport } from './user-preview-fixtures.mjs'
const now = '2026-10-10T12:00:00Z'
const groups = [
  { id: 1, name: 'Claude Standard · Preview', platform: 'anthropic', status: 'active', rate_multiplier: 1, subscription_type: 'standard', is_exclusive: false, account_count: 12, description: '合成标准分组', model_pricing: [], created_at: now, updated_at: now },
  { id: 2, name: 'OpenAI Pro · Preview', platform: 'openai', status: 'active', rate_multiplier: 1.5, subscription_type: 'standard', is_exclusive: false, account_count: 12, description: '合成高级分组', model_pricing: [], created_at: now, updated_at: now }
]
// Documentation-only IP ranges; location data is invented to verify that labels
// follow returned metadata even when a proxy's saved name says something else.
const proxies = [
  { id: 1, name: '旧东京节点', host: '192.0.2.10', ip_address: '203.0.113.10', country: '新加坡', country_code: 'SG', city: 'Singapore' },
  { id: 2, name: '备用节点', host: '192.0.2.20', ip_address: '198.51.100.20', country: '美国', country_code: 'US', region: 'California', city: 'Los Angeles' },
  { id: 3, name: '未探测代理', host: '192.0.2.30' },
].map(proxy => ({ ...proxy, protocol: 'socks5', port: 1080, username: null, status: 'active', account_count: 6, expires_at: null, fallback_mode: 'none', expiry_warn_days: 7, created_at: now, updated_at: now }))
const accounts = Array.from({length: 24}, (_, i) => ({
  id: 1024 + i, name: `${['Claude','OpenAI','Gemini'][i%3]}-${String(i+1).padStart(2,'0')}`,
  platform: ['anthropic','openai','gemini'][i%3], type: 'apikey', status: i===1 ? 'error' : 'active',
  schedulable: i!==1, concurrency: 10, current_concurrency: i%4, priority: 50, rate_multiplier: 1,
  proxy_id: i%4 === 3 ? null : proxies[i%4].id, group_ids: [i%2+1], groups: [groups[i%2]], notes: '独立合成数据，仅供界面验收',
  credentials: { base_url: 'https://example.test/v1' }, credentials_status: { has_api_key: true },
  created_at: now, updated_at: now, last_used_at: now, expires_at: null, auto_pause_on_expired: false,
  error_message: i===1 ? '合成示例：上游限流' : null,
  rate_limited_at: null, rate_limit_reset_at: null, overload_until: null, temp_unschedulable_until: null,
  temp_unschedulable_reason: null, session_window_start: null, session_window_end: null, session_window_status: null,
  quota_limit: 100, quota_used: 12.8+i, quota_daily_limit: 20, quota_daily_used: 3.28+i/10,
  scheduler_scores: [{group_id:i%2+1, group_name:groups[i%2].name, base_score:72.5}]
}))
const page = (items, params) => { const size=Number(params.get('page_size')||20), n=Number(params.get('page')||1); return {items:items.slice((n-1)*size,n*size),total:items.length,page:n,page_size:size,pages:Math.ceil(items.length/size)} }
const sortAccounts = (items, params) => {
  const requested = params.get('sort_by') || 'name'
  const keys = requested === 'platform_type' ? ['platform', 'type', 'id'] : [requested, 'id']
  const direction = params.get('sort_order') === 'desc' ? -1 : 1
  return [...items].sort((left, right) => {
    for (const key of keys) {
      const a = left[key], b = right[key]
      const comparison = typeof a === 'number' && typeof b === 'number'
        ? a - b
        : String(a ?? '').localeCompare(String(b ?? ''), 'en', { numeric: true })
      if (comparison) return comparison * direction
    }
    return 0
  })
}
// Read-only synthetic metadata; action endpoints deliberately remain unavailable.
function supplementaryAdminPreview(path, params, fixtures, adminGroups) {
  const R = {};
  const S = '/admin/settings', A = '/admin/accounts';
  const B = '/admin/backups', F = '/admin/affiliates';
  const P = '/admin/payment', O = '/admin/ops';
  const t = new Date().toISOString();
  const u = fixtures.users[101], admin = fixtures.users[201];

  Object.assign(R, {
    [S + '/web-search-emulation']: {
      enabled: false, providers: []
    },
    [A + '/upstream-billing-probe/settings']: {
      enabled: false, interval_minutes: 30
    },
    [A + '/ollama-cloud-usage/settings']: {
      enabled: false, interval_minutes: 60, debounce_minutes: 1
    },
    [A + '/opencode-go-usage/settings']: {
      enabled: false, interval_minutes: 60, debounce_minutes: 1
    },
    [S + '/admin-api-key']: {
      exists: false, masked_key: ''
    },
    [S + '/overload-cooldown']: {
      enabled: true, cooldown_minutes: 10
    },
    [S + '/rate-limit-429-cooldown']: {
      enabled: true, cooldown_seconds: 5
    },
    [S + '/panel-rate-limit']: {
      enabled: true, user_rpm: 240, heavy_rpm: 60,
      exempt_admin: true, public_ip_rpm: 300
    },
    [S + '/stream-timeout']: {
      enabled: true, action: 'temp_unsched', temp_unsched_minutes: 5,
      threshold_count: 3, threshold_window_minutes: 10
    },
    [S + '/rectifier']: {
      enabled: true, thinking_signature_enabled: true,
      thinking_budget_enabled: true, apikey_signature_enabled: false,
      apikey_signature_patterns: []
    },
    [S + '/beta-policy']: { rules: [] },
    [S + '/registration-email-blacklist']: {
      emails: ['blocked-preview@example.test'], max: 20000
    },
    [S + '/admin-api-keys']: {
      items: [{
        id: 'preview-admin-key-1', name: 'Synthetic preview disabled key',
        description: 'Local synthetic record; no usable credential',
        enabled: false, expires_at: null, allowed_ips: [],
        policy: { full_access: false, rules: [], global_permissions: [] },
        key_prefix: 'sk-preview-admin', created_by: admin.id,
        created_at: t, updated_at: t, last_used_at: null,
        last_used_ip: '', revoked_at: null
      }],
      permissions: [{
        id: 'accounts.read', label: '读取账户',
        category: '账户', scope: 'account'
      }]
    },
    [S + '/email-templates']: {
      events: ['auth.verify_code'], locales: ['zh-CN'],
      templates: [{
        event: 'auth.verify_code', locale: 'zh-CN',
        subject: 'Synthetic preview verification', is_custom: false,
        updated_at: t
      }],
      placeholders: ['{{verification_code}}']
    },
    // 列表加载后自动读取此详情，需一并提供。
    [S + '/email-templates/auth.verify_code/zh-CN']: {
      event: 'auth.verify_code', locale: 'zh-CN',
      subject: 'Synthetic preview verification',
      html: '<p>Local synthetic preview only: {{verification_code}}</p>',
      is_custom: false, updated_at: t,
      placeholders: ['{{verification_code}}']
    }
  });

  const s3 = {
    endpoint: 'https://storage.example.test', region: 'us-east-1',
    bucket: 'synthetic-preview-backups', access_key_id: '',
    prefix: 'preview/backups/', force_path_style: true
  };
  const backup = {
    id: 'preview-backup-1', status: 'completed', backup_type: 'postgres',
    file_name: 'preview-backup-1.sql.gz',
    s3_key: 'preview/backups/preview-backup-1.sql.gz',
    size_bytes: 1024, triggered_by: 'manual',
    started_at: new Date(Date.parse(t) - 60000).toISOString(),
    finished_at: t
  };
  const protocol = { plugin_protocol: 1, transport_api: 1, ui_bridge: 1 };
  const pluginName = 'Synthetic preview disabled plugin';

  Object.assign(R, {
    [B + '/s3-config']: s3,
    [B + '/image-storage']: {
      config: {
        enabled: false, reuse_backup_s3: true,
        bucket: 'synthetic-preview-images', prefix: 'preview/images/',
        public_base_url: '', presign_expiry_hours: 24,
        max_download_bytes: 10485760, endpoint: '', region: '',
        access_key_id: '', force_path_style: true
      },
      secret_configured: false
    },
    [B + '/schedule']: {
      enabled: false, cron_expr: '0 4 * * *',
      retain_days: 14, retain_count: 10,
      monthly_archive: {
        enabled: false, days: [1, 15],
        include_month_end: false, retain_count: 12
      }
    },
    [B]: { items: [backup] },
    [B + '/preview-backup-1']: backup,
    '/admin/plugins': [{
      id: 7, plugin_key: 'local.preview.disabled',
      name: pluginName, version: '1.0.0',
      description: 'Synthetic metadata only; no binary installed or executed',
      author: 'Synthetic preview',
      manifest: {
        schema_version: 1, id: 'local.preview.disabled',
        name: pluginName, version: '1.0.0',
        requires: { sub2api: '>=0.1.0', ...protocol },
        capabilities: [], ui: { entrypoint: '' }
      },
      binary_sha256: '0'.repeat(64), signature_status: 'unsigned',
      state: 'disabled', last_error: '',
      installed_at: t, updated_at: t, bindings: [],
      compatibility: {
        compatible: true, tested: false, status: 'untested',
        message: 'Synthetic compatibility metadata; never executed',
        current_sub2api_version: '0.1.0',
        required_sub2api_version: '>=0.1.0',
        recommended_sub2api_version: '0.1.0', ...protocol
      },
      runtime_healthy: false, runtime_message: 'Not running; synthetic preview'
    }]
  });

  const paidOrder = fixtures.orders.find(x => x.id === 601);
  const pair = {
    inviter_id: admin.id, inviter_email: admin.email,
    inviter_username: admin.username,
    invitee_id: u.id, invitee_email: u.email, invitee_username: u.username
  };
  const rebate = paidOrder.pay_amount * 0.05;

  Object.assign(R, {
    [F + '/users']: page([{
      user_id: admin.id, email: admin.email, username: admin.username,
      aff_code: 'PREVIEW-ADMIN-INVITE', aff_code_custom: true,
      aff_rebate_rate_percent: 5, aff_count: 1
    }], params),
    [F + '/invites']: page([{
      ...pair, aff_code: 'PREVIEW-ADMIN-INVITE',
      total_rebate: rebate, created_at: u.created_at
    }], params),
    [F + '/rebates']: page([{
      ...pair, order_id: paidOrder.id, out_trade_no: paidOrder.out_trade_no,
      order_amount: paidOrder.amount, pay_amount: paidOrder.pay_amount,
      rebate_amount: rebate, payment_type: paidOrder.payment_type,
      order_status: paidOrder.status, created_at: paidOrder.completed_at
    }], params),
    [F + '/transfers']: page([{
      ledger_id: 901, action: 'transfer', user_id: admin.id,
      user_email: admin.email, username: admin.username,
      amount: rebate, balance_after: admin.balance,
      available_quota_after: 0, frozen_quota_after: 0,
      history_quota_after: rebate, snapshot_available: true, created_at: t
    }], params),
    [P + '/orders']: page(fixtures.orders, params),
    [P + '/plans']: fixtures.plans,
    [P + '/dashboard']: {
      today_amount: { USD: 0 }, total_amount: { USD: paidOrder.pay_amount },
      today_count: 0, total_count: 1,
      avg_amount: { USD: paidOrder.pay_amount },
      daily_series: [{
        date: paidOrder.completed_at.slice(0, 10),
        amount: { USD: paidOrder.pay_amount }, count: 1
      }],
      payment_methods: [{
        type: paidOrder.payment_type,
        amount: { USD: paidOrder.pay_amount }, count: 1
      }],
      top_users: {
        USD: [{ user_id: u.id, email: u.email, amount: paidOrder.pay_amount }]
      }
    }
  });
  // 订单详情需要的只读 GET；返回 API 声明的 PaymentOrder。
  for (const order of fixtures.orders) R[P + '/orders/' + order.id] = order;

  const end = t;
  const durations = { '5m': 300, '30m': 1800, '1h': 3600, '6h': 21600, '24h': 86400 };
  const windowName = params.get('window') || params.get('time_range') || '1h';
  const start = params.get('start_time') ||
    new Date(Date.parse(end) - (durations[windowName] || 3600) * 1000).toISOString();

  R[O + '/dashboard/throughput-trend'] = {
    bucket: '5m',
    points: [{
      bucket_start: start, request_count: 12, token_consumed: 12000,
      switch_count: 0, qps: 0.1, tps: 100
    }]
  };
  R[O + '/realtime-traffic'] = {
    enabled: true, timestamp: end,
    summary: {
      window: params.get('window') || '1h', start_time: start,
      end_time: params.get('end_time') || end,
      platform: params.get('platform') || '',
      group_id: params.get('group_id') ? Number(params.get('group_id')) : null,
      qps: { current: 0.1, peak: 1, avg: 0.1 },
      tps: { current: 100, peak: 1000, avg: 100 }
    }
  };

  const conc = { enabled: true, timestamp: t, platform: {}, group: {}, account: {} };
  const avail = { enabled: true, timestamp: t, platform: {}, group: {}, account: {} };
  const selected = accounts.filter(a =>
    (!params.get('platform') || a.platform === params.get('platform')) &&
    (!params.get('group_id') || a.group_ids.includes(Number(params.get('group_id'))))
  );

  for (const a of selected) {
    const gid = Number(params.get('group_id')) || a.group_ids[0];
    const g = adminGroups.find(x => x.id === gid);
    const identity = {
      account_id: a.id, account_name: a.name,
      platform: a.platform, group_id: gid, group_name: g.name
    };
    const usable = a.status === 'active' && a.schedulable;
    conc.account[a.id] = {
      ...identity, current_in_use: a.current_concurrency,
      max_capacity: a.concurrency,
      load_percentage: 100 * a.current_concurrency / a.concurrency,
      waiting_in_queue: 0
    };
    avail.account[a.id] = {
      ...identity, status: a.status, is_available: usable,
      is_rate_limited: false, is_overloaded: false,
      has_error: a.status === 'error', error_message: a.error_message || ''
    };
    for (const [dim, key, meta] of [
      ['platform', a.platform, { platform: a.platform }],
      ['group', gid, {
        group_id: gid, group_name: g.name,
        platform: params.get('platform') || g.platform
      }]
    ]) {
      const c = conc[dim][key] ||= {
        ...meta, current_in_use: 0, max_capacity: 0,
        load_percentage: 0, waiting_in_queue: 0
      };
      c.current_in_use += a.current_concurrency;
      c.max_capacity += a.concurrency;
      c.load_percentage = 100 * c.current_in_use / c.max_capacity;
      const v = avail[dim][key] ||= {
        ...meta, total_accounts: 0, available_count: 0,
        rate_limit_count: 0, error_count: 0
      };
      v.total_accounts++;
      v.available_count += Number(usable);
      v.error_count += Number(a.status === 'error');
    }
  }
  R[O + '/concurrency'] = conc;
  R[O + '/account-availability'] = avail;

  return Object.prototype.hasOwnProperty.call(R, path) ? R[path] : undefined
}

export function adminPreview(method, path, params, fixtures, settings) {
  if (method !== 'GET') return undefined
  const adminGroups = [...groups, ...fixtures.groups.filter(group => group.id === 3 || group.id === 72).map(group => ({ ...group, account_count: 0 }))].map((group, index) => ({
    ...fixtures.groups.find(item => item.id === group.id), ...group, active_account_count: group.account_count,
    rate_limited_account_count: 0, force_openai_fast: false, free_openai_fast: false, force_openai_ultrafast: false, free_openai_ultrafast: false,
    model_pricing: [], profit_control_enabled: false, profit_min_margin: 0, profit_safety_buffer: 0,
    model_routing: null, model_routing_enabled: false, mcp_xml_inject: false, sort_order: index,
  }))
  if (path === '/admin/user-attributes' || path === '/admin/payment/providers') return []
  if (/^\/admin\/groups\/\d+\/model-allowlist-candidates$/.test(path)) return { models: fixtures.plaza.groups.filter(group => !params.get('platform') || group.platform === params.get('platform')).flatMap(group => group.models.map(model => model.name)) }
  const supplementary = supplementaryAdminPreview(path, params, fixtures, adminGroups)
  if (supplementary !== undefined) return supplementary
  const previewKeys = [...fixtures.keys.values()].flat()
  if (path === '/admin/settings') return {...settings, ops_monitoring_enabled:true, risk_control_enabled:true, payment_enabled:true, purchase_subscription_url:'', purchase_subscription_enabled:true}
  if (path === '/admin/usage/traffic') return buildTrafficReport(previewKeys, params)
  if (path === '/admin/usage/traffic/keys') return previewKeys.filter(key => (!params.get('user_id') || key.user_id === Number(params.get('user_id'))) && key.name.toLowerCase().includes((params.get('q') || '').toLowerCase())).map(key => ({ id: key.id, name: key.name, deleted: false }))
  if (path === '/admin/usage/traffic/health') return { status: 'ok', pending_events: 0, quarantined_events: 0, oldest_pending_at: null, oldest_pending_age_seconds: 0, delivered_events: 12, delivery_failures: 0, staging_failures: 0, persistence_failures: 0, last_delivery_at: now, checked_at: now, queued_events: 0, queue_capacity: 128, processing_events: 0, dropped_events: 0, queue_dropped_events: 0, shutdown_dropped_events: 0, spool_limit_dropped_events: 0, invalid_dropped_events: 0, unavailable_dropped_events: 0, worker_panics: 0, worker_running: true, storage_available: true }
  if (path === '/admin/risk-control/config') return {
    engine: 'openai', enabled: true, mode: 'observe', base_url: 'https://example.test', model: 'preview-moderation', proxy_id: null,
    api_key_configured: false, api_key_masked: '', api_key_count: 0, api_key_masks: [], api_key_statuses: [], timeout_ms: 3000,
    sample_rate: 1, all_groups: true, group_ids: [], record_non_hits: true, thresholds: { violence: 0.8 }, worker_count: 1, queue_size: 128,
    block_status: 403, block_message: 'Synthetic preview only', email_on_hit: false, auto_ban_enabled: false, ban_threshold: 3,
    violation_window_hours: 24, retry_count: 0, hit_retention_days: 7, non_hit_retention_days: 1, pre_hash_check_enabled: false,
    blocked_keywords: [], keyword_blocking_mode: 'api_only', model_filter: { type: 'all', models: [] }, cyber_policy_exclude_from_ban_count: false,
  }
  if (path === '/admin/risk-control/status') return { engine: 'openai', enabled: true, risk_control_enabled: true, mode: 'observe', worker_count: 1, max_workers: 1, active_workers: 0, idle_workers: 1, queue_size: 128, queue_length: 0, queue_usage_percent: 0, enqueued: 1, dropped: 0, processed: 1, errors: 0, pre_block_active: 0, pre_block_checked: 0, pre_block_allowed: 0, pre_block_blocked: 0, pre_block_errors: 0, pre_block_avg_latency_ms: 0, pre_block_api_key_active: 0, pre_block_api_key_available_count: 0, pre_block_api_key_total_calls: 0, pre_block_api_key_loads: [], api_key_statuses: [], flagged_hash_count: 0, last_cleanup_deleted_hit: 0, last_cleanup_deleted_non_hit: 0 }
  if (path === '/admin/risk-control/logs') return page([{ id: 901, request_id: 'preview-risk-request', user_id: 101, user_email: 'preview@example.test', api_key_id: 1001, api_key_name: 'Studio application', group_id: 1, group_name: groups[0].name, endpoint: '/v1/messages', provider: 'anthropic', model: 'claude-preview', mode: 'observe', action: 'allow', flagged: false, highest_category: 'violence', highest_score: 0.01, matched_keyword: '', category_scores: { violence: 0.01 }, threshold_snapshot: { violence: 0.8 }, input_excerpt: 'Synthetic local preview prompt. No real request was sent.', upstream_latency_ms: 120, error: '', violation_count: 0, auto_banned: false, email_sent: false, user_status: 'active', queue_delay_ms: 1, created_at: now }], params)
  if (path === '/admin/prompt-audit/config') return { enabled: true, blocking_enabled: false, blocking_latest_turn_only: true, store_pass_events: true, effective_mode: 'async_audit', strategy: 'priority', worker_count: 1, queue_capacity: 128, scanners: [], all_groups: true, group_ids: [], endpoints: [], config_version: 1, updated_at: now, updated_by: 201, change_summary: 'Synthetic local preview' }
  if (path === '/admin/prompt-audit/runtime') return { process_status: 'running', effective_mode: 'async_audit', expected_config_version: 1, active_config_version: 1, worker_total: 1, worker_active: 0, queue_capacity: 128, queue: { staging: 0, queued: 0, processing: 0, retry: 0, done: 1, failed: 0, active: 0 }, processed_total: 1, failed_total: 0, enqueued_total: 1, dropped_total: 0, database_status: 'ok', redis_status: 'ok', endpoints: {}, guard_metrics: { total: 1, allowed: 1, flagged: 0, blocked: 0, unavailable: 0, invalid: 0, timeouts: 0, failovers: 0, bulkhead_full: 0, record_failed: 0 } }
  if (path === '/admin/prompt-audit/events' || path === '/admin/prompt-audit/events/902') {
    const event = { id: 902, job_id: 1, snapshot: { request_id: 'preview-audit-request', user_id: 101, username: 'Preview Studio', user_email: 'preview@example.test', api_key_id: 1001, api_key_name: 'Studio application', group_id: 1, group_name: groups[0].name, provider: 'anthropic', endpoint: '/v1/messages', protocol: 'messages', model: 'claude-preview', prompt_hash: 'preview-hash', redacted_preview: 'Synthetic preview prompt', full_prompt: 'Synthetic preview prompt; no real user data or upstream request.', prompt_length: 66, message_count: 1, stage: 'input' }, decision: 'pass', risk_level: 'low', action: 'Allow', categories: [], matched_scanners: [], scanner_scores: {}, scanner_evidence: {}, scanner_backend: 'preview', scanner_version: 'preview', guard_endpoint_id: '', policy_id: 'preview', policy_version: 1, config_version: 1, chunk_total: 1, latency_ms: 120, issue_summaries: [], created_at: now }
    return path.endsWith('/902') ? event : page([event], params)
  }
  if (path === '/admin/ops/dashboard/snapshot-v2') {
    const end = new Date().toISOString(), start = new Date(Date.now() - 3600000).toISOString()
    return { generated_at: end, overview: { start_time: start, end_time: end, platform: '', success_count: 12, error_count_total: 0, business_limited_count: 0, error_count_sla: 0, request_count_total: 12, request_count_sla: 12, token_consumed: 12000, sla: 100, error_rate: 0, upstream_error_rate: 0, upstream_error_count_excl_429_529: 0, upstream_429_count: 0, upstream_529_count: 0, qps: { current: 0.1, peak: 1, avg: 0.1 }, tps: { current: 100, peak: 1000, avg: 100 }, duration: { p50_ms: 1200, p95_ms: 2400, avg_ms: 1400 }, ttft: { p50_ms: 120, p95_ms: 240, avg_ms: 140 } }, throughput_trend: { bucket: '5m', points: [{ bucket_start: start, request_count: 12, token_consumed: 12000, qps: 0.1, tps: 100 }] }, error_trend: { bucket: '5m', points: [{ bucket_start: start, error_count_total: 0, business_limited_count: 0, error_count_sla: 0, upstream_error_count_excl_429_529: 0, upstream_429_count: 0, upstream_529_count: 0 }] } }
  }
  if (path === '/admin/channels' || path === '/admin/channels/1') {
    const channel = { id: 1, name: 'Synthetic preview channel', description: 'Local synthetic data only', status: 'active', billing_model_source: 'requested', restrict_models: false, group_ids: [1], model_pricing: [], model_mapping: {}, apply_pricing_to_account_stats: false, account_stats_pricing_rules: [], created_at: now, updated_at: now }
    return path.endsWith('/1') ? channel : page([channel], params)
  }
  if (path === '/admin/channel-monitors') return page([{ id: 1, name: 'Synthetic preview monitor', provider: 'openai', api_mode: 'chat_completions', endpoint: 'https://example.test/v1', api_key_masked: 'sk-preview-***', primary_model: 'gpt-preview', extra_models: [], group_name: 'Synthetic preview', enabled: false, interval_seconds: 300, jitter_seconds: 0, last_checked_at: now, created_by: 201, created_at: now, updated_at: now, primary_status: 'operational', primary_latency_ms: 120, availability_7d: 100, extra_models_status: [], template_id: null, extra_headers: {}, body_override_mode: 'off', body_override: null, check_mode: 'probe', account_id: null }], params)
  if (path === '/admin/ops/settings/metric-thresholds') return { sla_percent_min: 99, ttft_p99_ms_max: 3000, request_error_rate_percent_max: 5, upstream_error_rate_percent_max: 5 }
  if (path === '/admin/ops/advanced-settings') return { data_retention: { cleanup_enabled: false, cleanup_schedule: '0 0 * * *', error_log_retention_days: 7, minute_metrics_retention_days: 7, hourly_metrics_retention_days: 30 }, aggregation: { aggregation_enabled: true }, openai_account_quota_auto_pause: { default_threshold_5h: 95, default_threshold_7d: 95 }, ignore_count_tokens_errors: false, ignore_context_canceled: false, ignore_no_available_accounts: false, ignore_invalid_api_key_errors: false, ignore_insufficient_balance_errors: false, display_openai_token_stats: false, display_alert_events: true, auto_refresh_enabled: false, auto_refresh_interval_seconds: 60 }
  if (path === '/admin/ops/dashboard/latency-histogram') return { start_time: now, end_time: now, platform: '', total_requests: 12, buckets: [{ range: '1000-2000', count: 12 }] }
  if (path === '/admin/ops/dashboard/error-distribution') return { total: 0, items: [] }
  if (path === '/admin/ops/alert-rules') return [{ id: 1, name: 'Synthetic preview alert', description: 'Local preview only', enabled: false, metric_type: 'error_rate', operator: '>', threshold: 5, window_minutes: 5, sustained_minutes: 1, severity: 'warning', cooldown_minutes: 10, notify_email: false, filters: {}, created_at: now, updated_at: now, last_triggered_at: now }]
  if (path === '/admin/ops/alert-events' || path === '/admin/ops/alert-events/1') {
    const event = { id: 1, rule_id: 1, severity: 'warning', status: 'resolved', fired_at: now, email_sent: false, created_at: now, title: 'Synthetic preview alert', description: 'Invented alert; no production monitoring data', metric_value: 6, threshold_value: 5, dimensions: { platform: 'openai' }, resolved_at: now }
    return path.endsWith('/1') ? event : [event]
  }
  if (path === '/admin/ops/system-logs') return page([{ id: 1, created_at: now, host: 'example.test', level: 'info', component: 'preview', message: 'Synthetic local preview; no production log data' }], params)
  if (path === '/admin/ops/system-logs/health') return { queue_depth: 0, queue_capacity: 128, dropped_count: 0, write_failed_count: 0, written_count: 1, avg_write_delay_ms: 1 }
  if (path === '/admin/ops/runtime/logging') return { level: 'info', persist_access_logs: false, enable_sampling: false, sampling_initial: 100, sampling_thereafter: 100, caller: false, stacktrace_level: 'error', retention_days: 7, request_retention_days: 1 }
  if (path === '/admin/payment/config') return { ...fixtures.paymentConfig, enabled: true, recharge_fee_rate: 0, load_balance_strategy: 'round_robin', product_name_prefix: 'Preview', product_name_suffix: '' }
  if (path === '/admin/dashboard/users-trend') return {trend:[]}
  if (path === '/admin/dashboard/users-ranking') return {users:[],total:0}
  if (path === '/admin/accounts') return page(sortAccounts(accounts.filter(a => (!params.get('platform') || a.platform===params.get('platform')) && (!params.get('search') || a.name.toLowerCase().includes(params.get('search').toLowerCase()))), params), params)
  if (/^\/admin\/accounts\/\d+$/.test(path)) return accounts.find(a=>a.id===Number(path.split('/').at(-1))) ?? null
  if (path === '/admin/accounts/platform-stats') return {total:24,by_platform:{anthropic:8,openai:8,gemini:8}}
  if (path === '/admin/accounts/today-stats') return {}
  if (path === '/admin/accounts/oauth-usage-summary') return {
    total_standard_cost: 0, average_standard_cost_per_oauth_account: 0,
    oauth_account_count: 0, oauth_accounts_with_usage: 0, usage_log_count: 0,
    deleted_oauth_account_count: 0, expired_oauth_account_count: 0,
  }
  if (path === '/admin/accounts/summary') return {
    total: 24, available: 23, active: 23, inactive: 0, error: 1, paused: 1,
    unschedulable: 1, rate_limited: 0, temp_unschedulable: 0, overloaded: 0, expired: 0, quota_exceeded: 0,
    openai: 8, recently_used: 24, never_used: 0,
    codex_5h: { sampled: 8, used_percent: 28, remaining_percent: 72, exhausted: 0 },
    codex_7d: { sampled: 8, used_percent: 39, remaining_percent: 61, exhausted: 0 },
    oauth_pool: { total: 0, available: 0, sampled: 0, exhausted: 0 },
    free_pool: { total: 0, available: 0, sampled: 0, exhausted: 0 },
    proxy_distribution: [
      ...proxies.map(proxy => ({ proxy_id: proxy.id, name: proxy.name, total: 2, available: 2, latency_ms: 85 + proxy.id * 12, latency_status: 'success' })),
      { proxy_id: null, name: '直连', total: 2, available: 1 },
    ],
  }
  if (path === '/admin/groups/all') return adminGroups.filter(group => !params.get('platform') || group.platform === params.get('platform'))
  if (path === '/admin/groups') return page(adminGroups,params)
  if (/^\/admin\/groups\/\d+$/.test(path)) return adminGroups.find(group => group.id === Number(path.split('/').at(-1))) ?? null
  if (path === '/admin/groups/live-capability') return { supported: false, reason: 'Synthetic preview only' }
  if (path === '/admin/groups/usage-summary') return adminGroups.map(group => ({ group_id: group.id, today_cost: 1.2, yesterday_cost: 2.4, total_cost: 12.8 }))
  if (path === '/admin/groups/capacity-summary') return adminGroups.map(group => ({ group_id: group.id, concurrency_used: 0, concurrency_max: 10, sessions_used: 0, sessions_max: 10, rpm_used: 0, rpm_max: 60, runtime_known: true }))
  if (/^\/admin\/groups\/\d+\/rate-multipliers$/.test(path)) return [{ user_id: 101, user_name: 'Preview Studio', user_email: 'preview@example.test', user_notes: 'Synthetic preview', user_status: 'active', rate_multiplier: 1, rpm_override: 60 }]
  if (path === '/admin/proxies/all') return proxies
  if (path === '/admin/proxies') return page(proxies,params)
  if (path === '/admin/users') return page(Object.values(fixtures.users),params)
  if (path === '/admin/usage') return page(fixtures.usage,params)
  // Deliberately large invented totals exercise metric wrapping at 360px.
  if (path === '/admin/usage/stats') return {
    total_requests: 1234567, total_tokens: 482350000,
    total_input_tokens: 51910000, total_output_tokens: 3400000,
    total_cache_tokens: 427040000, total_cache_creation_tokens: 12000000, total_cache_read_tokens: 415040000,
    total_cost: 1643.9282, total_actual_cost: 1283.5106,
    total_account_cost: 1442.9282, average_duration_ms: 23130,
    endpoints: ['/v1/messages', '/v1/responses', '/v1/chat/completions'].map((endpoint, i) => ({
      endpoint, requests: 1600 - i * 500, total_tokens: 397000000 / (i + 1),
      actual_cost: 15.14 / (i + 1), cost: 66.95 / (i + 1),
    })), upstream_endpoints: [], endpoint_paths: []
  }
  if (path === '/admin/announcements') return page(fixtures.announcements.map(item => ({ ...item, status: 'active', targeting: {}, created_by: 201, updated_by: 201, starts_at: now, ends_at: null })), params)
  if (/^\/admin\/announcements\/\d+\/read-status$/.test(path)) return page([{ user_id: 101, email: 'preview@example.test', username: 'Preview Studio', balance: 128.64, eligible: true, read_at: now }], params)
  if (path === '/admin/subscriptions') return page([{ ...fixtures.subscription, user: fixtures.users[101] }], params)
  if (path === '/admin/redeem-codes') return page(fixtures.redeemHistory.map(item => ({ ...item, used_by: 101 })), params)
  if (path === '/admin/audit-logs' || path === '/admin/audit-logs/903') {
    const entry = { id: 903, created_at: now, actor_user_id: 201, actor_email: 'admin@example.test', actor_role: 'admin', auth_method: 'session', credential_masked: 'preview-***', action: 'preview.read', method: 'GET', path: '/api/v1/admin/groups', request_id: 'preview-admin-request', client_ip: '192.0.2.18', user_agent: 'Synthetic preview client', request_body: '', status_code: 200, latency_ms: 12 }
    return path.endsWith('/903') ? entry : page([entry], params)
  }
  if (path === '/admin/promo-codes') return page([{ id: 1, code: 'PREVIEW-PROMO-NOT-REDEEMABLE', bonus_amount: 5, max_uses: 10, used_count: 1, status: 'active', expires_at: null, notes: 'Synthetic preview only', created_at: now, updated_at: now }], params)
  if (path === '/admin/promo-codes/1/usages') return page([{ id: 1, promo_code_id: 1, user_id: 101, bonus_amount: 5, used_at: now, user: fixtures.users[101] }], params)
  if (path === '/admin/models' || path === '/admin/tls-fingerprints') return []
  return undefined
}
