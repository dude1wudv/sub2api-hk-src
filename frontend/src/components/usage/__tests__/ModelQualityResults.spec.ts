import { afterEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import ModelQualityResults from '../ModelQualityResults.vue'

const { getAvailable, getExternalQuality, getExternalQualityArtwork } = vi.hoisted(() => ({ getAvailable: vi.fn(), getExternalQuality: vi.fn(), getExternalQualityArtwork: vi.fn() }))
vi.mock('@/api/groups', () => ({ userGroupsAPI: { getAvailable } }))
vi.mock('@/api/externalQuality', () => ({ getExternalQuality, getExternalQualityArtwork }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ locale: ref('en'), t: (key: string) => key }) }))
afterEach(() => { vi.clearAllMocks() })
const event = { id: 5, model_id: 'gpt-6.1-sol', status: 'completed', score: null, created_at: '2026-11-09T00:00:00Z', has_artwork: true }
const report = { group_id: 237, model: 'gpt-6.1-sol', status: 'stale', last_success_at: '2026-11-09T00:00:00Z', events: [event] }
describe('ModelQualityResults', () => {
  it('queries only accessible OpenAI groups and lazily loads artwork without inventing a verdict', async () => {
    Object.defineProperty(document, 'hidden', { configurable: true, get: () => false })
    getAvailable.mockResolvedValue([{ id: 237, name: 'Plus group' }, { id: 179, name: 'DeepSeek group' }, { id: 223, name: 'DeepSeek official' }])
    getExternalQuality.mockResolvedValue(report)
    getExternalQualityArtwork.mockResolvedValue({ id: 5, group_id: 237, response_text: '<svg><circle r="4" /></svg>' })
    const wrapper = mount(ModelQualityResults); await flushPromises()
    expect(getExternalQuality).toHaveBeenCalledTimes(1)
    expect(getExternalQuality).toHaveBeenCalledWith(237, expect.any(AbortSignal))
    expect(wrapper.text()).not.toContain('DeepSeek')
    expect(wrapper.text()).toContain('Generated · awaiting review')
    expect(wrapper.text()).not.toContain('Passed')
    expect(wrapper.text()).not.toContain('/ 100')
    expect(wrapper.find('iframe').exists()).toBe(false)
    expect(getExternalQualityArtwork).not.toHaveBeenCalled()
    const details = wrapper.get('details'); (details.element as HTMLDetailsElement).open = true
    await details.trigger('toggle'); await flushPromises()
    expect(getExternalQualityArtwork).toHaveBeenCalledWith(237, 5, expect.any(AbortSignal))
    expect(wrapper.get('iframe').attributes('sandbox')).toBe('')
    expect(wrapper.get('iframe').attributes('srcdoc')).toContain('default-src')
    wrapper.unmount()
  })
  it('shows an honest empty state rather than an inferred passing result', async () => {
    getAvailable.mockResolvedValue([{ id: 72, name: 'Pro group' }])
    getExternalQuality.mockResolvedValue({ group_id: 72, model: '', status: 'unavailable', events: [] })
    const wrapper = mount(ModelQualityResults); await flushPromises()
    expect(wrapper.text()).toContain('No test results yet')
    expect(wrapper.text()).not.toContain('Passed')
    wrapper.unmount()
  })
  it('filters by exact model without eagerly loading previews', async () => {
    getAvailable.mockResolvedValue([{ id: 237, name: 'Plus group' }])
    getExternalQuality.mockResolvedValue({ ...report, events: [event, { ...event, id: 6, model_id: 'gpt-6-astra' }] })
    const wrapper = mount(ModelQualityResults); await flushPromises()
    expect(wrapper.findAll('details')).toHaveLength(2)
    await wrapper.findAll('.quality-model-tabs button').find(b => b.text() === 'gpt-6-astra')!.trigger('click')
    expect(wrapper.findAll('details')).toHaveLength(1)
    expect(wrapper.get('details').text()).toContain('gpt-6-astra')
    expect(getExternalQualityArtwork).not.toHaveBeenCalled()
    wrapper.unmount()
  })
})
