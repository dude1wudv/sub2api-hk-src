import type { LocationQuery, LocationQueryRaw } from 'vue-router'

export interface UsageRouteFilters {
  start: string
  end: string
  key: string
  group: string
  model: string
  requestType: string
  billingType: string
  billingMode: string
  compaction: string
  status: string
  category: string
}

const fields = {
  start: 'start_date', end: 'end_date', key: 'api_key_id', group: 'group_id', model: 'model',
  requestType: 'request_type', billingType: 'billing_type', billingMode: 'billing_mode',
  compaction: 'compaction', status: 'status_code', category: 'category',
} as const

function scalar(value: LocationQuery[string]): string {
  return typeof value === 'string' ? value.trim() : ''
}

export function validUsageDate(value: string): boolean {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(value)) return false
  const parsed = new Date(`${value}T00:00:00Z`)
  return Number.isFinite(parsed.getTime()) && parsed.toISOString().slice(0, 10) === value
}

/** Start from defaults so a chart drilldown never inherits unrelated filters. */
export function parseUsageRoute(query: LocationQuery, defaults: UsageRouteFilters): UsageRouteFilters {
  const filters = { ...defaults }
  for (const [field, param] of Object.entries(fields) as Array<[keyof UsageRouteFilters, string]>) {
    const value = scalar(query[param])
    if (value) filters[field] = value
  }
  if (!validUsageDate(filters.start) || !validUsageDate(filters.end) || filters.start > filters.end) {
    filters.start = defaults.start
    filters.end = defaults.end
  }
  for (const field of ['key', 'group'] as const) {
    if (!/^[1-9]\d*$/.test(filters[field]) || !Number.isSafeInteger(Number(filters[field]))) filters[field] = ''
  }
  if (!['', 'sync', 'stream', 'ws_v2', 'live', 'cyber'].includes(filters.requestType)) filters.requestType = ''
  if (!['', '0', '1'].includes(filters.billingType)) filters.billingType = ''
  if (!['', 'token', 'per_request', 'image', 'video'].includes(filters.billingMode)) filters.billingMode = ''
  if (!['', 'true', 'false'].includes(filters.compaction)) filters.compaction = ''
  if (filters.status && !/^[1-5]\d\d$/.test(filters.status)) filters.status = ''
  return filters
}

export function usageFiltersQuery(filters: UsageRouteFilters): LocationQueryRaw {
  return Object.fromEntries(Object.entries(fields).flatMap(([field, param]) => {
    const value = filters[field as keyof UsageRouteFilters].trim()
    return value ? [[param, value]] : []
  }))
}

export interface UsageDrilldown { start: string; end: string; model?: string; key?: number }
export function usageDrilldownQuery({ start, end, model, key }: UsageDrilldown): LocationQueryRaw {
  const query: LocationQueryRaw = {}
  if (validUsageDate(start) && validUsageDate(end) && start <= end) {
    query.start_date = start
    query.end_date = end
  }
  if (model?.trim()) query.model = model.trim()
  if (Number.isSafeInteger(key) && (key ?? 0) > 0) query.api_key_id = String(key)
  return query
}
