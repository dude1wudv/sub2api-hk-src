import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import type { Component } from 'vue'
import ModelDistributionChart from '../ModelDistributionChart.vue'
import GroupDistributionChart from '../GroupDistributionChart.vue'
import EndpointDistributionChart from '../EndpointDistributionChart.vue'
import UserBreakdownSubTable from '../UserBreakdownSubTable.vue'
import DistributionChartLayout from '../DistributionChartLayout.vue'

const { getUserBreakdown } = vi.hoisted(() => ({ getUserBreakdown: vi.fn() }))
vi.mock('@/api/admin/dashboard', () => ({ getUserBreakdown }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key === 'common.unknown' ? 'Unknown' : key }) }))
vi.mock('vue-chartjs', () => ({
  Doughnut: { name: 'Doughnut', props: ['data', 'options'], template: '<div class="chart-stub" />' },
}))

const values = [
  { requests: 10, total_tokens: 9000, actual_cost: 0.2, cost: 1, input_tokens: 9000, output_tokens: 0, cache_creation_tokens: 0, cache_read_tokens: 0 },
  { requests: 5, total_tokens: 100, actual_cost: 4, cost: 5, input_tokens: 100, output_tokens: 0, cache_creation_tokens: 0, cache_read_tokens: 0 },
]
const cases: Array<{ name: string; component: Component; prop: string; rows: object[]; labels: string[]; drilldown: object }> = [
  { name: 'models', component: ModelDistributionChart, prop: 'modelStats', rows: values.map((value, index) => ({ ...value, model: `model-${index}` })), labels: ['model-0', 'model-1'], drilldown: { model: 'model-1', model_source: 'requested' } },
  { name: 'groups', component: GroupDistributionChart, prop: 'groupStats', rows: values.map((value, index) => ({ ...value, group_id: index + 1, group_name: `group-${index}` })), labels: ['group-0', 'group-1'], drilldown: { group_id: 2 } },
  { name: 'endpoints', component: EndpointDistributionChart, prop: 'endpointStats', rows: values.map((value, index) => ({ ...value, endpoint: `/v1/endpoint-${index}` })), labels: ['/v1/endpoint-0', '/v1/endpoint-1'], drilldown: { endpoint: '/v1/endpoint-1', endpoint_type: 'inbound' } },
]

