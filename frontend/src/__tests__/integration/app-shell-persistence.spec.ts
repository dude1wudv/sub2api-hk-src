/** App shell ownership stays isolated between administration and the Patrick user surface. */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { createMemoryHistory, createRouter } from 'vue-router'
import { createPinia, setActivePinia } from 'pinia'
import { mount, flushPromises } from '@vue/test-utils'
import { defineComponent, h, onMounted, onUnmounted, ref } from 'vue'

vi.mock('@/api/setup', () => ({ getSetupStatus: vi.fn().mockResolvedValue({ needs_setup: false }) }))
vi.mock('vue-i18n', async importOriginal => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return { ...actual, useI18n: () => ({ locale: ref('en') }) }
})

const lifecycle = {
  admin: { mounted: 0, unmounted: 0 },
  user: { mounted: 0, unmounted: 0 },
}

function shellStub(name: 'admin' | 'user') {
  return defineComponent({
    name: `${name}ShellStub`,
    setup(_, { slots }) {
      onMounted(() => { lifecycle[name].mounted++ })
      onUnmounted(() => { lifecycle[name].unmounted++ })
      return () => h('div', { class: `${name}-shell-stub` }, slots.default?.())
    },
  })
}

vi.mock('@/components/layout/AppShell.vue', () => ({ default: shellStub('admin') }))
vi.mock('@/components/layout/UserShell.vue', () => ({ default: shellStub('user') }))

import App from '@/App.vue'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useAdminSettingsStore } from '@/stores/adminSettings'

const page = (name: string) => defineComponent({ name, render: () => h('div', { class: name }, name) })

function deferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>(yes => { resolve = yes })
  return { promise, resolve }
}

function createTestRouter() {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      ...['/admin/dashboard', '/admin/users'].map(path => ({ path, component: page(`admin-${path.slice(7)}`), meta: { appLayout: true } })),
      ...['/dashboard', '/keys'].map(path => ({ path, component: page(path.slice(1)), meta: { appLayout: true } })),
      { path: '/custom/:id', component: page('custom'), meta: { appLayout: true } },
      { path: '/login', component: page('login') },
    ],
  })
}

