import { afterEach, describe, expect, it, vi } from 'vitest'
import { formatMultiplier, getKeyBillingInfo, isPeakRateActive } from '@/api/keyBilling'

describe('keyBilling API client', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('rejects empty key without issuing fetch', async () => {
    const fetchSpy = vi.spyOn(globalThis, 'fetch')
    const res = await getKeyBillingInfo('   ')
    expect(res.data).toBeNull()
    expect(res.error).toBe('Missing API key')
    expect(fetchSpy).not.toHaveBeenCalled()
  })

  it('parses valid /v1/sub2api/billing payload and detects active peak rate', async () => {
    const mockPayload = {
      object: 'sub2api.key_billing',
      schema_version: 1,
      billing_scope: 'token',
      group_rate_multiplier: 1.0,
      user_rate_multiplier: 0.8,
      resolved_rate_multiplier: 0.8,
      peak_rate_enabled: true,
      peak_start: '18:00',
      peak_end: '23:00',
      peak_rate_multiplier: 1.5,
      applied_peak_multiplier: 1.5,
      effective_rate_multiplier: 1.2,
      timezone: 'Asia/Shanghai',
      observed_at: '2026-11-19T12:00:00Z'
    }

    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(
      new Response(JSON.stringify(mockPayload), {
        status: 200,
        headers: { 'Content-Type': 'application/json' }
      })
    )

    const res = await getKeyBillingInfo('sk-demo-key')
    expect(res.status).toBe(200)
    expect(res.data?.effective_rate_multiplier).toBe(1.2)
    expect(isPeakRateActive(res.data)).toBe(true)
    expect(formatMultiplier(res.data?.effective_rate_multiplier)).toBe('1.2x')
  })

  it('handles non-OK responses gracefully', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(
      new Response(JSON.stringify({ error: { message: 'Invalid API key' } }), {
        status: 401,
        headers: { 'Content-Type': 'application/json' }
      })
    )

    const res = await getKeyBillingInfo('sk-bad')
    expect(res.data).toBeNull()
    expect(res.status).toBe(401)
    expect(res.error).toBe('Invalid API key')
  })
})
