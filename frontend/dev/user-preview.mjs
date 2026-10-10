import { createServer as createHttpServer } from 'node:http'
import { randomUUID } from 'node:crypto'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { createServer as createViteServer } from 'vite'
import vue from '@vitejs/plugin-vue'
import {
  PREVIEW_PASSWORD, PREVIEW_ACCOUNTS, publicSettings,
  createPreviewFixtures, paginate, filterUsage, aggregateUsage, buildUsageSnapshot,
} from './user-preview-fixtures.mjs'

const HOST = '127.0.0.1'
const VITE_PORT = 3410
const API_PORT = 3411
const PREVIEW_ORIGIN = `http://${HOST}:${VITE_PORT}`
const API_ORIGIN = `http://${HOST}:${API_PORT}`
const devDirectory = dirname(fileURLToPath(import.meta.url))
const frontendDirectory = resolve(devDirectory, '..')
const fixtures = createPreviewFixtures()
const sessions = new Map()
const refreshSessions = new Map()
const redeemedCodes = new Set()
const redemptionHistory = new Map(Object.keys(fixtures.users).map(id => [Number(id), [...fixtures.redeemHistory]]))

function json(response, status, data) {
  response.writeHead(status, { 'Content-Type': 'application/json; charset=utf-8', 'Cache-Control': 'no-store', 'X-Content-Type-Options': 'nosniff' })
  response.end(JSON.stringify(data))
}
const ok = (response, data) => json(response, 200, { code: 0, message: 'Synthetic local preview', data, preview: true })
const error = (response, status, code, message) => json(response, status, { code, message, preview: true })
const unavailable = (response) => error(response, 501, 'PREVIEW_UNAVAILABLE', '此操作在本地合成预览中不可用。不会执行真实注册、付款、验证码或外部操作。 / This action is unavailable in the synthetic preview; no real operation was performed.')

async function readJson(request) {
  let length = 0
  const chunks = []
  for await (const chunk of request) {
    length += chunk.length
    if (length > 32768) throw new Error('PREVIEW_BODY_TOO_LARGE')
    chunks.push(chunk)
  }
  if (!length) return {}
  const body = JSON.parse(Buffer.concat(chunks).toString('utf8'))
  if (!body || typeof body !== 'object' || Array.isArray(body)) throw new Error('PREVIEW_INVALID_BODY')
  return body
}

function currentUser(request) {
  const authorization = String(request.headers.authorization || '')
  const token = authorization.startsWith('Bearer ') ? authorization.slice(7) : ''
  const session = sessions.get(token)
  if (!session || session.expiresAt < Date.now()) return null
  return fixtures.users[session.userId] || null
}

function authResponse(session) {
  return {
    access_token: session.accessToken,
    refresh_token: session.refreshToken,
    token_type: 'Bearer',
    expires_in: 3600,
    user: { ...fixtures.users[session.userId], subscriptions: [{ ...fixtures.subscription, user_id: session.userId }] },
  }
}

function userUsage(user) {
  const keys = fixtures.keys.get(user.id)
  return fixtures.usage.map((row) => ({ ...row, user_id: user.id, api_key: keys.find((key) => key.id === row.api_key_id) || row.api_key }))
}

function dashboardStats(user) {
  const records = userUsage(user)
  const total = aggregateUsage(records)
  const today = aggregateUsage(records.filter((row) => row.created_at.slice(0, 10) === new Date().toISOString().slice(0, 10)))
  const keys = fixtures.keys.get(user.id)
  const result = { ...total, total_api_keys: keys.length, active_api_keys: keys.filter((key) => key.status === 'active').length, rpm: 1.4, tpm: 8103 }
  for (const name of ['requests', 'input_tokens', 'output_tokens', 'cache_creation_tokens', 'cache_read_tokens', 'tokens', 'cost', 'actual_cost']) result[`today_${name}`] = today[`total_${name}`]
  result.by_platform = fixtures.groups.map((group) => {
    const sum = aggregateUsage(records.filter((row) => row.group_id === group.id))
    const sumToday = aggregateUsage(records.filter((row) => row.group_id === group.id && row.created_at.slice(0, 10) === new Date().toISOString().slice(0, 10)))
    return { platform: group.platform, total_requests: sum.total_requests, total_tokens: sum.total_tokens, total_actual_cost: sum.total_actual_cost, today_requests: sumToday.total_requests, today_tokens: sumToday.total_tokens, today_actual_cost: sumToday.total_actual_cost }
  })
  return result
}

