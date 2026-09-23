import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import GlacierDisclosure from '../GlacierDisclosure.vue'
import { useAppearance } from '@/composables/useAppearance'

describe('GlacierDisclosure', () => {
  afterEach(() => useAppearance().setStyle('aurora'))

  it('starts collapsed in Glacier, expands on demand, and opens for active content', async () => {
    useAppearance().setStyle('glacier')
    const wrapper = mount(GlacierDisclosure, {
      props: { label: 'Proxy summary' },
      slots: { default: '<span>Proxy details</span>' },
    })
    const details = wrapper.get('details')
    expect((details.element as HTMLDetailsElement).open).toBe(false)
    await details.get('summary').trigger('click')
    expect((details.element as HTMLDetailsElement).open).toBe(true)

    const active = mount(GlacierDisclosure, {
      props: { label: 'Quota pool', active: true },
      slots: { default: '<span>Quota details</span>' },
    })
    expect((active.get('details').element as HTMLDetailsElement).open).toBe(true)
    wrapper.unmount()
    active.unmount()
  })

  it.each(['aurora', 'lagoon', 'graphite'] as const)('retains expandable content in %s', async theme => {
    useAppearance().setStyle(theme)
    const wrapper = mount(GlacierDisclosure, {
      props: { label: 'Proxy summary' },
      slots: { default: '<span>Proxy details</span>' },
    })
    expect((wrapper.get('details').element as HTMLDetailsElement).open).toBe(false)
    await wrapper.get('summary').trigger('click')
    expect((wrapper.get('details').element as HTMLDetailsElement).open).toBe(true)
    expect(wrapper.text()).toContain('Proxy details')
    wrapper.unmount()
  })
})
