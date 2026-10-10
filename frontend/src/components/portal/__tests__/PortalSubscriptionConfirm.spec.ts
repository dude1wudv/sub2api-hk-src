import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, RouterLinkStub } from '@vue/test-utils'
import PortalSubscriptionConfirm from '../PortalSubscriptionConfirm.vue'

const { purchaseSubscriptionWithBalance, refreshUser, fetchActiveSubscriptions } = vi.hoisted(() => ({
  purchaseSubscriptionWithBalance: vi.fn(),
  refreshUser: vi.fn(),
  fetchActiveSubscriptions: vi.fn(),
}))

vi.mock('@/api/payment', () => ({ paymentAPI: { purchaseSubscriptionWithBalance } }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => ({ user: { balance: 50 }, refreshUser }) }))
vi.mock('@/stores/subscriptions', () => ({ useSubscriptionStore: () => ({ fetchActiveSubscriptions }) }))
vi.mock('@/components/portal/PortalDialog.vue', () => ({ default: { props: ['open', 'title'], template: '<div class="portal-dialog-test"><slot /></div>' } }))
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const { ref } = await import('vue')
  return { ...actual, useI18n: () => ({ locale: ref('en-US'), t: (key: string) => key }) }
})

const plan = (purchase_mode: string) => ({ id: 17, name: 'API plan', price: 20, purchase_mode }) as any

describe('PortalSubscriptionConfirm', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    purchaseSubscriptionWithBalance.mockResolvedValue(undefined)
    refreshUser.mockResolvedValue(undefined)
    fetchActiveSubscriptions.mockResolvedValue([])
    vi.stubGlobal('crypto', { randomUUID: vi.fn(() => 'attempt-uuid') })
  })

  it.each(['balance', 'both'])('allows %s plans to use the existing balance-purchase API', async (purchase_mode) => {
    const wrapper = mount(PortalSubscriptionConfirm, { props: { plan: plan(purchase_mode) }, global: { stubs: { RouterLink: RouterLinkStub } } })
    const confirm = wrapper.findAll('button').find(button => button.text() === 'Confirm balance purchase')!
    expect(confirm.attributes('disabled')).toBeUndefined()
    await confirm.trigger('click')
    await flushPromises()

    expect(purchaseSubscriptionWithBalance).toHaveBeenCalledWith(17, 'balance-subscription-17-attempt-uuid')
    expect(wrapper.emitted('purchased')).toEqual([['']])
    expect(refreshUser).toHaveBeenCalledOnce()
    expect(fetchActiveSubscriptions).toHaveBeenCalledWith(true)
    wrapper.unmount()
  })

  it('does not offer the balance purchase API for plans with another payment mode', async () => {
    const wrapper = mount(PortalSubscriptionConfirm, { props: { plan: plan('card') }, global: { stubs: { RouterLink: RouterLinkStub } } })
    const confirm = wrapper.findAll('button').find(button => button.text() === 'Confirm balance purchase')!
    expect(confirm.attributes('disabled')).toBeDefined()
    expect(wrapper.get('[role="alert"]').text()).toContain('Balance purchase is not available for this plan')
    await confirm.trigger('click')
    expect(purchaseSubscriptionWithBalance).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('reuses its idempotency key after a failed request and suppresses concurrent submits', async () => {
    purchaseSubscriptionWithBalance.mockRejectedValueOnce(new Error('temporary payment failure'))
    const wrapper = mount(PortalSubscriptionConfirm, { props: { plan: plan('balance') }, global: { stubs: { RouterLink: RouterLinkStub } } })
    const confirm = wrapper.findAll('button').find(button => button.text() === 'Confirm balance purchase')!
    await confirm.trigger('click')
    await flushPromises()
    expect(wrapper.get('[role="alert"]').exists()).toBe(true)

    let resolvePurchase!: () => void
    purchaseSubscriptionWithBalance.mockImplementationOnce(() => new Promise<void>(resolve => { resolvePurchase = resolve }))
    await confirm.trigger('click')
    await confirm.trigger('click')
    expect(purchaseSubscriptionWithBalance).toHaveBeenCalledTimes(2)
    expect(purchaseSubscriptionWithBalance.mock.calls.map(([id, key]) => [id, key])).toEqual([
      [17, 'balance-subscription-17-attempt-uuid'],
      [17, 'balance-subscription-17-attempt-uuid'],
    ])

    resolvePurchase()
    await flushPromises()
    expect(wrapper.emitted('purchased')).toEqual([['']])
    wrapper.unmount()
  })

  it('reports refresh failure after a successful purchase without retrying the purchase', async () => {
    refreshUser.mockRejectedValueOnce(new Error('profile refresh failed'))
    const wrapper = mount(PortalSubscriptionConfirm, { props: { plan: plan('both') }, global: { stubs: { RouterLink: RouterLinkStub } } })
    const confirm = wrapper.findAll('button').find(button => button.text() === 'Confirm balance purchase')!
    await confirm.trigger('click')
    await flushPromises()

    expect(purchaseSubscriptionWithBalance).toHaveBeenCalledOnce()
    expect(wrapper.emitted('purchased')).toEqual([[
      'Subscription purchased. Refresh the page to update account information.',
    ]])
    expect(refreshUser).toHaveBeenCalledOnce()
    expect(fetchActiveSubscriptions).toHaveBeenCalledOnce()
    wrapper.unmount()
  })
})
