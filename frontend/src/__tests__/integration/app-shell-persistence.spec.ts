/**
 * 常驻 AppShell 集成测试：后台页面之间切换时侧边栏/顶栏不应卸载重建。
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import { createPinia, setActivePinia } from 'pinia'
import { mount, flushPromises } from '@vue/test-utils'
import { defineComponent, h, onMounted, onUnmounted } from 'vue'

vi.mock('@/api/setup', () => ({ getSetupStatus: vi.fn().mockResolvedValue({ needs_setup: false }) }))

const shellLifecycle = { mounted: 0, unmounted: 0 }

vi.mock('@/components/layout/AppShell.vue', () => ({
  default: defineComponent({
    name: 'AppShellStub',
    setup(_, { slots }) {
      onMounted(() => { shellLifecycle.mounted++ })
      onUnmounted(() => { shellLifecycle.unmounted++ })
      return () => h('div', { class: 'app-shell-stub' }, slots.default?.())
    }
  })
}))

import App from '@/App.vue'
import { useAppStore } from '@/stores/app'

const page = (name: string) => defineComponent({ name, render: () => h('div', { class: name }, name) })

function createTestRouter() {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/groups', component: page('groups'), meta: { appLayout: true } },
      { path: '/users', component: page('users'), meta: { appLayout: true } },
      {
        path: '/ops',
        component: page('ops'),
        meta: { appLayout: (route) => route.query.fullscreen !== '1' }
      },
      { path: '/login', component: page('login') }
    ]
  })
}

describe('App 常驻 AppShell', () => {
  beforeEach(() => {
    shellLifecycle.mounted = 0
    shellLifecycle.unmounted = 0
    setActivePinia(createPinia())
    vi.spyOn(useAppStore(), 'fetchPublicSettings').mockResolvedValue(null as never)
  })

  async function mountApp(initialPath: string) {
    const router = createTestRouter()
    await router.push(initialPath)
    await router.isReady()
    const wrapper = mount(App, {
      global: {
        plugins: [router],
        stubs: { NavigationProgress: true, Toast: true, AnnouncementPopup: true, AdminComplianceDialog: true }
      }
    })
    await flushPromises()
    return { router, wrapper }
  }

  it('后台页面之间切换时只替换内容区，不重建外壳', async () => {
    const { router, wrapper } = await mountApp('/groups')
    expect(wrapper.find('.app-shell-stub .groups').exists()).toBe(true)

    await router.push('/users')
    await flushPromises()

    expect(wrapper.find('.app-shell-stub .users').exists()).toBe(true)
    expect(wrapper.find('.groups').exists()).toBe(false)
    expect(shellLifecycle).toEqual({ mounted: 1, unmounted: 0 })
  })

  it('非工作区页面与全屏形态不渲染外壳', async () => {
    const { router, wrapper } = await mountApp('/login')
    expect(wrapper.find('.login').exists()).toBe(true)
    expect(wrapper.find('.app-shell-stub').exists()).toBe(false)

    await router.push('/ops')
    await flushPromises()
    expect(wrapper.find('.app-shell-stub .ops').exists()).toBe(true)

    await router.push('/ops?fullscreen=1')
    await flushPromises()
    expect(wrapper.find('.ops').exists()).toBe(true)
    expect(wrapper.find('.app-shell-stub').exists()).toBe(false)
  })
})
