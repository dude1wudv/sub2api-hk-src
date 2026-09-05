import { afterEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import BaseDialog from '../BaseDialog.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key })
}))

enableAutoUnmount(afterEach)

describe('BaseDialog', () => {
  afterEach(() => {
    document.body.innerHTML = ''
    document.body.classList.remove('modal-open')
  })

  it('resets body scroll position when reopened', async () => {
    const wrapper = mount(BaseDialog, {
      attachTo: document.body,
      props: { show: false, title: 'Details' },
      slots: { default: '<div style="height: 2000px">content</div>' },
      global: { stubs: { Icon: true } }
    })

    await wrapper.setProps({ show: true })
    await nextTick()
    const body = document.body.querySelector<HTMLElement>('.modal-body')
    expect(body).not.toBeNull()
    body!.scrollTop = 480

    await wrapper.setProps({ show: false })
    await wrapper.setProps({ show: true })
    await nextTick()

    expect(document.body.querySelector<HTMLElement>('.modal-body')?.scrollTop).toBe(0)
    wrapper.unmount()
  })

  it('traps focus within the active dialog', async () => {
    const wrapper = mount(BaseDialog, {
      attachTo: document.body,
      props: { show: true, showCloseButton: false, title: 'Details' },
      slots: { default: '<button id="first-action">First</button><button id="last-action">Last</button>' },
      global: { stubs: { Icon: true } }
    })

    await nextTick()
    const firstAction = document.body.querySelector<HTMLElement>('#first-action')!
    const lastAction = document.body.querySelector<HTMLElement>('#last-action')!

    lastAction.focus()
    const tabEvent = new KeyboardEvent('keydown', { key: 'Tab', bubbles: true, cancelable: true })
    document.dispatchEvent(tabEvent)
    expect(tabEvent.defaultPrevented).toBe(true)
    expect(document.activeElement).toBe(firstAction)

    firstAction.focus()
    const shiftTabEvent = new KeyboardEvent('keydown', {
      key: 'Tab',
      shiftKey: true,
      bubbles: true,
      cancelable: true
    })
    document.dispatchEvent(shiftTabEvent)
    expect(shiftTabEvent.defaultPrevented).toBe(true)
    expect(document.activeElement).toBe(lastAction)
    wrapper.unmount()
  })

  it('keeps the parent dialog locked and restores the nested trigger when a child unmounts', async () => {
    const opener = document.createElement('button')
    document.body.append(opener)
    opener.focus()

    const parentDialog = mount(BaseDialog, {
      attachTo: document.body,
      props: { show: false, showCloseButton: false, title: 'Parent' },
      slots: { default: '<button id="open-child">Open child</button>' },
      global: { stubs: { Icon: true } }
    })
    await parentDialog.setProps({ show: true })
    await nextTick()

    const childTrigger = document.body.querySelector<HTMLElement>('#open-child')!
    childTrigger.focus()
    const childDialog = mount(BaseDialog, {
      attachTo: document.body,
      props: { show: false, showCloseButton: false, title: 'Child' },
      slots: { default: '<button id="child-action">Confirm</button>' },
      global: { stubs: { Icon: true } }
    })
    await childDialog.setProps({ show: true })
    await nextTick()

    const childAction = document.body.querySelector<HTMLElement>('#child-action')!
    expect(document.activeElement).toBe(childAction)

    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    expect(childDialog.emitted('close')).toHaveLength(1)
    expect(parentDialog.emitted('close')).toBeUndefined()

    childDialog.unmount()
    await nextTick()
    expect(document.body.classList.contains('modal-open')).toBe(true)
    expect(document.activeElement).toBe(childTrigger)

    await parentDialog.setProps({ show: false })
    await nextTick()
    expect(document.body.classList.contains('modal-open')).toBe(false)
    expect(document.activeElement).toBe(opener)

    parentDialog.unmount()
  })

  it('focuses the dialog panel when it has no focusable content', async () => {
    const wrapper = mount(BaseDialog, {
      attachTo: document.body,
      props: { show: true, showCloseButton: false, title: 'Notice' },
      slots: { default: '<p>Read-only details</p>' },
      global: { stubs: { Icon: true } }
    })

    await nextTick()
    expect(document.activeElement).toBe(document.body.querySelector('.modal-content'))
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    expect(wrapper.emitted('close')).toHaveLength(1)
    wrapper.unmount()
  })

  it('skips hidden and disabled controls and wraps backwards from the panel', async () => {
    const wrapper = mount(BaseDialog, {
      attachTo: document.body,
      props: { show: true, showCloseButton: false, title: 'Actions' },
      slots: { default: '<div style="display:none"><button>Hidden</button></div><fieldset disabled><button>Disabled</button></fieldset><button id="available">Available</button><button hidden>Also hidden</button>' },
      global: { stubs: { Icon: true } }
    })
    await nextTick()
    const available = document.getElementById('available')!
    expect(document.activeElement).toBe(available)
    document.querySelector<HTMLElement>('.modal-content')!.focus()
    const event = new KeyboardEvent('keydown', { key: 'Tab', shiftKey: true, cancelable: true })
    document.dispatchEvent(event)
    expect(event.defaultPrevented).toBe(true)
    expect(document.activeElement).toBe(available)
    wrapper.unmount()
  })

  it('does not dismiss a covered dialog when the top dialog blocks Escape', async () => {
    const parent = mount(BaseDialog, {
      attachTo: document.body,
      props: { show: true, title: 'Parent', closeOnClickOutside: true },
      global: { stubs: { Icon: true } }
    })
    await nextTick()
    const child = mount(BaseDialog, {
      attachTo: document.body,
      props: { show: true, title: 'Child', closeOnEscape: false, closeOnClickOutside: true },
      global: { stubs: { Icon: true } }
    })
    await nextTick()
    const overlays = document.querySelectorAll<HTMLElement>('[role="dialog"]')
    expect(overlays[0].getAttribute('aria-labelledby')).not.toBe(overlays[1].getAttribute('aria-labelledby'))
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    overlays[0].click()
    expect(parent.emitted('close')).toBeUndefined()
    expect(child.emitted('close')).toBeUndefined()
    overlays[1].click()
    expect(child.emitted('close')).toHaveLength(1)
    child.unmount()
    parent.unmount()
    await nextTick()
  })

  it('preserves the outer opener when a covered dialog unmounts first', async () => {
    const opener = document.createElement('button')
    document.body.append(opener)
    opener.focus()
    const parent = mount(BaseDialog, {
      attachTo: document.body,
      props: { show: true, title: 'Parent', showCloseButton: false },
      slots: { default: '<button id="nested-opener">Open child</button>' },
      global: { stubs: { Icon: true } }
    })
    await nextTick()
    const child = mount(BaseDialog, {
      attachTo: document.body,
      props: { show: true, title: 'Child', showCloseButton: false },
      slots: { default: '<button id="nested-action">Action</button>' },
      global: { stubs: { Icon: true } }
    })
    await nextTick()
    parent.unmount()
    await nextTick()
    expect(document.body.classList.contains('modal-open')).toBe(true)
    expect(document.activeElement).toBe(document.getElementById('nested-action'))
    child.unmount()
    await nextTick()
    expect(document.body.classList.contains('modal-open')).toBe(false)
    expect(document.activeElement).toBe(opener)
  })
})
