import { describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import PortalKeyForm from '../PortalKeyForm.vue'

const { update, create } = vi.hoisted(() => ({ update: vi.fn(), create: vi.fn() }))

vi.mock('@/api/keys', () => ({ keysAPI: { update, create } }))
vi.mock('@/components/portal/PortalDialog.vue', () => ({ default: { props: ['open', 'title'], template: '<div class="portal-dialog-test"><slot /></div>' } }))
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  const { ref } = await import('vue')
  return { ...actual, useI18n: () => ({ locale: ref('en-US'), t: (key: string) => key }) }
})

const groups = [
  { id: 3, name: 'Standard group', platform: 'openai', status: 'active' },
  { id: 7, name: 'Smart route group', platform: 'anthropic', status: 'active' },
  { id: 9, name: 'Unsupported group', platform: 'other', status: 'active' },
  { id: 11, name: 'Inactive group', platform: 'openai', status: 'inactive' },
] as any

describe('PortalKeyForm', () => {
  it('loads the selected group, expiry and newline-separated IP data, then clears expiry explicitly', async () => {
    vi.clearAllMocks()
    update.mockResolvedValue({ id: 42 })
    const wrapper = mount(PortalKeyForm, {
      props: {
        open: true,
        groups,
        editing: {
          id: 42,
          name: 'Production key',
          group_id: 3,
          quota: 18,
          expires_at: '2030-03-04T12:30:00.000Z',
          rate_limit_5h: 4,
          rate_limit_1d: 5,
          rate_limit_7d: 6,
          ip_whitelist: ['192.0.2.1', '198.51.100.0/24'],
          ip_blacklist: ['203.0.113.7'],
          routing_group_ids: [],
        } as any,
      },
    })

    expect((wrapper.get('input[type="datetime-local"]').element as HTMLInputElement).value).toMatch(/^2030-03-04T/)
    expect((wrapper.findAll('textarea')[0].element as HTMLTextAreaElement).value).toBe('192.0.2.1\n198.51.100.0/24')
    expect((wrapper.findAll('textarea')[1].element as HTMLTextAreaElement).value).toBe('203.0.113.7')
    const groupSelect = wrapper.get('select')
    expect(groupSelect.findAll('option').map(option => option.text())).toEqual([
      'Select a group', 'Standard group', 'Smart route group', 'Unsupported group', 'Inactive group',
    ])
    expect(wrapper.get('fieldset').text()).toContain('Smart routing groups (up to 10)')
    expect(wrapper.get('fieldset').text()).not.toContain('Unsupported group')
    expect(wrapper.get('fieldset').text()).not.toContain('Inactive group')

    await wrapper.get('input[type="datetime-local"]').setValue('')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(update).toHaveBeenCalledWith(42, expect.objectContaining({
      group_id: 3,
      expires_at: '',
      ip_whitelist: ['192.0.2.1', '198.51.100.0/24'],
      ip_blacklist: ['203.0.113.7'],
    }))
    expect(wrapper.emitted('saved')).toHaveLength(1)
    wrapper.unmount()
  })

  it('uses actual available routing groups and rejects a stale unsupported routing ID', async () => {
    vi.clearAllMocks()
    const wrapper = mount(PortalKeyForm, {
      props: {
        open: true,
        groups,
        editing: { id: 5, name: 'Stale route', group_id: 3, routing_group_ids: [9] } as any,
      },
    })

    expect(wrapper.get('fieldset').text()).toContain('Smart route group')
    expect(wrapper.get('.routing-order').text()).toContain('Unsupported group')
    expect(wrapper.get('.routing-order').text()).toContain('Unavailable')
    expect(wrapper.get('.key-routing').findAll('input[type="checkbox"]').map(input => input.element.getAttribute('value'))).toEqual(['3', '7'])
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(wrapper.get('[role="alert"]').text()).toContain('Select up to 10 groups that support smart routing')
    expect(update).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('lets an edited key remove a stale routing group that is no longer selectable', async () => {
    vi.clearAllMocks()
    update.mockResolvedValue({ id: 5 })
    const wrapper = mount(PortalKeyForm, {
      props: {
        open: true,
        groups,
        editing: { id: 5, name: 'Stale route', group_id: 3, routing_group_ids: [9] } as any,
      },
    })

    expect(wrapper.get('.routing-order').text()).toContain('Unavailable')
    await wrapper.get('[aria-label="Remove group 1"]').trigger('click')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(update).toHaveBeenCalledWith(5, expect.objectContaining({
      group_id: 3,
      routing_group_ids: [],
    }))
    expect(wrapper.emitted('saved')).toHaveLength(1)
    wrapper.unmount()
  })

  it('sends reset flags only when opted in and preserves the edited routing order', async () => {
    vi.clearAllMocks()
    update.mockResolvedValue({ id: 6 })
    const wrapper = mount(PortalKeyForm, {
      props: {
        open: true,
        groups,
        editing: { id: 6, name: 'Ordered routes', group_id: 3, routing_group_ids: [3, 7] } as any,
      },
    })
    expect(wrapper.find('.routing-order').text()).toContain('1. Standard group')
    await wrapper.get('[aria-label="Move group 1 down"]').trigger('click')
    expect(wrapper.find('.routing-order').text()).toContain('1. Smart route group')

    const resetQuota = wrapper.findAll('label').find(label => label.text().includes('Reset used quota'))!.get('input')
    const resetRates = wrapper.findAll('label').find(label => label.text().includes('Reset rate-limit usage'))!.get('input')
    expect((resetQuota.element as HTMLInputElement).checked).toBe(false)
    expect((resetRates.element as HTMLInputElement).checked).toBe(false)
    await resetQuota.setValue(true)
    await resetRates.setValue(true)
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(update).toHaveBeenCalledWith(6, expect.objectContaining({
      group_id: 7,
      routing_group_ids: [7, 3],
      reset_quota: true,
      reset_rate_limit_usage: true,
    }))

    await wrapper.setProps({ open: false })
    await wrapper.setProps({ open: true })
    await flushPromises()
    expect((wrapper.findAll('label').find(label => label.text().includes('Reset used quota'))!.get('input').element as HTMLInputElement).checked).toBe(false)
    expect((wrapper.findAll('label').find(label => label.text().includes('Reset rate-limit usage'))!.get('input').element as HTMLInputElement).checked).toBe(false)
    wrapper.unmount()
  })
})