function adminStats(user) {
  return { ...dashboardStats(user), total_users: 2, today_new_users: 0, active_users: 1, hourly_active_users: 1,
    stats_updated_at: new Date().toISOString(), stats_stale: false, total_accounts: 3, normal_accounts: 3,
    error_accounts: 0, ratelimit_accounts: 0, overload_accounts: 0, total_account_cost: 0,
    today_account_cost: 0, uptime: 86400 }
}

function usageForGateway(key, params) {
  const records = filterUsage(fixtures.usage.filter((row) => row.api_key_id === key.id), params)
  const snapshot = buildUsageSnapshot(records)
  const sum = aggregateUsage(records)
  const total = { requests: sum.total_requests, input_tokens: sum.total_input_tokens, output_tokens: sum.total_output_tokens, cache_creation_tokens: sum.total_cache_creation_tokens, cache_read_tokens: sum.total_cache_read_tokens, total_tokens: sum.total_tokens, cost: sum.total_cost, actual_cost: sum.total_actual_cost }
  return {
    mode: 'quota_limited', isValid: key.status === 'active', status: key.status,
    quota: { limit: key.quota, used: key.quota_used, remaining: Math.max(0, key.quota - key.quota_used) },
    expires_at: key.expires_at, days_until_expiry: null, rate_limits: [],
    usage: { today: total, total, rpm: 1.4, tpm: 8103, average_duration_ms: sum.average_duration_ms },
    daily_usage: snapshot.trend, model_stats: snapshot.models, preview: true,
  }
}

function updateKey(key, body) {
  const next = { ...key }
  if ('name' in body) {
    if (typeof body.name !== 'string' || !body.name.trim() || body.name.length > 100) throw new Error('名称必须为 1–100 字符 / Name must contain 1–100 characters.')
    next.name = body.name.trim()
  }
  if ('status' in body) {
    if (!['active', 'inactive'].includes(body.status)) throw new Error('Unsupported preview key status.')
    next.status = body.status
  }
  if ('group_id' in body) {
    if (body.group_id !== null && !fixtures.groups.some((group) => group.id === body.group_id)) throw new Error('Unknown preview group.')
    next.group_id = body.group_id
    next.group = fixtures.groups.find((group) => group.id === body.group_id)
  }
  if ('routing_group_ids' in body) {
    if (!Array.isArray(body.routing_group_ids) || body.routing_group_ids.some((id) => !fixtures.groups.some((group) => group.id === id))) throw new Error('Unknown preview routing group.')
    next.routing_group_ids = [...new Set(body.routing_group_ids)]
  }
  for (const field of ['quota', 'rate_limit_5h', 'rate_limit_1d', 'rate_limit_7d']) {
    if (field in body) {
      if (typeof body[field] !== 'number' || !Number.isFinite(body[field]) || body[field] < 0) throw new Error('Preview limits must be non-negative numbers.')
      next[field] = body[field]
    }
  }
  for (const field of ['ip_whitelist', 'ip_blacklist']) {
    if (field in body) {
      if (!Array.isArray(body[field]) || body[field].some((entry) => typeof entry !== 'string')) throw new Error('IP rules must be a list of strings.')
      next[field] = [...body[field]]
    }
  }
  if ('expires_at' in body) {
    if (body.expires_at === '') next.expires_at = null
    else if (body.expires_at !== null) {
      if (!Number.isFinite(Date.parse(body.expires_at))) throw new Error('Invalid preview expiry.')
      next.expires_at = body.expires_at
    }
  }
  if (body.reset_quota) next.quota_used = 0
  if (body.reset_rate_limit_usage) Object.assign(next, { usage_5h: 0, usage_1d: 0, usage_7d: 0 })
  next.updated_at = new Date().toISOString()
  return next
}

