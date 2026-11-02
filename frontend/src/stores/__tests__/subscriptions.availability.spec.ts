import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useSubscriptionStore } from '../subscriptions'

const { getAvailability } = vi.hoisted(() => ({ getAvailability: vi.fn() }))
vi.mock('@/api/subscriptions', () => ({ default: { getAvailability } }))

describe('subscription navigation availability', () => {
  beforeEach(() => { setActivePinia(createPinia()); getAvailability.mockReset() })

  it('deduplicates requests and caches a successful empty catalogue', async () => {
    const value = { has_sale_plans: false, has_active_subscriptions: false, has_subscription_history: false }
    getAvailability.mockResolvedValue(value)
    const store = useSubscriptionStore()
    await Promise.all([store.fetchAvailability(), store.fetchAvailability()])
    await store.fetchAvailability()
    expect(getAvailability).toHaveBeenCalledTimes(1)
    expect(store.availability).toEqual(value)
  })

  it('does not turn a failed request into an empty catalogue and invalidates after redemption', async () => {
    const store = useSubscriptionStore()
    getAvailability.mockRejectedValueOnce(new Error('rate limited'))
    await expect(store.fetchAvailability()).rejects.toThrow('rate limited')
    expect(store.availability).toBeNull()
    expect(store.availabilityError).toBe(true)
    getAvailability.mockResolvedValue({ has_sale_plans: false, has_active_subscriptions: true, has_subscription_history: true })
    store.invalidateCache()
    await store.fetchAvailability()
    expect(store.availability?.has_active_subscriptions).toBe(true)
    expect(store.availabilityError).toBe(false)
  })

  it('ignores responses from before logout', async () => {
    let finish!: (value: unknown) => void
    getAvailability.mockImplementation(() => new Promise(resolve => { finish = resolve }))
    const store = useSubscriptionStore()
    const pending = store.fetchAvailability()
    store.clear()
    finish({ has_sale_plans: true, has_active_subscriptions: true, has_subscription_history: true })
    await pending
    expect(store.availability).toBeNull()
  })
})
