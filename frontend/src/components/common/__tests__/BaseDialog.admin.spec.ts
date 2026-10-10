import { afterEach, describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import BaseDialog from '../BaseDialog.vue'
import { adminSurfaceActive } from '@/composables/adminSurface'

const wrappers: ReturnType<typeof mount>[] = []
afterEach(() => { wrappers.splice(0).forEach(w => w.unmount()); document.body.innerHTML = ''; adminSurfaceActive.value = false })
function dialog(props = {}) {
  const wrapper = mount(BaseDialog, { attachTo: document.body, props: { show:true, title:'Account', ...props }, slots: { default:'<input aria-label="Name"><section hidden><button>Hidden</button></section>', footer:'<button id="last">Done</button>' }, global: { stubs: { transition: true } } })
  wrappers.push(wrapper); return wrapper
}
describe('admin editor and drawer interaction', () => {
  it('only applies a drawer on the admin surface', async () => {
    dialog({presentation:'drawer'}); await nextTick()
    expect(document.querySelector('.admin-detail-overlay')).toBeNull()
    adminSurfaceActive.value = true; await nextTick()
    expect(document.querySelector('.admin-detail-overlay')).not.toBeNull()
  })
  it('traps focus both ways and restores the opening control', async () => {
    const opener = document.createElement('button'); document.body.append(opener); opener.focus()
    const w = dialog(); await nextTick()
    const first = document.querySelector<HTMLButtonElement>('[aria-label="Close modal"]')!, last = document.querySelector<HTMLButtonElement>('#last')!
    last.focus(); document.dispatchEvent(new KeyboardEvent('keydown',{key:'Tab',bubbles:true,cancelable:true})); expect(document.activeElement).toBe(first)
    first.focus(); document.dispatchEvent(new KeyboardEvent('keydown',{key:'Tab',shiftKey:true,bubbles:true,cancelable:true})); expect(document.activeElement).toBe(last)
    await w.setProps({show:false}); expect(document.activeElement).toBe(opener)
  })
  it('keeps unsaved input on cancel and only closes after discard', async () => {
    const w = dialog({dirty:true}); await nextTick()
    document.querySelector<HTMLButtonElement>('[aria-label="Close modal"]')!.click(); await nextTick()
    expect(w.emitted('close')).toBeUndefined()
    const confirmation = document.querySelector('[role="alertdialog"]')!
    expect(confirmation).not.toBeNull()
    confirmation.querySelector<HTMLButtonElement>('button')!.click(); await nextTick()
    expect(w.emitted('close')).toBeUndefined(); expect(document.querySelector('[role="alertdialog"]')).toBeNull()
    ;(w.vm as unknown as {requestClose:()=>void}).requestClose(); await nextTick()
    document.querySelectorAll<HTMLButtonElement>('[role="alertdialog"] button')[1].click()
    expect(w.emitted('close')).toHaveLength(1)
  })
  it('blocks Escape and close requests during save, and allows successful parent close', async () => {
    const w = dialog({dirty:true,busy:true}); await nextTick()
    document.dispatchEvent(new KeyboardEvent('keydown',{key:'Escape'}))
    ;(w.vm as unknown as {requestClose:()=>void}).requestClose()
    expect(w.emitted('close')).toBeUndefined(); expect(document.querySelector('[role="alertdialog"]')).toBeNull()
    await w.setProps({show:false}); expect(document.querySelector('[role="dialog"]')).toBeNull()
  })
})