async function handleApi(request, response) {
  // Bind and validate both services locally; requests are never forwarded by this mock.
  const host = String(request.headers.host || '')
  const origin = request.headers.origin
  if (host !== `${HOST}:${API_PORT}` || (origin && ![PREVIEW_ORIGIN, API_ORIGIN].includes(origin))) {
    return error(response, 403, 'PREVIEW_LOCAL_ONLY', 'This preview accepts local-origin requests only.')
  }
  if (request.method === 'OPTIONS') return json(response, 204, {})
  const url = new URL(request.url || '/', API_ORIGIN)
  const path = url.pathname.replace(/\/+$/, '') || '/'
  const method = request.method || 'GET'
  const params = url.searchParams
  const user = currentUser(request)

  if (method === 'GET' && path === '/health') return ok(response, { status: 'ok', synthetic: true })
  if (method === 'GET' && path === '/setup/status') return ok(response, { needs_setup: false, step: 'complete' })
  if (method === 'GET' && path === '/api/v1/settings/public') return ok(response, publicSettings)
  if (method === 'GET' && path === '/api/v1/model-plaza') return ok(response, fixtures.plaza)
  if (path === '/api/v1/auth/login' && method === 'POST') {
    const body = await readJson(request)
    const account = PREVIEW_ACCOUNTS[String(body.email || '').trim().toLowerCase()]
    if (!account || body.password !== PREVIEW_PASSWORD) return error(response, 401, 'PREVIEW_LOGIN_REQUIRED', '仅接受文档中的合成预览账户。 / Use the synthetic account listed in dev/README.md.')
    const session = { userId: account.id, accessToken: `preview-access-${randomUUID()}`, refreshToken: `preview-refresh-${randomUUID()}`, expiresAt: Date.now() + 3600000 }
    sessions.set(session.accessToken, session)
    refreshSessions.set(session.refreshToken, session)
    return ok(response, authResponse(session))
  }
  if (path === '/api/v1/auth/refresh' && method === 'POST') {
    const body = await readJson(request)
    const session = refreshSessions.get(body.refresh_token)
    if (!session) return error(response, 401, 'PREVIEW_SESSION_EXPIRED', '预览会话已结束，请重新登录。 / Sign in to the preview again.')
    session.expiresAt = Date.now() + 3600000
    return ok(response, authResponse(session))
  }
  if (path === '/api/v1/auth/logout' && method === 'POST') {
    const body = await readJson(request)
    const session = refreshSessions.get(body.refresh_token)
    if (session) {
      sessions.delete(session.accessToken)
      refreshSessions.delete(session.refreshToken)
    }
    return ok(response, { message: 'Local preview session ended.' })
  }
  // Registration, reset, OAuth and authenticators must never simulate a successful real action.
  if (path.startsWith('/api/v1/auth/') && path !== '/api/v1/auth/me') return unavailable(response)
  if (path === '/v1/usage' && method === 'GET') {
    const rawKey = String(request.headers.authorization || '').replace(/^Bearer /, '')
    const key = [...fixtures.keys.values()].flat().find((item) => item.key === rawKey)
    if (!key) return error(response, 401, 'PREVIEW_KEY_REQUIRED', '只支持从预览密钥页复制的 sk-preview- 密钥。 / Use a synthetic sk-preview- key from this preview.')
    return json(response, 200, usageForGateway(key, params))
  }
  if (!user) return error(response, 401, 'PREVIEW_LOGIN_REQUIRED', '请先登录本地预览账户。 / Sign in to the local preview.')
  if (path.startsWith('/api/v1/admin/') && user.role !== 'admin') return error(response, 403, 'PREVIEW_ADMIN_REQUIRED', 'The synthetic administrator account is required.')
  const localPath = path.replace(/^\/api\/v1/, '')
  const records = filterUsage(userUsage(user), params)
  const snapshot = buildUsageSnapshot(records, params)

  if (method === 'GET' && ['/auth/me', '/user/profile'].includes(localPath)) return ok(response, { ...user, subscriptions: [{ ...fixtures.subscription, user_id: user.id }] })
  if (method === 'GET' && localPath === '/user/platform-quotas') return ok(response, { platform_quotas: [] })
  if (method === 'GET' && localPath === '/user/totp/status') return ok(response, { enabled: false, enabled_at: null, feature_enabled: false })
  if (method === 'GET' && localPath === '/user/totp/verification-method') return ok(response, { method: 'password' })
  if (method === 'GET' && localPath === '/user/passkeys') return ok(response, [])
  if (method === 'GET' && localPath === '/groups/available') return ok(response, fixtures.groups)
  if (method === 'GET' && localPath === '/groups/rates') return ok(response, {})
  if (method === 'GET' && localPath === '/channels/available') return ok(response, [{ name: 'Synthetic preview routes', description: 'No upstream is connected.', platforms: fixtures.plaza.groups.map((group) => ({ platform: group.platform, groups: [group], supported_models: group.models })) }])
  if (method === 'GET' && localPath === '/announcements') return ok(response, fixtures.announcements)

  if (localPath === '/keys' && method === 'GET') {
    let rows = fixtures.keys.get(user.id)
    const search = (params.get('search') || '').toLowerCase()
    if (search) rows = rows.filter((key) => `${key.name} ${key.key}`.toLowerCase().includes(search))
    if (params.get('status')) rows = rows.filter((key) => key.status === params.get('status'))
    if (params.get('group_id')) rows = rows.filter((key) => key.group_id === Number(params.get('group_id')) || key.routing_group_ids.includes(Number(params.get('group_id'))))
    const sort = ['id', 'name', 'created_at', 'last_used_at', 'status'].includes(params.get('sort_by')) ? params.get('sort_by') : 'id'
    rows = [...rows].sort((a, b) => String(a[sort] || '').localeCompare(String(b[sort] || ''), undefined, { numeric: true }) * (params.get('sort_order') === 'asc' ? 1 : -1))
    return ok(response, paginate(rows, params))
  }
  if (localPath === '/keys' && method === 'POST') {
    const body = await readJson(request)
    if (!body.name || (body.custom_key && !String(body.custom_key).startsWith('sk-preview-'))) return error(response, 422, 'PREVIEW_KEY_VALIDATION', '请填写名称；自定义密钥仅接受 sk-preview- 前缀。 / A name and a synthetic custom-key prefix are required.')
    try {
      const id = fixtures.nextKeyId++
      let key = fixtures.makeKey(id, user.id, body.name, body.group_id || 1)
      key = updateKey(key, body)
      if (body.custom_key) key.key = body.custom_key
      key.created_at = new Date().toISOString()
      key.last_used_at = null
      key.last_used_ip = null
      key.quota_used = 0
      if (body.expires_in_days != null) {
        if (!Number.isFinite(body.expires_in_days) || body.expires_in_days < 1) throw new Error('Invalid preview expiry days.')
        key.expires_at = new Date(Date.now() + body.expires_in_days * 86400000).toISOString()
      }
      fixtures.keys.get(user.id).push(key)
      return ok(response, key)
    } catch (cause) {
      return error(response, 422, 'PREVIEW_KEY_VALIDATION', cause.message)
    }
  }
  const keyMatch = localPath.match(/^\/keys\/(\d+)$/)
  if (keyMatch) {
    const rows = fixtures.keys.get(user.id)
    const index = rows.findIndex((key) => key.id === Number(keyMatch[1]))
    if (index < 0) return error(response, 404, 'PREVIEW_KEY_NOT_FOUND', 'Synthetic API key not found.')
    if (method === 'GET') return ok(response, rows[index])
    if (method === 'DELETE') { rows.splice(index, 1); return ok(response, { message: 'Synthetic API key deleted from memory.' }) }
    if (method === 'PUT') {
      const body = await readJson(request)
      try { rows[index] = updateKey(rows[index], body); return ok(response, rows[index]) }
      catch (cause) { return error(response, 422, 'PREVIEW_KEY_VALIDATION', cause.message) }
    }
  }
  if (method === 'POST' && ['/usage/dashboard/api-keys-usage', '/admin/dashboard/api-keys-usage'].includes(localPath)) {
    const body = await readJson(request)
    const stats = {}
    for (const id of Array.isArray(body.api_key_ids) ? body.api_key_ids : []) {
      const keyRecords = records.filter((row) => row.api_key_id === id)
      const today = keyRecords.filter((row) => row.created_at.slice(0, 10) === new Date().toISOString().slice(0, 10))
      stats[id] = { api_key_id: id, today_actual_cost: aggregateUsage(today).total_actual_cost, total_actual_cost: aggregateUsage(keyRecords).total_actual_cost }
    }
    return ok(response, { stats })
  }
  if (method === 'GET' && localPath === '/usage') return ok(response, paginate(records, params))
  if (method === 'GET' && localPath === '/usage/errors') return ok(response, paginate([], params))
  if (method === 'GET' && localPath === '/usage/stats') return ok(response, aggregateUsage(records))
  if (method === 'GET' && localPath === '/usage/dashboard/stats') return ok(response, dashboardStats(user))
  if (method === 'GET' && ['/usage/dashboard/trend', '/usage/dashboard/models', '/usage/dashboard/snapshot-v2'].includes(localPath)) return ok(response, snapshot)
  const usageMatch = localPath.match(/^\/usage\/(\d+)$/)
  if (method === 'GET' && usageMatch) {
    const row = records.find((entry) => entry.id === Number(usageMatch[1]))
    return row ? ok(response, row) : error(response, 404, 'PREVIEW_USAGE_NOT_FOUND', 'Synthetic usage record not found.')
  }
  const dailyMatch = localPath.match(/^\/user\/api-keys\/(\d+)\/usage\/daily$/)
  if (method === 'GET' && dailyMatch) {
    const daily = buildUsageSnapshot(records.filter((row) => row.api_key_id === Number(dailyMatch[1])))
    return ok(response, { items: daily.trend, days: Number(params.get('days')) || 30, start_date: daily.start_date, end_date: daily.end_date })
  }
  if (method === 'GET' && ['/subscriptions', '/subscriptions/active'].includes(localPath)) return ok(response, [{ ...fixtures.subscription, user_id: user.id }])
  if (method === 'GET' && localPath === '/subscriptions/progress') return ok(response, [{ subscription: { ...fixtures.subscription, user_id: user.id }, progress: fixtures.subscriptionProgress }])
  if (method === 'GET' && localPath === '/subscriptions/summary') return ok(response, { active_count: 1, subscriptions: [{ id: 301, group_name: fixtures.groups[2].name, status: 'active', daily_progress: 12.4, weekly_progress: 16.84, monthly_progress: 11.77, expires_at: fixtures.subscription.expires_at, days_remaining: 23 }] })
  if (method === 'GET' && localPath === '/redeem/history') return ok(response, paginate(redemptionHistory.get(user.id) || [], params))
  if (method === 'POST' && localPath === '/redeem') {
    const body = await readJson(request)
    const amounts = { 'PREVIEW-TOPUP-10': 10, 'PREVIEW-TOPUP-50': 50 }
    const code = String(body.code || '').trim()
    if (!Object.hasOwn(amounts, code)) return error(response, 422, 'PREVIEW_CODE_ONLY', '本地预览仅接受 PREVIEW-TOPUP-10 或 PREVIEW-TOPUP-50；不处理真实兑换码。')
    const id = `${user.id}:${code}`
    if (redeemedCodes.has(id)) return error(response, 409, 'CODE_ALREADY_USED', '该合成兑换码已使用。')
    redeemedCodes.add(id)
    user.balance += amounts[code]
    const history = redemptionHistory.get(user.id) || []
    history.unshift({ id: 900 + history.length, code, type: 'balance', value: amounts[code], status: 'used', used_at: new Date().toISOString(), created_at: new Date().toISOString() })
    redemptionHistory.set(user.id, history)
    return ok(response, { message: '合成兑换成功', type: 'balance', value: amounts[code], new_balance: user.balance })
  }
  if (method === 'GET' && localPath === '/payment/config') return ok(response, fixtures.paymentConfig)
  if (method === 'GET' && localPath === '/payment/plans') return ok(response, fixtures.plans)
  if (method === 'GET' && localPath === '/payment/checkout-info') return ok(response, fixtures.checkoutInfo)
  if (method === 'GET' && localPath === '/payment/limits') return ok(response, { methods: {}, global_min: 1, global_max: 100 })
  if (method === 'GET' && localPath === '/payment/orders/refund-eligible-providers') return ok(response, { provider_instance_ids: [] })
  if (method === 'GET' && localPath === '/payment/orders/my') return ok(response, paginate(fixtures.orders.filter((order) => !params.get('status') || order.status === params.get('status')).map((order) => ({ ...order, user_id: user.id })), params))
  const orderMatch = localPath.match(/^\/payment\/orders\/(\d+)$/)
  if (method === 'GET' && orderMatch) {
    const order = fixtures.orders.find((entry) => entry.id === Number(orderMatch[1]))
    return order ? ok(response, { ...order, user_id: user.id }) : error(response, 404, 'PREVIEW_ORDER_NOT_FOUND', 'Synthetic order not found.')
  }
  if (method === 'GET' && localPath === '/channel-monitors') return ok(response, { items: [] })
  const monitorMatch = localPath.match(/^\/(?:admin\/)?channel-monitor-v2\/(dimensions|snapshot|matrix|models|errors|users|config)$/)
  if (method === 'GET' && monitorMatch) {
    const monitor = fixtures.monitor
    switch (monitorMatch[1]) {
      case 'config': return ok(response, monitor.config)
      case 'snapshot': return ok(response, monitor)
      case 'dimensions': return ok(response, { platforms: fixtures.groups.map((group) => ({ value: group.platform, label: group.platform, request_count: 28 })), groups: fixtures.groups.map((group) => ({ id: group.id, name: group.name, platform: group.platform, request_count: 28 })), models: fixtures.plaza.groups.map((group) => ({ value: group.models[0].name, label: group.models[0].name, platform: group.platform, request_count: 28 })) })
      case 'matrix': return ok(response, { coverage: monitor.coverage, group_by: params.get('group_by') || 'platform', items: fixtures.plaza.groups.map((group) => ({ platform: group.platform, group_id: group.id, group_name: group.name, model: group.models[0].name, metrics: monitor.metrics, health: monitor.health, buckets: monitor.trend })) })
      case 'models': return ok(response, { coverage: monitor.coverage, items: fixtures.plaza.groups.map((group) => ({ platform: group.platform, model: group.models[0].name, metrics: monitor.metrics, health: monitor.health })) })
      default: return ok(response, { coverage: monitor.coverage, items: [] })
    }
  }
  if (method === 'GET' && localPath === '/admin/compliance') return ok(response, { required: false, version: 'preview', document_path_zh: '', document_path_en: '', document_url_zh: '', document_url_en: '', ack_phrase_zh: '', ack_phrase_en: '' })
  if (method === 'GET' && localPath === '/admin/settings') return ok(response, { ...publicSettings, ops_monitoring_enabled: false, purchase_subscription_url: '', purchase_subscription_enabled: true })
  if (method === 'GET' && localPath === '/admin/system/version') return ok(response, { version: 'local-preview' })
  if (method === 'GET' && localPath === '/admin/system/check-updates') return ok(response, { current_version: 'local-preview', latest_version: 'local-preview', has_update: false, cached: true, build_type: 'source' })
  if (method === 'GET' && localPath === '/admin/dashboard/stats') return ok(response, adminStats(user))
  if (method === 'GET' && ['/admin/dashboard/trend', '/admin/dashboard/models', '/admin/dashboard/groups', '/admin/dashboard/snapshot-v2'].includes(localPath)) return ok(response, { ...snapshot, stats: adminStats(user), users_trend: [] })
  if (method === 'GET' && localPath === '/admin/dashboard/upstream-balances') return ok(response, { total: 0, unit: 'USD', consumption: { last_24h: 0, yesterday: 0, today: 0, total: 0, unit: 'USD' }, items: [] })
  if (method === 'GET' && localPath === '/admin/dashboard/realtime') return ok(response, { active_requests: 0, requests_per_minute: 0, average_response_time: 0, error_rate: 0 })
  return unavailable(response)
}

