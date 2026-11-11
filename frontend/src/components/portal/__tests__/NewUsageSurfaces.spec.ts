import { describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import PortalKeyUsageDialog from '../PortalKeyUsageDialog.vue'
import GroupChangeIndicator from '@/components/common/GroupChangeIndicator.vue'
import InflightRequests from '@/components/usage/InflightRequests.vue'
const mocks = vi.hoisted(() => ({ usage: vi.fn(), inflight: vi.fn() }))
vi.mock('@/api/usage', () => ({ getKeyUsageDetail: mocks.usage }))
vi.mock('@/api/inflight', () => ({ getInflight: mocks.inflight }))
vi.mock('@/components/portal/PortalDialog.vue', () => ({ default: { props:['open','title'], template:'<div v-if="open"><slot /></div>' } }))
vi.mock('vue-i18n', async () => { const { ref } = await import('vue'); return { useI18n: () => ({ locale: ref('en-US') }) } })
const totals = { requests:1, input_tokens:1, output_tokens:1, cache_read_tokens:0, cache_write_tokens:0, total_tokens:2, cost:1, actual_cost:1 }
const result = (id:number) => ({api_key_id:id,start_date:'2026-11-01',end_date:'2026-11-07',timezone:'UTC',today:totals,range:totals,items:[]})
describe('new usage surfaces', () => {
  it('aborts old key request and ignores a late response after switching or closing', async () => {
    const pending:Array<{ resolve:(value:unknown)=>void; signal:AbortSignal }> = []
    mocks.usage.mockImplementation((_id,_params,signal) => new Promise(resolve => pending.push({ resolve, signal })))
    const wrapper = mount(PortalKeyUsageDialog, { props:{apiKey:{id:1,name:'One'} as any} })
    await wrapper.setProps({ apiKey:{id:2,name:'Two'} as any })
    expect(pending[0].signal.aborted).toBe(true)
    pending[1].resolve(result(2)); await flushPromises()
    expect(wrapper.text()).toContain('Two')
    pending[0].resolve(result(1)); await flushPromises()
    expect(wrapper.text()).not.toContain('One')
    await wrapper.setProps({ apiKey:null }); expect(pending[1].signal.aborted).toBe(true)
    wrapper.unmount()
  })
  it('shows NEW for subscriptions without a rate arrow and leaves unknown changes hidden', () => {
    const wrapper = mount(GroupChangeIndicator, {props:{group:{is_new:true,launched_at:'2026-11-11T00:00:00Z',launch_rate:2,subscription_type:'subscription',recent_rate_change:{old_rate:1,new_rate:2,changed_at:'2026-11-11T00:00:00Z'}}}})
    expect(wrapper.text()).toContain('NEW'); expect(wrapper.findAll('details')).toHaveLength(1)
    expect(wrapper.text()).toContain('Launched'); expect(wrapper.text()).not.toContain('Launch rate')
    wrapper.unmount()
  })
  it('shows a launch rate only when supplied for a non-subscription group', async () => {
    const wrapper = mount(GroupChangeIndicator, {props:{group:{is_new:true,launched_at:'2026-11-11T00:00:00Z',launch_rate:1.5}}})
    expect(wrapper.get('summary').attributes('aria-label')).toContain('launch details')
    expect(wrapper.text()).toContain('Launch rate: 1.5×')
    await wrapper.setProps({group:{is_new:true,launch_rate:null}})
    expect(wrapper.text()).not.toContain('Launch rate')
    wrapper.unmount()
  })
  it('keeps requests with the same ID from different users when loading more', async () => {
    const item = (user_id:number) => ({user_id,request_id:'same-id',model:`model-${user_id}`,started_at:'2026-11-11T00:00:00Z',api_key_id:user_id})
    mocks.inflight.mockReset().mockResolvedValueOnce({items:[item(1)],total:2,page:1,page_size:50,runtime_known:true}).mockResolvedValueOnce({items:[item(2)],total:2,page:2,page_size:50,runtime_known:true})
    const wrapper = mount(InflightRequests, {props:{admin:true}})
    const details = wrapper.get('details'); (details.element as HTMLDetailsElement).open = true; await details.trigger('toggle'); await flushPromises()
    await wrapper.findAll('button').find(button => button.text() === 'Load more')!.trigger('click'); await flushPromises()
    expect(wrapper.text()).toContain('model-1'); expect(wrapper.text()).toContain('model-2')
    expect(wrapper.findAll('tbody tr')).toHaveLength(2)
    wrapper.unmount()
  })
  it('loads in-flight requests only while opened and aborts on close', async () => {
    mocks.inflight.mockReset().mockResolvedValue({items:[],total:0,page:1,page_size:10,runtime_known:false})
    const wrapper = mount(InflightRequests)
    expect(mocks.inflight).not.toHaveBeenCalled()
    const details = wrapper.get('details'); (details.element as HTMLDetailsElement).open = true; await details.trigger('toggle'); await flushPromises()
    expect(mocks.inflight).toHaveBeenCalledTimes(1); expect(wrapper.text()).toContain('unavailable')
    const signal = mocks.inflight.mock.calls[0][2] as AbortSignal
    ;(details.element as HTMLDetailsElement).open = false; await details.trigger('toggle'); expect(signal.aborted).toBe(true)
    wrapper.unmount()
  })
})
