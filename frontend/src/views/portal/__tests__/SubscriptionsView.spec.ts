import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import SubscriptionsView from '../SubscriptionsView.vue'

const { getMySubscriptions, getCheckoutInfo, getSubscriptionsProgress, purchaseSubscriptionWithBalance, fetchActiveSubscriptions } = vi.hoisted(() => ({
  getMySubscriptions: vi.fn(),
  getCheckoutInfo: vi.fn(),
  getSubscriptionsProgress: vi.fn(),
  purchaseSubscriptionWithBalance: vi.fn(),
  fetchActiveSubscriptions: vi.fn(),
}))

vi.mock('@/api/subscriptions', () => ({ default: { getMySubscriptions, getSubscriptionsProgress } }))
vi.mock('@/api/payment', () => ({ paymentAPI: { getCheckoutInfo, purchaseSubscriptionWithBalance } }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ cachedPublicSettings: { subscription_enabled: true, server_utc_offset: '+00:00' } }) }))
vi.mock('@/stores/subscriptions', () => ({ useSubscriptionStore: () => ({ fetchActiveSubscriptions }) }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => ({ user: { balance: 50 }, refreshUser: vi.fn().mockResolvedValue(undefined) }) }))
vi.mock('@/components/portal/PortalDialog.vue', () => ({ default: { props: ['open', 'title'], template: '<div class="portal-dialog-test"><slot /></div>' } }))
vi.mock('vue-router', async () => {
  const actual = await vi.importActual<typeof import('vue-router')>('vue-router')
  return { ...actual, useRoute: () => ({ query: {} }) }
})
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const { ref } = await import('vue')
  return { ...actual, useI18n: () => ({ locale: ref('en-US'), t: (key: string) => key }) }
})

describe('portal SubscriptionsView purchase completion', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    getMySubscriptions.mockResolvedValue([])
    getSubscriptionsProgress.mockResolvedValue([])
    getCheckoutInfo.mockResolvedValue({ data: { plans: [{
      id: 17,
      name: 'API plan',
      description: 'Thirty days of API access',
      group_id: 8,
      group_name: 'Core group',
      price: 20,
      currency: 'USD',
      validity_days: 30,
      validity_unit: 'day',
      for_sale: true,
      sort_order: 1,
      purchase_mode: 'balance',
      features: [],
      rate_multiplier: 1,
    }] } })
    getMySubscriptions.mockResolvedValue([{
      id: 31,
      group_id: 8,
      status: 'active',
      created_at: '2026-01-01T00:00:00Z',
      starts_at: '2026-01-01T00:00:00Z',
      expires_at: '2027-01-01T00:00:00Z',
      daily_usage_usd: 2,
      weekly_usage_usd: 0,
      monthly_usage_usd: 0,
      daily_window_start: null,
      weekly_window_start: null,
      monthly_window_start: null,
      group: { name: 'Core group', daily_limit_usd: 5, weekly_limit_usd: null, monthly_limit_usd: null },
    }] as any)
    getSubscriptionsProgress.mockResolvedValue([{
      subscription: { id: 31 },
      progress: { daily: { used_usd: 3, limit_usd: 8, resets_at: new Date(Date.now() + 21 * 24 * 60 * 60 * 1000).toISOString() } },
    }])
    purchaseSubscriptionWithBalance.mockResolvedValue(undefined)
    fetchActiveSubscriptions
      .mockResolvedValueOnce([])
      .mockRejectedValueOnce(new Error('refresh temporarily unavailable'))
      .mockResolvedValue([])
  })

  it('closes the confirmation after a completed purchase even if post-purchase refresh fails', async () => {
    const wrapper = mount(SubscriptionsView, { global: { stubs: { RouterLink: true } } })
    await flushPromises()
    expect(wrapper.get('.subscription-quota').text()).toContain('$3.00 / $8.00')
    expect(wrapper.get('.subscription-quota').text()).toContain('Resets in 20d')
    await wrapper.get('.plan-button').trigger('click')
    await flushPromises()
    const confirm = wrapper.findAll('button').find(button => button.text() === 'Confirm balance purchase')!
    expect(confirm.exists()).toBe(true)

    await confirm.trigger('click')
    await flushPromises()

    expect(purchaseSubscriptionWithBalance).toHaveBeenCalledOnce()
    expect(fetchActiveSubscriptions).toHaveBeenCalledTimes(3)
    expect(wrapper.findAll('button').some(button => button.text() === 'Confirm balance purchase')).toBe(false)
    expect(wrapper.get('[role="status"]').text()).toContain('Subscription purchased. Refresh the page to update account information.')
    expect(getMySubscriptions).toHaveBeenCalledTimes(2)
    wrapper.unmount()
  })

  it('shows an unavailable or first-use label instead of estimating a reset time', async () => {
    getMySubscriptions.mockResolvedValue([{
      id: 31, group_id: 8, status: 'active',
      daily_usage_usd: 1, daily_window_start: null,
      group: { name: 'Core group', daily_limit_usd: 5 },
    }] as any)
    getSubscriptionsProgress.mockRejectedValueOnce(new Error('progress unavailable'))
    const failed = mount(SubscriptionsView, { global: { stubs: { RouterLink: true } } })
    await flushPromises()
    expect(failed.get('.subscription-quota').text()).toContain('Reset time unavailable')
    expect(failed.get('.subscription-quota').text()).not.toContain('Resets in')
    failed.unmount()

    getSubscriptionsProgress.mockResolvedValueOnce([{ subscription: { id: 31 }, progress: {} }])
    const notStarted = mount(SubscriptionsView, { global: { stubs: { RouterLink: true } } })
    await flushPromises()
    expect(notStarted.get('.subscription-quota').text()).toContain('Period starts after first use')
    expect(notStarted.get('.subscription-quota').text()).not.toContain('Resets in')
    notStarted.unmount()
  })
})