const apiServer = createHttpServer((request, response) => {
  handleApi(request, response).catch(() => {
    // Never log request bodies, keys, authorization headers or passwords.
    if (!response.headersSent) error(response, 400, 'PREVIEW_INVALID_REQUEST', 'Invalid request in synthetic preview.')
    else response.end()
  })
})
apiServer.requestTimeout = 10000
apiServer.headersTimeout = 10000

// This guard exists only in transformed development HTML. CSP blocks cross-origin resource
// requests; click/window.open guards also prevent accidental external navigation from examples.
const browserGuard = `(() => {
  const origin = ${JSON.stringify(PREVIEW_ORIGIN)};
  const allowed = value => { try { const url = new URL(String(value || ''), origin); return url.origin === origin && ['http:', 'ws:'].includes(url.protocol) || url.protocol === 'blob:'; } catch { return false; } };
  const notice = () => { const el = document.getElementById('patrick-preview-badge'); if (el) { el.textContent = '本地预览：外部操作不可用 / External action unavailable'; setTimeout(() => { el.textContent = '本地预览 · 合成数据 / Local preview · synthetic data'; }, 4000); } };
  document.addEventListener('click', event => { const anchor = event.target instanceof Element ? event.target.closest('a[href]') : null; if (anchor && !allowed(anchor.href)) { event.preventDefault(); event.stopImmediatePropagation(); notice(); } }, true);
  document.addEventListener('submit', event => { const form = event.target; if (form instanceof HTMLFormElement && !allowed(form.action)) { event.preventDefault(); event.stopImmediatePropagation(); notice(); } }, true);
  const originalOpen = window.open.bind(window);
  window.open = (url, ...rest) => { if (!allowed(url)) { notice(); return null; } return originalOpen(url, ...rest); };
})();`

