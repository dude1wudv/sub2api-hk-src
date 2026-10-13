// All records are invented. There are no upstream credentials or production calls.
const now = '2026-10-10T12:00:00Z'
const groups = [
  { id: 1, name: 'Claude Standard', platform: 'anthropic', status: 'active', rate_multiplier: 1, subscription_type: 'standard', is_exclusive: false, accounts_count: 8, description: '合成标准分组', model_pricing: [], created_at: now, updated_at: now },
  { id: 2, name: 'OpenAI Pro', platform: 'openai', status: 'active', rate_multiplier: 1.5, subscription_type: 'standard', is_exclusive: false, accounts_count: 12, description: '合成高级分组', model_pricing: [], created_at: now, updated_at: now }
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
export function adminPreview(method, path, params, fixtures, settings) {
  if (method !== 'GET') return undefined
  if (path === '/admin/settings') return {...settings, ops_monitoring_enabled:false, payment_enabled:true, purchase_subscription_url:'', purchase_subscription_enabled:true}
  if (path === '/admin/payment/config') return {enabled:true,methods:[],plans:[]}
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
  if (path === '/admin/groups/all') return groups
  if (path === '/admin/groups') return page(groups,params)
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
  if (/^\/admin\/(announcements|subscriptions|redeem|promo-codes|audit-logs)$/.test(path)) return page([],params)
  if (path === '/admin/models' || path === '/admin/tls-fingerprints') return []
  return undefined
}
