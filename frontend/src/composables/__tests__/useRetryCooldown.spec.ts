import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { useRetryCooldown } from '../useRetryCooldown'

function createCooldown() {
  let cooldown!: ReturnType<typeof useRetryCooldown>
  const wrapper = mount(defineComponent({
    setup() { cooldown = useRetryCooldown(); return () => null },
  }))
  return { cooldown, wrapper }
}

describe('useRetryCooldown', () => {
  beforeEach(() => { vi.useFakeTimers(); vi.setSystemTime(new Date('2026-11-02T00:00:00Z')) })
  afterEach(() => vi.useRealTimers())

  it('does not erase an active server cooldown when another parallel request fails without Retry-After', () => {
    const { cooldown, wrapper } = createCooldown()
    cooldown.start({ retry_after: 60 })
    vi.advanceTimersByTime(5000)
    cooldown.start({ status: 503, message: 'Unavailable' })
    expect(cooldown.remaining.value).toBe(55)
    vi.advanceTimersByTime(55000)
    expect(cooldown.remaining.value).toBe(0)
    expect(vi.getTimerCount()).toBe(0)
    wrapper.unmount()
  })

  it('keeps the later deadline across shorter limits and extends it when needed', () => {
    const { cooldown, wrapper } = createCooldown()
    cooldown.start({ retry_after: 60 })
    vi.advanceTimersByTime(10000)
    cooldown.start({ retry_after: 5 })
    expect(cooldown.remaining.value).toBe(50)
    cooldown.start({ retry_after: 90 })
    expect(cooldown.remaining.value).toBe(90)
    expect(vi.getTimerCount()).toBe(1)
    vi.advanceTimersByTime(90000)
    expect(cooldown.remaining.value).toBe(0)
    cooldown.start({ retry_after: 15 })
    wrapper.unmount()
    expect(vi.getTimerCount()).toBe(0)
    expect(cooldown.remaining.value).toBe(0)
  })
})
