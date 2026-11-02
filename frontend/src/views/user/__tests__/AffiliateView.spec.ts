import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import AffiliateView from '../AffiliateView.vue'

const { copyToClipboard, getAffiliateDetail, transferAffiliateQuota, refreshUser } = vi.hoisted(() => ({
  copyToClipboard: vi.fn(),
  transferAffiliateQuota: vi.fn(),
  refreshUser: vi.fn(),
  getAffiliateDetail: vi.fn(),
}))

vi.mock('@/api/user', () => ({
  default: {
    getAffiliateDetail,
    transferAffiliateQuota,
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
  }),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    refreshUser,
  }),
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({ copyToClipboard }),
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
      locale: { value: 'en' },
    }),
  }
})

function mountAffiliate() {
  return mount(AffiliateView, {
    global: { stubs: { Icon: true, BaseDialog: { props: ['show'], template: '<div v-if="show" role="dialog"><slot /><slot name="footer" /></div>' } } },
  })
}

describe('AffiliateView', () => {
  const affiliateCode = 'affiliate-code-that-is-long-enough-to-overflow-a-mobile-viewport'

  beforeEach(() => {
    vi.clearAllMocks()
    copyToClipboard.mockResolvedValue(true)
    getAffiliateDetail.mockResolvedValue({
      user_id: 1,
      aff_code: affiliateCode,
      inviter_id: null,
      aff_count: 0,
      aff_quota: 0,
      aff_frozen_quota: 0,
      aff_history_quota: 0,
      effective_rebate_rate_percent: 10,
      invitees: [],
    })
  })

  it('stacks long values and copy controls on mobile while retaining desktop rows', async () => {
    const wrapper = mountAffiliate()

    await flushPromises()

    const values = wrapper.findAll('code')
    expect(values).toHaveLength(2)
    for (const value of values) {
      expect(value.classes()).toEqual(expect.arrayContaining([
        'min-w-0',
        'break-all',
        'sm:flex-1',
        'sm:truncate',
      ]))
      expect(Array.from(value.element.parentElement?.classList ?? [])).toEqual(expect.arrayContaining([
        'flex-col',
        'items-stretch',
        'sm:flex-row',
        'sm:items-center',
      ]))
    }

    const copyButtons = wrapper.findAll('button').filter((button) =>
      ['affiliate.copyCode', 'affiliate.copyLink'].includes(button.text()),
    )
    expect(copyButtons).toHaveLength(2)
    for (const button of copyButtons) {
      expect(button.classes()).toEqual(expect.arrayContaining([
        'w-full',
        'sm:w-auto',
        'sm:shrink-0',
      ]))
    }

    await copyButtons[0].trigger('click')
    await copyButtons[1].trigger('click')
    await flushPromises()

    expect(copyToClipboard).toHaveBeenNthCalledWith(1, affiliateCode, 'affiliate.codeCopied')
    expect(copyToClipboard).toHaveBeenNthCalledWith(
      2,
      `${window.location.origin}/register?aff=${encodeURIComponent(affiliateCode)}`,
      'affiliate.linkCopied',
    )
  })
  it('requires confirmation and cancellation never transfers funds', async () => {
    getAffiliateDetail.mockResolvedValueOnce({ aff_quota: 1, aff_count: 0, aff_history_quota: 1, aff_frozen_quota: 0, aff_code: 'abc', invitees: [] })
    const wrapper = mountAffiliate()
    await flushPromises()
    const transfer = wrapper.findAll('button').find(button => button.text() === 'affiliate.transfer.button')!
    await transfer.trigger('click')
    expect(wrapper.get('[role="dialog"]').text()).toContain('Transfer all currently available rebate')
    expect(transferAffiliateQuota).not.toHaveBeenCalled()
    await wrapper.get('[role="dialog"]').findAll('button').find(button => button.text() === 'Cancel')!.trigger('click')
    expect(wrapper.find('[role="dialog"]').exists()).toBe(false)
    expect(transferAffiliateQuota).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('submits only once and separates a confirmed transfer from refresh failure', async () => {
    getAffiliateDetail.mockResolvedValueOnce({ aff_quota: 10, aff_count: 0, aff_history_quota: 10, aff_frozen_quota: 0, aff_code: 'abc', invitees: [] })
    getAffiliateDetail.mockRejectedValueOnce({ status: 503 })
    let finish!: (value: { transferred_quota: number }) => void
    transferAffiliateQuota.mockReturnValueOnce(new Promise(resolve => { finish = resolve }))
    const wrapper = mountAffiliate()
    await flushPromises()
    await wrapper.findAll('button').find(button => button.text() === 'affiliate.transfer.button')!.trigger('click')
    const confirm = wrapper.get('[data-testid="confirm-transfer"]')
    await confirm.trigger('click')
    await confirm.trigger('click')
    expect(transferAffiliateQuota).toHaveBeenCalledOnce()
    finish({ transferred_quota: 10 })
    await flushPromises()
    expect(wrapper.text()).toContain('affiliate.transfer.success')
    expect(wrapper.text()).toContain('Account information could not be refreshed')
    expect(transferAffiliateQuota).toHaveBeenCalledOnce()
    expect(refreshUser).toHaveBeenCalledOnce()
    await wrapper.findAll('button').find(button => button.text() === 'Refresh rebate and balance')!.trigger('click')
    await flushPromises()
    expect(wrapper.text()).not.toContain('Account information could not be refreshed')
    expect(transferAffiliateQuota).toHaveBeenCalledOnce()
    wrapper.unmount()
  })

  it('queries state after a timeout without retrying the financial request', async () => {
    getAffiliateDetail.mockResolvedValueOnce({ aff_quota: 1, aff_count: 0, aff_history_quota: 1, aff_frozen_quota: 0, aff_code: 'abc', invitees: [] })
    transferAffiliateQuota.mockRejectedValueOnce(new Error('network timeout'))
    const wrapper = mountAffiliate()
    await flushPromises()
    await wrapper.findAll('button').find(button => button.text() === 'affiliate.transfer.button')!.trigger('click')
    await wrapper.get('[data-testid="confirm-transfer"]').trigger('click')
    await flushPromises()
    expect(transferAffiliateQuota).toHaveBeenCalledOnce()
    expect(getAffiliateDetail).toHaveBeenCalledTimes(2)
    expect(refreshUser).toHaveBeenCalledOnce()
    expect(wrapper.text()).toContain('The transfer result is uncertain')
    wrapper.unmount()
  })

})
