import { describe, expect, it } from 'vitest'
import { parseUsageRoute, usageDrilldownQuery, usageFiltersQuery, validUsageDate, type UsageRouteFilters } from '../usageRoute'

const defaults: UsageRouteFilters = { start: '2026-10-01', end: '2026-10-11', key: '', group: '', model: '', requestType: '', billingType: '', billingMode: '', compaction: '', status: '', category: '' }

describe('user usage route filters', () => {
  it('restores dates, model and key from a reloadable URL', () => {
    expect(parseUsageRoute({ start_date: '2026-09-05', end_date: '2026-09-07', model: 'claude-sonnet', api_key_id: '42' }, defaults)).toEqual({ ...defaults, start: '2026-09-05', end: '2026-09-07', model: 'claude-sonnet', key: '42' })
  })
  it('builds chart drilldowns without retaining unrelated list conditions', () => {
    expect(usageDrilldownQuery({ start: '2026-10-04', end: '2026-10-04', model: 'gpt-5' })).toEqual({ start_date: '2026-10-04', end_date: '2026-10-04', model: 'gpt-5' })
    expect(parseUsageRoute({ model: 'gpt-5' }, defaults)).toEqual({ ...defaults, model: 'gpt-5' })
  })
  it('rejects impossible/reversed dates and ambiguous or invalid key identifiers', () => {
    expect(validUsageDate('2026-02-30')).toBe(false)
    expect(validUsageDate('2024-02-29')).toBe(true)
    for (const api_key_id of ['-1', '0', '1.5', '9007199254740992', ['4', '5']]) {
      const restored = parseUsageRoute({ start_date: '2026-10-08', end_date: '2026-10-01', api_key_id }, defaults)
      expect(restored).toEqual(defaults)
    }
  })
  it('preserves explicitly applied filters for forward/back navigation', () => {
    const applied = { ...defaults, group: '7', billingType: '0', requestType: 'stream', compaction: 'false' }
    expect(parseUsageRoute(usageFiltersQuery(applied) as Record<string, string>, defaults)).toEqual(applied)
  })
})
