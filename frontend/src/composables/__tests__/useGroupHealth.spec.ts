import { afterEach, describe, expect, it, vi } from 'vitest'
import { latestGroupHealth } from '../useGroupHealth'
import type { MonitorMatrixRow } from '@/api/channelMonitorV2'

vi.mock('@/api/channelMonitorV2', () => ({ getMatrix: vi.fn() }))

describe('latestGroupHealth', () => {
  afterEach(() => vi.useRealTimers())
  it('uses the most recent bucket, never the historical row aggregate', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-11-15T12:00:30Z'))
    const row = {
      health: { overall: 'critical' },
      buckets: [
        { bucket_start: '2026-11-15T12:00:00Z', health: { overall: 'healthy' } },
        { bucket_start: '2026-11-15T11:59:00Z', health: { overall: 'critical' } },
      ],
    } as MonitorMatrixRow
    expect(latestGroupHealth(row, '2026-11-15T12:00:20Z', 60)).toBe('healthy')
    expect(latestGroupHealth(row, '2026-11-15T11:50:20Z', 60)).toBe('unknown')
    expect(latestGroupHealth({ ...row, buckets: [] }, '2026-11-15T12:00:20Z', 60)).toBe('unknown')
    expect(latestGroupHealth({ ...row, buckets: [row.buckets[1]!] }, '2026-11-15T12:05:20Z', 60)).toBe('unknown')
  })
})
