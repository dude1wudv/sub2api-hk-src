export interface KeyBillingInfo {
  object: string
  schema_version: number
  billing_scope: string
  group_rate_multiplier: number
  user_rate_multiplier?: number | null
  resolved_rate_multiplier: number
  peak_rate_enabled: boolean
  peak_start?: string | null
  peak_end?: string | null
  peak_rate_multiplier?: number | null
  applied_peak_multiplier?: number | null
  effective_rate_multiplier: number
  timezone?: string | null
  observed_at: string
}

export interface KeyBillingResult {
  data: KeyBillingInfo | null
  error?: string
  status?: number
}

/**
 * Fetches effective token billing multiplier and peak-hour status for a given API key
 * via GET /v1/sub2api/billing. Never persists or logs the raw key.
 */
export async function getKeyBillingInfo(apiKey: string, signal?: AbortSignal): Promise<KeyBillingResult> {
  const trimmed = apiKey.trim()
  if (!trimmed) {
    return { data: null, error: 'Missing API key' }
  }

  try {
    const response = await fetch('/v1/sub2api/billing', {
      method: 'GET',
      headers: {
        Authorization: `Bearer ${trimmed}`,
        Accept: 'application/json'
      },
      cache: 'no-store',
      signal
    })

    if (!response.ok) {
      let message = `HTTP ${response.status}`
      try {
        const body = await response.json()
        message = body?.error?.message || body?.message || message
      } catch {
        // fallback to status code
      }
      return { data: null, error: message, status: response.status }
    }

    const payload = (await response.json()) as KeyBillingInfo
    return { data: payload, status: response.status }
  } catch (err: unknown) {
    if (err instanceof DOMException && err.name === 'AbortError') {
      return { data: null, error: 'Aborted' }
    }
    return {
      data: null,
      error: err instanceof Error ? err.message : 'Network error'
    }
  }
}

export function isPeakRateActive(info: KeyBillingInfo | null | undefined): boolean {
  if (!info || !info.peak_rate_enabled) return false
  return Number(info.applied_peak_multiplier ?? 1) > 1
}

export function formatMultiplier(value: number | null | undefined): string {
  const num = Number(value ?? 1)
  if (!Number.isFinite(num)) return '1x'
  const rounded = Number(num.toFixed(4))
  return `${rounded}x`
}