describe('shared distribution ranking', () => {
  beforeEach(() => {
    getUserBreakdown.mockReset().mockResolvedValue({ users: [] })
  })

  it.each(cases)('keeps $name chart, rows, colors and drilldown aligned through both metric switches', async ({ component, prop, rows, labels, drilldown }) => {
    const source = Object.freeze(rows.map(row => Object.freeze({ ...row })))
    const wrapper = mount(component, { props: { [prop]: source, showMetricToggle: true }, global: { stubs: { LoadingSpinner: true, UserBreakdownSubTable: true } } })
    const chart = wrapper.findComponent({ name: 'Doughnut' })
    const initial = chart.props('data')
    const tableLabels = () => wrapper.findAll('tbody td[title]').map(cell => cell.attributes('title'))
    expect(initial.labels).toEqual(labels)
    expect(initial.datasets[0].data).toEqual([9000, 100])
    expect(tableLabels()).toEqual(labels)

    await wrapper.findAll('button').find(button => button.text() === 'admin.dashboard.metricActualCost')!.trigger('click')
    expect(wrapper.emitted('update:metric')).toEqual([['actual_cost']])
    await wrapper.setProps({ metric: 'actual_cost' })
    expect(chart.props('data').labels).toEqual([...labels].reverse())
    expect(chart.props('data').datasets[0].data).toEqual([4, 0.2])
    expect(chart.props('data').datasets[0].backgroundColor).toEqual([...initial.datasets[0].backgroundColor].reverse())
    expect(tableLabels()).toEqual([...labels].reverse())
    await wrapper.find('tbody tr').trigger('click')
    await flushPromises()
    expect(getUserBreakdown).toHaveBeenCalledWith(expect.objectContaining(drilldown))
    expect(wrapper.findComponent(UserBreakdownSubTable).props('metric')).toBe('actual_cost')

    await wrapper.setProps({ metric: 'tokens' })
    expect(chart.props('data')).toEqual(initial)
    expect(tableLabels()).toEqual(labels)
    expect(wrapper.findComponent(UserBreakdownSubTable).props('metric')).toBe('tokens')
    expect(source).toEqual(rows)
    expect(source[0]).not.toHaveProperty('chartColor')
    wrapper.unmount()
  })

  it.each(cases)('keeps all $name columns and expanded users in a labelled keyboard scroll region', async ({ component, prop, rows, name }) => {
    const email = `${'long-account-name-'.repeat(12)}@example.test`
    getUserBreakdown.mockResolvedValue({ users: [{
      ...values[0], user_id: 9, email, cache_tokens: 0, account_cost: 1234.56,
    }] })
    const wrapper = mount(component, { props: { [prop]: rows, showAccountCost: true } })
    try {
      const region = wrapper.get('[role="region"]')
      expect(region.attributes('tabindex')).toBe('0')
      expect(region.attributes('aria-label')).toBe(name === 'endpoints' ? 'usage.endpointDistribution' : `admin.dashboard.${name === 'models' ? 'model' : 'group'}Distribution`)
      expect(region.find('table').exists()).toBe(true)
      expect(region.find('.chart-stub').exists()).toBe(false)
      expect(region.findAll('thead th')).toHaveLength(name === 'endpoints' ? 5 : 6)
      await region.get('tbody tr').trigger('click')
      await flushPromises()
      expect(region.getComponent(UserBreakdownSubTable).text()).toContain(email)
      expect(region.getComponent(UserBreakdownSubTable).findAll('td')).toHaveLength(name === 'endpoints' ? 5 : 6)
      expect(wrapper.findAll('[role="region"]')).toHaveLength(1)
    } finally {
      wrapper.unmount()
    }
  })

  it('keeps the scroll label and column width policy when switching to user spending ranking', async () => {
    const wrapper = mount(ModelDistributionChart, { props: {
      modelStats: cases[0].rows,
      enableRankingView: true,
      rankingItems: [{ user_id: 1, email: 'long-name-'.repeat(20) + '@example.test', actual_cost: 12, requests: 1234567890123, tokens: 1e9 }],
    } })
    try {
      expect(wrapper.getComponent(DistributionChartLayout).props('columns')).toBe(6)
      await wrapper.findAll('button').find(button => button.text() === 'admin.dashboard.viewSpendingRanking')!.trigger('click')
      expect(wrapper.getComponent(DistributionChartLayout).props('columns')).toBe(4)
      const region = wrapper.get('[role="region"]')
      expect(region.attributes('aria-label')).toBe('admin.dashboard.spendingRankingTitle')
      expect(region.get('tbody tr').text()).toContain('1,234,567,890,123')
      expect(region.get('td [title]').attributes('title')).toContain('@example.test')
    } finally {
      wrapper.unmount()
    }
  })

  it('uses the same unknown-endpoint fallback in the chart and table after sorting', async () => {
    const wrapper = mount(EndpointDistributionChart, { props: {
      endpointStats: [{ ...values[0], endpoint: '/v1/responses' }, { ...values[1], endpoint: '' }],
      metric: 'actual_cost',
    } })
    expect(wrapper.findComponent({ name: 'Doughnut' }).props('data').labels).toEqual(['Unknown', '/v1/responses'])
    expect(wrapper.find('tbody td').text()).toBe('Unknown')
    expect(wrapper.find('tbody td').attributes('title')).toBe('Unknown')
    wrapper.unmount()
  })

  it('sorts expanded user rows by the parent metric without mutating the source', async () => {
    const items = values.map((value, index) => Object.freeze({ ...value, user_id: index + 1, email: `user-${index}@example.com`, cache_tokens: 0, account_cost: 0 }))
    Object.freeze(items)
    const wrapper = mount(UserBreakdownSubTable, { props: { items } })
    const labels = () => wrapper.findAll('tbody tr').map(row => row.get('td').text())
    expect(labels()).toEqual(['user-0@example.com', 'user-1@example.com'])
    await wrapper.setProps({ metric: 'actual_cost' })
    expect(labels()).toEqual(['user-1@example.com', 'user-0@example.com'])
    await wrapper.setProps({ metric: 'tokens' })
    expect(labels()).toEqual(['user-0@example.com', 'user-1@example.com'])
    expect(items.map(item => item.user_id)).toEqual([1, 2])
    wrapper.unmount()
  })

  it('sorts the spending ranking by actual cost and retains its user click mapping', async () => {
    const rankingItems = [
      { user_id: 1, username: 'token-first', email: '', requests: 20, tokens: 9000, actual_cost: 0.2 },
      { user_id: 2, username: 'cost-first', email: '', requests: 5, tokens: 100, actual_cost: 4 },
    ]
    Object.freeze(rankingItems)
    const wrapper = mount(ModelDistributionChart, { props: { modelStats: [], enableRankingView: true, rankingItems } })
    await wrapper.findAll('button').find(button => button.text() === 'admin.dashboard.viewSpendingRanking')!.trigger('click')
    expect(wrapper.findComponent({ name: 'Doughnut' }).props('data').labels).toEqual(['#1 cost-first', '#2 token-first'])
    expect(wrapper.findAll('tbody tr')[0].text()).toContain('cost-first')
    await wrapper.find('tbody tr').trigger('click')
    expect(wrapper.emitted('ranking-click')).toEqual([[rankingItems[1]]])
    expect(rankingItems.map(item => item.user_id)).toEqual([1, 2])
    wrapper.unmount()
  })
})
