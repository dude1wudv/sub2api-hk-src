import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import InflightRequests from '../InflightRequests.vue'
import type { InflightItem, InflightResponse } from '@/api/inflight'

const { getInflight } = vi.hoisted(() => ({ getInflight: vi.fn() }))
vi.mock('@/api/inflight', () => ({ getInflight }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ locale: { value: 'en-US' } }) }))
const item = (overrides: Partial<InflightItem> = {}): InflightItem => ({
  request_id: 'request-123', user_id: 7, api_key_id: 8, api_key_name: 'Production key',
  group_id: 9, group_name: 'GPT group', model: 'gpt-test', stream: true,
  started_at: '2026-11-11T12:00:00Z', input_tokens: 34000, output_tokens: 80,
  tokens_estimated: true, reserved_amount: 0.000001, email: 'user@example.test',
  account_id: 10, account_name: 'Provider account', ...overrides,
})
const response = (items = [item()], overrides: Partial<InflightResponse> = {}): InflightResponse => ({ items, total: items.length, page: 1, page_size: 10, runtime_known: true, ...overrides })
let wrappers: ReturnType<typeof mount>[] = []
const render = (admin = false) => { const wrapper = mount(InflightRequests, { props: { admin } }); wrappers.push(wrapper); return wrapper }
async function toggle(wrapper: ReturnType<typeof mount>, open: boolean) {
  const details = wrapper.get('details')
  ;(details.element as HTMLDetailsElement).open = open
  await details.trigger('toggle')
  // jsdom schedules the native details toggle separately from the component timer.
  if (vi.isFakeTimers()) await vi.advanceTimersByTimeAsync(0)
  await flushPromises()
}
beforeEach(() => { vi.clearAllMocks(); getInflight.mockResolvedValue(response()) })
afterEach(() => { wrappers.forEach(wrapper => wrapper.unmount()); wrappers = []; vi.useRealTimers() })

