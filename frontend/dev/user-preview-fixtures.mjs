// Entirely synthetic fixtures. This module never reads files, environment values or remote data.
export const PREVIEW_PASSWORD = 'PreviewOnly123!'
export const PREVIEW_ACCOUNTS = Object.freeze({
  'preview@example.test': { id: 101, role: 'user', username: 'Preview Studio' },
  'admin@example.test': { id: 201, role: 'admin', username: 'Preview Admin' },
})

export const publicSettings = Object.freeze({
  registration_enabled: true,
  email_verify_enabled: false,
  force_email_on_third_party_signup: false,
  registration_email_suffix_whitelist: [],
  registration_email_domain_quota_enabled: false,
  promo_code_enabled: false,
  password_reset_enabled: true,
  invitation_code_enabled: false,
  login_agreement_enabled: false,
  login_agreement_mode: 'checkbox',
  login_agreement_documents: [],
  login_agreement_updated_at: '',
  login_agreement_revision: '',
  turnstile_enabled: false,
  turnstile_site_key: '',
  tencent_captcha_enabled: false,
  tencent_captcha_app_id: '',
  aliyun_captcha_enabled: false,
  aliyun_captcha_scene_id: '',
  aliyun_captcha_prefix: '',
  passkey_enabled: false,
  site_name: 'patrickapi',
  site_logo: '',
  site_subtitle: 'API workspace · synthetic local preview',
  api_base_url: 'https://patrickapi.microedulab.com',
  contact_info: '本地预览 · 合成数据 / Local preview · synthetic data',
  doc_url: '',
  home_content: '',
  compact_home_enabled: false,
  hide_ccs_import_button: true,
  payment_enabled: true,
  payment_balance_disabled: false,
  risk_control_enabled: false,
  table_default_page_size: 20,
  table_page_size_options: [10, 20, 50, 100],
  custom_menu_items: [],
  custom_endpoints: [],
  linuxdo_oauth_enabled: false,
  dingtalk_oauth_enabled: false,
  wechat_oauth_enabled: false,
  wechat_oauth_open_enabled: false,
  wechat_oauth_mp_enabled: false,
  wechat_oauth_mobile_enabled: false,
  oidc_oauth_enabled: false,
  oidc_oauth_provider_name: 'OIDC',
  github_oauth_enabled: false,
  google_oauth_enabled: false,
  backend_mode_enabled: false,
  version: 'local-preview',
  server_timezone: 'Etc/GMT+8',
  server_utc_offset: '-08:00',
  balance_low_notify_enabled: true,
  account_quota_notify_enabled: false,
  balance_low_notify_threshold: 10,
  channel_monitor_enabled: true,
  channel_monitor_mode: 'v2',
  channel_monitor_default_interval_seconds: 300,
  channel_monitor_hide_throughput: false,
  channel_monitor_show_quota: false,
  channel_monitor_hide_user_ranking: true,
  available_channels_enabled: true,
  subscription_enabled: true,
  model_plaza_enabled: true,
  model_plaza_require_auth: false,
  plugin_management_enabled: false,
  service_quota_enabled: false,
  affiliate_enabled: false,
  allow_user_view_error_requests: false,
})

const round = (value) => Math.round(value * 1e6) / 1e6
const iso = (value) => new Date(value).toISOString()
const day = (value) => iso(value).slice(0, 10)
const requestedDay = (value, timezone) => new Intl.DateTimeFormat('en-CA', { timeZone: timezone || 'UTC', year: 'numeric', month: '2-digit', day: '2-digit' }).format(new Date(value))

export function paginate(items, params) {
  const page = Math.max(1, Math.floor(Number(params.get('page')) || 1))
  const pageSize = Math.min(100, Math.max(1, Math.floor(Number(params.get('page_size')) || 20)))
  return {
    items: items.slice((page - 1) * pageSize, page * pageSize),
    total: items.length,
    page,
    page_size: pageSize,
    pages: Math.ceil(items.length / pageSize),
  }
}

