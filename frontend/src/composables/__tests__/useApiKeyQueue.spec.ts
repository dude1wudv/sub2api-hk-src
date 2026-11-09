import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { useApiKeyQueue } from '../useApiKeyQueue'
import type { ApiKey } from '@/types'

const { getConcurrencyQueue } = vi.hoisted(() => ({ getConcurrencyQueue: vi.fn() }))
vi.mock('@/api/keys', () => ({ getConcurrencyQueue }))
let hidden = false
let wrappers: ReturnType<typeof mount>[] = []
beforeEach(() => {
  vi.useFakeTimers(); hidden = false
  Object.defineProperty(document, 'hidden', { configurable: true, get: () => hidden })
  getConcurrencyQueue.mockReset().mockResolvedValue({ queue_policy: { max_waiting: 5, timeout_seconds: 30 }, items: [{ id: 7, current_concurrency: 2, current_waiting: 3 }] })
})
afterEach(() => { wrappers.forEach(wrapper => wrapper.unmount()); wrappers = []; vi.useRealTimers() })
function render() {
  const keys = ref([{ id: 7 }] as ApiKey[])
  const wrapper = mount(defineComponent({ setup: () => useApiKeyQueue(keys), template: '<div>{{ JSON.stringify(counts) }} {{ unavailable }}</div>' }))
  wrappers.push(wrapper)
  return { wrapper, keys }
}
describe('useApiKeyQueue', () => {
  it('refreshes displayed keys and pauses requests when the page is hidden', async () => {
    const { wrapper } = render(); await flushPromises()
    expect(getConcurrencyQueue).toHaveBeenCalledWith([7], expect.any(AbortSignal))
    expect(wrapper.text()).toContain('"current_waiting":3')
    await vi.advanceTimersByTimeAsync(5000)
    expect(getConcurrencyQueue).toHaveBeenCalledTimes(2)
    hidden = true; document.dispatchEvent(new Event('visibilitychange'))
    await vi.advanceTimersByTimeAsync(15000)
    expect(getConcurrencyQueue).toHaveBeenCalledTimes(2)
    hidden = false; document.dispatchEvent(new Event('visibilitychange')); await flushPromises()
    expect(getConcurrencyQueue).toHaveBeenCalledTimes(3)
  })
  it('drops stale counts when the displayed keys or request availability change', async () => {
    const { wrapper, keys } = render(); await flushPromises()
    getConcurrencyQueue.mockRejectedValue(new Error('Unavailable'))
    keys.value = [{ id: 8 }] as ApiKey[]; await flushPromises()
    expect(getConcurrencyQueue).toHaveBeenLastCalledWith([8], expect.any(AbortSignal))
    expect(wrapper.text()).toBe('{} true')
  })
})
