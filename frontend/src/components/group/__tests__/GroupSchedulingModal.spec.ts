import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { getGroupSchedulingMock, saveGroupSchedulingMock } = vi.hoisted(() => ({
  getGroupSchedulingMock: vi.fn(),
  saveGroupSchedulingMock: vi.fn()
}))
vi.mock('@/api/admin/scheduling', () => ({
  getGroupScheduling: getGroupSchedulingMock,
  saveGroupScheduling: saveGroupSchedulingMock
}))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
import GroupSchedulingModal from '../GroupSchedulingModal.vue'

const initial = () => ({
  enabled: false,
  version: 'ver-1',
  accounts: [
    { account_id: 10, name: 'alpha', priority: 10, load_factor: 1, concurrency: 4 },
    { account_id: 11, name: 'beta', priority: 10, load_factor: 2, concurrency: 4 },
    { account_id: 12, name: 'gamma', priority: 20, load_factor: 1, concurrency: 2 }
  ]
})
const mountModal = () => mount(GroupSchedulingModal, {
  props: { groupId: 5 },
  global: { stubs: { BaseDialog: { template: '<div><slot/><slot name="footer"/></div>' } } }
})

beforeEach(() => {
  vi.clearAllMocks()
  getGroupSchedulingMock.mockResolvedValue(initial())
  saveGroupSchedulingMock.mockImplementation(async (_id, value) => value)
})

describe('GroupSchedulingModal', () => {
  it('loads and persists enabled state with the complete group membership', async () => {
    const wrapper = mountModal()
    await flushPromises()
    await wrapper.get('input[type="checkbox"]').setValue(true)
    await wrapper.get('button.btn-primary').trigger('click')
    await flushPromises()
    expect(getGroupSchedulingMock).toHaveBeenCalledWith(5)
    expect(saveGroupSchedulingMock).toHaveBeenCalledTimes(1)
    const submitted = saveGroupSchedulingMock.mock.calls[0][1]
    expect(submitted.enabled).toBe(true)
    expect(submitted.version).toBe('ver-1')
    expect(submitted.accounts).toEqual(initial().accounts)
    expect(wrapper.emitted('saved')).toHaveLength(1)
  })

  it('moves whole priority levels up/down while preserving same-level membership', async () => {
    const wrapper = mountModal()
    await flushPromises()
    const down = wrapper.findAll('button[aria-label="admin.accounts.groupScheduling.down"]')[0]
    await down.trigger('click')
    const alpha = wrapper.get('input[aria-label="alpha admin.accounts.priority"]')
    const beta = wrapper.get('input[aria-label="beta admin.accounts.priority"]')
    const gamma = wrapper.get('input[aria-label="gamma admin.accounts.priority"]')
    expect((alpha.element as HTMLInputElement).value).toBe('20')
    expect((beta.element as HTMLInputElement).value).toBe('20')
    expect((gamma.element as HTMLInputElement).value).toBe('10')
    await wrapper.findAll('button[aria-label="admin.accounts.groupScheduling.up"]')[1].trigger('click')
    const alphaAfterMove = wrapper.get('input[aria-label="alpha admin.accounts.priority"]')
    const betaAfterMove = wrapper.get('input[aria-label="beta admin.accounts.priority"]')
    const gammaAfterMove = wrapper.get('input[aria-label="gamma admin.accounts.priority"]')
    expect((alphaAfterMove.element as HTMLInputElement).value).toBe('10')
    expect((betaAfterMove.element as HTMLInputElement).value).toBe('10')
    expect((gammaAfterMove.element as HTMLInputElement).value).toBe('20')
  })

  it('shows flat apiClient 409 conflict text and retains unsaved edits', async () => {
    const wrapper = mountModal()
    await flushPromises()
    await wrapper.findAll('button[aria-label="admin.accounts.groupScheduling.down"]')[0].trigger('click')
    saveGroupSchedulingMock.mockRejectedValue({ status: 409 })
    await wrapper.get('button.btn-primary').trigger('click')
    await flushPromises()
    expect(wrapper.get('[role="alert"]').text()).toBe('admin.accounts.groupScheduling.conflict')
    expect((wrapper.get('input[aria-label="alpha admin.accounts.priority"]').element as HTMLInputElement).value).toBe('20')
    expect(wrapper.emitted('close')).toBeUndefined()
  })
})
