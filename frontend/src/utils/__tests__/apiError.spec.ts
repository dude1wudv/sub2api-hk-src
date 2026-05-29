import { describe, expect, it } from 'vitest'
import { explainHumanApiError } from '@/utils/apiError'

describe('explainHumanApiError', () => {
  it('explains missing redeem codes with a practical next step', () => {
    const info = explainHumanApiError(
      { reason: 'REDEEM_CODE_NOT_FOUND', message: 'redeem code not found' },
      '兑换失败'
    )

    expect(info.title).toBe('兑换码不存在')
    expect(info.description).toContain('输错')
    expect(info.action).toContain('重新复制')
    expect(info.severity).toBe('warning')
  })

  it('explains insufficient balance without exposing only raw backend text', () => {
    const info = explainHumanApiError(
      { reason: 'INSUFFICIENT_BALANCE', message: 'insufficient balance' },
      '请求失败'
    )

    expect(info.title).toBe('余额不足')
    expect(info.description).toContain('余额')
    expect(info.action).toContain('兑换码')
  })

  it('maps numeric rate-limit errors to a human-readable throttling message', () => {
    const info = explainHumanApiError(
      { code: 429, message: 'rate limit exceeded' },
      '请求失败'
    )

    expect(info.title).toBe('请求太频繁')
    expect(info.action).toContain('降低并发')
  })

  it('keeps unknown backend messages as fallback context', () => {
    const info = explainHumanApiError(
      { message: 'upstream returned unexpected payload' },
      '操作失败'
    )

    expect(info.title).toBe('操作失败')
    expect(info.description).toBe('upstream returned unexpected payload')
    expect(info.action).toContain('管理员')
  })
})
