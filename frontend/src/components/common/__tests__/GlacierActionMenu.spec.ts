import { mount } from '@vue/test-utils'
import { defineComponent, h } from 'vue'
import { afterEach, describe, expect, it, vi } from 'vitest'
import GlacierActionMenu from '../GlacierActionMenu.vue'

const mounted: ReturnType<typeof mount>[] = []

function mountMenu(onSelect = vi.fn()) {
  const Host = defineComponent({
    setup() {
      return () => h(GlacierActionMenu, { label: 'More actions' }, {
        default: ({ close }: { close: () => void }) => [
          h('button', { disabled: true }, 'Unavailable'),
          h('button', { onClick: () => { close(); onSelect() } }, 'Edit'),
          h('button', null, 'Delete'),
        ],
      })
    },
  })
  const wrapper = mount(Host, { attachTo: document.body, global: { stubs: { Icon: true } } })
  mounted.push(wrapper)
  return wrapper
}

async function openMenu(wrapper: ReturnType<typeof mount>) {
  await wrapper.get('.glacier-action-trigger').trigger('click')
  await new Promise(resolve => setTimeout(resolve, 0))
  return document.body.querySelector<HTMLElement>('[role="menu"]')!
}

afterEach(() => {
  mounted.splice(0).forEach(wrapper => wrapper.unmount())
})

describe('GlacierActionMenu', () => {
  it('skips disabled actions and wraps through arrow, Home, and End navigation', async () => {
    const wrapper = mountMenu()
    const menu = await openMenu(wrapper)
    const items = Array.from(menu.querySelectorAll<HTMLButtonElement>('button:not(:disabled)'))

    expect(document.activeElement).toBe(items[0])
    expect(menu.querySelector('button:disabled')?.hasAttribute('role')).toBe(false)
    items[0].dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowUp', bubbles: true }))
    expect(document.activeElement).toBe(items[1])
    items[1].dispatchEvent(new KeyboardEvent('keydown', { key: 'ArrowDown', bubbles: true }))
    expect(document.activeElement).toBe(items[0])
    items[0].dispatchEvent(new KeyboardEvent('keydown', { key: 'End', bubbles: true }))
    expect(document.activeElement).toBe(items[1])
    items[1].dispatchEvent(new KeyboardEvent('keydown', { key: 'Home', bubbles: true }))
    expect(document.activeElement).toBe(items[0])
  })

  it('closes on Escape and Tab and restores focus to its trigger', async () => {
    const wrapper = mountMenu()
    const menu = await openMenu(wrapper)
    menu.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true, cancelable: true }))
    await wrapper.vm.$nextTick()
    expect(document.body.querySelector('[role="menu"]')).toBeNull()
    expect(document.activeElement).toBe(wrapper.get('.glacier-action-trigger').element)

    await openMenu(wrapper)
    document.body.querySelector('[role="menu"]')!.dispatchEvent(new KeyboardEvent('keydown', { key: 'Tab', bubbles: true, cancelable: true }))
    await wrapper.vm.$nextTick()
    expect(document.body.querySelector('[role="menu"]')).toBeNull()
  })

  it('closes after a menu selection without dropping the caller action', async () => {
    const onSelect = vi.fn()
    const wrapper = mountMenu(onSelect)
    const menu = await openMenu(wrapper)
    menu.querySelectorAll<HTMLButtonElement>('button:not(:disabled)')[0].click()
    await wrapper.vm.$nextTick()

    expect(onSelect).toHaveBeenCalledOnce()
    expect(document.body.querySelector('[role="menu"]')).toBeNull()
  })

  it('dismisses on outside pointer input and removes global listeners on unmount', async () => {
    const remove = vi.spyOn(document, 'removeEventListener')
    const wrapper = mountMenu()
    await openMenu(wrapper)
    document.body.dispatchEvent(new Event('pointerdown', { bubbles: true }))
    await wrapper.vm.$nextTick()
    expect(document.body.querySelector('[role="menu"]')).toBeNull()

    await openMenu(wrapper)
    wrapper.unmount()
    expect(remove).toHaveBeenCalledWith('pointerdown', expect.any(Function))
    remove.mockRestore()
  })
})
