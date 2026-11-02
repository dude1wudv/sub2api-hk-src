import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, RouterLinkStub } from '@vue/test-utils'
import ApiView from '../ApiView.vue'
import PortalKeyForm from '@/components/portal/PortalKeyForm.vue'
import PortalApiKeyTable from '@/components/portal/PortalApiKeyTable.vue'
import UseKeyModal from '@/components/keys/UseKeyModal.vue'

const api = vi.hoisted(() => ({ list: vi.fn(), groups: vi.fn(), rates: vi.fn() }))
const usage = vi.hoisted(() => ({ batch: vi.fn() }))
const navigation = vi.hoisted(() => ({ route: { path: '/keys', hash: '', query: {} as Record<string, string> }, replace: vi.fn() }))
const app = vi.hoisted(() => ({
  docUrl: '',
  siteName: 'sub2api',
  cachedPublicSettings: {
    model_plaza_enabled: true,
    api_base_url: 'https://api.example.com',
    hide_ccs_import_button: false,
    site_name: 'sub2api',
  },
  showSuccess: vi.fn(),
  showError: vi.fn(),
}))
vi.mock('@/api/keys', () => ({ keysAPI: { list: api.list } }))
vi.mock('@/api/usage', () => ({ usageAPI: { getDashboardApiKeysUsage: usage.batch } }))
vi.mock('@/api/groups', () => ({ userGroupsAPI: { getAvailable: api.groups, getUserGroupRates: api.rates } }))
vi.mock('@/stores/app', () => ({ useAppStore: () => app }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => ({ isSimpleMode: false }) }))
vi.mock('@/composables/useClipboard', () => ({ useClipboard: () => ({ copyToClipboard: vi.fn() }) }))
vi.mock('vue-router', async () => ({ ...await vi.importActual('vue-router'), useRoute: () => navigation.route, useRouter: () => ({ replace: navigation.replace }) }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ locale: { value: 'zh-CN' }, t: (key: string) => key }) }))

function render() {
  return mount(ApiView, { global: { stubs: {
    RouterLink: RouterLinkStub, Icon: true, PortalKeyForm: true, PortalApiKeyTable: true,
    UseKeyModal: true, PortalApiBulkForm: true,
    PortalDialog: { props: ['open', 'title'], template: '<div v-if="open"><slot /></div>' },
    PortalApiExamples: true,
  } } })
}

