import { describe, expect, it, vi } from 'vitest'
import { singleFlight } from '../singleFlight'
describe('single submission across async validation', () => {
  it('ignores another submit during validation, then permits a later retry', async () => {
    let finish!: () => void
    const validate = vi.fn(() => new Promise<void>(resolve => { finish = resolve }))
    const request = vi.fn()
    const submit = singleFlight(async () => { await validate(); request() })
    const first = submit(); await submit()
    expect(validate).toHaveBeenCalledOnce(); expect(request).not.toHaveBeenCalled()
    finish(); await first; expect(request).toHaveBeenCalledOnce()
    const retry = submit(); finish(); await retry; expect(request).toHaveBeenCalledTimes(2)
  })
  it('releases the guard after a rejected validation', async () => {
    const request = vi.fn().mockRejectedValueOnce(new Error('invalid')).mockResolvedValueOnce('saved')
    const submit = singleFlight(request)
    await expect(submit()).rejects.toThrow('invalid')
    await expect(submit()).resolves.toBe('saved')
  })
})
