import { afterEach, describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick, ref } from 'vue'
import PortalUsageChart from '../PortalUsageChart.vue'

const state = vi.hoisted(() => ({ resize: undefined as undefined | ((entries: { contentRect: { width: number } }[]) => void) }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ locale: ref('en') }) }))
vi.mock('@vueuse/core', () => ({ useMediaQuery: () => ref(true), useResizeObserver: (_target: unknown, callback: typeof state.resize) => { state.resize = callback } }))
afterEach(() => vi.restoreAllMocks())
const months = Array.from({ length: 12 }, (_, index) => ({ key: `2026-${String(index + 1).padStart(2, '0')}`, label: `${index + 1}月`, cost: index + 1, requests: index * 3, tokens: index * 100 }))

describe('responsive monthly usage chart', () => {
  it.each([0, 1, 12])('uses a 320px container without invalid coordinates for %s months', async count => {
    const wrapper = mount(PortalUsageChart, { props: { months: months.slice(0, count) } })
    state.resize?.([{ contentRect: { width: 320 } }])
    await nextTick()
    expect(wrapper.get('svg').attributes('viewBox')).toBe('0 0 320 220')
    expect(wrapper.html()).not.toMatch(/NaN|Infinity/)
    expect(wrapper.findAll('.chart-point')).toHaveLength(count)
    if (count === 12) expect(wrapper.findAll('.chart-point text')).toHaveLength(5)
    wrapper.unmount()
  })
  it('selects every month without hover and retains all data points on resize', async () => {
    const wrapper = mount(PortalUsageChart, { props: { months } })
    expect(wrapper.findAll('option')).toHaveLength(12)
    for (const month of months) {
      await wrapper.get('select').setValue(month.key)
      expect(wrapper.get('.chart-detail').text()).toContain(month.key)
    }
    await wrapper.findAll('.chart-point')[0].trigger('click')
    expect(wrapper.get('.chart-detail').text()).toContain('2026-01')
    state.resize?.([{ contentRect: { width: 900 } }])
    await nextTick()
    expect(wrapper.get('svg').attributes('viewBox')).toBe('0 0 900 220')
    expect(wrapper.findAll('.chart-point')).toHaveLength(12)
    wrapper.unmount()
  })
})
