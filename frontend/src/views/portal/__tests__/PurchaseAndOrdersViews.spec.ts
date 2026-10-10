import { describe, expect, it, vi } from 'vitest'
import { mount, RouterLinkStub } from '@vue/test-utils'
import PurchaseView from '../PurchaseView.vue'
import OrdersView from '../OrdersView.vue'
import { PATRICK_CARD_ORDER_URL, PATRICK_CARD_SHOP_URL } from '@/utils/portalPurchase'

const authState = vi.hoisted(() => ({ user: { balance: 12.5 } }))

vi.mock('@/stores/auth', () => ({ useAuthStore: () => authState }))
vi.mock('@/components/portal/PortalRedeemForm.vue', () => ({
  default: {
    emits: ['redeemed'],
    template: '<button class="mock-redeem-success" @click="$emit(\'redeemed\')">redeem success</button>',
  },
}))
vi.mock('@/components/portal/PortalRedeemHistory.vue', () => ({
  default: {
    props: ['compact', 'refreshKey'],
    template: '<div class="mock-history" :data-compact="compact" :data-refresh-key="refreshKey" />',
  },
}))
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const { ref } = await import('vue')
  return { ...actual, useI18n: () => ({ locale: ref('en-US'), t: (key: string) => key }) }
})

const global = { stubs: { RouterLink: RouterLinkStub } }

describe('portal purchase and order links', () => {
  it('uses the public card-shop URLs and reloads local redemption history after success', async () => {
    const wrapper = mount(PurchaseView, { global })
    const shop = wrapper.findAll('a').find(link => link.text().includes('Visit card shop'))!
    const orders = wrapper.findAll('a').find(link => link.text().includes('Find your card-shop order'))!
    expect(shop.attributes()).toMatchObject({ href: PATRICK_CARD_SHOP_URL, target: '_blank', rel: 'noopener noreferrer' })
    expect(orders.attributes()).toMatchObject({ href: PATRICK_CARD_ORDER_URL, target: '_blank', rel: 'noopener noreferrer' })
    expect(wrapper.get('.purchase-balance strong').text()).toContain('$12.50')
    expect(wrapper.get('.mock-history').attributes('data-compact')).toBe('')

    await wrapper.get('.mock-redeem-success').trigger('click')
    expect(wrapper.get('.mock-history').attributes('data-refresh-key')).toBe('1')
    wrapper.unmount()
  })

  it('keeps redemption history on patrickapi while card-shop orders link externally', () => {
    const wrapper = mount(OrdersView, { global })
    const external = wrapper.findAll('a').find(link => link.attributes('href') === PATRICK_CARD_ORDER_URL)!
    expect(external.attributes()).toMatchObject({ target: '_blank', rel: 'noopener noreferrer' })
    expect(wrapper.get('.mock-history').exists()).toBe(true)
    expect(wrapper.get('.mock-history').attributes('data-compact')).toBeUndefined()
    expect(wrapper.get('.redemption-tabs').findComponent(RouterLinkStub).props('to')).toBe('/purchase')
    wrapper.unmount()
  })
})
