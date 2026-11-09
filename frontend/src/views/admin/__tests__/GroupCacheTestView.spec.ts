import { afterEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import GroupCacheTestView from '../GroupCacheTestView.vue'

const { getAll, getModels, list, executeCacheRun } = vi.hoisted(() => ({ getAll: vi.fn(), getModels: vi.fn(), list: vi.fn(), executeCacheRun: vi.fn() }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => ({ user: { id: 1 } }) }))
vi.mock('@/api/admin/groups', () => ({ getAll, getModelAllowlistCandidates: getModels }))
vi.mock('@/api/keys', () => ({ list }))
vi.mock('@/api/groupCacheTest', () => ({ executeCacheRun, reconcileCacheBills: vi.fn() }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ locale: ref('en'), t: (key: string) => key }) }))
afterEach(() => { vi.clearAllMocks(); localStorage.clear() })
describe('manual group cache tests', () => {
  it('loads only selected groups and never starts a model request on mount or selection', async () => {
    getAll.mockResolvedValue([{ id: 72, name: 'GPT', platform: 'openai' }, { id: 179, name: 'DeepSeek', platform: 'deepseek' }, { id: 999, name: 'Other', platform: 'openai' }])
    getModels.mockResolvedValue(['deepseek-v4.1-flash'])
    list.mockResolvedValue({ items: [], pages: 1 })
    const wrapper = mount(GroupCacheTestView, { global: { stubs: { RouterLink: true } } })
    await flushPromises()
    expect(wrapper.get('[data-testid="cache-group"]').findAll('option').map(option => option.attributes('value'))).toEqual([undefined, '72', '179'])
    await wrapper.get('[data-testid="cache-group"]').setValue(179); await flushPromises()
    expect((wrapper.get('[data-testid="cache-model"]').element as HTMLInputElement).value).toBe('deepseek-v4.1-flash')
    expect((wrapper.get('[data-testid="cache-protocol"]').element as HTMLSelectElement).value).toBe('chat')
    expect(executeCacheRun).not.toHaveBeenCalled()
    expect(wrapper.get('[data-testid="cache-start"]').attributes('disabled')).toBeDefined()
    wrapper.unmount()
  })
})
