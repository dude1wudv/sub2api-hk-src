import { afterEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import GroupCacheTestView from '../GroupCacheTestView.vue'
import { MOBILE_CONTENT_QUERY } from '@/utils/responsive'
import { DEFAULT_CACHE_SETTINGS, saveCacheHistory, type CacheTestRun } from '@/utils/groupCacheTest'

const { getAll, getModels, list, executeCacheRun } = vi.hoisted(() => ({ getAll: vi.fn(), getModels: vi.fn(), list: vi.fn(), executeCacheRun: vi.fn() }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => ({ user: { id: 1 } }) }))
vi.mock('@/api/admin/groups', () => ({ getAll, getModelAllowlistCandidates: getModels }))
vi.mock('@/api/keys', () => ({ list }))
vi.mock('@/api/groupCacheTest', () => ({ executeCacheRun, reconcileCacheBills: vi.fn() }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ locale: ref('en'), t: (key: string) => key }) }))
afterEach(() => { vi.clearAllMocks(); vi.restoreAllMocks(); localStorage.clear() })
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
  it('shows all eight fields and expands the last mobile round without executing requests', async () => {
    const matchMedia = window.matchMedia
    vi.spyOn(window, 'matchMedia').mockImplementation(query => ({ ...matchMedia(query), matches: query === MOBILE_CONTENT_QUERY }))
    getAll.mockResolvedValue([{ id: 72, name: 'Synthetic cache', platform: 'openai' }])
    list.mockResolvedValue({ items: [], pages: 1 })
    const run: CacheTestRun = {
      version: 1, id: 'mobile-run', startedAt: '2026-10-10T12:00:00Z', finishedAt: '2026-10-10T12:01:00Z',
      groupId: 72, groupName: 'Synthetic cache', keyId: 1, keyName: 'Synthetic key', status: 'completed',
      settings: { ...DEFAULT_CACHE_SETTINGS, model: 'preview-model' },
      rounds: [1, 2].map(index => ({ index, status: 'completed', startedAt: '2026-10-10T12:00:00Z',
        requestId: `request-${index}`, tokens: { input: 1024, read: 512, write: 128, output: 32 },
        responseModel: 'preview-model', serviceTier: null, firstTextMs: 21, durationMs: 42,
        error: null, httpStatus: 200, bill: null, billingState: 'unavailable' })),
    }
    expect(saveCacheHistory(1, [run])).toBe(true)
    const wrapper = mount(GroupCacheTestView, { global: { stubs: { RouterLink: true } } })
    await flushPromises()
    await wrapper.findAll('button').find(button => button.text().includes('Synthetic cache'))!.trigger('click')
    const cards = wrapper.findAll('.mobile-record')
    expect(cards).toHaveLength(2)
    expect(cards[1]!.findAll('dt')).toHaveLength(8)
    expect(wrapper.find('table').exists()).toBe(false)
    await cards[1]!.get('button').trigger('click')
    expect(wrapper.get('[data-testid="cache-detail"]').text()).toContain('request-2')
    await cards[1]!.get('button').trigger('click')
    expect(wrapper.find('[data-testid="cache-detail"]').exists()).toBe(false)
    expect(executeCacheRun).not.toHaveBeenCalled()
    wrapper.unmount()
  })
})
