import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { defineComponent, nextTick, ref } from 'vue'
import BaseDialog from '../BaseDialog.vue'
import Select from '../Select.vue'
import PortalDialog from '@/components/portal/PortalDialog.vue'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key, locale: ref('en') }) }))
const wrappers: VueWrapper[] = []
const showDescriptor = Object.getOwnPropertyDescriptor(HTMLDialogElement.prototype, 'showModal')
const closeDescriptor = Object.getOwnPropertyDescriptor(HTMLDialogElement.prototype, 'close')
afterEach(() => {
  wrappers.splice(0).forEach(w => w.unmount())
  for (const [key, descriptor] of [['showModal', showDescriptor], ['close', closeDescriptor]] as const) {
    if (descriptor) Object.defineProperty(HTMLDialogElement.prototype, key, descriptor)
    else Reflect.deleteProperty(HTMLDialogElement.prototype, key)
  }
  vi.restoreAllMocks()
  document.body.innerHTML = ''
})

for (const native of [false, true]) {
  describe(native ? 'native dialog overlay host' : 'BaseDialog overlay host', () => {
    async function open() {
      if (native) {
        Object.defineProperty(HTMLDialogElement.prototype, 'showModal', { configurable: true, value() { this.open = true } })
        Object.defineProperty(HTMLDialogElement.prototype, 'close', { configurable: true, value() { this.open = false } })
      }
      const wrapper = mount(defineComponent({
        components: { BaseDialog, PortalDialog, AppSelect: Select },
        setup: () => ({ open: ref(true), native, value: ref('a') }),
        template: `<component :is="native ? 'PortalDialog' : 'BaseDialog'" v-bind="native ? { open } : { show: open }" title="Details" @close="open = false"><AppSelect v-model="value" :searchable="true" :options="[{value:'a',label:'Alpha'},{value:'b',label:'Beta'}]"/><input aria-label="Next" /></component>`,
      }), { attachTo: document.body, global: { stubs: { transition: true } } })
      wrappers.push(wrapper)
      await flushPromises()
      await wrapper.findComponent(Select).get('button').trigger('click')
      await flushPromises()
      return document.querySelector(native ? 'dialog' : '.modal-overlay')!
    }
    it('keeps the menu inside its host and closes it before its parent', async () => {
      const host = await open()
      const input = host.querySelector<HTMLInputElement>('.select-search-input')!
      expect(input).not.toBeNull()
      input.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true, cancelable: true }))
      await flushPromises()
      expect(host.querySelector('.select-search-input')).toBeNull()
      expect(document.activeElement?.classList.contains('select-trigger')).toBe(true)
      expect(native ? (host as HTMLDialogElement).open : host.isConnected).toBe(true)
      if (native) host.dispatchEvent(new Event('cancel', { cancelable: true }))
      else document.activeElement!.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
      await flushPromises()
      expect(native ? (host as HTMLDialogElement).open : host.isConnected).toBe(false)
    })
    it('does not cancel native Tab navigation and restores its trigger', async () => {
      const host = await open()
      const event = new KeyboardEvent('keydown', { key: 'Tab', bubbles: true, cancelable: true })
      host.querySelector('.select-search-input')!.dispatchEvent(event)
      await nextTick()
      expect(event.defaultPrevented).toBe(false)
      expect(document.activeElement?.classList.contains('select-trigger')).toBe(true)
      expect(host.querySelector('.select-search-input')).toBeNull()
    })
  })
}

it('submits the associated footer form once, preserving native validation', async () => {
  Object.defineProperty(HTMLDialogElement.prototype, 'showModal', { configurable: true, value() { this.open = true } })
  Object.defineProperty(HTMLDialogElement.prototype, 'close', { configurable: true, value() { this.open = false } })
  const submit = vi.fn()
  const wrapper = mount(defineComponent({ components: { PortalDialog }, setup: () => ({ submit }), template: `<PortalDialog :open="true" title="Form"><form id="footer-form" @submit.prevent="submit"><input required name="name" /></form><template #footer><button type="submit" form="footer-form">Save</button></template></PortalDialog>` }), { attachTo: document.body })
  wrappers.push(wrapper)
  await flushPromises()
  const button = document.querySelector<HTMLButtonElement>('.portal-dialog-footer button')!
  button.click()
  expect(submit).not.toHaveBeenCalled()
  document.querySelector<HTMLInputElement>('form input')!.value = 'Valid'
  button.click()
  expect(submit).toHaveBeenCalledTimes(1)
})
