import { mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import { describe, expect, it, vi } from 'vitest'
import type { SubscriptionPlan } from '@/types/payment'

const { updatePlan } = vi.hoisted(() => ({ updatePlan: vi.fn() }))

vi.mock('@/api/admin/payment', () => ({
  adminPaymentAPI: { updatePlan },
}))

import PlanEditDialog from '../orders/PlanEditDialog.vue'

const i18n = createI18n({
  legacy: false,
  locale: 'en',
  fallbackWarn: false,
  missingWarn: false,
  messages: { en: {} },
})

const plan: SubscriptionPlan = {
  id: 1,
  group_id: 10,
  name: 'Pro',
  description: 'Plan description',
  price: 10,
  validity_days: 30,
  validity_unit: 'days',
  features: ['Existing'],
  one_purchase_per_user: false,
  for_sale: true,
  sort_order: 0,
}

describe('PlanEditDialog', () => {
  it('submits the purchase limit and normalized feature entries to the admin API', async () => {
    updatePlan.mockResolvedValue({})
    const wrapper = mount(PlanEditDialog, {
      props: { show: false, plan, groups: [] },
      global: {
        plugins: [createPinia(), i18n],
        stubs: {
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
          Select: {
            props: ['modelValue', 'options'],
            template: '<div><slot /></div>',
          },
          Icon: true,
          GroupBadge: true,
        },
      },
    })

    await wrapper.setProps({ show: true })
    await wrapper.findAll('button[type="button"]')[0].trigger('click')
    await wrapper.findAll('textarea')[1].setValue('First\\r\\nSecond\r\nThird')
    await wrapper.get('form').trigger('submit')

    expect(updatePlan).toHaveBeenCalledWith(1, expect.objectContaining({
      one_purchase_per_user: true,
      features: 'First\nSecond\nThird',
    }))
  })
})