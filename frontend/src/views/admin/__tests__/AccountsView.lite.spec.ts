import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { DOMWrapper, flushPromises, mount } from '@vue/test-utils'
import { defineComponent } from 'vue'

import AccountsView from '../AccountsView.vue'
import AccountActionMenu from '@/components/admin/account/AccountActionMenu.vue'

const {
  listAccounts,
  listWithEtag,
  getById,
  getBatchTodayStats,
  getUpstreamBillingProbeSettings,
  getSummary,
  getAllProxies,
  getAllGroups,
  refreshCredentials,
  exportData,
  setSchedulable,
  showError,
  showWarning
} = vi.hoisted(() => ({
  listAccounts: vi.fn(),
  listWithEtag: vi.fn(),
  getById: vi.fn(),
  getBatchTodayStats: vi.fn(),
  getUpstreamBillingProbeSettings: vi.fn(),
  getSummary: vi.fn(),
  getAllProxies: vi.fn(),
  getAllGroups: vi.fn(),
  refreshCredentials: vi.fn(),
  exportData: vi.fn(),
  setSchedulable: vi.fn(),
  showError: vi.fn(),
  showWarning: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      list: listAccounts,
      getById,
      listWithEtag,
      getBatchTodayStats,
      getUpstreamBillingProbeSettings,
      getSummary,
      getOAuthUsageSummary: vi.fn().mockResolvedValue({ oauth_account_count: 0, total_standard_cost: 0 }),
      delete: vi.fn(),
      batchClearError: vi.fn(),
      batchRefresh: vi.fn(),
      toggleSchedulable: vi.fn(),
      refreshCredentials,
      exportData,
      setSchedulable
    },
    proxies: { getAllWithCount: getAllProxies },
    groups: { getAll: getAllGroups }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError, showWarning, showSuccess: vi.fn(), showInfo: vi.fn() })
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({ token: 'test-token', isSimpleMode: false })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({
    t: (key: string, params?: { count?: number }) => params?.count == null ? key : `${key} (${params.count})`
  }) }
})

const DataTableStub = defineComponent({
  props: { data: { type: Array, default: () => [] } },
  template: `
    <div>
      <div v-for="row in data" :key="row.id" :data-account-name="row.name">
        <slot name="cell-groups" :row="row" />
        <slot name="cell-schedulable" :row="row" />
        <slot name="cell-actions" :row="row" />
      </div>
    </div>
  `
})

const AccountGroupsCellStub = defineComponent({
  props: { groups: { type: Array, default: () => [] } },
  template: '<span data-test="account-groups">{{ groups.map(group => group.name).join(",") }}</span>'
})

const EditAccountModalStub = defineComponent({
  props: { show: Boolean, account: { type: Object, default: null } },
  template: '<div data-test="edit-account">{{ show ? account?.name : "" }}</div>'
})

const AccountTestModalStub = defineComponent({
  props: { show: Boolean, account: { type: Object, default: null } },
  template: '<div data-test="test-account">{{ show ? account?.name : "" }}</div>'
})

const AccountStatsModalStub = defineComponent({
  props: { show: Boolean, account: { type: Object, default: null } },
  template: '<div data-test="stats-account">{{ show ? account?.name : "" }}</div>'
})

function mountView(stubActionMenu = true) {
  return mount(AccountsView, {
    attachTo: document.body,
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>' },
        DataTable: DataTableStub,
        AccountTableActions: { template: '<div><slot name="after" /></div>' },
        AccountTableFilters: true,
        AccountBulkActionsBar: true,
        Pagination: true,
        ConfirmDialog: true,
        AccountActionMenu: stubActionMenu,
        ImportDataModal: true,
        ReAuthAccountModal: true,
        AccountTestModal: AccountTestModalStub,
        AccountStatsModal: AccountStatsModalStub,
        ScheduledTestsPanel: true,
        SyncFromCrsModal: true,
        TempUnschedStatusModal: true,
        ErrorPassthroughRulesModal: true,
        TLSFingerprintProfilesModal: true,
        CreateAccountModal: true,
        EditAccountModal: EditAccountModalStub,
        BulkEditAccountModal: true,
        PlatformTypeBadge: true,
        AccountCapacityCell: true,
        AccountStatusIndicator: true,
        AccountTodayStatsCell: true,
        AccountGroupsCell: AccountGroupsCellStub,
        AccountUsageCell: true,
        UpstreamBillingRateCell: true,
        HelpTooltip: true,
        Icon: true,
        Teleport: stubActionMenu
      }
    }
  })
}

