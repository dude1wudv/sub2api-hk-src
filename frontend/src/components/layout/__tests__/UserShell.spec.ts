import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { mount, flushPromises, type VueWrapper } from '@vue/test-utils'
import { computed, nextTick, reactive, ref } from 'vue'
import { createMemoryHistory, createRouter } from 'vue-router'
import type { PublicSettings, User } from '@/types'
import UserShell from '../UserShell.vue'

const spies = vi.hoisted(() => ({
  launchHelios: vi.fn().mockResolvedValue(true),
  refreshAccess: vi.fn().mockResolvedValue(true),
  replayTour: vi.fn(),
  prefetch: vi.fn(),
  logout: vi.fn().mockResolvedValue(undefined),
  showError: vi.fn(),
}))

const desktop = ref(true)
const railViewport = ref(true)
const batchAllowed = ref(true)
const appStore = reactive({
  cachedPublicSettings: {} as Partial<PublicSettings>,
  backendModeEnabled: false,
  docUrl: 'https://docs.example.test/',
  contactInfo: '',
  showError: spies.showError,
})
const authStore = reactive({
  user: null as Partial<User> | null,
  isAuthenticated: true,
  isAdmin: false,
  isSimpleMode: false,
  logout: spies.logout,
})
const onboardingStore = {
  isCurrentStep: vi.fn().mockReturnValue(false),
  nextStep: vi.fn(),
  setReplayCallback: vi.fn(),
}

vi.mock('@/stores', () => ({ useAppStore: () => appStore, useAuthStore: () => authStore, useOnboardingStore: () => onboardingStore }))
vi.mock('@/stores/app', () => ({ useAppStore: () => appStore }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => authStore }))
vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key, locale: ref('zh-CN') }) }
})
vi.mock('@vueuse/core', () => ({ useMediaQuery: (query: string) => query === '(min-width: 1100px)' ? desktop : railViewport }))
vi.mock('@/composables/useBatchImageAccess', () => ({ useBatchImageAccess: () => ({ canUseBatchImage: batchAllowed, refreshBatchImageAccess: spies.refreshAccess }) }))
vi.mock('@/composables/useGlacierPreferences', () => ({
  tableDensityKey: Symbol('table-density'),
  useGlacierPreferences: () => ({ effectiveDensity: computed(() => 'comfortable'), isGlassLayout: computed(() => false) }),
}))
vi.mock('@/composables/useGlacierInteraction', () => ({ useGlacierInteraction: vi.fn() }))
vi.mock('@/composables/useOnboardingTour', () => ({ useOnboardingTour: () => ({ replayTour: spies.replayTour }) }))
vi.mock('@/composables/useRoutePrefetch', () => ({ prefetchRoutePath: spies.prefetch }))
vi.mock('@/router/title', () => ({ resolveRouteMetaKeys: () => ({ titleKey: 'dashboard.title' }) }))
vi.mock('@/utils/heliosLaunch', () => ({ launchHeliosWorkbench: spies.launchHelios }))
vi.mock('@/components/brand/PatrickBrand.vue', () => ({ default: { props: ['compact'], template: '<span>patrickapi</span>' } }))
vi.mock('@/components/common/LocaleSwitcher.vue', () => ({ default: { template: '<button aria-label="language">中文</button>' } }))
vi.mock('@/components/common/AnnouncementBell.vue', () => ({ default: { template: '<button aria-label="announcements">公告</button>' } }))
vi.mock('@/components/common/SubscriptionProgressMini.vue', () => ({ default: { template: '<span data-testid="subscriptions" />' } }))

const wrappers: VueWrapper[] = []

async function mountShell(openMore = false) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      ...['/home', '/dashboard', '/keys', '/usage', '/profile', '/login', '/purchase', '/model-plaza', '/admin/dashboard'].map(path => ({
        path,
        component: { template: '<div />' },
        meta: { title: 'Workspace' },
      })),
      { path: '/:pathMatch(.*)*', component: { template: '<div />' } },
    ],
  })
  await router.push('/keys')
  await router.isReady()
  const wrapper = mount(UserShell, { attachTo: document.body, global: { plugins: [router] }, slots: { default: '<div>Workspace page</div>' } })
  wrappers.push(wrapper)
  await flushPromises()
  if (openMore) {
    await wrapper.get('.user-more-toggle').trigger('click')
    await flushPromises()
  }
  return { wrapper, router }
}

function navPaths(wrapper: VueWrapper) {
  return wrapper.findAll('[data-nav-path]').map(item => item.attributes('data-nav-path'))
}

