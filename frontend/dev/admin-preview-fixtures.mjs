// All records are invented. There are no upstream credentials or production calls.
const now = '2026-10-10T12:00:00Z'
const groups = [
  { id: 1, name: 'Claude Standard', platform: 'anthropic', status: 'active', rate_multiplier: 1, subscription_type: 'standard', is_exclusive: false, accounts_count: 8, description: '合成标准分组', model_pricing: [], created_at: now, updated_at: now },
  { id: 2, name: 'OpenAI Pro', platform: 'openai', status: 'active', rate_multiplier: 1.5, subscription_type: 'standard', is_exclusive: false, accounts_count: 12, description: '合成高级分组', model_pricing: [], created_at: now, updated_at: now }
]
const accounts = Array.from({length: 24}, (_, i) => ({
  id: 1024 + i, name: `${['Claude','OpenAI','Gemini'][i%3]}-${String(i+1).padStart(2,'0')}`,
  platform: ['anthropic','openai','gemini'][i%3], type: 'apikey', status: i===1 ? 'error' : 'active',
  schedulable: i!==1, concurrency: 10, current_concurrency: i%4, priority: 50, rate_multiplier: 1,
  proxy_id: null, group_ids: [i%2+1], groups: [groups[i%2]], notes: '独立合成数据，仅供界面验收',
  credentials: { base_url: 'https://example.test/v1' }, credentials_status: { has_api_key: true },
  created_at: now, updated_at: now, last_used_at: now, expires_at: null, auto_pause_on_expired: false,
  error_message: i===1 ? '合成示例：上游限流' : null,
  rate_limited_at: null, rate_limit_reset_at: null, overload_until: null, temp_unschedulable_until: null,
  temp_unschedulable_reason: null, session_window_start: null, session_window_end: null, session_window_status: null,
  quota_limit: 100, quota_used: 12.8+i, quota_daily_limit: 20, quota_daily_used: 3.28+i/10,
  scheduler_scores: [{group_id:i%2+1, group_name:groups[i%2].name, base_score:72.5}]
}))
const page = (items, params) => { const size=Number(params.get('page_size')||20), n=Number(params.get('page')||1); return {items:items.slice((n-1)*size,n*size),total:items.length,page:n,page_size:size,pages:Math.ceil(items.length/size)} }
export function adminPreview(method, path, params, fixtures, settings) {
  if (method !== 'GET') return undefined
  if (path === '/admin/settings') return {...settings, ops_monitoring_enabled:false, payment_enabled:true, purchase_subscription_url:'', purchase_subscription_enabled:true}
  if (path === '/admin/payment/config') return {enabled:true,methods:[],plans:[]}
  if (path === '/admin/dashboard/users-trend') return {trend:[]}
  if (path === '/admin/dashboard/users-ranking') return {users:[],total:0}
  if (path === '/admin/accounts') return page(accounts.filter(a => (!params.get('platform') || a.platform===params.get('platform')) && (!params.get('search') || a.name.toLowerCase().includes(params.get('search').toLowerCase()))), params)
  if (/^\/admin\/accounts\/\d+$/.test(path)) return accounts.find(a=>a.id===Number(path.split('/').at(-1))) ?? null
  if (path === '/admin/accounts/platform-stats') return {total:24,by_platform:{anthropic:8,openai:8,gemini:8}}
  if (path === '/admin/accounts/today-stats') return {}
  if (path === '/admin/groups/all') return groups
  if (path === '/admin/groups') return page(groups,params)
  if (path === '/admin/proxies/all') return []
  if (path === '/admin/proxies') return page([],params)
  if (path === '/admin/users') return page(Object.values(fixtures.users),params)
  if (path === '/admin/usage') return page(fixtures.usage,params)
  if (/^\/admin\/(announcements|subscriptions|redeem|promo-codes|audit-logs)$/.test(path)) return page([],params)
  if (path === '/admin/models' || path === '/admin/tls-fingerprints') return []
  return undefined
}
