import type { TrendDataPoint, UsageLog, UserErrorRequest, UsageRequestType } from '@/types'
import { getDisplayBillingMode, isImageUsage } from '@/utils/billingMode'
import { resolveUsageRequestType } from '@/utils/usageRequestType'
import { numericRequestTypeKind } from '@/utils/errorBadges'

export function finiteNumber(value: unknown): number {
  return typeof value === 'number' && Number.isFinite(value) ? value : 0
}

export function localDate(value: Date): string {
  return `${value.getFullYear()}-${String(value.getMonth() + 1).padStart(2, '0')}-${String(value.getDate()).padStart(2, '0')}`
}

export function usageMonthWindow(now = new Date()) {
  return {
    start: localDate(new Date(now.getFullYear(), now.getMonth() - 11, 1)),
    end: localDate(now),
    currentStart: localDate(new Date(now.getFullYear(), now.getMonth(), 1)),
  }
}

export interface PortalUsageMonth {
  key: string
  label: string
  cost: number
  requests: number
  tokens: number
}

/** Daily trend dates already use the timezone supplied to the API. */
export function aggregateUsageMonths(trend: TrendDataPoint[], now = new Date()): PortalUsageMonth[] {
  const months = Array.from({ length: 12 }, (_, index) => {
    const date = new Date(now.getFullYear(), now.getMonth() - 11 + index, 1)
    return { key: localDate(date).slice(0, 7), label: `${date.getMonth() + 1}月`, cost: 0, requests: 0, tokens: 0 }
  })
  const lookup = new Map(months.map(month => [month.key, month]))
  for (const point of trend) {
    const month = lookup.get(point.date.slice(0, 7))
    if (!month) continue
    month.cost += finiteNumber(point.actual_cost)
    month.requests += finiteNumber(point.requests)
    month.tokens += finiteNumber(point.total_tokens)
  }
  return months
}

export function usageBillingMode(row: UsageLog): string {
  return getDisplayBillingMode(row) || 'token'
}

export function usageBillingLabel(row: UsageLog, english = false): string {
  const labels: Record<string, string> = english ? { token: 'Token', per_request: 'Per request', image: 'Image', video: 'Video' } : { token: 'Token', per_request: '按次', image: '图片', video: '视频' }
  return labels[usageBillingMode(row)] || usageBillingMode(row)
}

export function isTokenUsage(row: UsageLog): boolean {
  return usageBillingMode(row) === 'token' && !isImageUsage(row)
}

export const portalUsageColumns = [
  { key: 'created_at', label: '时间', default: true, fixed: true, sortable: true },
  { key: 'type', label: '类型', default: true },
  { key: 'model', label: '模型', default: true, sortable: true },
  { key: 'tokens', label: 'Token', default: true },
  { key: 'cost', label: '费用', default: true },
  { key: 'latency', label: '延迟', default: true },
  { key: 'api_key', label: 'API Key' },
  { key: 'group', label: '分组' },
  { key: 'endpoint', label: '端点' },
  { key: 'ip_address', label: 'IP' },
  { key: 'reasoning_effort', label: '推理强度' },
  { key: 'billing_mode', label: '计费模式' },
  { key: 'user_agent', label: 'User-Agent' },
] as const
export type PortalUsageColumnKey = (typeof portalUsageColumns)[number]['key']

export const portalErrorColumns = [
  { key: 'created_at', label: '时间', default: true, fixed: true, sortable: true },
  { key: 'type', label: '类型', default: true },
  { key: 'model', label: '模型', default: true, sortable: true },
  { key: 'status_code', label: '状态', default: true, fixed: true, sortable: true },
  { key: 'category', label: '分类', default: true },
  { key: 'message', label: '消息', default: true },
  { key: 'key_name', label: 'API Key' },
  { key: 'endpoint', label: '端点' },
  { key: 'client_ip', label: 'IP' },
  { key: 'group_name', label: '分组' },
  { key: 'platform', label: '平台' },
  { key: 'user_agent', label: 'User-Agent' },
] as const
export type PortalErrorColumnKey = (typeof portalErrorColumns)[number]['key']

export function usageColumnLabel(key: string, english = false): string {
  const labels: Record<string, string> = { created_at: 'Time', type: 'Type', model: 'Model', tokens: 'Token', cost: 'Cost', latency: 'Latency', api_key: 'API Key', group: 'Group', endpoint: 'Endpoint', ip_address: 'IP', reasoning_effort: 'Reasoning effort', billing_mode: 'Billing mode', user_agent: 'User-Agent', status_code: 'Status', category: 'Category', message: 'Message', key_name: 'API Key', client_ip: 'IP', group_name: 'Group', platform: 'Platform' }
  return english ? labels[key] || key : [...portalUsageColumns, ...portalErrorColumns].find(column => column.key === key)?.label || key
}

export function usageCacheRate(row: UsageLog): number | null {
  if (!isTokenUsage(row)) return null
  const denominator = finiteNumber(row.input_tokens) + finiteNumber(row.cache_creation_tokens) + finiteNumber(row.cache_read_tokens)
  return denominator > 0 ? finiteNumber(row.cache_read_tokens) / denominator * 100 : null
}

export function usageOutputStageRate(row: UsageLog): number | null {
  if (!isTokenUsage(row) || row.duration_ms == null || row.first_token_ms == null) return null
  const duration = row.duration_ms - row.first_token_ms
  return Number.isFinite(duration) && duration > 0 ? finiteNumber(row.output_tokens) / (duration / 1000) : null
}

export function usageOutputRate(row: UsageLog): number | null {
  if (!isTokenUsage(row) || row.duration_ms == null || !Number.isFinite(row.duration_ms) || row.duration_ms <= 0) return null
  return finiteNumber(row.output_tokens) / (row.duration_ms / 1000)
}