describe('API key workspace', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    navigation.route.query = {}
    app.cachedPublicSettings.hide_ccs_import_button = false
    app.cachedPublicSettings.api_base_url = 'https://api.example.com'
    api.list.mockResolvedValue({ items: [], total: 0 })
    api.groups.mockResolvedValue([{ id: 7, name: 'OpenAI', rate_multiplier: 1 }])
    api.rates.mockResolvedValue({ 7: 0.7 })
    usage.batch.mockResolvedValue({ stats: {} })
  })

  it('keeps an unknown count when the first list request fails', async () => {
    api.list.mockRejectedValueOnce({ status: 429, message: 'Too many requests' })
    const wrapper = render()
    await flushPromises()
    expect(wrapper.get('.portal-api-count').text()).toBe('—')
    expect(wrapper.text()).toContain('操作过于频繁')
    expect(wrapper.text()).not.toContain('还没有 API Key')
    wrapper.unmount()
  })

  it('retains the key list and exposes a separate usage retry after statistics fail', async () => {
    api.list.mockResolvedValue({ items: [{ id: 3, key: 'sk-test-key-value', name: 'pro' }], total: 1 })
    usage.batch.mockRejectedValueOnce(new Error('temporary network error'))
    const wrapper = render()
    await flushPromises()
    expect(wrapper.get('.portal-api-count').text()).toBe('1')
    expect(wrapper.getComponent(PortalApiKeyTable).props('keys')).toHaveLength(1)
    expect(wrapper.text()).toContain('用量加载失败')
    const retry = wrapper.findAll('button').find(button => button.text().includes('重试用量'))!
    usage.batch.mockResolvedValueOnce({ stats: { '3': { today_actual_cost: 0, total_actual_cost: 0 } } })
    await retry.trigger('click')
    await flushPromises()
    expect(wrapper.text()).not.toContain('用量加载失败')
    expect(wrapper.getComponent(PortalApiKeyTable).props('usageStats')['3'].today_actual_cost).toBe(0)
    wrapper.unmount()
  })

  it('shows the usage retry cooldown and prevents another request before Retry-After elapses', async () => {
    api.list.mockResolvedValue({ items: [{ id: 3, key: 'sk-test-key-value', name: 'pro' }], total: 1 })
    usage.batch.mockRejectedValueOnce({ status: 429, retry_after: 60 })
    const wrapper = render()
    await flushPromises()
    const retry = wrapper.findAll('button').find(button => button.text().includes('重试用量'))!
    expect(retry.text()).toContain('60s')
    expect(retry.attributes('disabled')).toBeDefined()
    expect(wrapper.get('[data-test="refresh-keys"]').attributes('disabled')).toBeDefined()
    await retry.trigger('click')
    expect(usage.batch).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })

  it.each(['empty', 'failure'])('clears obsolete usage loading when the next key result is %s', async (result) => {
    let finishUsage!: (value: unknown) => void
    api.list.mockResolvedValueOnce({ items: [{ id: 3, key: 'sk-test-key-value', name: 'pro' }], total: 1 })
    usage.batch.mockImplementationOnce(() => new Promise(resolve => { finishUsage = resolve }))
    const wrapper = render()
    await flushPromises()
    expect(wrapper.text()).toContain('正在刷新密钥用量')
    if (result === 'empty') api.list.mockResolvedValueOnce({ items: [], total: 0 })
    else api.list.mockRejectedValueOnce(new Error('list unavailable'))
    await wrapper.get('[data-test="refresh-keys"]').trigger('click')
    await flushPromises()
    expect(wrapper.text()).not.toContain('正在刷新密钥用量')
    finishUsage({ stats: { '3': { today_actual_cost: 123 } } })
    await flushPromises()
    if (result === 'empty') expect(wrapper.findComponent(PortalApiKeyTable).exists()).toBe(false)
    else expect(wrapper.getComponent(PortalApiKeyTable).props('usageStats')).toEqual({})
    wrapper.unmount()
  })

  it('waits for available groups before opening and consuming a create shortcut', async () => {
    navigation.route.query = { create: '1', source: 'command' }
    let finishGroups!: (value: unknown) => void
    api.groups.mockImplementationOnce(() => new Promise(resolve => { finishGroups = resolve }))
    const wrapper = render()
    await flushPromises()
    expect(wrapper.getComponent(PortalKeyForm).props('open')).toBe(false)
    expect(navigation.replace).not.toHaveBeenCalled()
    finishGroups([{ id: 7, name: 'OpenAI', rate_multiplier: 1 }])
    await flushPromises()
    expect(wrapper.getComponent(PortalKeyForm).props('open')).toBe(true)
    expect(navigation.replace).toHaveBeenCalledOnce()
    expect(navigation.replace).toHaveBeenCalledWith({ path: '/keys', hash: '', query: { source: 'command' } })
    wrapper.unmount()
  })

  it('explains why a creation shortcut is unavailable when there are no groups', async () => {
    navigation.route.query = { create: '1' }
    api.groups.mockResolvedValue([])
    const wrapper = render()
    await flushPromises()
    expect(wrapper.getComponent(PortalKeyForm).props('open')).toBe(false)
    expect(wrapper.text()).toContain('当前账户没有可绑定分组')
    expect(navigation.replace).toHaveBeenCalledWith({ path: '/keys', hash: '', query: {} })
    wrapper.unmount()
  })

  it('starts with key management and opens group configuration only on create', async () => {
    const wrapper = render()
    await flushPromises()
    expect(wrapper.find('#portal-api-models-title').exists()).toBe(false)
    expect(wrapper.find('portal-api-examples-stub').exists()).toBe(false)
    expect(wrapper.getComponent(PortalKeyForm).props('open')).toBe(false)
    const create = wrapper.findAll('button').find(button => button.text().includes('创建密钥'))!
    await create.trigger('click')
    expect(wrapper.getComponent(PortalKeyForm).props()).toMatchObject({ open: true, groupRates: { 7: 0.7 } })
    expect(wrapper.findAllComponents(RouterLinkStub).some(link => link.props('to') === '/model-plaza?embedded=1')).toBe(true)
    expect(api.list).toHaveBeenCalledWith(1, 20, expect.objectContaining({ sort_by: 'created_at' }), expect.anything())
    expect(usage.batch).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('does not show misleading default multipliers when personal rates fail to load', async () => {
    api.rates.mockRejectedValue(new Error('倍率读取失败'))
    const wrapper = render()
    await flushPromises()
    expect(wrapper.text()).toContain('倍率读取失败')
    const create = wrapper.findAll('button').find(button => button.text().includes('创建密钥'))!
    expect(create.attributes('disabled')).toBeDefined()
    expect(wrapper.getComponent(PortalKeyForm).props('open')).toBe(false)
    wrapper.unmount()
  })

  it('opens the key editor when a group is clicked', async () => {
    const key = { id: 3, key: 'sk-test-key-value', name: 'pro', group_id: 7 }
    api.list.mockResolvedValue({ items: [key], total: 1 })
    const wrapper = render()
    await flushPromises()
    wrapper.getComponent(PortalApiKeyTable).vm.$emit('edit', key)
    await flushPromises()
    expect(wrapper.getComponent(PortalKeyForm).props()).toMatchObject({ open: true, editing: key })
    wrapper.unmount()
  })

  it('opens the use-key dialog from a table row', async () => {
    api.list.mockResolvedValue({ items: [{ id: 3, key: 'sk-test-key-value', name: 'pro' }], total: 1 })
    const wrapper = render()
    await flushPromises()
    const key = { id: 3, key: 'sk-test-key-value', name: 'pro', group: { platform: 'openai', claude_code_only: true, allow_messages_dispatch: false } }
    wrapper.getComponent(PortalApiKeyTable).vm.$emit('use', key)
    await flushPromises()
    expect(wrapper.getComponent(UseKeyModal).props()).toMatchObject({
      show: true,
      apiKey: 'sk-test-key-value',
      baseUrl: 'https://api.example.com',
      platform: 'openai',
      claudeCodeOnly: true,
    })
    wrapper.unmount()
  })

  it.each([
    ['anthropic', 'claude', 'https://api.example.com'],
    ['gemini', 'gemini', 'https://api.example.com'],
    ['antigravity', 'claude', 'https://api.example.com/antigravity'],
    ['antigravity', 'gemini', 'https://api.example.com/antigravity'],
    ['openai', 'codex', 'https://api.example.com'],
    ['grok', 'grokbuild', 'https://api.example.com/v1'],
  ])('keeps protocol-specific base paths for %s / %s imports', async (platform, client, endpoint) => {
    // The public setting may already contain /v1; the display is OpenAI-specific,
    // while both integrations expect a root and select their own protocol paths.
    app.cachedPublicSettings.api_base_url = 'https://api.example.com/v1/'
    const key = { id: 3, key: 'sk-test-key-value', name: 'pro', group: { platform } }
    api.list.mockResolvedValue({ items: [key], total: 1 })
    const open = vi.spyOn(window, 'open').mockReturnValue(null)
    const wrapper = render()
    await flushPromises()
    expect(wrapper.get('.portal-api-endpoint code').text()).toBe('https://api.example.com/v1')
    wrapper.getComponent(PortalApiKeyTable).vm.$emit('use', key)
    await flushPromises()
    expect(wrapper.getComponent(UseKeyModal).props('baseUrl')).toBe('https://api.example.com')
    wrapper.getComponent(PortalApiKeyTable).vm.$emit('import-ccs', key)
    await flushPromises()
    if (platform === 'antigravity') {
      const choiceKey = client === 'gemini' ? 'keys.ccsClientSelect.geminiCli' : 'keys.ccsClientSelect.claudeCode'
      await wrapper.findAll('.portal-api-ccs-choices button').find(button => button.text().includes(choiceKey))!.trigger('click')
    }
    expect(open).toHaveBeenCalledOnce()
    const link = new URL(String(open.mock.calls[0][0]))
    expect(link.searchParams.get('app')).toBe(client)
    expect(link.searchParams.get('endpoint')).toBe(endpoint)
    expect(link.searchParams.get('apiKey')).toBe(key.key)
    expect(link.searchParams.get('endpoint')).not.toContain('/v1/v1')
    wrapper.unmount()
    open.mockRestore()
  })

  it('hides CCS import when the public setting disables it', async () => {
    app.cachedPublicSettings.hide_ccs_import_button = true
    api.list.mockResolvedValue({ items: [{ id: 3, name: 'pro' }], total: 1 })
    const wrapper = render()
    await flushPromises()
    expect(wrapper.getComponent(PortalApiKeyTable).props('hideCcs')).toBe(true)
    wrapper.unmount()
  })

  it('still renders keys when usage stats fail to load', async () => {
    usage.batch.mockRejectedValue(new Error('用量失败'))
    api.list.mockResolvedValue({ items: [{ id: 9, name: 'pro', key: 'sk-1234567890abcd' }], total: 1 })
    const wrapper = render()
    await flushPromises()
    expect(wrapper.text()).not.toContain('密钥加载失败')
    expect(wrapper.getComponent(PortalApiKeyTable).props('keys')).toEqual([expect.objectContaining({ id: 9 })])
    expect(usage.batch).toHaveBeenCalledWith([9], expect.anything())
    wrapper.unmount()
  })
})