export function filterUsage(records, params) {
  const filtered = records.filter((row) => {
    const rowDay = requestedDay(row.created_at, params.get('timezone'))
    if (params.get('api_key_id') && row.api_key_id !== Number(params.get('api_key_id'))) return false
    if (params.get('group_id') && row.group_id !== Number(params.get('group_id'))) return false
    if (params.get('model') && row.model !== params.get('model')) return false
    if (params.get('request_type') && row.request_type !== params.get('request_type')) return false
    if (params.get('billing_type') && row.billing_type !== Number(params.get('billing_type'))) return false
    if (params.get('stream') && row.stream !== (params.get('stream') === 'true')) return false
    if (params.get('start_date') && rowDay < params.get('start_date').slice(0, 10)) return false
    if (params.get('end_date') && rowDay > params.get('end_date').slice(0, 10)) return false
    if (params.get('billing_mode') && row.billing_mode !== params.get('billing_mode')) return false
    if (params.has('native_compaction_v2') && !!row.native_compaction_v2 !== (params.get('native_compaction_v2') === 'true')) return false
    return true
  })
  const sortBy = params.get('sort_by') === 'model' ? 'model' : 'created_at'
  return filtered.sort((a, b) => String(a[sortBy]).localeCompare(String(b[sortBy])) * (params.get('sort_order') === 'asc' ? 1 : -1))
}

export function aggregateUsage(records) {
  const fields = ['input_tokens', 'output_tokens', 'cache_creation_tokens', 'cache_read_tokens']
  const result = {
    total_requests: records.length,
    total_tokens: 0,
    total_cache_tokens: 0,
    total_cost: 0,
    total_actual_cost: 0,
    average_duration_ms: records.length ? Math.round(records.reduce((sum, row) => sum + row.duration_ms, 0) / records.length) : 0,
    models: {},
    endpoints: [],
    upstream_endpoints: [],
    endpoint_paths: [],
  }
  for (const field of fields) result[`total_${field}`] = records.reduce((sum, row) => sum + row[field], 0)
  result.total_tokens = fields.reduce((sum, field) => sum + result[`total_${field}`], 0)
  result.total_cache_tokens = result.total_cache_creation_tokens + result.total_cache_read_tokens
  result.total_cost = round(records.reduce((sum, row) => sum + row.total_cost, 0))
  result.total_actual_cost = round(records.reduce((sum, row) => sum + row.actual_cost, 0))
  for (const row of records) result.models[row.model] = (result.models[row.model] || 0) + 1
  return result
}

function compactStats(records) {
  const stats = aggregateUsage(records)
  return {
    requests: stats.total_requests,
    input_tokens: stats.total_input_tokens,
    output_tokens: stats.total_output_tokens,
    cache_creation_tokens: stats.total_cache_creation_tokens,
    cache_read_tokens: stats.total_cache_read_tokens,
    cache_write_tokens: stats.total_cache_creation_tokens,
    total_tokens: stats.total_tokens,
    cost: stats.total_cost,
    actual_cost: stats.total_actual_cost,
  }
}

export function buildUsageSnapshot(records, params = new URLSearchParams()) {
  const granularity = params.get('granularity') === 'hour' ? 'hour' : 'day'
  const bucket = (row) => granularity === 'hour' ? `${row.created_at.slice(0, 13)}:00:00Z` : requestedDay(row.created_at, params.get('timezone'))
  const dates = [...new Set(records.map(bucket))].sort()
  const models = [...new Set(records.map((row) => row.model))].map((model) => ({
    model, ...compactStats(records.filter((row) => row.model === model)),
  }))
  const groups = [...new Set(records.map((row) => row.group_id))].map((groupId) => {
    const entries = records.filter((row) => row.group_id === groupId)
    return { group_id: groupId, group_name: entries[0]?.group?.name || 'Preview group', ...compactStats(entries) }
  })
  return {
    generated_at: iso(Date.now()),
    start_date: params.get('start_date') || dates[0]?.slice(0, 10) || day(Date.now()),
    end_date: params.get('end_date') || day(Date.now()),
    granularity,
    trend: dates.map((date) => ({ date, ...compactStats(records.filter((row) => bucket(row) === date)) })),
    models,
    groups,
  }
}

