import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { nextTick, reactive } from 'vue'
import Toast from '../Toast.vue'
import type { Toast as ToastEntry } from '@/types'
import { initMotionPreferences, useMotionPreferences } from '@/composables/useMotionPreferences'
import { mockMotionMedia } from '@/composables/__tests__/motionTestUtils'

const store = reactive({ toasts: [] as ToastEntry[], hideToast: vi.fn() })
vi.mock('@/stores/app', () => ({ useAppStore: () => store }))
let cleanup: () => void
let wrapper: VueWrapper | undefined
beforeEach(() => {
  localStorage.clear()
  mockMotionMedia()
  cleanup = initMotionPreferences()
  store.toasts = []
  store.hideToast.mockClear()
})
afterEach(() => { wrapper?.unmount(); cleanup(); document.body.innerHTML = ''; vi.restoreAllMocks() })

describe('toast success feedback', () => {
  it('responds only to a real success entry, retains its node on updates, and preserves dismissal', async () => {
    wrapper = mount(Toast, { global: { stubs: { Icon: true } } })
    expect(document.querySelector('.toast-success-mark')).toBeNull()
    store.toasts.push({ id: 'saving', type: 'info', message: 'Saving…' })
    await nextTick()
    expect(document.querySelector('.toast-success-mark')).toBeNull()
    store.toasts.push({ id: 'saved', type: 'success', message: 'Account saved', duration: 3000 })
    await nextTick()
    const mark = document.querySelector('.toast-success-mark')!
    expect(mark).not.toBeNull()
    expect(document.querySelector<HTMLElement>('.toast-progress')?.style.animationDuration).toBe('3000ms')
    expect(document.querySelector('.toast-progress')?.parentElement?.getAttribute('aria-hidden')).toBe('true')
    store.toasts[0].message = 'Ready'
    await nextTick()
    expect(document.querySelector('.toast-success-mark')).toBe(mark)
    expect(document.querySelector('[aria-live]')?.getAttribute('aria-live')).toBe('polite')
    const notification = mark.closest('.toast-notification')!
    notification.querySelector<HTMLButtonElement>('button')!.click()
    expect(store.hideToast).toHaveBeenCalledWith('saved')
  })

  it('keeps success content and countdown duration while disabling visual motion', async () => {
    useMotionPreferences().setPreference('reduced')
    store.toasts.push({ id: 'saved', type: 'success', message: 'Saved', duration: 4000 })
    wrapper = mount(Toast, { global: { stubs: { Icon: true } } })
    expect(document.querySelector('.toast-success-mark')).toBeNull()
    expect(document.querySelector('.toast-notification--reduced')?.textContent).toContain('Saved')
    expect(document.querySelector<HTMLElement>('.toast-progress')?.style.animationDuration).toBe('4000ms')
    useMotionPreferences().setPreference('auto')
    await nextTick()
    expect(document.querySelector('.toast-success-mark')).toBeNull()
  })

  it('does not replay completed success feedback when content or preferences change', async () => {
    store.toasts.push({ id: 'saved', type: 'success', message: 'Saved', duration: 4000 })
    wrapper = mount(Toast, { global: { stubs: { Icon: true } } })
    document.querySelector('.toast-success-mark')!.dispatchEvent(new Event('animationend'))
    await nextTick()
    expect(document.querySelector('.toast-success-mark')).toBeNull()
    store.toasts[0].message = 'Saved successfully'
    useMotionPreferences().setPreference('reduced')
    await nextTick()
    useMotionPreferences().setPreference('auto')
    await nextTick()
    expect(document.querySelector('.toast-success-mark')).toBeNull()
  })
})
