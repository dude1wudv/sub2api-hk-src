import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { RouterLinkStub } from '@vue/test-utils'
import PortalRedeemHistory from '../PortalRedeemHistory.vue'

const { getHistory } = vi.hoisted(() => ({ getHistory: vi.fn() }))

vi.mock('@/api/redeem', () => ({ redeemAPI: { redeem: vi.fn(), getHistory } }))
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const { ref } = await import('vue')
  return { ...actual, useI18n: () => ({ locale: ref('en-US'), t: (key: string) => key }) }
})

describe('PortalRedeemHistory', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    getHistory.mockResolvedValue({
      total: 61,
      items: [{ id: 1, code: 'FULL-REDEEM-SECRET', type: 'balance', value: 25, used_at: '2026-03-08T00:00:00Z' }],
    })
  })

  it('masks redeem codes before rendering them', async () => {
    const wrapper = mount(PortalRedeemHistory, { global: { stubs: { RouterLink: RouterLinkStub } } })
    await flushPromises()

    expect(wrapper.text()).toContain('•••• •••• CRET')
    expect(wrapper.text()).not.toContain('FULL-REDEEM-SECRET')
    expect(getHistory).toHaveBeenCalledWith(1, 20)
    wrapper.unmount()
  })

  it('keeps the current page and loaded rows after a later page request fails', async () => {
    const wrapper = mount(PortalRedeemHistory, { global: { stubs: { RouterLink: RouterLinkStub } } })
    await flushPromises()
    await wrapper.get('.redeem-history-pagination button:last-of-type').trigger('click')
    await flushPromises()
    expect(getHistory).toHaveBeenLastCalledWith(2, 20)
    expect(wrapper.text()).toContain('2 / 4')

    getHistory.mockRejectedValueOnce(new Error('temporary network failure'))
    await wrapper.get('.redeem-history-pagination button:last-of-type').trigger('click')
    await flushPromises()

    expect(getHistory).toHaveBeenLastCalledWith(3, 20)
    expect(wrapper.get('.redeem-history-error').text()).toContain('temporary network failure')
    expect(wrapper.text()).toContain('2 / 4')
    expect(wrapper.text()).toContain('•••• •••• CRET')
    expect(wrapper.get('.redeem-history-pagination button:last-of-type').attributes('disabled')).toBeUndefined()

    await wrapper.get('.redeem-history-error button').trigger('click')
    await flushPromises()
    expect(getHistory).toHaveBeenLastCalledWith(3, 20)
    expect(wrapper.text()).toContain('3 / 4')
    wrapper.unmount()
  })
})
