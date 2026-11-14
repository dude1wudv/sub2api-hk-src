import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { Mock } from 'vitest'
import type { VueWrapper } from '@vue/test-utils'
import router from '@/router'
import { flushPromises, mount, RouterLinkStub } from '@vue/test-utils'
import { reactive } from 'vue'
import type * as VueRouter from 'vue-router'
import type * as VueI18n from 'vue-i18n'
import type { ChatHostBridge } from '@/api/chatBridge'
import ChatView from '../ChatView.vue'

const ctx = vi.hoisted(() => ({
  mount: vi.fn(), destroy: vi.fn(), disconnect: vi.fn(), frameLoaded: vi.fn(),
  push: vi.fn(), replace: vi.fn(), stopActions: vi.fn(),
  options: undefined as ConstructorParameters<typeof ChatHostBridge>[0] | undefined,
}))
let auth: { token: string | null; user: { id: number } | null; logout: Mock; $onAction: Mock }
let action: ((context: { name: string }) => void) | undefined
vi.mock('@/stores/auth', () => ({ useAuthStore: () => auth }))
vi.mock('@/api/chatBridge', () => ({
  ChatHostBridge: vi.fn(function (options: ConstructorParameters<typeof ChatHostBridge>[0]) {
    ctx.options = options
    return { mount: ctx.mount, destroy: ctx.destroy, disconnect: ctx.disconnect, frameLoaded: ctx.frameLoaded }
  }),
}))
vi.mock('vue-router', async () => ({
  ...await vi.importActual<typeof VueRouter>('vue-router'),
  useRouter: () => ({ push: ctx.push, replace: ctx.replace }),
}))
vi.mock('vue-i18n', async () => ({
  ...await vi.importActual<typeof VueI18n>('vue-i18n'),
  useI18n: () => ({ locale: { value: 'zh-CN' } }),
}))
const wrappers: VueWrapper[] = []
function render() {
  const wrapper = mount(ChatView, { global: { stubs: { RouterLink: RouterLinkStub } } })
  wrappers.push(wrapper)
  return wrapper
}

describe('ChatView standalone Vue host', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    ctx.options = undefined
    action = undefined
    auth = reactive({
      token: 'host-only-token', user: { id: 7 },
      logout: vi.fn(async () => { auth.token = null; auth.user = null }),
      $onAction: vi.fn((callback: (context: { name: string }) => void) => { action = callback; return ctx.stopActions }),
    })
  })
  afterEach(() => {
    wrappers.splice(0).forEach((wrapper) => wrapper.unmount())
    vi.unstubAllGlobals()
  })

  it('uses only the fixed full-height iframe and leaves navigation to in-app settings', async () => {
    const wrapper = render()
    const frame = wrapper.get('iframe')
    expect(frame.attributes('src')).toBe('/chat-app/')
    expect(frame.attributes('allow')).toBe("microphone 'self'; camera 'none'; geolocation 'none'")
    expect(frame.attributes('src')).not.toContain('token')
    expect(wrapper.get('main').classes()).toContain('chat-host')
    expect(wrapper.find('nav').exists()).toBe(false)
    expect(wrapper.findAllComponents(RouterLinkStub)).toHaveLength(0)
    expect(ctx.mount).toHaveBeenCalledTimes(1)
    expect(ctx.options!.userID()).toBe(7)
    await frame.trigger('load')
    expect(ctx.frameLoaded).toHaveBeenCalledTimes(1)
    await ctx.options!.navigate('/dashboard')
    expect(ctx.push).toHaveBeenCalledWith('/dashboard')
  })

  it('keeps the same iframe within the visible viewport when the keyboard opens and closes', async () => {
    const viewport = Object.assign(new EventTarget(), { width: 390, height: 844, offsetLeft: 0, offsetTop: 0, scale: 1 })
    vi.stubGlobal('visualViewport', viewport)
    const wrapper = render()
    const frame = wrapper.get('iframe').element
    const host = wrapper.get('main').element
    expect(host.style.height).toBe('844px')
    Object.assign(viewport, { height: 360, offsetTop: 40 })
    viewport.dispatchEvent(new Event('resize'))
    viewport.dispatchEvent(new Event('scroll'))
    await new Promise<void>(resolve => requestAnimationFrame(() => resolve()))
    await flushPromises()
    expect(host.style.height).toBe('360px')
    expect(host.style.top).toBe('40px')
    expect(wrapper.get('iframe').element).toBe(frame)
    Object.assign(viewport, { height: 844, offsetTop: 0 })
    viewport.dispatchEvent(new Event('resize'))
    await new Promise<void>(resolve => requestAnimationFrame(() => resolve()))
    await flushPromises()
    expect(host.style.height).toBe('844px')
    expect(host.style.top).toBe('0px')
    expect(ctx.mount).toHaveBeenCalledTimes(1)
    expect(ctx.disconnect).not.toHaveBeenCalled()
  })

  it('does not tear down a same-user refresh while setToken reloads /auth/me', async () => {
    const wrapper = render()
    const initialFrame = wrapper.get('iframe').element
    auth.token = null
    auth.user = null
    auth.token = 'refreshed-token'
    await flushPromises()
    expect(wrapper.get('iframe').element).toBe(initialFrame)
    expect(ctx.disconnect).not.toHaveBeenCalled()
    auth.user = { id: 7 }
    await flushPromises()
    expect(ctx.disconnect).not.toHaveBeenCalled()
    expect(ctx.options!.userID()).toBe(7)
  })

  it('disconnects and remounts the frame on an account switch instead of sharing memory', async () => {
    const wrapper = render()
    const initialFrame = wrapper.get('iframe').element
    auth.user = { id: 8 }
    await flushPromises()
    expect(ctx.disconnect).toHaveBeenCalledTimes(1)
    expect(wrapper.get('iframe').element).not.toBe(initialFrame)
    expect(ctx.options!.userID()).toBe(8)
  })

  it('disconnects at logout initiation before waiting for the logout API', async () => {
    const wrapper = render()
    action!({ name: 'logout' })
    expect(ctx.disconnect).toHaveBeenCalledTimes(1)
    expect(ctx.options!.userID()).toBeNull()
    await flushPromises()
    expect(wrapper.find('iframe').exists()).toBe(false)
  })

  it('uses the main auth store for bridged logout and returns to sign in', async () => {
    const wrapper = render()
    await ctx.options!.logout()
    await flushPromises()
    expect(auth.logout).toHaveBeenCalledTimes(1)
    expect(ctx.replace).toHaveBeenCalledWith('/login')
    expect(wrapper.find('iframe').exists()).toBe(false)
  })

  it('removes the old frame on cross-tab invalidation and destroys the bridge on unmount', async () => {
    const wrapper = render()
    ctx.options!.onSessionInvalid!()
    await flushPromises()
    expect(wrapper.find('iframe').exists()).toBe(false)
    wrapper.unmount()
    wrappers.splice(wrappers.indexOf(wrapper), 1)
    expect(ctx.destroy).toHaveBeenCalledTimes(1)
    expect(ctx.stopActions).toHaveBeenCalledTimes(1)
  })

  it('keeps /chat protected but outside the workspace shell after real route adaptation', () => {
    const chat = router.getRoutes().find((route) => route.path === '/chat')
    expect(chat).toMatchObject({ name: 'Chat', meta: { requiresAuth: true, requiresAdmin: false, appLayout: false } })
    const models = router.getRoutes().find((route) => route.path === '/chat-models')
    expect(models?.meta.appLayout).toBe(true)
  })
})
