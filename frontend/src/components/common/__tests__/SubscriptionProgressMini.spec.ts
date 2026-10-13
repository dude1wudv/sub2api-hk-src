import { enableAutoUnmount, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import SubscriptionProgressMini from '../SubscriptionProgressMini.vue'
const store = vi.hoisted(() => ({ activeSubscriptions: [] as unknown[], hasActiveSubscriptions: true, fetchActiveSubscriptions: vi.fn().mockResolvedValue(undefined) }))
vi.mock('@/stores', () => ({ useSubscriptionStore: () => store }))
vi.mock('@/utils/featureFlags', () => ({ FeatureFlags: { subscription: 'subscription' }, isFeatureFlagEnabled: () => true }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
enableAutoUnmount(afterEach)
beforeEach(() => { vi.useFakeTimers(); vi.setSystemTime(new Date(2026, 8, 22, 12)); store.fetchActiveSubscriptions.mockResolvedValue(undefined) })
afterEach(() => { vi.useRealTimers(); vi.restoreAllMocks(); vi.unstubAllGlobals() })
describe('subscription expiry calendar labels', () => {
  it.each([
    [new Date(2026, 8, 22, 18), 'expiresToday'],
    [new Date(2026, 8, 23, 18), 'expiresTomorrow'],
    [new Date(2026, 8, 22, 12), 'expired'],
    [new Date(2026, 8, 25, 12), 'daysRemaining'],
  ])('labels %s as %s', async (expires, label) => {
    store.activeSubscriptions = [{ id: 1, group_id: 1, expires_at: expires.toISOString(), group: { name: 'Plan' } }]
    const w = mount(SubscriptionProgressMini, { global: { stubs: { Icon: true, RouterLink: true, Transition: true } } })
    await w.get('button').trigger('click')
    expect(document.body.querySelector('[role="dialog"]')?.textContent).toContain('subscriptionProgress.' + label)
  })

  it.each([320, 360, 390, 1280])('keeps details within a %spx viewport and closes on resize', async (width) => {
    vi.stubGlobal('innerWidth', width)
    vi.stubGlobal('innerHeight', 300)
    store.activeSubscriptions = Array.from({ length: 20 }, (_, id) => ({ id, group_id: id, group: { name: `Long subscription ${id}` } }))
    const w = mount(SubscriptionProgressMini, { global: { stubs: { Icon: true, RouterLink: true, Transition: true } } })
    vi.spyOn(w.element, 'getBoundingClientRect').mockReturnValue({ top: 120, bottom: 160, left: width - 44, right: width, width: 44, height: 40 } as DOMRect)
    await w.get('button').trigger('click')
    const panel = document.body.querySelector<HTMLElement>('[role="dialog"]')!
    expect(Number.parseFloat(panel.style.left)).toBeGreaterThanOrEqual(8)
    expect(Number.parseFloat(panel.style.left) + Number.parseFloat(panel.style.width)).toBeLessThanOrEqual(width - 8)
    expect(Number.parseFloat(panel.style.maxHeight)).toBeGreaterThan(0)
    expect(Number.parseFloat(panel.style.maxHeight)).toBeLessThanOrEqual(132)
    expect(panel.classList.contains('overflow-y-auto')).toBe(true)
    panel.dispatchEvent(new Event('scroll'))
    await w.vm.$nextTick()
    expect(document.body.querySelector('[role="dialog"]')).not.toBeNull()
    window.dispatchEvent(new Event('resize'))
    await w.vm.$nextTick()
    expect(document.body.querySelector('[role="dialog"]')).toBeNull()
  })
})
