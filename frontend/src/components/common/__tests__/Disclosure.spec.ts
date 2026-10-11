import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it } from 'vitest'
import Disclosure from '../Disclosure.vue'
import { useColorMode } from '@/composables/useColorMode'

describe('Disclosure', () => {
  afterEach(() => useColorMode().setMode('light'))

  it('starts collapsed, expands on demand, and opens for active content', async () => {
    useColorMode().setMode('dark')
    const wrapper = mount(Disclosure, {
      props: { label: 'Proxy summary' },
      slots: { default: '<span>Proxy details</span>' },
    })
    const details = wrapper.get('details')
    expect((details.element as HTMLDetailsElement).open).toBe(false)
    await details.get('summary').trigger('click')
    expect((details.element as HTMLDetailsElement).open).toBe(true)

    const active = mount(Disclosure, {
      props: { label: 'Quota pool', active: true },
      slots: { default: '<span>Quota details</span>' },
    })
    expect((active.get('details').element as HTMLDetailsElement).open).toBe(true)
    wrapper.unmount()
    active.unmount()
  })

  it.each(['light', 'dark'] as const)('retains expandable content in %s mode', async theme => {
    useColorMode().setMode(theme)
    const wrapper = mount(Disclosure, {
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
