import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import UsageView from '../UsageView.vue'
import PortalUsageAnalytics from '@/components/portal/PortalUsageAnalytics.vue'
import { usageMonthWindow } from '@/utils/portalUsage'

const { query, listMyErrorRequests, getStats, getDashboardTrend, getDashboardSnapshotV2, listKeys, getAvailable } = vi.hoisted(() => ({
  query: vi.fn(),
  listMyErrorRequests: vi.fn(),
  getStats: vi.fn(),
  getDashboardTrend: vi.fn(),
  getDashboardSnapshotV2: vi.fn(),
  listKeys: vi.fn(),
  getAvailable: vi.fn(),
}))
const storeSettings = vi.hoisted(() => ({ allow_user_view_error_requests: false, subscription_enabled: true }))

vi.mock('@/api', () => ({
  usageAPI: { query, listMyErrorRequests, getStats, getDashboardTrend, getDashboardSnapshotV2, getById: vi.fn(), getMyErrorDetail: vi.fn() },
  keysAPI: { list: listKeys },
  userGroupsAPI: { getAvailable },
}))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => ({ user: { balance: 10, is_simple_mode: false }, isSimpleMode: false }) }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ cachedPublicSettings: storeSettings }) }))
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const { ref } = await import('vue')
  return { ...actual, useI18n: () => ({ locale: ref('en-US'), t: (key: string) => key }) }
})

function mountUsage() {
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/usage', component: UsageView }] })
  void router.push('/usage')
  return mount(UsageView, {
    global: {
      plugins: [router],
      stubs: {
        RouterLink: true,
        PortalUsageChart: true,
        PortalUsageAnalytics: true,
        PortalUsageDetail: true,
        PortalDialog: true,
        PortalUsageTable: { props: ['rows'], template: '<div data-testid="usage-table">{{ rows.map(row => row.id).join(",") }}</div>' },
        PortalUsageErrorTable: { props: ['rows'], template: '<div data-testid="error-table">{{ rows.map(row => row.id).join(",") }}</div>' },
      },
    },
  })
}

