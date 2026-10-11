import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, RouterLinkStub } from '@vue/test-utils'

import HomeView from '../HomeView.vue'

const { appStore, authStore, routerPush, copyToClipboard } = vi.hoisted(() => ({
  appStore: {
    cachedPublicSettings: {} as Record<string, unknown>,
    siteName: 'Fallback site',
    siteLogo: '',
    docUrl: '',
    publicSettingsLoaded: true,
    fetchPublicSettings: vi.fn(),
  },
  authStore: {
    isAuthenticated: false,
    isAdmin: false,
    user: null as { email?: string } | null,
    checkAuth: vi.fn(),
  },
  routerPush: vi.fn(),
  copyToClipboard: vi.fn().mockResolvedValue(true),
}))

vi.mock('@/composables/useClipboard', () => ({ useClipboard: () => ({ copyToClipboard }) }))

vi.mock('@/stores', () => ({
  useAppStore: () => appStore,
  useAuthStore: () => authStore,
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => appStore,
}))

vi.mock('vue-router', async () => {
  const actual = await vi.importActual<typeof import('vue-router')>('vue-router')
  return { ...actual, useRouter: () => ({ push: routerPush }) }
})

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key, locale: { value: 'en-US' } }),
  }
})

function mountHome(settings: Record<string, unknown> = {}) {
  appStore.cachedPublicSettings = {
    site_name: 'Test site',
    site_subtitle: 'Test subtitle',
    ...settings,
  }

  return mount(HomeView, {
    global: {
      stubs: {
        RouterLink: RouterLinkStub,
        PatrickBrand: { template: '<span data-testid="patrick-brand">patrickapi</span>' },
        LocaleSwitcher: { template: '<div data-testid="locale-switcher" />' },
        Icon: { template: '<span data-testid="icon" />' },
      },
    },
  })
}

function compactDestination(wrapper: ReturnType<typeof mountHome>) {
  return wrapper.get('[data-testid="home-console"]').findComponent(RouterLinkStub).props('to')
}

function modelPlazaDestination(wrapper: ReturnType<typeof mountHome>) {
  return wrapper
    .findAllComponents(RouterLinkStub)
    .find((link) => link.props('to') === '/model-plaza')
    ?.props('to')
}