export function formatUsageRate(value: number | null): string { return value == null ? '—' : `${value.toFixed(1)} tok/s` }
export function formatUsagePercent(value: number | null): string { return value == null ? '—' : `${value.toFixed(1)}%` }
export function formatUsageNumber(value: unknown): string { return finiteNumber(value).toLocaleString('zh-CN') }
export function formatUsageMoney(value: unknown, digits = 6): string { return `$${finiteNumber(value).toFixed(digits)}` }
export function formatUsageDuration(value: number | null | undefined): string {
  if (value == null || !Number.isFinite(value) || value < 0) return '—'
  if (value < 1000) return `${Math.round(value)}ms`
  if (value < 60000) return `${(value / 1000).toFixed(2)}s`
  const seconds = Math.round(value / 1000)
  return seconds < 3600 ? `${Math.floor(seconds / 60)}m ${seconds % 60}s` : `${Math.floor(seconds / 3600)}h ${Math.floor(seconds % 3600 / 60)}m`
}

export function usageRequestLabel(value: UsageRequestType | null | undefined, english = false): string {
  return (english ? { sync: 'Sync', stream: 'Stream', ws_v2: 'WS', live: 'Live', cyber: 'Cyber', unknown: 'Unknown' } : { sync: '同步', stream: '流式', ws_v2: 'WS', live: 'Live', cyber: 'Cyber', unknown: '未知' })[value || 'unknown']
}
export function usageRowRequestLabel(row: UsageLog, english = false): string { return usageRequestLabel(resolveUsageRequestType(row), english) }
export function usageErrorRequestLabel(row: UserErrorRequest, english = false): string { return usageRequestLabel(numericRequestTypeKind(row.request_type, row.stream), english) }

/** Quote every CSV cell and neutralize formula prefixes, including hidden leading whitespace. */
export function escapeUsageCsv(value: unknown): string {
  const text = value == null ? '' : String(value)
  // Spreadsheet apps can ignore leading whitespace and control bytes before a formula.
  const prefix = Array.from(text).find(character => character.charCodeAt(0) > 32 && !/\s/.test(character)) || ''
  const safe = '=+@-'.includes(prefix) && prefix !== '' || /^[\t\r\n]/.test(text) ? `'${text}` : text
  return `"${safe.replace(/"/g, '""')}"`
}

export function usageLogsCsv(rows: UsageLog[], english = false): string {
  const headers = english ? ['Time', 'Request ID', 'API Key', 'Group', 'Model', 'Request type', 'Billing mode', 'Billing source', 'Reasoning effort', 'Inbound endpoint', 'IP', 'Input tokens', 'Output tokens', 'Cache read tokens', 'Cache write tokens', '5m cache write', '1h cache write', 'Cache hit rate %', 'Output stage tok/s', 'Output TPS', 'Billed cost USD', 'Standard cost USD', 'Rate multiplier', 'First token ms', 'Duration ms', 'Image count', 'Image size', 'Image input tokens', 'Image output tokens', 'User-Agent'] : ['时间', '请求 ID', 'API Key', '分组', '模型', '请求类型', '计费模式', '扣费方式', '推理强度', '入站端点', 'IP', '输入 Token', '输出 Token', '缓存读取 Token', '缓存写入 Token', '5m 缓存写入', '1h 缓存写入', '缓存命中率 %', '输出阶段 tok/s', '输出 TPS', '实际扣费 USD', '标准费用 USD', '计费倍率', '首字 ms', '总耗时 ms', '图片数量', '图片尺寸', '图片输入 Token', '图片输出 Token', 'User-Agent']
  const values = rows.map(row => [
    row.created_at, row.request_id, row.api_key?.name || '', row.group?.name || '', row.model, usageRowRequestLabel(row, english), usageBillingLabel(row, english), row.billing_type === 1 ? english ? 'Subscription' : '订阅' : english ? 'Balance' : '余额', row.reasoning_effort || '', row.inbound_endpoint || '', row.ip_address || '',
    row.input_tokens, row.output_tokens, row.cache_read_tokens, row.cache_creation_tokens, row.cache_creation_5m_tokens, row.cache_creation_1h_tokens,
    usageCacheRate(row), usageOutputStageRate(row), usageOutputRate(row), finiteNumber(row.actual_cost).toFixed(8), finiteNumber(row.total_cost).toFixed(8), row.rate_multiplier, row.first_token_ms, row.duration_ms,
    row.image_count, row.image_size || '', row.image_input_tokens, row.image_output_tokens, row.user_agent || '',
  ])
  return '\uFEFF' + [headers, ...values].map(row => row.map(escapeUsageCsv).join(',')).join('\r\n')
}

export function usageErrorsCsv(rows: UserErrorRequest[], english = false): string {
  const headers = english ? ['Time', 'API Key', 'Model', 'Request type', 'Status code', 'Category', 'Message', 'Endpoint', 'IP', 'Group', 'Platform', 'User-Agent'] : ['时间', 'API Key', '模型', '请求类型', '状态码', '分类', '消息', '端点', 'IP', '分组', '平台', 'User-Agent']
  return '\uFEFF' + [headers, ...rows.map(row => [row.created_at, row.key_name, row.model, usageErrorRequestLabel(row, english), row.status_code, row.category, row.message, row.inbound_endpoint, row.client_ip || '', row.group_name || '', row.platform, row.user_agent || ''])]
    .map(row => row.map(escapeUsageCsv).join(',')).join('\r\n')
}

export function isUsageAbort(cause: unknown): boolean {
  return !!cause && typeof cause === 'object' && (('name' in cause && cause.name === 'AbortError') || ('code' in cause && cause.code === 'ERR_CANCELED'))
}