describe('portal UsageView access and request races', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    storeSettings.allow_user_view_error_requests = false
    query.mockResolvedValue({ items: [{ id: 1 }], total: 1, pages: 1, page_size: 10 })
    listMyErrorRequests.mockResolvedValue({ items: [{ id: 77 }], total: 1, pages: 1, page_size: 10 })
    getStats.mockResolvedValue({ total_actual_cost: 0, total_requests: 0 })
    getDashboardTrend.mockResolvedValue({ trend: [] })
    getDashboardSnapshotV2.mockResolvedValue({ models: [], groups: [] })
    listKeys.mockResolvedValue({ items: [], pages: 0 })
    getAvailable.mockResolvedValue([])
  })

  it('does not expose or request error records when that feature is disabled', async () => {
    const wrapper = mountUsage()
    await flushPromises()
    const dates = usageMonthWindow()
    expect(getStats).toHaveBeenCalledWith(expect.objectContaining({
      start_date: dates.currentStart,
      end_date: dates.end,
      timezone: expect.any(String),
    }))
    expect(wrapper.find('[role="tablist"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('Error requests')
    expect(listMyErrorRequests).not.toHaveBeenCalled()
    expect(wrapper.get('[data-testid="usage-table"]').text()).toBe('1')
    wrapper.unmount()
  })

  it('restores same-route drilldown conditions and clears unrelated filters on back navigation', async () => {
    const wrapper = mountUsage()
    await flushPromises()
    const router = wrapper.vm.$router
    await router.push('/usage?start_date=2026-09-04&end_date=2026-09-04&model=gpt-5&api_key_id=7')
    await flushPromises()
    expect(query).toHaveBeenLastCalledWith(expect.objectContaining({ page: 1, start_date: '2026-09-04', end_date: '2026-09-04', model: 'gpt-5', api_key_id: 7 }), expect.anything())
    expect(wrapper.get('.usage-filter-summary').text()).toContain('gpt-5')
    await router.push('/usage?start_date=2026-09-07&end_date=2026-09-07&model=claude-sonnet')
    await flushPromises()
    expect(query).toHaveBeenLastCalledWith(expect.objectContaining({ page: 1, start_date: '2026-09-07', model: 'claude-sonnet', api_key_id: undefined, group_id: undefined }), expect.anything())
    router.back()
    await flushPromises()
    expect(query).toHaveBeenLastCalledWith(expect.objectContaining({ page: 1, start_date: '2026-09-04', model: 'gpt-5', api_key_id: 7 }), expect.anything())
    wrapper.unmount()
  })

  it('resets to the first page when the same chart drilldown is selected again', async () => {
    query.mockResolvedValue({ items: [{ id: 1 }], total: 30, pages: 3, page_size: 10 })
    const wrapper = mountUsage()
    await flushPromises()
    const selection = { start: '2026-10-04', end: '2026-10-04', model: 'gpt-5' }
    wrapper.getComponent(PortalUsageAnalytics).vm.$emit('drilldown', selection)
    await flushPromises()
    await wrapper.get('[aria-label="Next page"]').trigger('click')
    await flushPromises()
    expect(query).toHaveBeenLastCalledWith(expect.objectContaining({ page: 2 }), expect.anything())
    wrapper.getComponent(PortalUsageAnalytics).vm.$emit('drilldown', selection)
    await flushPromises()
    expect(query).toHaveBeenLastCalledWith(expect.objectContaining({ page: 1, model: 'gpt-5' }), expect.anything())
    wrapper.unmount()
  })

  it('cancels stale usage rows when switching to enabled errors and ignores the old response', async () => {
    storeSettings.allow_user_view_error_requests = true
    let resolveUsage!: (value: { items: Array<{ id: number }>; total: number; pages: number; page_size: number }) => void
    let usageSignal!: AbortSignal
    query.mockImplementationOnce((_params, options) => {
      return new Promise(resolve => {
        resolveUsage = resolve
        usageSignal = options.signal
        expect(options.signal).toBeInstanceOf(AbortSignal)
      })
    })
    const wrapper = mountUsage()
    await flushPromises()
    const tabs = wrapper.findAll('[role="tab"]')
    expect(tabs).toHaveLength(2)
    await tabs[1].trigger('click')
    await flushPromises()
    expect(usageSignal.aborted).toBe(true)
    expect(listMyErrorRequests).toHaveBeenCalledOnce()
    expect(wrapper.get('[data-testid="error-table"]').text()).toBe('77')

    resolveUsage({ items: [{ id: 55 }], total: 1, pages: 1, page_size: 10 })
    await flushPromises()
    expect(wrapper.get('[data-testid="error-table"]').text()).toBe('77')
    expect(wrapper.find('[data-testid="usage-table"]').exists()).toBe(false)
    wrapper.unmount()
  })

  it('exports every page once and neutralizes spreadsheet formulas in the downloaded CSV', async () => {
    const wrapper = mountUsage()
    await flushPromises()
    query.mockClear()
    query.mockImplementation(async params => params.page === 1
      ? { items: [{ id: 1, request_id: '=1+1', model: 'model-a' }], total: 2, pages: 2, page_size: 100 }
      : { items: [
        { id: 1, request_id: '=1+1', model: 'duplicate-row' },
        { id: 2, request_id: 'safe-id', model: 'model-b' },
      ], total: 2, pages: 2, page_size: 100 })

    let csv = ''
    const OriginalBlob = globalThis.Blob
    vi.stubGlobal('Blob', vi.fn((parts: BlobPart[], options?: BlobPropertyBag) => {
      csv = parts.map(part => String(part)).join('')
      return new OriginalBlob(parts, options)
    }))
    const originalCreateObjectURL = window.URL.createObjectURL
    const originalRevokeObjectURL = window.URL.revokeObjectURL
    window.URL.createObjectURL = vi.fn(() => 'blob:portal-usage-export')
    window.URL.revokeObjectURL = vi.fn()
    const click = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})

    try {
      await wrapper.findAll('button').find(button => button.text() === 'Export CSV')!.trigger('click')
      await flushPromises()

      expect(query.mock.calls.map(([params]) => [params.page, params.page_size])).toEqual([[1, 100], [2, 100]])
      expect(csv).toContain('"\'=1+1"')
      expect(csv).toContain('"safe-id"')
      expect(csv).not.toContain('duplicate-row')
      expect(wrapper.get('[role="status"]').text()).toContain('Exported 2 records')
      expect(click).toHaveBeenCalledOnce()
    } finally {
      window.URL.createObjectURL = originalCreateObjectURL
      window.URL.revokeObjectURL = originalRevokeObjectURL
      vi.unstubAllGlobals()
      click.mockRestore()
      wrapper.unmount()
    }
  })
})