describe('App shell surface isolation', () => {
  beforeEach(() => {
    lifecycle.admin = { mounted: 0, unmounted: 0 }
    lifecycle.user = { mounted: 0, unmounted: 0 }
    delete document.body.dataset.patrickSurface
    setActivePinia(createPinia())
    vi.spyOn(useAppStore(), 'fetchPublicSettings').mockResolvedValue(null as never)
    useAuthStore().user = { id: 1, username: 'Tester', email: 'tester@example.test', role: 'user' } as never
    useAdminSettingsStore().customMenuItems = []
  })

  async function mountApp(initialPath: string) {
    const router = createTestRouter()
    await router.push(initialPath)
    await router.isReady()
    const wrapper = mount(App, {
      global: {
        plugins: [router],
        stubs: { NavigationProgress: true, Toast: true, AnnouncementPopup: true, AdminComplianceDialog: true, LoadingSpinner: true },
      },
    })
    await flushPromises()
    return { router, wrapper }
  }

  it('keeps the admin shell mounted between real admin routes, then switches to the user shell', async () => {
    const { router, wrapper } = await mountApp('/admin/dashboard')
    expect(wrapper.find('.admin-shell-stub .admin-dashboard').exists()).toBe(true)
    expect(wrapper.find('.user-shell-stub').exists()).toBe(false)
    expect(document.body.dataset.patrickSurface).toBeUndefined()

    await router.push('/admin/users')
    await flushPromises()
    expect(wrapper.find('.admin-shell-stub .admin-users').exists()).toBe(true)
    expect(lifecycle.admin).toEqual({ mounted: 1, unmounted: 0 })

    await router.push('/dashboard')
    await flushPromises()
    expect(wrapper.find('.user-shell-stub .dashboard').exists()).toBe(true)
    expect(wrapper.find('.admin-shell-stub').exists()).toBe(false)
    expect(document.body.dataset.patrickSurface).toBe('true')
    expect(lifecycle.admin).toEqual({ mounted: 1, unmounted: 1 })
    expect(lifecycle.user).toEqual({ mounted: 1, unmounted: 0 })

    await router.push('/keys')
    await flushPromises()
    expect(wrapper.find('.user-shell-stub .keys').exists()).toBe(true)
    expect(lifecycle.user).toEqual({ mounted: 1, unmounted: 0 })
  })

  it('uses the admin shell for an administrator-only custom page and clears the user marker', async () => {
    useAuthStore().user = { id: 2, username: 'Admin', email: 'admin@example.test', role: 'admin' } as never
    Object.assign(useAdminSettingsStore(), {
      loaded: true,
      customMenuItems: [{ id: 'ops-home', page_slug: 'ops-home', visibility: 'admin' }],
    })
    const { router, wrapper } = await mountApp('/dashboard')
    expect(document.body.dataset.patrickSurface).toBe('true')

    await router.push('/custom/ops-home')
    await flushPromises()
    expect(wrapper.find('.admin-shell-stub .custom').exists()).toBe(true)
    expect(wrapper.find('.user-shell-stub').exists()).toBe(false)
    expect(document.body.dataset.patrickSurface).toBeUndefined()
  })

  it('waits for admin menus on a direct custom-page visit before choosing a shell', async () => {
    useAuthStore().user = { id: 3, username: 'Admin', email: 'admin@example.test', role: 'admin' } as never
    const adminStore = useAdminSettingsStore()
    adminStore.loaded = false
    adminStore.customMenuItems = []
    const pending = deferred<void>()
    const fetchSettings = vi.spyOn(adminStore, 'fetch').mockImplementation(async () => {
      adminStore.loading = true
      await pending.promise
      adminStore.customMenuItems = [{ id: 'ops-home', page_slug: 'ops-home', visibility: 'admin' } as never]
      adminStore.loaded = true
      adminStore.loading = false
    })

    const { wrapper } = await mountApp('/custom/ops-home')
    expect(fetchSettings).toHaveBeenCalledOnce()
    expect(wrapper.get('[data-testid="workspace-surface-loading"]').exists()).toBe(true)
    expect(wrapper.find('.admin-shell-stub, .user-shell-stub').exists()).toBe(false)
    expect(document.body.dataset.patrickSurface).toBeUndefined()

    pending.resolve()
    await flushPromises()
    expect(wrapper.find('.admin-shell-stub .custom').exists()).toBe(true)
    expect(wrapper.find('.user-shell-stub').exists()).toBe(false)
    expect(wrapper.find('[data-testid="workspace-surface-loading"]').exists()).toBe(false)
    expect(document.body.dataset.patrickSurface).toBeUndefined()
  })

  it('keeps the custom route in a retry state after an admin-settings failure', async () => {
    useAuthStore().user = { id: 4, username: 'Admin', email: 'admin@example.test', role: 'admin' } as never
    const adminStore = useAdminSettingsStore()
    adminStore.loaded = false
    adminStore.customMenuItems = []
    const fetchSettings = vi.spyOn(adminStore, 'fetch').mockImplementation(async () => {
      adminStore.loading = true
      await Promise.resolve()
      if (fetchSettings.mock.calls.length > 1) {
        adminStore.customMenuItems = [{ id: 'ops-home', page_slug: 'ops-home', visibility: 'admin' } as never]
        adminStore.loaded = true
      }
      adminStore.loading = false
    })

    const { wrapper } = await mountApp('/custom/ops-home')
    expect(fetchSettings).toHaveBeenCalledOnce()
    expect(wrapper.get('[data-testid="workspace-surface-loading"]').text()).toContain('Try again')
    expect(wrapper.find('.admin-shell-stub, .user-shell-stub').exists()).toBe(false)
    expect(wrapper.find('.custom').exists()).toBe(false)

    await wrapper.get('[data-testid="workspace-surface-loading"] button').trigger('click')
    await flushPromises()
    expect(fetchSettings).toHaveBeenNthCalledWith(2, true)
    expect(wrapper.find('.admin-shell-stub .custom').exists()).toBe(true)
    expect(wrapper.find('.user-shell-stub').exists()).toBe(false)
    expect(wrapper.find('[data-testid="workspace-surface-loading"]').exists()).toBe(false)
  })

  it('leaves anonymous public auth routes outside either shell while retaining the Patrick marker', async () => {
    useAuthStore().user = null
    const { router, wrapper } = await mountApp('/dashboard')
    expect(document.body.dataset.patrickSurface).toBe('true')

    await router.push('/login')
    await flushPromises()
    expect(wrapper.find('.login').exists()).toBe(true)
    expect(wrapper.find('.admin-shell-stub, .user-shell-stub').exists()).toBe(false)
    expect(document.body.dataset.patrickSurface).toBe('true')
  })
})
