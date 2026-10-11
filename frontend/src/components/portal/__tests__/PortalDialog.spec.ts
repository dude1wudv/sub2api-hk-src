import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises, type VueWrapper } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import PortalDialog from '../PortalDialog.vue'
import PortalAccountDialog from '../PortalAccountDialog.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'

const wrappers: VueWrapper[] = []
const prototype = HTMLDialogElement.prototype
const originalShow = Object.getOwnPropertyDescriptor(prototype, 'showModal')
const originalClose = Object.getOwnPropertyDescriptor(prototype, 'close')
beforeEach(() => {
  Object.defineProperty(prototype, 'showModal', { configurable: true, value: vi.fn(function(this: HTMLDialogElement) { this.open = true }) })
  Object.defineProperty(prototype, 'close', { configurable: true, value: vi.fn(function(this: HTMLDialogElement) { this.open = false }) })
})
afterEach(() => {
  wrappers.splice(0).forEach(wrapper => wrapper.unmount())
  if (originalShow) Object.defineProperty(prototype, 'showModal', originalShow)
  else delete (prototype as unknown as { showModal?: unknown }).showModal
  if (originalClose) Object.defineProperty(prototype, 'close', originalClose)
  else delete (prototype as unknown as { close?: unknown }).close
  document.body.innerHTML = ''
})
const i18n = () => createI18n({ legacy: false, locale: 'en', messages: { en: {} } })
function portal(open = true) {
  const wrapper = mount(PortalDialog, { attachTo: document.body, props: { open, title: 'Details' }, slots: { default: '<input aria-label="Name" />' }, global: { plugins: [i18n()] } })
  wrappers.push(wrapper)
  return wrapper
}

describe('native portal dialog', () => {
  it('keeps native modal semantics, locks scrolling and restores the opener on close', async () => {
    const opener = document.createElement('button')
    document.body.append(opener)
    opener.focus()
    const wrapper = portal()
    await flushPromises()
    const dialog = document.querySelector('dialog')!
    expect(dialog.open).toBe(true)
    expect(dialog.getAttribute('aria-labelledby')).toBe(dialog.querySelector('h2')!.id)
    expect(document.body.style.overflow).toBe('hidden')
    ;(dialog.querySelector('input') as HTMLInputElement).focus()
    dialog.dispatchEvent(new Event('cancel', { cancelable: true }))
    expect(wrapper.emitted('close')).toHaveLength(1)
    await wrapper.setProps({ open: false })
    expect(dialog.open).toBe(false)
    expect(document.activeElement).toBe(opener)
    expect(document.body.classList.contains('modal-open')).toBe(false)
  })

  it('keeps a BaseDialog lock when the native dialog closes', async () => {
    const base = mount(BaseDialog, { props: { show: true, title: 'Parent' } })
    wrappers.push(base)
    const child = portal()
    await flushPromises()
    await child.setProps({ open: false })
    expect(document.body.classList.contains('modal-open')).toBe(true)
    await base.setProps({ show: false })
    expect(document.body.classList.contains('modal-open')).toBe(false)
  })

  it('releases the shared lock when unmounted while open', async () => {
    const wrapper = portal()
    await flushPromises()
    wrapper.unmount()
    expect(document.body.classList.contains('modal-open')).toBe(false)
  })

  it('keeps an account editor busy, shares the lock and restores its opener on unmount', async () => {
    const parent = portal()
    await flushPromises()
    const opener = document.querySelector('dialog button') as HTMLButtonElement
    opener.focus()
    const account = mount(PortalAccountDialog, { attachTo: document.body, props: { title: 'Account', busy: true }, slots: { default: '<input autofocus aria-label="Name" />' } })
    wrappers.push(account)
    await flushPromises()
    const modal = document.querySelector('.portal-account-modal') as HTMLDialogElement
    expect(modal.open).toBe(true)
    expect(modal.getAttribute('aria-labelledby')).toBe(modal.querySelector('h2')!.id)
    expect(document.activeElement).toBe(modal.querySelector('input'))
    modal.dispatchEvent(new Event('cancel', { cancelable: true }))
    expect(account.emitted('close')).toBeUndefined()
    await account.setProps({ busy: false })
    modal.dispatchEvent(new Event('cancel', { cancelable: true }))
    expect(account.emitted('close')).toHaveLength(1)
    account.unmount()
    expect(document.activeElement).toBe(opener)
    expect(document.body.classList.contains('modal-open')).toBe(true)
    await parent.setProps({ open: false })
    expect(document.body.classList.contains('modal-open')).toBe(false)
  })
})
