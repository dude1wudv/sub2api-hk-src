import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import UserDashboardCharts from '../UserDashboardCharts.vue'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ locale: { value: 'en' }, t: (key: string) => key }) }))
vi.mock('vue-chartjs', () => ({ Line: { name: 'Line', props: ['data', 'options'], template: '<div />' } }))

describe('UserDashboardCharts', () => {
  it('orders models by its stated request metric without changing time order or click targets', async () => {
    const points = [
      { date: '2026-10-09', requests: 2, input_tokens: 1000, output_tokens: 0, cache_creation_tokens: 0, cache_read_tokens: 0, total_tokens: 1000, actual_cost: 4, cost: 4 },
      { date: '2026-10-10', requests: 9, input_tokens: 50, output_tokens: 0, cache_creation_tokens: 0, cache_read_tokens: 0, total_tokens: 50, actual_cost: 0.2, cost: 0.2 },
    ]
    const models = points.map((point, index) => Object.freeze({ ...point, model: ['token-first', 'request-first'][index] }))
    Object.freeze(models)
    const wrapper = mount(UserDashboardCharts, {
      props: { loading: false, startDate: '2026-10-09', endDate: '2026-10-10', granularity: 'day', trend: points, models },
      global: { stubs: { DateRangePicker: true } },
    })
    expect(wrapper.findAll('.workspace-model-name strong').map(label => label.text())).toEqual(['request-first', 'token-first'])
    await wrapper.get('.workspace-models button').trigger('click')
    expect(wrapper.emitted('drilldown')?.[0]?.[0]).toEqual({ model: 'request-first', start: '2026-10-09', end: '2026-10-10' })
    const chart = wrapper.findComponent({ name: 'Line' })
    expect(chart.props('data').labels).toEqual(['2026-10-09', '2026-10-10'])
    chart.props('options').onClick({}, [{ index: 0 }])
    expect(wrapper.emitted('drilldown')?.[1]?.[0]).toEqual({ start: '2026-10-09', end: '2026-10-09' })
    expect(models.map(model => model.model)).toEqual(['token-first', 'request-first'])
    wrapper.unmount()
  })
})
