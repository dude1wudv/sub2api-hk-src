import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { RouterLinkStub } from '@vue/test-utils'
import PortalRedeemForm from '../PortalRedeemForm.vue'

const { redeem, refreshUser, invalidateCache, fetchActiveSubscriptions } = vi.hoisted(() => ({
  redeem: vi.fn(),
  refreshUser: vi.fn(),
  invalidateCache: vi.fn(),
  fetchActiveSubscriptions: vi.fn(),
}))

vi.mock('@/api/redeem', () => ({ redeemAPI: { redeem, getHistory: vi.fn() } }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => ({ refreshUser }) }))
vi.mock('@/stores/subscriptions', () => ({ useSubscriptionStore: () => ({ invalidateCache, fetchActiveSubscriptions }) }))
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const { ref } = await import('vue')
  return { ...actual, useI18n: () => ({ locale: ref('en-US'), t: (key: string) => key }) }
})

describe('PortalRedeemForm', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    redeem.mockResolvedValue({ type: 'balance', value: 20 })
    refreshUser.mockResolvedValue(undefined)
    fetchActiveSubscriptions.mockResolvedValue([])
  })

  function mountForm() {
    return mount(PortalRedeemForm, { global: { stubs: { RouterLink: RouterLinkStub } } })
  }

  it('validates empty input and ignores repeated submission while the request is pending', async () => {
    const wrapper = mountForm()
    await wrapper.get('form').trigger('submit')
    expect(wrapper.get('[role="alert"]').text()).toContain('Enter a redemption code')
    expect(redeem).not.toHaveBeenCalled()

    let resolveRedeem!: (value: { type: string; value: number }) => void
    redeem.mockImplementationOnce(() => new Promise((resolve) => { resolveRedeem = resolve }))
    await wrapper.get('input#portal-redeem-code').setValue(' CODE-12345 ')
    await wrapper.get('form').trigger('submit')
    await wrapper.get('form').trigger('submit')
    expect(redeem).toHaveBeenCalledOnce()
    expect(redeem).toHaveBeenCalledWith('CODE-12345')

    resolveRedeem({ type: 'balance', value: 20 })
    await flushPromises()
    expect(wrapper.get('[role="status"]').text()).toContain('has been credited')
    expect(wrapper.emitted('redeemed')).toHaveLength(1)
    wrapper.unmount()
  })

  it('keeps a failed code editable and masks it if the API echoes it in the error', async () => {
    redeem.mockRejectedValue({ response: { data: { detail: 'Rejected CODE-SECRET-1234' } } })
    const wrapper = mountForm()
    await wrapper.get('input#portal-redeem-code').setValue('CODE-SECRET-1234')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(wrapper.get('[role="alert"]').text()).toContain('Rejected •••• •••• 1234')
    expect(wrapper.get('[role="alert"]').text()).not.toContain('CODE-SECRET-1234')
    expect((wrapper.get('input#portal-redeem-code').element as HTMLInputElement).value).toBe('CODE-SECRET-1234')
    expect(wrapper.emitted('redeemed')).toBeUndefined()
    expect(refreshUser).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('refreshes account information separately after redemption when the first refresh fails', async () => {
    refreshUser.mockRejectedValueOnce(new Error('profile refresh failed')).mockResolvedValueOnce(undefined)
    const wrapper = mountForm()
    await wrapper.get('input#portal-redeem-code').setValue('BALANCE-CODE')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(redeem).toHaveBeenCalledOnce()
    expect(wrapper.get('.redeem-refresh-warning').text()).toContain('Redemption succeeded; do not submit the code again')
    expect(wrapper.get('.redeem-refresh-warning button').exists()).toBe(true)
    await wrapper.get('.redeem-refresh-warning button').trigger('click')
    await flushPromises()

    expect(redeem).toHaveBeenCalledOnce()
    expect(refreshUser).toHaveBeenCalledTimes(2)
    expect(wrapper.find('.redeem-refresh-warning').exists()).toBe(false)
    expect(wrapper.emitted('redeemed')).toHaveLength(1)
    wrapper.unmount()
  })

  it('refreshes subscriptions only for subscription redemption', async () => {
    redeem.mockResolvedValue({ type: 'subscription', value: 30 })
    const wrapper = mountForm()
    await wrapper.get('input#portal-redeem-code').setValue('SUBSCRIPTION-CODE')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(invalidateCache).toHaveBeenCalledOnce()
    expect(fetchActiveSubscriptions).toHaveBeenCalledWith(true)
    expect(wrapper.get('[role="status"]').text()).toContain('subscription code has been redeemed')
    wrapper.unmount()
  })
})
