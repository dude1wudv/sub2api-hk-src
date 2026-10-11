import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, RouterLinkStub } from '@vue/test-utils'
import ApiView from '../ApiView.vue'
import PortalKeyForm from '@/components/portal/PortalKeyForm.vue'
import PortalApiKeyTable from '@/components/portal/PortalApiKeyTable.vue'
import UseKeyModal from '@/components/keys/UseKeyModal.vue'

const api = vi.hoisted(() => ({ list: vi.fn(), groups: vi.fn(), rates: vi.fn() }))
const usage = vi.hoisted(() => ({ batch: vi.fn() }))
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
vi.mock('vue-router', async () => ({ ...await vi.importActual('vue-router'), useRoute: () => ({ query: {} }), useRouter: () => ({ replace: vi.fn() }) }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ locale: { value: 'zh-CN' }, t: (key: string) => key }) }))

function render() {
  return mount(ApiView, { global: { stubs: {
    RouterLink: RouterLinkStub, Icon: true, PortalKeyForm: true, PortalApiKeyTable: true,
    UseKeyModal: true, PortalApiBulkForm: true, PortalDialog: true, PortalApiExamples: true,
  } } })
}

describe('API key workspace', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    app.cachedPublicSettings.hide_ccs_import_button = false
    api.list.mockResolvedValue({ items: [], total: 0 })
    api.groups.mockResolvedValue([{ id: 7, name: 'OpenAI', rate_multiplier: 1 }])
    api.rates.mockResolvedValue({ 7: 0.7 })
    usage.batch.mockResolvedValue({ stats: {} })
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