beforeEach(() => {
  vi.clearAllMocks()
  desktop.value = true
  railViewport.value = true
  batchAllowed.value = true
  authStore.user = { id: 7, username: 'Demo', email: 'demo@example.test', balance: 12, role: 'user' }
  authStore.isAuthenticated = true
  authStore.isAdmin = false
  authStore.isSimpleMode = false
  appStore.backendModeEnabled = false
  appStore.cachedPublicSettings = {
    subscription_enabled: true,
    payment_enabled: true,
    channel_monitor_enabled: true,
    available_channels_enabled: true,
    affiliate_enabled: true,
    model_plaza_enabled: true,
    custom_menu_items: [
      { id: 'later', label: 'Later', visibility: 'user', sort_order: 2, icon_svg: '', url: '' },
      { id: 'admin-only', label: 'Admin only', visibility: 'admin', sort_order: 0, icon_svg: '', url: '' },
      { id: 'first', label: 'First', visibility: 'user', sort_order: 1, icon_svg: '', url: '' },
    ],
  }
})

afterEach(() => {
  wrappers.splice(0).forEach(wrapper => wrapper.unmount())
})

describe('personal workspace navigation', () => {
  it('retains every personal feature and only sorted user custom menus', async () => {
    const { wrapper, router } = await mountShell(true)
    expect(navPaths(wrapper)).toEqual([
      '/keys', '/usage', '/purchase', '/subscriptions', '/profile', '/model-plaza', '/available-channels', '/monitor',
      '/image2', '/helios', '/batch-image', '/affiliate',
      '/custom/first', '/custom/later',
    ])
    expect(wrapper.get('[data-nav-path="/image2"]').attributes('href')).toBe('/image2/')
    expect(wrapper.get('[data-nav-path="/model-plaza"]').attributes('href')).toBe('/model-plaza?embedded=1')
    expect(wrapper.get('[data-nav-path="/keys"]').attributes('data-tour')).toBe('sidebar-my-keys')
    expect(wrapper.find('[aria-label="切换语言"]').exists()).toBe(true)
    expect(wrapper.find('[aria-label="announcements"]').exists()).toBe(true)

    await wrapper.get('[data-nav-path="/model-plaza"]').trigger('click')
    await flushPromises()
    expect(router.currentRoute.value.path).toBe('/model-plaza')
    expect(router.currentRoute.value.query).toEqual({ embedded: '1' })
  })

  it('applies feature switches and the batch image permission', async () => {
    Object.assign(appStore.cachedPublicSettings, {
      subscription_enabled: false, payment_enabled: false, channel_monitor_enabled: false,
      available_channels_enabled: false, affiliate_enabled: false, model_plaza_enabled: false,
    })
    batchAllowed.value = false
    const { wrapper } = await mountShell(true)
    const paths = navPaths(wrapper)
    for (const path of ['/subscriptions', '/orders', '/monitor', '/available-channels', '/affiliate', '/model-plaza', '/batch-image']) {
      expect(paths).not.toContain(path)
    }
    expect(paths).toContain('/purchase')
    expect(wrapper.get('.user-balance-pill').attributes('href')).toBe('/purchase')
    expect(wrapper.find('[data-testid="subscriptions"]').exists()).toBe(false)
  })

  it('keeps simple-mode limits without hiding the remaining personal entries', async () => {
    authStore.isSimpleMode = true
    const { wrapper } = await mountShell(true)
    expect(navPaths(wrapper)).toEqual(['/keys', '/profile', '/model-plaza', '/monitor', '/custom/first', '/custom/later'])
    expect(wrapper.find('.user-balance-pill').exists()).toBe(false)
    expect(navPaths(wrapper)).not.toContain('/purchase')
    expect(navPaths(wrapper)).not.toContain('/redeem')
  })

  it('keeps the recharge label when subscriptions are disabled', async () => {
    appStore.cachedPublicSettings.subscription_enabled = false
    const { wrapper } = await mountShell(true)
    expect(wrapper.get('[data-nav-path="/purchase"]').text()).toBe('充值')
  })

  it('opens API quick start at the real key guide without invoking the onboarding tour', async () => {
    const { wrapper, router } = await mountShell()
    await wrapper.get('.user-account-trigger').trigger('click')
    await wrapper.get('button.user-account-action:not(.user-signout)').trigger('click')
    await flushPromises()

    expect(router.currentRoute.value.path).toBe('/keys')
    expect(router.currentRoute.value.hash).toBe('#api-quick-start')
    expect(spies.replayTour).not.toHaveBeenCalled()
  })

  it('gives administrators personal navigation and an explicit return to administration', async () => {
    authStore.isAdmin = true
    authStore.user!.role = 'admin'
    appStore.backendModeEnabled = true
    const { wrapper } = await mountShell()
    expect(navPaths(wrapper)).toContain('/keys')
    expect(navPaths(wrapper)).toContain('/profile')
    expect(navPaths(wrapper)).not.toContain('/admin/accounts')
    expect(wrapper.get('.user-admin-return').attributes('href')).toBe('/admin/dashboard')
  })

  it('opens Helios through the existing popup handoff and reports a blocked popup', async () => {
    const { wrapper } = await mountShell(true)
    await wrapper.get('[data-nav-path="/helios"]').trigger('click')
    expect(spies.launchHelios).toHaveBeenCalledWith(expect.objectContaining({ mode: 'popup' }))
    spies.launchHelios.mock.calls[0][0].notify('popup-blocked')
    expect(spies.showError).toHaveBeenCalledWith('helios.popupBlocked')
  })

  it('signs out through the existing auth store and returns to login', async () => {
    const { wrapper, router } = await mountShell()
    await wrapper.get('.user-account-trigger').trigger('click')
    await wrapper.get('.user-signout').trigger('click')
    await flushPromises()
    expect(spies.logout).toHaveBeenCalledOnce()
    expect(router.currentRoute.value.path).toBe('/login')
  })
})

