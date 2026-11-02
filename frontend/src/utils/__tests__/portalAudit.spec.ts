import { describe, expect, it } from 'vitest'
import { formatUsageMoney } from '../portalUsage'
import { timezoneLabel } from '../timezoneLabel'
import { retryAfterSeconds, userFacingError } from '../userFacingError'
import { parseHeliosLaunchUrl } from '../heliosLaunch'

describe('portal audit boundaries', () => {
  it('keeps unknown money distinct from zero and preserves tiny charges', () => {
    expect(formatUsageMoney(null)).toBe('—')
    expect(formatUsageMoney(0, 2)).toBe('USD 0.00')
    expect(formatUsageMoney(0.000012, 2)).toBe('USD 0.000012')
    expect(formatUsageMoney(0.00000001, 2)).toBe('< USD 0.000001')
  })

  it('honors the reversed POSIX sign and date-specific daylight saving time', () => {
    expect(timezoneLabel('Etc/GMT+7')).toBe('UTC−7')
    expect(timezoneLabel('Asia/Bangkok')).toBe('UTC+7')
    expect(timezoneLabel('America/Denver', new Date('2026-01-10T12:00:00Z'))).toBe('UTC−7')
    expect(timezoneLabel('America/Denver', new Date('2026-07-10T12:00:00Z'))).toBe('UTC−6')
  })

  it('localizes throttling while retaining its retry delay', () => {
    const error = { status: 429, code: 'RATE_LIMITED', message: 'Too many requests', retry_after: '30' }
    expect(retryAfterSeconds(error)).toBe(30)
    expect(userFacingError(error, 'zh-CN', '失败')).toBe('操作过于频繁，请在 30 秒后重试。')
    expect(userFacingError({ message: 'INTERNAL_ERROR' }, 'zh-CN', '加载失败，请重试。')).toBe('加载失败，请重试。')
  })

  it('only accepts the configured bootstrap origin and exact subpath', () => {
    const trusted = 'https://patrickapi.microedulab.com/cancas/bootstrap'
    expect(parseHeliosLaunchUrl(`${trusted}#code=one-time`, trusted)).not.toBeNull()
    for (const value of [
      'https://canvas.sub.sunmmyapi.xyz/bootstrap#code=one-time',
      'https://patrickapi.microedulab.com/bootstrap#code=one-time',
      `${trusted}?redirect=https://example.com#code=one-time`,
      'https://user:password@patrickapi.microedulab.com/cancas/bootstrap#code=one-time',
    ]) expect(parseHeliosLaunchUrl(value, trusted)).toBeNull()
    expect(parseHeliosLaunchUrl(trusted)).toBeNull()
  })
})