export function createPreviewFixtures() {
  const now = Date.now()
  const createdAt = iso(now - 45 * 86400000)
  const timestamp = iso(now)
  const groups = [
    { id: 1, name: 'OpenAI · Preview', platform: 'openai' },
    { id: 2, name: 'Claude · Preview', platform: 'anthropic' },
    { id: 3, name: 'Creative · Preview', platform: 'gemini' },
  ].map((group) => ({
    ...group,
    description: '合成分组，仅供界面预览 / Synthetic preview group',
    rate_multiplier: 1,
    is_exclusive: false,
    status: 'active',
    subscription_type: group.id === 3 ? 'subscription' : 'standard',
    daily_limit_usd: group.id === 3 ? 10 : null,
    weekly_limit_usd: group.id === 3 ? 50 : null,
    monthly_limit_usd: group.id === 3 ? 150 : null,
    long_context_pricing_enabled: true,
    allow_image_generation: false,
    allow_batch_image_generation: false,
    image_rate_independent: false,
    image_rate_multiplier: 1,
    batch_image_discount_multiplier: 1,
    batch_image_hold_multiplier: 1,
    image_price_1k: null,
    image_price_2k: null,
    image_price_4k: null,
    video_rate_independent: false,
    video_rate_multiplier: 1,
    video_price_480p: null,
    video_price_720p: null,
    video_price_1080p: null,
    web_search_price_per_call: null,
    search_price_per_1k: null,
    audio_realtime_price_per_min: null,
    audio_tts_price_per_million_chars: null,
    audio_stt_price_per_hour: null,
    peak_rate_enabled: false,
    peak_start: '',
    peak_end: '',
    peak_rate_multiplier: 1,
    claude_code_only: false,
    fallback_group_id: null,
    fallback_group_id_on_invalid_request: null,
    allow_messages_dispatch: true,
    allow_live: false,
    require_oauth_only: false,
    require_privacy_set: false,
    created_at: createdAt,
    updated_at: timestamp,
  }))
  const users = Object.fromEntries(Object.entries(PREVIEW_ACCOUNTS).map(([email, account]) => [account.id, {
    ...account, email, balance: 128.64, frozen_balance: 0, concurrency: 10, rpm_limit: 60,
    status: 'active', allowed_groups: [1, 2, 3], email_bound: true,
    auth_bindings: { email: true, github: false, google: false, oidc: false, linuxdo: false, wechat: false, dingtalk: false },
    balance_notify_enabled: true, balance_notify_threshold: 10, balance_notify_extra_emails: [],
    avatar_url: null, created_at: createdAt, updated_at: timestamp, last_active_at: timestamp, run_mode: 'standard',
  }]))
  const makeKey = (id, userId, name, groupId, status = 'active') => ({
    id, user_id: userId, key: `sk-preview-${userId}-${id}-not-a-real-key`, name, group_id: groupId,
    routing_group_ids: [groupId], status, ip_whitelist: [], ip_blacklist: [],
    last_used_at: iso(now - 18 * 60000), last_used_ip: '192.0.2.18',
    quota: 50, quota_used: 8.42, expires_at: null, created_at: createdAt, updated_at: timestamp,
    current_concurrency: 0, group: groups.find((group) => group.id === groupId),
    rate_limit_5h: 0, rate_limit_1d: 0, rate_limit_7d: 0, usage_5h: 0, usage_1d: 0, usage_7d: 0,
    window_5h_start: null, window_1d_start: null, window_7d_start: null,
    reset_5h_at: null, reset_1d_at: null, reset_7d_at: null,
  })
  const keys = new Map(Object.values(users).map((user) => [user.id, [
    makeKey(1001, user.id, 'Studio application', 1),
    makeKey(1002, user.id, 'Local development', 2),
    makeKey(1003, user.id, 'Archive key', 1, 'inactive'),
  ]]))
  const modelNames = ['gpt-preview', 'claude-sonnet-preview', 'gemini-pro-preview']
  // Token pricing follows the API's USD/token unit; the plaza displays USD/1M tokens.
  const pricing = (index) => ({
    billing_mode: 'token', input_price: (1.5 + index) / 1_000_000, output_price: (6 + index * 2) / 1_000_000,
    cache_write_price: null, cache_read_price: .15 / 1_000_000, image_input_price: null,
    image_output_price: null, per_request_price: null, intervals: [],
  })
  const plaza = {
    description: '**本地预览 · 合成数据。** 下列模型、价格与分组仅用于布局验收，不代表实际供应或计费。 / Synthetic models and prices for UI review only.',
    groups: groups.map((group, index) => ({ ...group, models: [
      { name: modelNames[index], platform: group.platform, pricing: pricing(index), official_pricing: null },
      { name: `${modelNames[index]}-fast`, platform: group.platform, pricing: pricing(index + 1), official_pricing: null },
    ] })),
  }
  const usage = Array.from({ length: 84 }, (_, index) => {
    const group = groups[index % 3]
    const input = 620 + (index * 173) % 9000
    const output = 150 + (index * 127) % 3100
    const cacheRead = index % 3 === 0 ? input * 2 : 0
    const cost = round(input * .000002 + output * .000008 + cacheRead * .0000002)
    return {
      id: 5000 + index, user_id: 101, api_key_id: 1001 + (index % 2), account_id: null,
      request_id: `preview-request-${index + 1}`, session_id: null, model: modelNames[index % 3],
      service_tier: null, reasoning_effort: index % 2 ? 'medium' : null,
      inbound_endpoint: '/v1/chat/completions', upstream_endpoint: '/v1/chat/completions',
      group_id: group.id, subscription_id: group.id === 3 ? 301 : null,
      input_tokens: input, output_tokens: output, cache_creation_tokens: 0, cache_read_tokens: cacheRead,
      cache_creation_5m_tokens: 0, cache_creation_1h_tokens: 0,
      input_cost: round(input * .000002), output_cost: round(output * .000008), cache_creation_cost: 0,
      cache_read_cost: round(cacheRead * .0000002), total_cost: cost, actual_cost: cost,
      rate_multiplier: 1, long_context_billing_applied: false, billing_type: 0, billing_mode: 'token',
      request_type: index % 4 ? 'stream' : 'sync', stream: index % 4 !== 0, native_compaction_v2: false,
      duration_ms: 540 + (index * 149) % 4200, first_token_ms: 130 + (index * 31) % 700,
      image_count: 0, image_size: null, image_input_size: null, image_output_size: null,
      image_size_source: null, image_size_breakdown: null, image_input_tokens: 0, image_input_cost: 0,
      image_output_tokens: 0, image_output_cost: 0, user_agent: 'Synthetic preview client',
      ip_address: '192.0.2.18', cache_ttl_overridden: false,
      created_at: iso(now - index * 4 * 3600000), group,
      api_key: keys.get(101)[index % 2],
    }
  })
  const subscription = {
    id: 301, user_id: 101, group_id: 3, status: 'active', starts_at: iso(now - 7 * 86400000),
    daily_usage_usd: 1.24, weekly_usage_usd: 8.42, monthly_usage_usd: 17.65,
    daily_window_start: iso(now - 3 * 3600000), weekly_window_start: iso(now - 3 * 86400000),
    monthly_window_start: iso(now - 7 * 86400000), created_at: createdAt, updated_at: timestamp,
    expires_at: iso(now + 23 * 86400000), group: groups[2],
  }
  const subscriptionProgress = {
    id: 301, group_name: groups[2].name,
    daily: { used_usd: 1.24, limit_usd: 10, remaining_usd: 8.76, percentage: 12.4, window_start: subscription.daily_window_start, resets_at: iso(now + 21 * 3600000), resets_in_seconds: 21 * 3600 },
    weekly: { used_usd: 8.42, limit_usd: 50, remaining_usd: 41.58, percentage: 16.84, window_start: subscription.weekly_window_start, resets_at: iso(now + 4 * 86400000), resets_in_seconds: 4 * 86400 },
    monthly: { used_usd: 17.65, limit_usd: 150, remaining_usd: 132.35, percentage: 11.77, window_start: subscription.monthly_window_start, resets_at: iso(now + 23 * 86400000), resets_in_seconds: 23 * 86400 },
    expires_at: subscription.expires_at, expires_in_days: 23,
  }
  const plans = [19, 49].map((price, index) => ({
    id: 401 + index, group_id: 3, group_platform: 'gemini', group_name: groups[2].name,
    rate_multiplier: 1, name: index ? 'Studio · Preview' : 'Explore · Preview',
    description: '合成套餐，仅供预览；购买操作不可用 / Synthetic plan; purchase unavailable.',
    price, original_price: price, currency: 'USD', validity_days: 30, validity_unit: 'day',
    features: ['本地合成数据 / Synthetic data', '仅用于界面预览 / UI preview only'],
    purchase_mode: 'balance', for_sale: true, sort_order: index,
    daily_limit_usd: 10, weekly_limit_usd: 50, monthly_limit_usd: 150,
  }))
  const paymentConfig = {
    payment_enabled: true, min_amount: 1, max_amount: 100, daily_limit: 100,
    max_pending_orders: 0, order_timeout_minutes: 15, balance_disabled: false,
    balance_recharge_multiplier: 1, subscription_usd_to_cny_rate: 0,
    enabled_payment_types: [], help_image_url: '', stripe_publishable_key: '',
    help_text: '本地预览不支持真实充值、下单或付款。 / Real payments are unavailable in this preview.',
  }
  const orders = [
    { id: 601, status: 'COMPLETED', amount: 20, order_type: 'balance' },
    { id: 602, status: 'CANCELLED', amount: 19, order_type: 'subscription', plan_id: 401 },
  ].map((order, index) => ({
    ...order, user_id: 101, pay_amount: order.amount, currency: 'USD', fee_rate: 0,
    payment_type: 'preview', out_trade_no: `PREVIEW-${order.id}`, created_at: iso(now - (index + 2) * 86400000),
    expires_at: iso(now - 86400000), paid_at: index ? undefined : iso(now - 2 * 86400000),
    completed_at: index ? undefined : iso(now - 2 * 86400000), refund_amount: 0, provider_instance_id: '',
  }))
  const thresholds = { minimum_sample: 5, warning_error_rate: .05, critical_error_rate: .15, target_ttft_ms: 800, warning_ttft_ms: 2000, critical_ttft_ms: 5000, warning_cache_rate: .1, critical_cache_rate: .05, error_weight: .6, ttft_weight: .3, cache_weight: .1 }
  const metrics = {
    success_requests: 84, error_requests: 0, request_count: 84, token_count: 486200,
    rpm: 1.4, tpm: 8103, error_rate: 0, cache_rate: .24, cache_rate_numerator: 24, cache_rate_denominator: 100,
    ttft: { sample_count: 84, p50_ms: 420, p90_ms: 760, p95_ms: 890, avg_ms: 460 },
    duration: { sample_count: 84, p50_ms: 2300, p90_ms: 4100, p95_ms: 4900, avg_ms: 2500 },
  }
  const health = { overall: 'healthy', error_rate: 'healthy', ttft: 'healthy', cache: 'healthy', score: 97, error_rate_score: 100, ttft_score: 95, cache_score: 92, minimum_sample: 5, thresholds }
  const coverage = { requested_start: iso(now - 86400000), requested_end: timestamp, coverage_start: iso(now - 86400000), data_through: timestamp, computed_at: timestamp, aggregation_lag_seconds: 0, coverage_complete: true, bucket_seconds: 3600 }
  const monitor = {
    config: { version: 1, enabled: true, refresh_interval_seconds: 300, platforms: groups.map((group, index) => ({ platform: group.platform, enabled: true, models: [modelNames[index]] })), group_ids: [1, 2, 3], health_thresholds: thresholds, ignored_error_categories: [] },
    coverage, metrics, health,
    trend: Array.from({ length: 24 }, (_, index) => ({ bucket_start: iso(now - (23 - index) * 3600000), metrics: { ...metrics, request_count: 3 + index % 4, success_requests: 3 + index % 4 }, health })),
  }
  return {
    users, groups, keys, makeKey, usage, plaza, subscription, subscriptionProgress, plans, orders,
    paymentConfig, monitor, nextKeyId: 1004,
    checkoutInfo: { methods: {}, global_min: 1, global_max: 100, plans, balance_disabled: false, balance_recharge_multiplier: 1, subscription_usd_to_cny_rate: 0, recharge_fee_rate: 0, recharge_bonus_tiers: [], help_text: paymentConfig.help_text, help_image_url: '', stripe_publishable_key: '' },
    announcements: [{ id: 701, title: '本地预览 · 合成数据', content: '当前账户、密钥、金额、调用和模型均为合成示例。此环境不连接生产服务。', notify_mode: 'silent', created_at: timestamp, updated_at: timestamp, read_at: timestamp }],
    redeemHistory: [{ id: 801, code: 'PREVIEW-NOT-REDEEMABLE', type: 'balance', value: 20, status: 'used', used_at: iso(now - 2 * 86400000), created_at: createdAt, notes: 'Synthetic history; no real balance was credited.' }],
  }
}
