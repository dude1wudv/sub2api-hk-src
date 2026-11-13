import { beforeEach, describe, expect, it, vi } from 'vitest'
import { chatAPI } from '../chat'
const { client } = vi.hoisted(() => ({ client: { get: vi.fn(), post: vi.fn(), patch: vi.fn(), put: vi.fn(), delete: vi.fn() } }))
vi.mock('../client', () => ({ apiClient: client }))
describe('Chat API', () => {
  beforeEach(() => { vi.clearAllMocks(); for (const fn of Object.values(client)) fn.mockResolvedValue({ data: [] }) })
  it('uses authenticated panel endpoints and forwards catalog cancellation', async () => {
    const controller = new AbortController()
    await chatAPI.catalog(11, 'shared', controller.signal)
    expect(client.get).toHaveBeenCalledWith('/chat/catalog', { params: { group_id: 11, search: 'shared' }, signal: controller.signal })
    await chatAPI.presets(controller.signal)
    expect(client.get).toHaveBeenLastCalledWith('/chat/presets', { signal: controller.signal })
  })
  it('submits explicit bindings and complete ordering without credentials', async () => {
    await chatAPI.createPreset({ group_id: 12, model: 'shared-model' })
    expect(client.post).toHaveBeenCalledWith('/chat/presets', { group_id: 12, model: 'shared-model' })
    await chatAPI.updatePreset('uuid', { display_name: 'mine', is_default: true })
    expect(client.patch).toHaveBeenCalledWith('/chat/presets/uuid', { display_name: 'mine', is_default: true })
    await chatAPI.orderPresets(['a', 'b'])
    expect(client.put).toHaveBeenCalledWith('/chat/presets/order', { ids: ['a', 'b'] })
    await chatAPI.deletePreset('a')
    expect(client.delete).toHaveBeenCalledWith('/chat/presets/a')
    await chatAPI.reconnectPreset('a')
    expect(client.post).toHaveBeenLastCalledWith('/chat/presets/a/reconnect')
  })
})
