import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises, type VueWrapper } from '@vue/test-utils'
import { reactive, ref } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import AdminShell from '../AdminShell.vue'

const desktop = ref(true), rail = ref(true), isDark = ref(false)
const toggleMode = vi.fn(() => { isDark.value = !isDark.value })
const app = reactive({ sidebarScrollTop: 0, docUrl: '', cachedPublicSettings: {}, showError: vi.fn() })
const auth = reactive({ isSimpleMode: false, user: { username: 'Admin', email: 'admin@example.test' }, logout: vi.fn() })
const settings = reactive({ customMenuItems: [], opsMonitoringEnabled: true, paymentEnabled: true, fetch: vi.fn() })
vi.mock('@/stores', () => ({ useAppStore: () => app, useAuthStore: () => auth, useAdminSettingsStore: () => settings, useOnboardingStore: () => ({ setReplayCallback: vi.fn() }) }))
vi.mock('@/stores/app', () => ({ useAppStore: () => app }))
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key, locale: ref('zh') }) }
})
vi.mock('@vueuse/core', () => ({ useMediaQuery: (query: string) => query === '(min-width: 1200px)' ? desktop : rail, onClickOutside: vi.fn() }))
vi.mock('@/composables/useColorMode', () => ({ useColorMode: () => ({ isDark, toggleMode }) }))
vi.mock('@/composables/useBatchImageAccess', () => ({ useBatchImageAccess: () => ({ canUseBatchImage: ref(false), refreshBatchImageAccess: vi.fn() }) }))
vi.mock('@/composables/useOnboardingTour', () => ({ useOnboardingTour: () => ({ replayTour: vi.fn() }) }))
vi.mock('@/composables/useRoutePrefetch', () => ({ prefetchRoutePath: vi.fn() }))
vi.mock('@/utils/heliosLaunch', () => ({ launchHeliosWorkbench: vi.fn() }))
vi.mock('@/utils/featureFlags', () => ({ FeatureFlags: {}, isFeatureFlagEnabled: () => true, resolveFeatureFlag: () => true }))
vi.mock('@/router/title', () => ({ resolveRouteMetaKeys: () => ({ titleKey: 'nav.usage' }) }))
vi.mock('@/components/brand/PatrickBrand.vue', () => ({ default: { template: '<span>patrickapi</span>' } }))
vi.mock('@/components/common/LocaleSwitcher.vue', () => ({ default: { template: '<button>Language</button>' } }))
vi.mock('@/components/common/AnnouncementBell.vue', () => ({ default: { template: '<button>Announcements</button>' } }))
const wrappers: VueWrapper[] = []
beforeEach(() => { desktop.value = true; rail.value = true; isDark.value = false; vi.clearAllMocks(); localStorage.removeItem('patrick-admin-sidebar-collapsed') })
afterEach(() => wrappers.splice(0).forEach(wrapper => wrapper.unmount()))
async function shell() {
  const router = createRouter({ history: createMemoryHistory(), routes: [{ path: '/:pathMatch(.*)*', component: { template: '<div />' } }] })
  await router.push('/admin/usage')
  await router.isReady()
  const wrapper = mount(AdminShell, { attachTo: document.body, global: { plugins: [router] } })
  wrappers.push(wrapper)
  await flushPromises()
  return { wrapper, router }
}

describe('administration responsive shell', () => {
  it('keeps desktop collapse separate from the shared color mode', async () => {
    const { wrapper } = await shell()
    expect(wrapper.attributes('data-mode')).toBe('desktop')
    await wrapper.get('.admin-topbar > button').trigger('click')
    expect(wrapper.attributes('data-collapsed')).toBe('true')
    expect(localStorage.getItem('patrick-admin-sidebar-collapsed')).toBe('true')
    await wrapper.get('.admin-mode-toggle').trigger('click')
    expect(toggleMode).toHaveBeenCalledOnce()
    expect(wrapper.get('.admin-mode-toggle').attributes('aria-pressed')).toBe('true')
  })

  it('keeps the rail visible and locks an opened drawer until Escape restores focus', async () => {
    desktop.value = false
    const { wrapper } = await shell()
    expect(wrapper.attributes('data-mode')).toBe('rail')
    expect(wrapper.attributes('data-collapsed')).toBe('true')
    expect(wrapper.get('.admin-sidebar').isVisible()).toBe(true)
    const toggle = wrapper.get('.admin-topbar > button')
    await toggle.trigger('click')
    expect(wrapper.get('.admin-sidebar').attributes('aria-modal')).toBe('true')
    expect(wrapper.get('.admin-workspace').attributes()).toHaveProperty('inert')
    expect(document.body.style.overflow).toBe('hidden')
    await wrapper.get('.admin-sidebar').trigger('keydown', { key: 'Escape' })
    expect(document.activeElement).toBe(toggle.element)
    expect(document.body.style.overflow).not.toBe('hidden')
  })

  it('traps keyboard focus on phones and moves focus into content after navigation', async () => {
    desktop.value = false; rail.value = false
    const { wrapper, router } = await shell()
    expect(wrapper.attributes('data-mode')).toBe('mobile')
    expect(wrapper.get('.admin-sidebar').isVisible()).toBe(false)
    await wrapper.get('.admin-topbar > button').trigger('click')
    const controls = wrapper.get('.admin-sidebar').findAll('a[href],button:not(:disabled),input:not(:disabled)')
    ;(controls[controls.length - 1].element as HTMLElement).focus()
    await controls[controls.length - 1].trigger('keydown', { key: 'Tab' })
    expect(document.activeElement).toBe(controls[0].element)
    await router.push('/admin/accounts')
    await flushPromises()
    expect(wrapper.get('.admin-sidebar').isVisible()).toBe(false)
    expect(document.activeElement).toBe(wrapper.get('#admin-content').element)
    expect(document.body.classList.contains('modal-open')).toBe(false)
  })
})