const listRow = {
  id: 42,
  name: 'compact row',
  platform: 'openai',
  type: 'oauth',
  status: 'active',
  schedulable: true,
  concurrency: 2,
  priority: 1,
  group_ids: [7],
  extra: {},
  credentials: {}
}

const fullAccount = {
  ...listRow,
  groups: [{ id: 7, name: 'codex', platform: 'openai' }],
  account_groups: [{ account_id: 42, group_id: 7 }],
  credentials: { api_key: 'redacted' },
  extra: { detail_only: true }
}

describe('admin AccountsView lite account list', () => {
  beforeEach(() => {
    localStorage.clear()
    listAccounts.mockReset().mockResolvedValue({ items: [listRow], total: 1, page: 1, page_size: 20, pages: 1 })
    listWithEtag.mockReset().mockResolvedValue({ notModified: true, etag: 'compact-etag', data: null })
    getById.mockReset().mockResolvedValue(fullAccount)
    getBatchTodayStats.mockReset().mockResolvedValue({ stats: {} })
    getUpstreamBillingProbeSettings.mockReset().mockResolvedValue({ enabled: true })
    getSummary.mockReset().mockResolvedValue({ total: 0, available: 0, proxy_distribution: [] })
    getAllProxies.mockReset().mockResolvedValue([])
    getAllGroups.mockReset().mockResolvedValue([{ id: 7, name: 'codex', platform: 'openai' }])
    refreshCredentials.mockReset()
    exportData.mockReset()
    setSchedulable.mockReset()
    showError.mockReset()
    showWarning.mockReset()
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  it('keeps lite=1 on the initial list request', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(listAccounts).toHaveBeenCalledWith(
      1,
      20,
      expect.objectContaining({ lite: '1' }),
      expect.objectContaining({ signal: expect.any(AbortSignal) })
    )
    wrapper.unmount()
  })

  it('restores platform sorting and requests a fresh first page when mobile sorting changes', async () => {
    localStorage.setItem('account-table-sort', JSON.stringify({ key: 'platform_type', order: 'desc' }))
    const wrapper = mountView()
    await flushPromises()
    expect(listAccounts).toHaveBeenLastCalledWith(
      1, 20,
      expect.objectContaining({ lite: '1', sort_by: 'platform_type', sort_order: 'desc' }),
      expect.objectContaining({ signal: expect.any(AbortSignal) })
    )

    wrapper.getComponent(DataTableStub).vm.$emit('sort', 'id', 'asc')
    await flushPromises()
    expect(listAccounts).toHaveBeenLastCalledWith(
      1, 20,
      expect.objectContaining({ lite: '1', sort_by: 'id', sort_order: 'asc' }),
      expect.objectContaining({ signal: expect.any(AbortSignal) })
    )
    wrapper.unmount()
  })

  it('maps group_ids through the group catalog for the table cell', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.get('[data-test="account-groups"]').text()).toBe('codex')
    wrapper.unmount()
  })

  it('shows every proxy node by its cached region and address without inferring location from names', async () => {
    getSummary.mockResolvedValue({
      openai: 6,
      proxy_distribution: [
        { proxy_id: 1, name: 'old Tokyo node', total: 1, available: 1 },
        { proxy_id: 2, name: 'backup', total: 1, available: 1 },
        { proxy_id: 3, name: 'Japan label without geo data', total: 1, available: 1 },
        { proxy_id: 4, name: 'missing metadata', total: 1, available: 0 },
        { proxy_id: 5, name: 'account-attached proxy', total: 1, available: 1 },
        { proxy_id: null, name: 'Direct', total: 1, available: 1 }
      ]
    })
    getAllProxies.mockResolvedValue([
      { id: 1, name: 'old Tokyo node', host: '192.0.2.10', ip_address: '203.0.113.10', country: 'Singapore', city: 'Singapore' },
      { id: 2, name: 'backup', host: '192.0.2.20', ip_address: '198.51.100.20', country_code: 'us', region: 'California', city: 'Los Angeles' },
      { id: 3, name: 'Japan label without geo data', host: '192.0.2.30' }
    ])
    listAccounts.mockResolvedValue({
      items: [{ ...listRow, proxy: { id: 5, name: 'account-attached proxy', host: '192.0.2.50', country: 'Germany' } }],
      total: 1, page: 1, page_size: 20, pages: 1
    })
    const wrapper = mountView()
    await flushPromises()

    expect(getAllProxies).toHaveBeenCalledOnce()
    expect(wrapper.get('[data-test="proxy-node-count"]').text()).toBe('admin.accounts.summary.proxyNodeCount (5)')
    const nodes = wrapper.findAll('.proxy-health-row')
    expect(nodes).toHaveLength(6)
    expect(nodes[0].get('[data-test="proxy-node-location"]').text()).toBe('Singapore')
    expect(nodes[0].get('[data-test="proxy-node-address"]').text()).toBe('admin.accounts.summary.exitIp: 203.0.113.10')
    expect(nodes[0].text()).not.toContain('Tokyo')
    expect(nodes[1].get('[data-test="proxy-node-location"]').text()).toBe('US · California · Los Angeles')
    expect(nodes[2].get('[data-test="proxy-node-location"]').text()).toBe('admin.accounts.summary.unknownProxyRegion')
    expect(nodes[2].get('[data-test="proxy-node-address"]').text()).toBe('admin.accounts.summary.proxyAddress: 192.0.2.30')
    expect(nodes[2].text()).not.toContain('admin.accounts.summary.exitIp')
    expect(nodes[3].get('[data-test="proxy-node-location"]').text()).toBe('admin.accounts.summary.unknownProxyRegion')
    expect(nodes[3].find('[data-test="proxy-node-address"]').exists()).toBe(false)
    expect(nodes[4].get('[data-test="proxy-node-location"]').text()).toBe('Germany')
    expect(nodes[4].get('[data-test="proxy-node-address"]').text()).toBe('admin.accounts.summary.proxyAddress: 192.0.2.50')
    expect(nodes[5].get('[data-test="proxy-node-location"]').text()).toBe('admin.accounts.noProxy')
    expect(nodes[5].find('[data-test="proxy-node-address"]').exists()).toBe(false)
    wrapper.unmount()
  })

  it('exposes a labelled scheduling switch and disables it while the existing update is pending', async () => {
    let finishUpdate!: (result: { schedulable: boolean }) => void
    setSchedulable.mockImplementationOnce(() => new Promise(resolve => { finishUpdate = resolve }))
    const wrapper = mountView()
    await flushPromises()

    const control = wrapper.get('[role="switch"]')
    expect(control.attributes('type')).toBe('button')
    expect(control.attributes('aria-label')).toBe('admin.accounts.columns.schedulable · compact row')
    expect(control.attributes('aria-checked')).toBe('true')
    expect(control.attributes('aria-busy')).toBe('false')
    expect(control.get('.account-schedule-track').element.tagName).toBe('SPAN')
    expect(control.get('.account-schedule-thumb').element.tagName).toBe('SPAN')

    await control.trigger('click')
    expect(setSchedulable).toHaveBeenCalledWith(42, false)
    expect(control.attributes('disabled')).toBeDefined()
    expect(control.attributes('aria-busy')).toBe('true')
    expect(control.attributes('aria-checked')).toBe('true')

    finishUpdate({ schedulable: false })
    await flushPromises()
    expect(control.attributes('disabled')).toBeUndefined()
    expect(control.attributes('aria-busy')).toBe('false')
    expect(control.attributes('aria-checked')).toBe('false')
    wrapper.unmount()
  })

  it('keeps the scheduling state and restores the control when an update fails', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => {})
    setSchedulable.mockRejectedValueOnce(new Error('update failed'))
    const wrapper = mountView()
    await flushPromises()

    const control = wrapper.get('[role="switch"]')
    await control.trigger('click')
    await flushPromises()
    expect(setSchedulable).toHaveBeenCalledWith(42, false)
    expect(control.attributes('aria-checked')).toBe('true')
    expect(control.attributes('aria-busy')).toBe('false')
    expect(control.attributes('disabled')).toBeUndefined()
    expect(showError).toHaveBeenCalledWith('admin.accounts.failedToToggleSchedulable')
    wrapper.unmount()
  })

  it('keeps account tools and column settings collapsed until the tools menu is opened', async () => {
    const wrapper = mountView()
    await flushPromises()

    const trigger = wrapper.get('[title="admin.accounts.moreActions"]')
    expect(trigger.attributes('aria-expanded')).toBe('false')
    expect(wrapper.find('.account-tools-menu-item').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('admin.accounts.viewColumns')
    await trigger.trigger('click')
    expect(trigger.attributes('aria-expanded')).toBe('true')
    expect(wrapper.findAll('.account-tools-menu-item').map(button => button.text())).toEqual([
      'admin.accounts.syncFromCrs',
      'admin.accounts.dataImport',
      'admin.accounts.dataExport',
      'admin.errorPassthrough.title',
      'admin.tlsFingerprintProfiles.title'
    ])
    expect(wrapper.text()).toContain('admin.accounts.viewColumns')
    await trigger.trigger('click')

    for (const [label, modalName] of [
      ['admin.accounts.syncFromCrs', 'SyncFromCrsModal'],
      ['admin.accounts.dataImport', 'ImportDataModal'],
      ['admin.errorPassthrough.title', 'ErrorPassthroughRulesModal'],
      ['admin.tlsFingerprintProfiles.title', 'TLSFingerprintProfilesModal']
    ]) {
      const modal = wrapper.findComponent({ name: modalName })
      expect(modal.props('show')).toBe(false)
      await trigger.trigger('click')
      await wrapper.findAll('.account-tools-menu-item').find(button => button.text() === label)!.trigger('click')
      expect(modal.props('show')).toBe(true)
      expect(trigger.attributes('aria-expanded')).toBe('false')
      modal.vm.$emit('close')
      await flushPromises()
      expect(modal.props('show')).toBe(false)
    }

    await trigger.trigger('click')
    await wrapper.findAll('.account-tools-menu-item').find(button => button.text() === 'admin.accounts.dataExport')!.trigger('click')
    const exportDialog = wrapper.findAllComponents({ name: 'ConfirmDialog' })
      .find(dialog => dialog.props('title') === 'admin.accounts.dataExport')!
    expect(exportDialog.props('show')).toBe(true)
    expect(exportData).not.toHaveBeenCalled()
    exportDialog.vm.$emit('cancel')
    await flushPromises()
    expect(exportDialog.props('show')).toBe(false)
    wrapper.unmount()
  })

  it('preserves column visibility changes through the collapsed tools menu', async () => {
    const wrapper = mountView()
    await flushPromises()

    await wrapper.get('[title="admin.accounts.moreActions"]').trigger('click')
    const scoreToggle = wrapper.findAll('button')
      .find(button => button.text() === 'admin.accounts.columns.schedulerScore')!
    expect(listAccounts.mock.calls[0]?.[2]).toEqual(expect.objectContaining({ include_scheduler_score: '0' }))

    await scoreToggle.trigger('click')
    await flushPromises()
    expect(JSON.parse(localStorage.getItem('account-hidden-columns') || '[]')).not.toContain('scheduler_score')
    expect(listAccounts.mock.lastCall?.[2]).toEqual(expect.objectContaining({ include_scheduler_score: '1' }))

    await scoreToggle.trigger('click')
    await flushPromises()
    expect(JSON.parse(localStorage.getItem('account-hidden-columns') || '[]')).toContain('scheduler_score')
    expect(listAccounts.mock.lastCall?.[2]).toEqual(expect.objectContaining({ include_scheduler_score: '0' }))
    wrapper.unmount()
  })

  it('keeps the action menu open during internal scrolling but closes it on table scrolling', async () => {
    const wrapper = mountView(false)
    await flushPromises()

    const trigger = wrapper.findAll('button').find(button => button.text() === 'common.more')!
    await trigger.trigger('click')
    const menu = new DOMWrapper(document.body.querySelector('.action-menu-content')!)
    menu.element.dispatchEvent(new Event('scroll'))
    await flushPromises()
    expect(wrapper.findComponent(AccountActionMenu).props('show')).toBe(true)

    menu.get('button').element.dispatchEvent(new Event('scroll'))
    await flushPromises()
    expect(wrapper.findComponent(AccountActionMenu).props('show')).toBe(true)

    wrapper.getComponent(DataTableStub).element.dispatchEvent(new Event('scroll'))
    await flushPromises()
    expect(wrapper.findComponent(AccountActionMenu).props('show')).toBe(false)
    wrapper.unmount()
  })

  it('keeps lite=1 on automatic ETag refreshes', async () => {
    vi.useFakeTimers()
    vi.spyOn(document, 'hidden', 'get').mockReturnValue(false)
    localStorage.setItem('account-auto-refresh', JSON.stringify({ enabled: true, interval_seconds: 5 }))
    const wrapper = mountView()
    await flushPromises()

    await vi.advanceTimersByTimeAsync(6000)
    await flushPromises()

    expect(listWithEtag).toHaveBeenCalledWith(
      1,
      20,
      expect.objectContaining({ lite: '1' }),
      expect.objectContaining({ etag: null })
    )
    wrapper.unmount()
  })

  it('loads the full account by id before opening edit, test, and stats actions', async () => {
    const wrapper = mountView()
    await flushPromises()

    const editButton = wrapper.findAll('button').find(button => button.text().includes('common.edit'))
    expect(editButton).toBeTruthy()
    await editButton!.trigger('click')
    await flushPromises()
    expect(getById).toHaveBeenCalledWith(42)
    expect(wrapper.get('[data-test="edit-account"]').text()).toBe('compact row')

    const menu = wrapper.findComponent(AccountActionMenu)
    menu.vm.$emit('test', listRow)
    await flushPromises()
    expect(getById).toHaveBeenCalledTimes(2)
    expect(wrapper.get('[data-test="test-account"]').text()).toBe('compact row')

    menu.vm.$emit('stats', listRow)
    await flushPromises()
    expect(getById).toHaveBeenCalledTimes(3)
    expect(wrapper.get('[data-test="stats-account"]').text()).toBe('compact row')
    wrapper.unmount()
  })

  it('shows the warning and patches the account after a partial Antigravity refresh', async () => {
    refreshCredentials.mockResolvedValue({
      account: { ...fullAccount, name: 'refreshed account' },
      message: 'Token refreshed, but project_id is temporarily unavailable',
      warning: 'missing_project_id_temporary'
    })
    const wrapper = mountView(false)
    await flushPromises()

    wrapper.findComponent(AccountActionMenu).vm.$emit('refresh-token', listRow)
    await flushPromises()

    expect(refreshCredentials).toHaveBeenCalledWith(42)
    expect(wrapper.get('[data-account-name]').attributes('data-account-name')).toBe('refreshed account')
    expect(showWarning).toHaveBeenCalledWith('Token refreshed, but project_id is temporarily unavailable')
    wrapper.unmount()
  })

  it('shows an error and keeps the modal closed when detail loading fails', async () => {
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {})
    getById.mockRejectedValueOnce(new Error('detail failed'))
    const wrapper = mountView()
    await flushPromises()

    const editButton = wrapper.findAll('button').find(button => button.text().includes('common.edit'))
    await editButton!.trigger('click')
    await flushPromises()

    expect(showError).toHaveBeenCalledWith('detail failed')
    expect(wrapper.get('[data-test="edit-account"]').text()).toBe('')
    consoleError.mockRestore()
    wrapper.unmount()
  })
})
