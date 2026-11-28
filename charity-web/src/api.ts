export type PriceCurrency = 'CNY' | 'USD'

export interface CharityPublicModel {
  name: string
  platform: string
  price_currency: PriceCurrency
  input_price_per_million?: number | null
  output_price_per_million?: number | null
  cache_read_per_million?: number | null
  cache_write_per_million?: number | null
  supports_vision?: boolean
  supports_reasoning?: boolean
}

export interface CharityPublicGroup {
  id: number
  group_id: number
  name: string
  description: string
  platform: string
  enabled: boolean
  daily_limit_usd: number | null
  used_today_usd: number
  remaining_today_usd: number | null
  request_count_today: number
  max_concurrency: number
  current_concurrency: number
  public_api_key: string
  last_rotated_date: string
  last_rotated_at?: string | null
  exhausted: boolean
  exhausted_reason?: string
  models: CharityPublicModel[]
}

export interface CharityPublicIPStatus {
  masked_ip: string
  used_today_usd: number
  request_count_today: number
  daily_limit_usd: number
  remaining_today_usd: number | null
  max_concurrency: number
  current_concurrency: number
  rpm_limit: number
}

export interface CharityPublicOverview {
  enabled: boolean
  site_title: string
  site_notice: string
  usage_date: string
  timezone: string
  auto_rotate_daily: boolean
  next_reset_at: string
  seconds_until_reset: number
  daily_budget_usd: number
  used_today_usd: number
  remaining_today_usd: number | null
  request_count_today: number
  max_concurrency: number
  current_concurrency: number
  pool_exhausted: boolean
  groups: CharityPublicGroup[]
  ip_status: CharityPublicIPStatus
}

export interface CharityKeyLookupResult {
  found: boolean
  status: 'active' | 'rotated_expired' | 'disabled' | 'not_found'
  group_id?: number
  group_name?: string
  platform?: string
  usage_date: string
  pool_daily_budget_usd: number
  pool_used_today_usd: number
  pool_remaining_usd: number | null
  group_daily_limit_usd: number | null
  group_used_today_usd: number
  group_remaining_usd: number | null
  last_rotated_date?: string
  next_reset_at: string
}

interface Envelope<T> {
  code: number
  message: string
  data: T
}

async function parseJSON<T>(res: Response): Promise<T> {
  const payload = (await res.json()) as Envelope<T> & { error?: { message?: string } }
  if (!res.ok || (typeof payload.code === 'number' && payload.code !== 0)) {
    throw new Error(payload.message || payload.error?.message || `请求失败 (${res.status})`)
  }
  return payload.data
}

export async function fetchCharityOverview(): Promise<CharityPublicOverview> {
  const res = await fetch('/api/v1/charity/overview', {
    headers: { Accept: 'application/json' },
    cache: 'no-store'
  })
  return parseJSON<CharityPublicOverview>(res)
}

export async function lookupCharityKeyStatus(apiKey: string): Promise<CharityKeyLookupResult> {
  const res = await fetch('/api/v1/charity/key-status', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Accept: 'application/json'
    },
    body: JSON.stringify({ api_key: apiKey.trim() })
  })
  return parseJSON<CharityKeyLookupResult>(res)
}
