import { afterEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, nextTick, ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { useAdminDraft } from '../useAdminDraft'
import { useAdminFormTabs } from '../useAdminFormTabs'
import { initAppearance, useAppearance } from '../useAppearance'
import { adminSurfaceActive } from '../adminSurface'
vi.mock('vue-i18n',()=>({useI18n:()=>({locale:ref('en'),t:(k:string)=>k})}))
afterEach(()=>{vi.restoreAllMocks();adminSurfaceActive.value=false;localStorage.clear()})
describe('admin draft lifecycle',()=>{
  it('waits for hydration and ignores presentation changes without erasing edits',async()=>{
    const Editor=defineComponent({setup(){
      const value=ref(''), ready=ref(false), hint=ref('Loading')
      const {dirty}=useAdminDraft(()=>true,()=>({value:value.value}),()=>false,()=>ready.value)
      return {value,ready,hint,dirty}
    },template:'<span>{{hint}} {{dirty}}</span>'})
    const router=createRouter({history:createMemoryHistory(),routes:[{path:'/',component:Editor}]})
    await router.push('/');const w=mount({template:'<router-view />'},{global:{plugins:[router]}});await flushPromises()
    const editor=w.getComponent(Editor)
    editor.vm.value='loaded';editor.vm.ready=true;await flushPromises();expect(editor.vm.dirty).toBe(false)
    editor.vm.hint='Translated capability label';await flushPromises();expect(editor.vm.dirty).toBe(false)
    editor.vm.value='edited';await flushPromises();expect(editor.vm.dirty).toBe(true)
    editor.vm.ready=false;await flushPromises();editor.vm.ready=true;await flushPromises();expect(editor.vm.dirty).toBe(true)
    w.unmount()
  })
  it('blocks navigation for dirty input, supports undo, and accepts a confirmed discard',async()=>{
    const Editor=defineComponent({setup(){const value=ref('original');const {dirty}=useAdminDraft(()=>true,()=>value.value);return {value,dirty}},template:'<input v-model="value"><span>{{dirty}}</span>'})
    const router=createRouter({history:createMemoryHistory(),routes:[{path:'/',component:Editor},{path:'/other',component:{template:'<p>Other</p>'}}]})
    await router.push('/');const w=mount({template:'<router-view />'},{global:{plugins:[router]}});await flushPromises()
    const confirm=vi.spyOn(window,'confirm').mockReturnValue(false)
    await w.get('input').setValue('edited');expect(w.text()).toBe('true')
    await router.push('/other');expect(router.currentRoute.value.path).toBe('/');expect(confirm).toHaveBeenCalledOnce()
    await w.get('input').setValue('original');expect(w.text()).toBe('false')
    await w.get('input').setValue('changed again');confirm.mockReturnValue(true);await router.push('/other');expect(router.currentRoute.value.path).toBe('/other');w.unmount()
  })
  it('restores the saved theme after leaving administration without overwriting preference',()=>{
    initAppearance();useAppearance().setStyle('lagoon');localStorage.setItem('theme','dark');initAppearance()
    const appearance=useAppearance();adminSurfaceActive.value=true
    expect(appearance.style.value).toBe('graphite');expect(appearance.isDark.value).toBe(false)
    expect(localStorage.getItem('appearance-style')).toBe('lagoon');expect(localStorage.getItem('theme')).toBe('dark')
    adminSurfaceActive.value=false;expect(appearance.style.value).toBe('lagoon');expect(appearance.isDark.value).toBe(true)
  })
  it('reveals the first invalid hidden tab while preserving input on other tabs',async()=>{
    const Editor=defineComponent({setup(){return {a:ref('kept'),...useAdminFormTabs(()=>true)}},template:`<form @invalid.capture="revealInvalid"><section data-admin-tab="basic" v-show="formTab === 'basic'"><input v-model="a"></section><section data-admin-tab="auth" v-show="formTab === 'auth'"><input id="invalid" required></section></form>`})
    const w=mount(Editor,{attachTo:document.body});const input=w.get<HTMLInputElement>('#invalid').element
    vi.spyOn(input,'reportValidity').mockReturnValue(false)
    input.dispatchEvent(new Event('invalid',{cancelable:true}));await nextTick()
    expect(w.vm.formTab).toBe('auth');expect(document.activeElement).toBe(input);expect(w.vm.a).toBe('kept');w.unmount()
  })
})
