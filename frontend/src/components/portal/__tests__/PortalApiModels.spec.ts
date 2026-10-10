import { describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import PortalApiModels from '../PortalApiModels.vue'

vi.mock('@/components/portal/PortalDialog.vue', () => ({
  default: { props: ['open', 'title'], template: '<div class="portal-dialog-test"><slot /></div>' },
}))
vi.mock('@/components/icons/Icon.vue', () => ({ default: { template: '<span />' } }))
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const { ref } = await import('vue')
  return { ...actual, useI18n: () => ({ locale: ref('en-US'), t: (key: string) => key }) }
})

const group = {
  id: 1,
  name: 'API group',
  description: '',
  platform: 'openai',
  subscription_type: 'standard',
  rate_multiplier: 2,
  user_rate_multiplier: 1.5,
  peak_rate_enabled: false,
  peak_start: '',
  peak_end: '',
  peak_rate_multiplier: 1,
  is_exclusive: false,
  image_rate_independent: true,
  image_rate_multiplier: 0.5,
  video_rate_independent: true,
  video_rate_multiplier: 3,
  long_context_pricing_enabled: true,
  models: [
    {
      name: 'token-model',
      platform: 'openai',
      pricing: {
        billing_mode: 'token',
        input_price: 0.000001,
        output_price: 0.000002,
        cache_write_price: 0.000001,
        cache_read_price: 0.0000005,
        cache_write_1h_price: 0.000002,
        image_input_price: null,
        image_output_price: null,
        per_request_price: null,
        intervals: [],
      },
      official_pricing: null,
    },
    {
      name: 'image-model',
      platform: 'openai',
      pricing: {
        billing_mode: 'image', input_price: null, output_price: null,
        cache_write_price: null, cache_read_price: null, image_input_price: null,
        image_output_price: null, per_request_price: 2, intervals: [],
      },
      official_pricing: null,
    },
    {
      name: 'video-model',
      platform: 'openai',
      pricing: {
        billing_mode: 'video', input_price: null, output_price: null,
        cache_write_price: null, cache_read_price: null, image_input_price: null,
        image_output_price: null, per_request_price: 2, intervals: [],
      },
      official_pricing: null,
    },
  ],
} as any

describe('PortalApiModels pricing display', () => {
  it('applies user-specific token rates and independent image/video rates in list and detail views', async () => {
    const wrapper = mount(PortalApiModels, {
      props: { groups: [group], selectedGroupId: 1, bindableGroupIds: [1] },
    })

    const table = wrapper.get('.portal-api-model-table')
    expect(table.find('tbody tr').text()).toContain('token-model')
    expect(table.find('tbody tr').text()).toContain('$1.50')
    expect(table.find('tbody tr').text()).toContain('1h $3.00')
    const imageRow = table.findAll('tbody tr').find(row => row.text().includes('image-model'))!
    const videoRow = table.findAll('tbody tr').find(row => row.text().includes('video-model'))!
    expect(imageRow.text()).toContain('$1.00 / image')
    expect(videoRow.text()).toContain('$6.00 / request')

    await wrapper.get('button.portal-api-model-name').trigger('click')
    await flushPromises()
    expect(wrapper.get('.portal-api-price-summary').text()).toContain('×1.5')
    expect(wrapper.text()).toContain('Replaces the default group multiplier')
    wrapper.unmount()
  })

  it('renders pricing descriptions as sanitized Markdown', () => {
    const wrapper = mount(PortalApiModels, {
      props: {
        groups: [group], selectedGroupId: 1,
        description: '**Actual prices**<img src=x onerror=alert(1)>',
      },
    })

    expect(wrapper.get('.portal-api-price-description strong').text()).toBe('Actual prices')
    expect(wrapper.get('.portal-api-price-description').html()).not.toContain('onerror')
    wrapper.unmount()
  })
})