function previewHtml() {
  return {
    name: 'patrick-synthetic-preview-only',
    transformIndexHtml: {
      order: 'pre',
      handler(html) {
        const config = JSON.stringify(publicSettings).replace(/</g, '\\u003c')
        return html.replace(/<title>[^<]*<\/title>/i, '<title>patrickapi · Local synthetic preview</title>')
          .replace('</head>', `<script>window.__APP_CONFIG__=${config};</script><script>${browserGuard}</script></head>`)
          .replace('</body>', '<div id="patrick-preview-badge" role="status" style="position:fixed;z-index:2147483647;left:50%;bottom:8px;transform:translateX(-50%);max-width:calc(100vw - 24px);padding:6px 12px;border:1px solid #c6d7e7;border-radius:99px;background:#eff6fdf2;color:#365975;font:10px/1.5 system-ui,sans-serif;box-shadow:0 3px 16px #24486a14;text-align:center;pointer-events:none">本地预览 · 合成数据 / Local preview · synthetic data</div></body>')
      },
    },
    configureServer(server) {
      server.middlewares.use((request, response, next) => {
        if (request.headers.host !== `${HOST}:${VITE_PORT}`) {
          response.statusCode = 403
          response.end('Local preview host only.')
          return
        }
        next()
      })
    },
  }
}

