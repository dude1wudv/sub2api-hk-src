import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import PortalApiBulkForm from '../PortalApiBulkForm.vue'

const { bulkUpdate, showSuccess } = vi.hoisted(() => ({ bulkUpdate: vi.fn(), showSuccess: vi.fn() }))

vi.mock('@/api/keys', () => ({ keysAPI: { bulkUpdate } }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showSuccess }) }))
vi.mock('@/components/portal/PortalDialog.vue', () => ({ default: { props: ['open', 'title'], template: '<div class="portal-dialog-test"><slot /></div>' } }))
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const { ref } = await import('vue')
  return { ...actual, useI18n: () => ({ locale: ref('en-US'), t: (key: string) => key }) }
})

describe('PortalApiBulkForm', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('prevents concurrent submits and retries only failed keys after a partial result', async () => {
    let resolveFirst!: (result: { succeededIds: number[]; failures: Array<{ id: number; error: Error }> }) => void
    bulkUpdate.mockImplementationOnce(() => new Promise(resolve => { resolveFirst = resolve }))
    bulkUpdate.mockResolvedValueOnce({ succeededIds: [2], failures: [] })
    const wrapper = mount(PortalApiBulkForm, {
      props: {
        open: true,
        selectedKeys: [{ id: 1, name: 'one' }, { id: 2, name: 'two' }, { id: 3, name: 'three' }] as any,
        groups: [{ id: 7, name: 'Group 7' }] as any,
      },
    })
    await wrapper.get('[data-test="enable-status"]').setValue(true)
    await wrapper.get('[data-test="status-input"]').setValue('inactive')
    await wrapper.get('form').trigger('submit')
    await wrapper.get('form').trigger('submit')
    expect(bulkUpdate).toHaveBeenCalledOnce()
    expect(bulkUpdate).toHaveBeenCalledWith([1, 2, 3], { status: 'inactive' })

    resolveFirst({ succeededIds: [1, 3], failures: [{ id: 2, error: new Error('temporary failure') }] })
    await flushPromises()
    expect(wrapper.get('.portal-api-bulk-results').text()).toContain('Succeeded: 2; pending retry: 1.')
    expect(wrapper.get('.portal-api-bulk-failures').text()).toContain('#2 two')

    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(bulkUpdate.mock.calls).toEqual([
      [[1, 2, 3], { status: 'inactive' }],
      [[2], { status: 'inactive' }],
    ])
    expect(wrapper.get('.portal-api-bulk-results').text()).toContain('Succeeded: 3; pending retry: 0.')
    expect(showSuccess).toHaveBeenCalledWith('Updated 3 keys')
    wrapper.unmount()
  })

  it('sends explicit zero, empty IP allowlist and cleared expiry only when their fields are enabled', async () => {
    bulkUpdate.mockResolvedValue({ succeededIds: [8], failures: [] })
    const wrapper = mount(PortalApiBulkForm, {
      props: { open: true, selectedKeys: [{ id: 8, name: 'eight' }] as any, groups: [] },
    })
    await wrapper.get('[data-test="enable-quota"]').setValue(true)
    await wrapper.get('[data-test="quota-input"]').setValue('0')
    await wrapper.get('[data-test="enable-ip_whitelist"]').setValue(true)
    await wrapper.get('[data-test="enable-expiration"]').setValue(true)
    await wrapper.get('[data-test="never-expires"]').setValue(true)
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(bulkUpdate).toHaveBeenCalledWith([8], { quota: 0, ip_whitelist: [], expires_at: '' })
    wrapper.unmount()
  })

  it('does not allow bulk group changes when selection contains smart-routing keys', () => {
    const wrapper = mount(PortalApiBulkForm, {
      props: {
        open: true,
        selectedKeys: [{ id: 9, name: 'routed', routing_group_ids: [2, 4] }] as any,
        groups: [{ id: 7, name: 'Group 7' }] as any,
      },
    })
    expect(wrapper.get('[data-test="enable-group"]').attributes('disabled')).toBeDefined()
    expect(wrapper.text()).toContain('This selection contains smart-routing keys')
    wrapper.unmount()
  })
})
