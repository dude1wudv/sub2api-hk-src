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
  it('defaults to a populated connection without hiding empty connections', () => {
    const wrapper = mount(PortalApiModels, { props: { groups: [{ ...group, id: 2, models: [] }, group], selectedGroupId: null } })
    expect(wrapper.emitted('update:selectedGroupId')).toEqual([[1]])
    expect(wrapper.findAll('.portal-api-group-nav button')).toHaveLength(2)
    wrapper.unmount()
  })

  it('distinguishes same-named models on different connection platforms', async () => {
    const models = ['anthropic', 'openai'].map(platform => ({ ...group.models[0], name: 'claude-shared', platform }))
    const wrapper = mount(PortalApiModels, { props: { groups: [{ ...group, platform: 'composite', models }], selectedGroupId: 1 } })
    const rows = wrapper.findAll('.portal-api-model-row')
    expect(rows).toHaveLength(2)
    expect(rows[0].text()).toContain('Platform: Anthropic')
    expect(rows[1].text()).toContain('Platform: OpenAI')
    await rows[1].get('.portal-api-model-name').trigger('click')
    expect(wrapper.get('.portal-api-detail-platform').text()).toContain('Connection platform: OpenAI')
    wrapper.unmount()
  })

  it('groups models by vendor rather than the OpenAI-compatible transport', async () => {
    const models = ['deepseek-v4.1-flash', 'claude-sonnet', 'gpt-6-sol'].map(name => ({ ...group.models[0], name, platform: 'openai' }))
    const wrapper = mount(PortalApiModels, { props: { groups: [{ ...group, models }], selectedGroupId: 1 } })
    const vendors = wrapper.get('.portal-model-platforms')
    expect(vendors.text()).toContain('DeepSeek')
    expect(vendors.text()).toContain('Claude')
    await vendors.findAll('button').find(button => button.text().includes('DeepSeek'))!.trigger('click')
    expect(wrapper.findAll('.portal-api-model-name').map(button => button.text())).toEqual([expect.stringContaining('deepseek-v4.1-flash')])
    wrapper.unmount()
  })

  it('shows twelve-hour cache prices in both the catalogue and pricing details', async () => {
    const model = { ...group.models[0], pricing: { ...group.models[0].pricing, cache_write_12h_price: 0.000004 } }
    const wrapper = mount(PortalApiModels, { props: { groups: [{ ...group, models: [model] }], selectedGroupId: 1 } })
    expect(wrapper.get('.portal-api-model-table').text()).toContain('12h $6.00')
    await wrapper.get('.portal-api-model-name').trigger('click')
    expect(wrapper.get('.portal-api-detail-table').text()).toContain('Cache write 12h')
    expect(wrapper.get('.portal-api-detail-table').text()).toContain('$6.00')
    wrapper.unmount()
  })

  it('keeps empty published groups selectable and distinguishes missing prices from zero', async () => {
    const wrapper = mount(PortalApiModels, { props: { groups: [group, { ...group, id: 2, name: 'New connection', models: [] }], selectedGroupId: 2 } })
    expect(wrapper.get('.portal-api-group-nav').text()).toContain('New connection')
    expect(wrapper.text()).toContain('No model pricing is published for this group.')
    await wrapper.setProps({ selectedGroupId: 1, groups: [{ ...group, models: [{ ...group.models[0], pricing: { ...group.models[0].pricing, input_price: 0, output_price: null } }] }] })
    expect(wrapper.get('.portal-api-model-table').text()).toContain('$0.00')
    expect(wrapper.get('.portal-api-model-table').text()).toContain('—')
    wrapper.unmount()
  })

  it('applies user-specific token rates and independent image/video rates in list and detail views', async () => {
    const wrapper = mount(PortalApiModels, {
      props: { groups: [group], selectedGroupId: 1, bindableGroupIds: [1] },
    })

    const table = wrapper.get('.portal-api-model-table')
    expect(table.find('.portal-api-model-row').text()).toContain('token-model')
    expect(table.find('.portal-api-model-row').text()).toContain('$1.50')
    expect(table.find('.portal-api-model-row').text()).toContain('1h $3.00')
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
