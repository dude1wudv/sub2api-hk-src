import { afterEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import ModelQualityResults from '../ModelQualityResults.vue'

const { getAvailable, getExternalQuality } = vi.hoisted(() => ({ getAvailable: vi.fn(), getExternalQuality: vi.fn() }))
vi.mock('@/api/groups', () => ({ userGroupsAPI: { getAvailable } }))
vi.mock('@/api/externalQuality', () => ({ getExternalQuality }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ locale: ref('en'), t: (key: string) => key }) }))
afterEach(() => { vi.clearAllMocks() })
describe('ModelQualityResults', () => {
  it('queries only accessible configured groups and displays results without origin labels', async () => {
    Object.defineProperty(document, 'hidden', { configurable: true, get: () => false })
    getAvailable.mockResolvedValue([{ id: 179, name: 'DeepSeek group' }, { id: 999, name: 'Other' }])
    getExternalQuality.mockResolvedValue({ group_id: 179, model: 'deepseek-v4.1-flash', status: 'ready', updated_at: '2026-11-09T00:00:00Z', last_success_at: '2026-11-09T00:00:00Z', source_url: 'https://example.com', events: [{ id: 'a', model_id: 'deepseek-v4.1-flash', status: 'success', score: 96, created_at: '2026-11-09T00:00:00Z', response_text: '<svg><circle r="4" /></svg>' }] })
    const wrapper = mount(ModelQualityResults); await flushPromises()
    expect(getExternalQuality).toHaveBeenCalledTimes(1)
    expect(getExternalQuality).toHaveBeenCalledWith(179, expect.any(AbortSignal))
    expect(wrapper.text()).toContain('96 / 100')
    expect(wrapper.text()).not.toContain('example.com')
    expect(wrapper.get('iframe').attributes('sandbox')).toBe('')
    expect(wrapper.get('iframe').attributes('srcdoc')).toContain('default-src')
    wrapper.unmount()
  })
  it('shows an honest empty state rather than an inferred passing result', async () => {
    getAvailable.mockResolvedValue([{ id: 179, name: 'DeepSeek group' }])
    getExternalQuality.mockResolvedValue({ group_id: 179, model: 'deepseek-v4.1-flash', status: 'unavailable', events: [] })
    const wrapper = mount(ModelQualityResults); await flushPromises()
    expect(wrapper.text()).toContain('No test results yet')
    expect(wrapper.text()).not.toContain('Passed')
    wrapper.unmount()
  })
})