let viteServer
let stopping = false
async function closeServers() {
  if (stopping) return
  stopping = true
  sessions.clear()
  refreshSessions.clear()
  if (viteServer) await viteServer.close()
  if (apiServer.listening) {
    apiServer.closeAllConnections()
    await new Promise((done) => apiServer.close(done))
  }
}

for (const signal of ['SIGINT', 'SIGTERM']) {
  process.once(signal, () => {
    closeServers().then(() => { process.exitCode = 0 }).catch(() => { process.exitCode = 1 })
  })
}

try {
  await new Promise((done, reject) => {
    apiServer.once('error', reject)
    apiServer.listen(API_PORT, HOST, () => { apiServer.off('error', reject); done() })
  })
  viteServer = await createViteServer({
    root: frontendDirectory,
    configFile: false,
    envFile: false,
    envDir: devDirectory,
    envPrefix: '__PATRICK_PREVIEW_NO_ENV__',
    mode: 'preview',
    cacheDir: resolve(frontendDirectory, 'node_modules/.vite-user-preview'),
    plugins: [vue(), previewHtml()],
    resolve: { alias: { '@': resolve(frontendDirectory, 'src'), 'vue-i18n': 'vue-i18n/dist/vue-i18n.runtime.esm-bundler.js' } },
    define: { __INTLIFY_JIT_COMPILATION__: true, 'import.meta.env.VITE_API_BASE_URL': JSON.stringify('/api/v1'), 'import.meta.env.VITE_WS_BASE_URL': JSON.stringify(`ws://${HOST}:${VITE_PORT}`) },
    server: {
      host: HOST,
      port: VITE_PORT,
      strictPort: true,
      open: false,
      cors: false,
      hmr: { host: HOST, port: VITE_PORT },
      fs: { strict: true, allow: [frontendDirectory], deny: ['.env', '.env.*', '*.{crt,pem,key}', '**/.git/**'] },
      headers: {
        'Content-Security-Policy': `default-src 'self'; connect-src 'self' ws://${HOST}:${VITE_PORT}; img-src 'self' data: blob:; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; font-src 'self' data:; media-src 'self' data: blob:; worker-src 'self' blob:; object-src 'none'; base-uri 'self'; form-action 'self'; frame-src 'none'`,
        'Referrer-Policy': 'no-referrer',
        'Cache-Control': 'no-store',
      },
      proxy: Object.fromEntries(['/api', '/v1', '/setup', '/health'].map((prefix) => [prefix, { target: API_ORIGIN, changeOrigin: true, ws: false }])),
    },
  })
  await viteServer.listen()
  console.log(`Synthetic preview: ${PREVIEW_ORIGIN}/home`)
  console.log(`Mock API: ${API_ORIGIN} (loopback only; no upstream connections)`)
  console.log('Accounts: preview@example.test / admin@example.test. Demo password: see dev/README.md.')
  console.log('Press Ctrl+C to stop both services. All mock mutations are memory-only.')
} catch (cause) {
  console.error(`Preview startup failed (${cause.code || cause.name || 'error'}). Ports 3410 and 3411 must both be available.`)
  await closeServers()
  process.exitCode = 1
}
