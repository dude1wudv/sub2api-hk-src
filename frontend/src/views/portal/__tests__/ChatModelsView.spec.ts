import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, RouterLinkStub } from '@vue/test-utils'
import type * as VueI18n from 'vue-i18n'
import ChatModelsView from '../ChatModelsView.vue'

const { chat, getAvailable, app } = vi.hoisted(() => ({
  chat: { presets: vi.fn(), catalog: vi.fn(), createPreset: vi.fn(), updatePreset: vi.fn(), orderPresets: vi.fn(), deletePreset: vi.fn(), reconnectPreset: vi.fn() },
  getAvailable: vi.fn(),
  app: { showSuccess: vi.fn(), showError: vi.fn() },
}))
vi.mock('@/api/chat', () => ({ chatAPI: chat }))
vi.mock('@/api/groups', () => ({ getAvailable }))
vi.mock('@/stores', () => ({ useAppStore: () => app }))
vi.mock('vue-i18n', async () => ({
  ...await vi.importActual<typeof VueI18n>('vue-i18n'),
  useI18n: () => ({ locale: { value: 'zh-CN' } }),
}))
const group = (id: number) => ({ id, name: `Group ${id}`, platform: 'openai', subscription_type: 'standard', rate_multiplier: 2, user_rate_multiplier: 0.5 })
const model = { id: 'shared-model', available: true, unavailable_reason: '', pricing: { billing_mode: 'token', input_price: 0.000001, output_price: 0.000002 }, capabilities: { image_input: false, document_input: false, search: false } }
const preset = (id: string, groupId: number, available = true) => ({ id, group_id: groupId, model: 'shared-model', display_name: '', is_default: false, sort_order: 0, group: group(groupId), available, unavailable_reason: available ? '' : '分组权限已撤销' })
const render = () => mount(ChatModelsView, { global: { stubs: { RouterLink: RouterLinkStub } } })

describe('ChatModelsView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    chat.presets.mockResolvedValue([])
    getAvailable.mockResolvedValue([group(11), group(12)])
    chat.catalog.mockImplementation(async (id: number) => ({ group: group(id), models: [model] }))
    chat.createPreset.mockResolvedValue({})
    chat.updatePreset.mockResolvedValue({})
    chat.orderPresets.mockResolvedValue(undefined)
    chat.deletePreset.mockResolvedValue(undefined)
    chat.reconnectPreset.mockResolvedValue(undefined)
  })

  it('requires an explicit group and never auto-adds a model', async () => {
    const wrapper = render(); await flushPromises()
    expect(wrapper.text()).toContain('还没有常用模型')
    expect(chat.catalog).not.toHaveBeenCalled()
    expect(chat.createPreset).not.toHaveBeenCalled()
    await wrapper.get('#chat-group').setValue(12); await flushPromises()
    expect(chat.catalog).toHaveBeenLastCalledWith(12, '', expect.any(AbortSignal))
    expect(wrapper.text()).toContain('你的倍率 ×0.5')
    const add = wrapper.findAll('button').find(button => button.text() === '添加常用')!
    await add.trigger('click'); await flushPromises()
    expect(chat.createPreset).toHaveBeenCalledTimes(1)
    expect(chat.createPreset).toHaveBeenCalledWith({ group_id: 12, model: 'shared-model' })
    wrapper.unmount()
  })

  it('keeps same-name presets distinct and marks revoked items unavailable', async () => {
    chat.presets.mockResolvedValue([preset('a', 11), preset('b', 12, false)])
    const wrapper = render(); await flushPromises()
    expect(wrapper.findAll('[data-preset-id]')).toHaveLength(2)
    expect(wrapper.get('[data-preset-id=a]').text()).toContain('Group 11')
    expect(wrapper.get('[data-preset-id=b]').text()).toContain('Group 12')
    expect(wrapper.get('[data-preset-id=b]').text()).toContain('分组权限已撤销')
    const makeDefault = wrapper.get('[data-preset-id=b]').findAll('button').find(button => button.text() === '设为默认')!
    expect(makeDefault.attributes('disabled')).toBeDefined()
    await wrapper.get('#chat-group').setValue(11); await flushPromises()
    const added = wrapper.findAll('button').find(button => button.text() === '已添加')!
    expect(added.attributes('disabled')).toBeDefined()
    wrapper.unmount()
  })

  it('sends the full order and preserves history when removing a preset', async () => {
    chat.presets.mockResolvedValue([preset('a', 11), preset('b', 12)])
    const wrapper = render(); await flushPromises()
    await wrapper.get('[data-preset-id=b] button[aria-label^=上移]').trigger('click'); await flushPromises()
    expect(chat.orderPresets).toHaveBeenCalledWith(['b', 'a'])
    await wrapper.get('[data-preset-id=a]').findAll('button').find(button => button.text() === '移除')!.trigger('click'); await flushPromises()
    expect(chat.deletePreset).toHaveBeenCalledWith('a')
    wrapper.unmount()
  })

  it('reconnects only after an explicit user action', async () => {
    chat.presets.mockResolvedValue([preset('a', 11)])
    const wrapper = render(); await flushPromises()
    expect(chat.reconnectPreset).not.toHaveBeenCalled()
    await wrapper.get('[data-preset-id=a]').findAll('button').find(button => button.text() === '重新连接')!.trigger('click'); await flushPromises()
    expect(chat.reconnectPreset).toHaveBeenCalledWith('a')
    wrapper.unmount()
  })

  it('ignores stale catalog responses after group changes', async () => {
    const oldRequest = Promise.withResolvers<unknown>()
    chat.catalog.mockImplementationOnce(() => oldRequest.promise)
    const wrapper = render(); await flushPromises()
    await wrapper.get('#chat-group').setValue(11); await flushPromises()
    await wrapper.get('#chat-group').setValue(12); await flushPromises()
    oldRequest.resolve({ group: group(11), models: [{ ...model, id: 'stale-model' }] }); await flushPromises()
    expect(wrapper.text()).not.toContain('stale-model')
    expect(wrapper.text()).toContain('shared-model')
    wrapper.unmount()
  })

  it('renders untrusted names as text and keeps mutation failures visible', async () => {
    chat.presets.mockResolvedValue([{ ...preset('a', 11), display_name: '<img src=x onerror=alert(1)>' }])
    chat.deletePreset.mockRejectedValueOnce(new Error('权限已变化'))
    const wrapper = render(); await flushPromises()
    expect(wrapper.find('img').exists()).toBe(false)
    expect(wrapper.text()).toContain('<img src=x onerror=alert(1)>')
    await wrapper.get('[data-preset-id=a]').findAll('button').find(button => button.text() === '移除')!.trigger('click'); await flushPromises()
    expect(app.showError).toHaveBeenCalledWith('权限已变化')
    expect(wrapper.findAll('[data-preset-id]')).toHaveLength(1)
    wrapper.unmount()
  })
})