describe('InflightRequests', () => {
  it('loads only on expansion, scopes user/admin requests, and only offers supported ID filtering', async () => {
    const user = render()
    expect(getInflight).not.toHaveBeenCalled()
    expect(user.get('.inflight-summary-meta').text()).toContain('Expand to view')
    expect(user.findAll('input')).toHaveLength(1)
    await toggle(user, true)
    expect(user.get('.inflight-summary-meta').text()).toContain('Collapse panel')
    expect(getInflight).toHaveBeenLastCalledWith(false, { page: 1, page_size: 10, model: undefined, request_id: undefined }, expect.any(AbortSignal))
    expect(user.text()).not.toContain('user@example.test')
    expect(user.text()).not.toContain('Provider account')

    const admin = render(true)
    await toggle(admin, true)
    await admin.findAll('input')[0]!.setValue('gpt-test')
    await admin.findAll('input')[1]!.setValue('request-')
    await admin.get('form').trigger('submit')
    await flushPromises()
    expect(getInflight).toHaveBeenLastCalledWith(true, { page: 1, page_size: 50, model: 'gpt-test', request_id: 'request-' }, expect.any(AbortSignal))
    expect(admin.text()).toContain('user@example.test')
    expect(admin.text()).toContain('Provider account')
  })

  it('labels reservations and estimates, preserves tiny amounts, and does not invent first-token or rate data', async () => {
    const wrapper = render(true)
    await toggle(wrapper, true)
    expect(wrapper.get('.inflight-reserved').text()).toBe('USD 0.000001')
    expect(wrapper.get('.inflight-estimate').text()).toBe('Estimated')
    expect(wrapper.get('.inflight-hint').text()).toContain('Reserved amounts are not final charges')
    expect(wrapper.get('.inflight-stage').text()).toBe('Processing')
    expect(wrapper.get('.inflight-metrics').text()).toContain('Not recorded')
    expect(wrapper.text()).not.toMatch(/TPS|tok\/s/)
    getInflight.mockResolvedValueOnce(response([item({ first_token_ms: 0, reserved_amount: 1e-8 })]))
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(wrapper.get('.inflight-stage').text()).toBe('First token received')
    expect(wrapper.get('.inflight-first-token').text()).toBe('0ms')
    expect(wrapper.get('.inflight-reserved').text()).toBe('< USD 0.000001')
  })

  it('distinguishes unknown runtime from known empty requests and supports retry after failure', async () => {
    getInflight.mockResolvedValueOnce(response([], { runtime_known: false }))
    const wrapper = render()
    await toggle(wrapper, true)
    expect(wrapper.get('.inflight-state').text()).toContain('Current status is unavailable')
    expect(wrapper.find('.inflight-count').exists()).toBe(false)
    getInflight.mockRejectedValueOnce(new Error('unavailable'))
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(wrapper.get('[role="alert"]').text()).toContain('Unable to load requests')
    getInflight.mockResolvedValueOnce(response([]))
    await wrapper.get('[role="alert"] button').trigger('click')
    await flushPromises()
    expect(wrapper.get('.inflight-state').text()).toBe('No requests are currently in progress.')
    expect(wrapper.get('.inflight-count').text()).toBe('0')
  })

  it('keeps the previous snapshot during refresh and ignores a closed request response', async () => {
    const wrapper = render()
    await toggle(wrapper, true)
    let resolveRefresh!: (result: InflightResponse) => void
    getInflight.mockImplementationOnce(() => new Promise(resolve => { resolveRefresh = resolve }))
    await wrapper.get('form').trigger('submit')
    expect(wrapper.findAll('.inflight-request')).toHaveLength(1)
    const signal = getInflight.mock.calls.at(-1)![2] as AbortSignal
    await toggle(wrapper, false)
    expect(signal.aborted).toBe(true)
    getInflight.mockResolvedValueOnce(response([item({ model: 'new-request' })]))
    await toggle(wrapper, true)
    resolveRefresh(response([item({ model: 'stale-request' })]))
    await flushPromises()
    expect(wrapper.get('.inflight-request-model').text()).toContain('new-request')
    expect(wrapper.text()).not.toContain('stale-request')
  })

  it('retries a failed next page without skipping it and merges the same request ID across users separately', async () => {
    getInflight.mockResolvedValueOnce(response([item()], { total: 3 }))
    const wrapper = render(true)
    await toggle(wrapper, true)
    await wrapper.findAll('input')[0]!.setValue('unapplied-draft')
    getInflight.mockRejectedValueOnce(new Error('network'))
    await wrapper.get('.inflight-list-footer button').trigger('click')
    await flushPromises()
    getInflight.mockResolvedValueOnce(response([item(), item({ user_id: 99, email: 'other@example.test' })], { total: 3, page: 2 }))
    await wrapper.get('.inflight-list-footer button').trigger('click')
    await flushPromises()
    expect(getInflight.mock.calls.slice(-2).map(call => call[1].page)).toEqual([2, 2])
    expect(getInflight).toHaveBeenLastCalledWith(true, expect.objectContaining({ model: undefined }), expect.any(AbortSignal))
    expect(wrapper.findAll('.inflight-request')).toHaveLength(2)
    expect(wrapper.text()).toContain('other@example.test')
  })

  it('updates elapsed display without polling and releases its timer on collapse/unmount', async () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-11-11T12:00:10Z'))
    const wrapper = render()
    await toggle(wrapper, true)
    expect(wrapper.get('.inflight-elapsed').text()).toBe('10s')
    await vi.advanceTimersByTimeAsync(3000)
    expect(wrapper.get('.inflight-elapsed').text()).toBe('13s')
    expect(getInflight).toHaveBeenCalledTimes(1)
    await toggle(wrapper, false)
    expect(vi.getTimerCount()).toBe(0)
    await toggle(wrapper, true)
    wrapper.unmount()
    wrappers = []
    expect(vi.getTimerCount()).toBe(0)
  })
})