describe('personal workspace navigation at narrow widths', () => {
  beforeEach(() => { desktop.value = false; railViewport.value = false })

  it('opens with a labelled control, closes with Escape and restores focus', async () => {
    const { wrapper } = await mountShell()
    const toggle = wrapper.get('.user-nav-toggle')
    expect(wrapper.get('.user-sidebar').isVisible()).toBe(false)
    await toggle.trigger('click')
    await nextTick()
    expect(wrapper.get('.user-sidebar').isVisible()).toBe(true)
    expect(wrapper.get('.user-sidebar').attributes('role')).toBe('dialog')
    expect(wrapper.get('.user-sidebar').attributes('aria-modal')).toBe('true')
    expect(wrapper.get('.user-drawer-scrim').exists()).toBe(true)
    expect(document.activeElement).toBe(wrapper.get('.user-sidebar [aria-label="关闭导航"]').element)
    expect(wrapper.get('.user-workspace').attributes()).toHaveProperty('inert')
    await wrapper.get('.user-sidebar').trigger('keydown', { key: 'Escape' })
    await nextTick()
    expect(wrapper.get('.user-sidebar').isVisible()).toBe(false)
    expect(document.activeElement).toBe(toggle.element)
  })

  it('keeps Tab inside the drawer and closes it after navigation', async () => {
    const { wrapper, router } = await mountShell()
    await wrapper.get('.user-nav-toggle').trigger('click')
    const controls = wrapper.get('.user-sidebar').findAll('a[href], button:not(:disabled)')
    ;(controls[controls.length - 1].element as HTMLElement).focus()
    await controls[controls.length - 1].trigger('keydown', { key: 'Tab' })
    expect(document.activeElement).toBe(controls[0].element)
    await router.push('/usage')
    await flushPromises()
    expect(router.currentRoute.value.path).toBe('/usage')
    expect(wrapper.get('.user-sidebar').isVisible()).toBe(false)
    expect(document.activeElement).toBe(wrapper.get('#user-workspace-content').element)
  })
})

describe('personal workspace navigation at rail widths (740–1099px)', () => {
  beforeEach(() => { desktop.value = false; railViewport.value = true })

  it('keeps the collapsed navigation rail visible without treating it as a modal', async () => {
    const { wrapper } = await mountShell()
    expect(wrapper.get('.user-portal').attributes('data-navigation-mode')).toBe('rail')
    expect(wrapper.get('.user-portal').attributes('data-sidebar-collapsed')).toBe('true')
    expect(wrapper.get('.user-sidebar').isVisible()).toBe(true)
    expect(wrapper.get('.user-sidebar').attributes('role')).toBeUndefined()
    expect(wrapper.get('.user-sidebar').attributes('aria-modal')).toBeUndefined()
    expect(wrapper.get('.user-workspace').attributes('inert')).toBeUndefined()

    await wrapper.get('.user-nav-toggle').trigger('click')
    await nextTick()
    expect(wrapper.get('.user-portal').attributes('data-sidebar-collapsed')).toBe('false')
    expect(wrapper.get('.user-portal').attributes('data-nav-open')).toBe('true')
    expect(wrapper.get('.user-sidebar').attributes('role')).toBe('dialog')
  })
})
