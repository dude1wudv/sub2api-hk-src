import { describe, expect, it } from 'vitest'
import type { UsageLog } from '@/types'
import {
  aggregateUsageMonths,
  escapeUsageCsv,
  formatUsageRate,
  usageCacheRate,
  usageLogsCsv,
  usageOutputRate,
  usageOutputStageRate,
} from '../portalUsage'

function usage(overrides: Record<string, unknown> = {}) {
  return {
    id: 1,
    billing_mode: 'token',
    input_tokens: 30,
    cache_creation_tokens: 10,
    cache_read_tokens: 60,
    output_tokens: 80,
    duration_ms: 5000,
    first_token_ms: 1000,
    actual_cost: 0.2,
    total_cost: 0.3,
    rate_multiplier: 1,
    ...overrides,
  } as unknown as UsageLog
}

describe('portal usage calculations', () => {
  it('uses all input and cache tokens as the cache-hit-rate denominator', () => {
    expect(usageCacheRate(usage())).toBe(60)
    expect(usageCacheRate(usage({ input_tokens: 0, cache_creation_tokens: 0, cache_read_tokens: 0 }))).toBeNull()
  })

  it('uses the post-first-token duration for output-stage throughput and total duration for TPS', () => {
    expect(usageOutputStageRate(usage())).toBe(20)
    expect(usageOutputRate(usage())).toBe(16)
    expect(usageOutputStageRate(usage({ first_token_ms: 5000 }))).toBeNull()
    expect(usageOutputStageRate(usage({ first_token_ms: null }))).toBeNull()
    expect(usageOutputRate(usage({ duration_ms: 0 }))).toBeNull()
    expect(formatUsageRate(null)).toBe('—')
  })

  it.each(['per_request', 'image', 'video'])(
    'shows token-derived metrics as unavailable for %s billing',
    billing_mode => {
      const row = usage({ billing_mode })
      expect(usageCacheRate(row)).toBeNull()
      expect(usageOutputStageRate(row)).toBeNull()
      expect(usageOutputRate(row)).toBeNull()
    },
  )

  it('neutralizes spreadsheet formulas after leading whitespace and quotes delimiters', () => {
    expect(escapeUsageCsv('\t =1+1')).toBe('"\'\t =1+1"')
    expect(escapeUsageCsv('a,"b"')).toBe('"a,""b"""')
    const csv = usageLogsCsv([usage({ request_id: '+SUM(A1:A2)' })], true)
    expect(csv.startsWith('\uFEFF"Time","Request ID"')).toBe(true)
    expect(csv).toContain('"\'+SUM(A1:A2)"')
  })

  it('aggregates trend data only into the requested rolling year', () => {
    const months = aggregateUsageMonths([
      { date: '2025-04-02', actual_cost: 1.5, requests: 2, total_tokens: 100 } as any,
      { date: '2024-01-01', actual_cost: 9, requests: 9, total_tokens: 9 } as any,
    ], new Date(2025, 3, 17))
    expect(months).toHaveLength(12)
    expect(months[11]).toMatchObject({ key: '2025-04', cost: 1.5, requests: 2, tokens: 100 })
    expect(months.slice(0, 11).reduce((total, month) => total + month.cost, 0)).toBe(0)
  })
})
