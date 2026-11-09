import { afterEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import ModelQualityResults from '../ModelQualityResults.vue'
const { getExternalQuality, getExternalQualityArtwork } = vi.hoisted(() => ({ getExternalQuality: vi.fn(), getExternalQualityArtwork: vi.fn() }))
vi.mock('@/api/externalQuality', () => ({ getExternalQuality, getExternalQualityArtwork }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ locale: ref('en'), t: (key: string) => key }) }))
afterEach(() => { vi.clearAllMocks(); vi.useRealTimers() })
const event = { id: 5, model_id: 'gpt-6-astra', status: 'completed', score: null, created_at: '2026-11-09T00:00:00Z', finished_at: '2026-11-09T00:02:00Z', duration_ms: 120000, has_artwork: true }
const report = { group_id: 237, model: '', status: 'stale', last_success_at: event.finished_at, events: [event] }
const mountView = () => mount(ModelQualityResults, { props: { groupId: 237, groupName: 'Plus group' }, global: { stubs: { BaseDialog: { name: 'BaseDialog', template: '<div><slot /></div>' } } } })
describe('ModelQualityResults dialog', () => {
  it('plays selected output in opaque frame and stops/remounts playback', async () => {
    Object.defineProperty(document, 'hidden', { configurable: true, get: () => false })
    getExternalQuality.mockResolvedValue(report)
    getExternalQualityArtwork.mockResolvedValue({ id: 5, group_id: 237, response_text: '<svg><circle r="4" /></svg><script>window.counter=1</script>' })
    const wrapper = mountView(); await flushPromises()
    expect(getExternalQuality).toHaveBeenCalledWith(237, expect.any(AbortSignal))
    expect(wrapper.text()).toContain('Generated · awaiting review')
    expect(wrapper.text()).not.toContain('Passed')
    expect(wrapper.text()).not.toContain('/ 100')
    expect(getExternalQualityArtwork).toHaveBeenCalledWith(237, 5, expect.any(AbortSignal))
    expect(wrapper.get('iframe').attributes('sandbox')).toBe('allow-scripts')
    expect(wrapper.get('iframe').attributes('srcdoc')).toContain("connect-src 'none'")
    const iframe = wrapper.get('iframe').element
    await wrapper.findAll('button').find(button => button.text() === 'Stop preview')!.trigger('click')
    expect(wrapper.find('iframe').exists()).toBe(false)
    await wrapper.findAll('button').find(button => button.text() === 'Play output')!.trigger('click'); await flushPromises()
    expect(wrapper.get('iframe').element).not.toBe(iframe)
    wrapper.unmount()
  })
  it('shows honest empty state without fabricated test type', async () => {
    getExternalQuality.mockResolvedValue({ ...report, events: [], status: 'unavailable' })
    const wrapper = mountView(); await flushPromises()
    expect(wrapper.text()).toContain('No test results yet')
    expect(wrapper.text()).not.toContain('Passed')
    expect(wrapper.text()).not.toContain('Candy')
    expect(getExternalQualityArtwork).not.toHaveBeenCalled()
    wrapper.unmount()
  })
  it('switches exact model and aborts polling when closed', async () => {
    vi.useFakeTimers()
    getExternalQuality.mockResolvedValue({ ...report, events: [event, { ...event, id: 6, model_id: 'gpt-6.1-sol' }] })
    getExternalQualityArtwork.mockResolvedValue({ response_text: '<div>output</div>' })
    const wrapper = mountView(); await flushPromises()
    expect(wrapper.findAll('.quality-history-item')).toHaveLength(1)
    await wrapper.findAll('.quality-model-tabs button').find(button => button.text() === 'gpt-6.1-sol')!.trigger('click'); await flushPromises()
    expect(getExternalQualityArtwork).toHaveBeenLastCalledWith(237, 6, expect.any(AbortSignal))
    wrapper.findComponent({ name: 'BaseDialog' }).vm.$emit('close'); await flushPromises()
    expect(wrapper.emitted('close')).toHaveLength(1)
    await vi.advanceTimersByTimeAsync(120000)
    expect(getExternalQuality).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })
})
