import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import PortalUsageAnalytics from '../PortalUsageAnalytics.vue'

const { getStats, getDashboardModels, getDashboardSnapshotV2 } = vi.hoisted(() => ({ getStats: vi.fn(), getDashboardModels: vi.fn(), getDashboardSnapshotV2: vi.fn() }))
vi.mock('@/api', () => ({ usageAPI: { getStats, getDashboardModels, getDashboardSnapshotV2 } }))
vi.mock('vue-i18n', async () => {
  const { ref } = await import('vue')
  return { useI18n: () => ({ locale: ref('en-US') }) }
})
vi.mock('vue-chartjs', () => ({
  Doughnut: { name: 'Doughnut', props: ['data', 'options'], template: '<div class="doughnut-stub" />' },
  Line: { name: 'Line', props: ['data', 'options'], template: '<div class="line-stub" />' },
}))

const summary = (requests = 42) => ({ total_requests: requests, total_tokens: 1800, total_input_tokens: 1000, total_output_tokens: 100, total_cache_creation_tokens: 200, total_cache_read_tokens: 500, total_cache_tokens: 700, total_cost: 1.5, total_actual_cost: 0.5, average_duration_ms: 2000, endpoints: [{ endpoint: '/v1/responses', requests, total_tokens: 1800, cost: 1.5, actual_cost: 0.5 }] })
const model = { model: 'claude-sonnet', requests: 42, total_tokens: 1800, cost: 1.5, actual_cost: 0.5 }
const point = { date: '2026-10-10', requests: 42, input_tokens: 1000, output_tokens: 100, cache_creation_tokens: 200, cache_read_tokens: 500, total_tokens: 1800, cost: 1.5, actual_cost: 0.5 }
const mountAnalytics = () => mount(PortalUsageAnalytics)

describe('PortalUsageAnalytics', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    getStats.mockResolvedValue(summary())
    getDashboardModels.mockResolvedValue({ models: [model] })
    getDashboardSnapshotV2.mockResolvedValue({ groups: [{ group_id: 1, group_name: 'Claude group', requests: 42, total_tokens: 1800, cost: 1.5, actual_cost: 0.5 }], trend: [point] })
  })

  it('uses complete aggregate APIs, requested models, colored model icons and token/cache trend', async () => {
    const wrapper = mountAnalytics()
    await flushPromises()
    expect(getDashboardModels).toHaveBeenCalledWith(expect.objectContaining({ model_source: 'requested', timezone: expect.any(String) }))
    expect(getDashboardSnapshotV2).toHaveBeenCalledWith(expect.objectContaining({ include_trend: true, include_group_stats: true, include_model_stats: false }))
    expect(wrapper.get('[data-testid="analytics-requests"]').text()).toBe('42')
    expect(wrapper.text()).toContain('Independent of the record filters below')
    expect(wrapper.get('[data-distribution="models"] .model-icon').exists()).toBe(true)
    expect(wrapper.text()).toContain('/v1/responses')
    const line = wrapper.findComponent({ name: 'Line' })
    const datasets = line.props('data').datasets
    expect(datasets).toHaveLength(5)
    expect(datasets[4].data[0]).toBeCloseTo(500 / 1700 * 100)
    expect(new Set(datasets.map((item: { borderColor: string }) => item.borderColor)).size).toBe(5)
    wrapper.unmount()
  })

  it('changes distribution metrics without requesting or changing the source amounts', async () => {
    const wrapper = mountAnalytics()
    await flushPromises()
    const section = wrapper.get('[data-distribution="models"]')
    await section.findAll('button')[1].trigger('click')
    expect(section.findAll('button')[1].attributes('aria-pressed')).toBe('true')
    expect(wrapper.findComponent({ name: 'Doughnut' }).props('data').datasets[0].data).toEqual([0.5])
    expect(getStats).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })

  it('applies calendar date ranges to every aggregate and reloads the selected granularity', async () => {
    const wrapper = mountAnalytics()
    await flushPromises()
    await wrapper.get('[data-testid="analytics-preset"]').setValue('custom')
    await wrapper.get('[data-testid="analytics-start"]').setValue('2026-09-01')
    await wrapper.get('[data-testid="analytics-end"]').setValue('2026-09-03')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(getStats).toHaveBeenLastCalledWith(expect.objectContaining({ start_date: '2026-09-01', end_date: '2026-09-03' }))
    expect(getDashboardModels).toHaveBeenLastCalledWith(expect.objectContaining({ start_date: '2026-09-01', end_date: '2026-09-03' }))
    await wrapper.get('[data-testid="analytics-granularity"]').setValue('hour')
    await flushPromises()
    expect(getDashboardSnapshotV2).toHaveBeenLastCalledWith(expect.objectContaining({ granularity: 'hour', start_date: '2026-09-01' }))
    await wrapper.get('[data-testid="analytics-start"]').setValue('2026-09-04')
    const count = getStats.mock.calls.length
    await wrapper.get('form').trigger('submit')
    expect(getStats).toHaveBeenCalledTimes(count)
    expect(wrapper.text()).toContain('Choose a valid date range')
    wrapper.unmount()
  })

  it('rejects stale responses after a newer range finishes', async () => {
    let resolveOld!: (value: ReturnType<typeof summary>) => void
    getStats.mockImplementationOnce(() => new Promise(resolve => { resolveOld = resolve }))
    const wrapper = mountAnalytics()
    await flushPromises()
    await wrapper.get('[data-testid="analytics-preset"]').setValue('today')
    await flushPromises()
    expect(wrapper.get('[data-testid="analytics-requests"]').text()).toBe('42')
    resolveOld(summary(999))
    await flushPromises()
    expect(wrapper.get('[data-testid="analytics-requests"]').text()).toBe('42')
    wrapper.unmount()
  })

  it('reports partial failures without presenting failed totals as zero or hiding healthy sections', async () => {
    getStats.mockRejectedValueOnce(new Error('network'))
    const wrapper = mountAnalytics()
    await flushPromises()
    expect(wrapper.get('[data-testid="analytics-requests"]').text()).toBe('—')
    expect(wrapper.text()).toContain('Summary and endpoint statistics could not be loaded')
    expect(wrapper.get('[data-distribution="models"]').text()).toContain('claude-sonnet')
    expect(wrapper.get('[data-distribution="groups"]').text()).toContain('Claude group')
    wrapper.unmount()
  })

  it('keeps zero-metric records visible, distinguishes empty sections, and never prints NaN for absent summary values', async () => {
    getStats.mockResolvedValueOnce({ total_requests: 0, endpoints: [] })
    getDashboardModels.mockResolvedValueOnce({ models: [{ ...model, total_tokens: 0, actual_cost: 0 }] })
    getDashboardSnapshotV2.mockResolvedValueOnce({ groups: [], trend: [] })
    const wrapper = mountAnalytics()
    await flushPromises()
    expect(wrapper.get('[data-testid="analytics-requests"]').text()).toBe('0')
    expect(wrapper.text()).not.toContain('NaN')
    expect(wrapper.get('[data-distribution="models"]').text()).toContain('Metric total is 0')
    expect(wrapper.get('[data-distribution="models"]').text()).toContain('claude-sonnet')
    expect(wrapper.get('[data-distribution="groups"]').text()).toContain('No requests in this period')
    expect(wrapper.findComponent({ name: 'Line' }).exists()).toBe(false)
    wrapper.unmount()
  })
})