describe('HomeView compact mode', () => {
  beforeEach(() => {
    authStore.isAuthenticated = false
    authStore.isAdmin = false
    authStore.user = null
    authStore.checkAuth.mockClear()
    appStore.fetchPublicSettings.mockClear()
    routerPush.mockClear()
    copyToClipboard.mockReset().mockResolvedValue(true)
    localStorage.clear()
    vi.spyOn(window, 'matchMedia').mockReturnValue({ matches: false, addEventListener: vi.fn(), removeEventListener: vi.fn() } as unknown as MediaQueryList)
  })

  it('renders custom HTML ahead of compact mode', () => {
    const wrapper = mountHome({
      compact_home_enabled: true,
      home_content: '<section id="custom-home">Custom home</section>',
    })

    expect(wrapper.get('#custom-home').text()).toBe('Custom home')
    expect(wrapper.find('[data-testid="compact-home"]').exists()).toBe(false)
  })

  it('renders custom URL content ahead of compact mode', () => {
    const wrapper = mountHome({
      compact_home_enabled: true,
      home_content: ' https://example.com/home ',
    })

    expect(wrapper.get('iframe').attributes('src')).toBe('https://example.com/home')
    expect(wrapper.find('[data-testid="compact-home"]').exists()).toBe(false)
  })

  it('treats whitespace-only custom content as empty and selects compact mode', () => {
    const wrapper = mountHome({ compact_home_enabled: true, home_content: ' \n\t ' })

    expect(wrapper.get('[data-testid="compact-home"]').get('h1').text()).toBe('Your ideas.In motion.')
    expect(wrapper.get('[data-testid="compact-home"] .home-entry').exists()).toBe(true)
  })

  it.each([undefined, false])('selects the patrickapi home when compact mode is %s', (enabled) => {
    const settings = enabled === undefined ? {} : { compact_home_enabled: enabled }
    const wrapper = mountHome(settings)

    expect(wrapper.get('[data-testid="patrick-home"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="home-console"]').findComponent(RouterLinkStub).props('to')).toBe('/login')
    expect(wrapper.get('h1').text()).toBe('Your ideas.In motion.')
    expect(wrapper.get('.home-endpoint code').text()).toMatch(/^https?:\/\//)
  })

  it('links unauthenticated visitors to login', () => {
    expect(compactDestination(mountHome({ compact_home_enabled: true }))).toBe('/login')
  })

  it('links authenticated users to their overview', () => {
    authStore.isAuthenticated = true

    expect(compactDestination(mountHome({ compact_home_enabled: true }))).toBe('/dashboard')
  })

  it('routes model search to the dedicated catalog', async () => {
    const wrapper = mountHome({ model_plaza_enabled: true })
    await wrapper.get('input[aria-label="Search models"]').setValue('  gpt-5.4  ')
    await wrapper.get('.home-entry').trigger('submit')

    expect(routerPush).toHaveBeenCalledWith({ path: '/model-plaza', query: { model: 'gpt-5.4' } })
    wrapper.unmount()
  })

  it('keeps an empty entry directed to key management', async () => {
    const wrapper = mountHome({ model_plaza_enabled: true })
    await wrapper.get('.home-entry').trigger('submit')
    expect(routerPush).toHaveBeenCalledWith({ path: '/keys' })
    wrapper.unmount()
  })

  it('links administrators to the admin dashboard', () => {
    authStore.isAuthenticated = true
    authStore.isAdmin = true

    const wrapper = mountHome({ compact_home_enabled: true })
    expect(compactDestination(wrapper)).toBe('/admin/dashboard')
  })

  it('keeps the connection diagram, endpoint and placeholder example on the same protocol', async () => {
    const wrapper = mountHome()
    expect(wrapper.get('.portal-api-code pre').text()).toContain('/v1/chat/completions')
    await wrapper.get('.ink-destination--anthropic').trigger('click')
    expect(wrapper.get('.ink-connection').attributes('data-protocol')).toBe('anthropic')
    expect(wrapper.get('.home-endpoint code').text()).not.toMatch(/\/v1$/)
    expect(wrapper.get('.portal-api-code pre').text()).toContain('/v1/messages')
    expect(wrapper.get('.portal-api-code pre').text()).toContain('YOUR_API_KEY')
    expect(wrapper.get('.portal-api-code pre').text()).toContain('anthropic-version')
    await wrapper.get('.portal-api-protocols button[data-active-indicator-item="openai"]').trigger('click')
    expect(wrapper.get('.ink-connection').attributes('data-protocol')).toBe('openai')
    expect(wrapper.get('.home-endpoint code').text()).toMatch(/\/v1$/)
    wrapper.unmount()
  })

  it.each(['protocol', 'unmount'])('discards pending copy feedback after %s changes', async change => {
    let finish!: (value: boolean) => void
    copyToClipboard.mockReturnValueOnce(new Promise<boolean>(resolve => { finish = resolve }))
    const wrapper = mountHome()
    const timer = vi.spyOn(globalThis, 'setTimeout')
    await wrapper.get('.home-endpoint button').trigger('click')
    if (change === 'protocol') await wrapper.get('.ink-destination--anthropic').trigger('click')
    else wrapper.unmount()
    finish(true)
    await flushPromises()
    expect(timer.mock.calls.filter(call => call[1] === 2200)).toHaveLength(0)
    if (change === 'protocol') {
      expect(wrapper.get('.home-endpoint button').attributes('aria-label')).toBe('Copy API base URL')
      wrapper.unmount()
    }
    timer.mockRestore()
  })

  it('shows the model plaza link to anonymous visitors when public access is enabled', () => {
    const wrapper = mountHome({
      compact_home_enabled: true,
      model_plaza_enabled: true,
      model_plaza_require_auth: false,
    })

    expect(modelPlazaDestination(wrapper)).toBe('/model-plaza')
  })

  it('hides the model plaza link from anonymous visitors when sign-in is required', () => {
    const wrapper = mountHome({
      compact_home_enabled: true,
      model_plaza_enabled: true,
      model_plaza_require_auth: true,
    })

    expect(modelPlazaDestination(wrapper)).toBeUndefined()
  })

  it('shows the model plaza link to authenticated visitors when sign-in is required', () => {
    authStore.isAuthenticated = true

    const wrapper = mountHome({
      compact_home_enabled: true,
      model_plaza_enabled: true,
      model_plaza_require_auth: true,
    })

    expect(modelPlazaDestination(wrapper)).toBe('/model-plaza')
  })

  it('shows the model plaza link in the default home header', () => {
    const wrapper = mountHome({
      model_plaza_enabled: true,
      model_plaza_require_auth: false,
    })

    expect(modelPlazaDestination(wrapper)).toBe('/model-plaza')
  })

  it('hides the model plaza link when the feature is disabled', () => {
    const wrapper = mountHome({
      compact_home_enabled: true,
      model_plaza_enabled: false,
      model_plaza_require_auth: false,
    })

    expect(modelPlazaDestination(wrapper)).toBeUndefined()
  })
})
