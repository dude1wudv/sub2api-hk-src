import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, RouterLinkStub } from '@vue/test-utils'
import ApiView from '../ApiView.vue'
import PortalKeyForm from '@/components/portal/PortalKeyForm.vue'

const api = vi.hoisted(() => ({ list: vi.fn(), groups: vi.fn(), rates: vi.fn() }))
vi.mock('@/api/keys', () => ({ keysAPI: { list: api.list } }))
vi.mock('@/api/groups', () => ({ userGroupsAPI: { getAvailable: api.groups, getUserGroupRates: api.rates } }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ docUrl: '', cachedPublicSettings: { model_plaza_enabled: true } }) }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => ({ isSimpleMode: false }) }))
vi.mock('@/composables/useClipboard', () => ({ useClipboard: () => ({ copyToClipboard: vi.fn() }) }))
vi.mock('vue-router', async () => ({ ...await vi.importActual('vue-router'), useRoute: () => ({ query: {} }), useRouter: () => ({ replace: vi.fn() }) }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ locale: { value: 'zh-CN' } }) }))

function render() {
  return mount(ApiView, { global: { stubs: {
    RouterLink: RouterLinkStub, Icon: true, PortalKeyForm: true, PortalApiKeyCard: true,
    PortalApiBulkForm: true, PortalDialog: true, PortalApiExamples: true,
  } } })
}

describe('API key workspace', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    api.list.mockResolvedValue({ items: [], total: 0 })
    api.groups.mockResolvedValue([{ id: 7, name: 'OpenAI', rate_multiplier: 1 }])
    api.rates.mockResolvedValue({ 7: 0.7 })
  })

  it('starts with key management and opens group configuration only on create', async () => {
    const wrapper = render()
    await flushPromises()
    expect(wrapper.find('#portal-api-models-title').exists()).toBe(false)
    expect(wrapper.find('portal-api-examples-stub').exists()).toBe(false)
    expect(wrapper.getComponent(PortalKeyForm).props('open')).toBe(false)
    const create = wrapper.findAll('button').find(button => button.text().includes('创建 API Key'))!
    await create.trigger('click')
    expect(wrapper.getComponent(PortalKeyForm).props()).toMatchObject({ open: true, groupRates: { 7: 0.7 } })
    expect(wrapper.findAllComponents(RouterLinkStub).some(link => link.props('to') === '/model-plaza?embedded=1')).toBe(true)
    expect(api.list).toHaveBeenCalledWith(1, 6, expect.objectContaining({ sort_by: 'created_at' }), expect.anything())
    wrapper.unmount()
  })

  it('does not show misleading default multipliers when personal rates fail to load', async () => {
    api.rates.mockRejectedValue(new Error('倍率读取失败'))
    const wrapper = render()
    await flushPromises()
    expect(wrapper.text()).toContain('倍率读取失败')
    const create = wrapper.findAll('button').find(button => button.text().includes('创建 API Key'))!
    expect(create.attributes('disabled')).toBeDefined()
    expect(wrapper.getComponent(PortalKeyForm).props('open')).toBe(false)
    wrapper.unmount()
  })
})
