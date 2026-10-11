import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { ref, type Component } from 'vue'
import OAuthCallbackView from '../OAuthCallbackView.vue'
import LinuxDoCallbackView from '../LinuxDoCallbackView.vue'
import DingTalkCallbackView from '../DingTalkCallbackView.vue'
import OidcCallbackView from '../OidcCallbackView.vue'
import WechatCallbackView from '../WechatCallbackView.vue'
import DingTalkEmailCompletionView from '../DingTalkEmailCompletionView.vue'

const state = vi.hoisted(() => ({
  route: { path: '/auth/oauth/callback', query: {} as Record<string, unknown> },
  isAdmin: false,
  loginAsAdmin: true,
  replace: vi.fn(),
  setToken: vi.fn(),
  pending: vi.fn(),
  clearPending: vi.fn(),
  exchange: vi.fn(),
  post: vi.fn(),
  login2FA: vi.fn(),
  success: vi.fn(),
  error: vi.fn(),
}))
vi.mock('vue-router', () => ({ useRoute: () => state.route, useRouter: () => ({ replace: state.replace }) }))
vi.mock('vue-i18n', async () => ({
  ...await vi.importActual<typeof import('vue-i18n')>('vue-i18n'),
  useI18n: () => ({ t: (key: string) => key, te: () => false, locale: ref('en') }),
}))
vi.mock('@/stores', () => ({
  useAuthStore: () => ({
    get isAdmin() { return state.isAdmin },
    setToken: state.setToken,
    setPendingAuthSession: state.pending,
    clearPendingAuthSession: state.clearPending,
  }),
  useAppStore: () => ({
    showSuccess: state.success,
    showError: state.error,
    publicSettingsLoaded: true,
    cachedPublicSettings: { wechat_oauth_open_enabled: true, wechat_oauth_mp_enabled: false },
  }),
}))
vi.mock('@/api/client', () => ({ apiClient: { post: (...args: unknown[]) => state.post(...args) } }))
vi.mock('@/api/auth', async () => ({
  ...await vi.importActual<typeof import('@/api/auth')>('@/api/auth'),
  exchangePendingOAuthCompletion: (...args: unknown[]) => state.exchange(...args),
  login2FA: (...args: unknown[]) => state.login2FA(...args),
  getPublicSettings: async () => ({ oidc_oauth_provider_name: 'OIDC', invitation_code_enabled: false, turnstile_enabled: false }),
  persistOAuthTokenContext: vi.fn(),
}))
vi.mock('@/composables/useClipboard', () => ({ useClipboard: () => ({ copyToClipboard: vi.fn() }) }))

const providers = [
  ['email', OAuthCallbackView],
  ['linuxdo', LinuxDoCallbackView],
  ['dingtalk', DingTalkCallbackView],
  ['oidc', OidcCallbackView],
  ['wechat', WechatCallbackView],
] as const
const wrappers: VueWrapper[] = []
function render(component: Component) {
  const wrapper = mount(component, { global: { stubs: {
    AuthLayout: { template: '<div><slot /></div>' },
    Icon: true,
    PatrickBrand: true,
    RouterLink: { template: '<a><slot /></a>' },
    PendingOAuthCreateAccountForm: { name: 'PendingOAuthCreateAccountForm', emits: ['submit'], template: '<form />' },
    transition: false,
  } } })
  wrappers.push(wrapper)
  return wrapper
}
beforeEach(() => {
  vi.clearAllMocks()
  state.route.path = '/auth/oauth/callback'
  state.route.query = {}
  state.isAdmin = false
  state.loginAsAdmin = true
  state.setToken.mockImplementation(async () => {
    await Promise.resolve()
    state.isAdmin = state.loginAsAdmin
    return { role: state.isAdmin ? 'admin' : 'user' }
  })
  state.exchange.mockResolvedValue({ access_token: 'test-token', redirect: '/auth/landing' })
  state.login2FA.mockResolvedValue({ access_token: 'verified-token' })
  state.post.mockResolvedValue({ data: { access_token: 'completed-token' } })
  window.location.hash = ''
  localStorage.clear()
  sessionStorage.clear()
})
afterEach(() => { wrappers.splice(0).forEach(wrapper => wrapper.unmount()); window.location.hash = '' })

describe.each(providers)('%s OAuth landing', (_provider, component) => {
  it.each([
    ['/auth/landing', true, '/admin/dashboard'],
    ['/auth/landing', false, '/dashboard'],
    ['/dashboard', true, '/dashboard'],
    ['/profile/connections', true, '/profile/connections'],
  ] as const)('resolves %s for admin=%s only after the user is loaded', async (redirect, isAdmin, expected) => {
    state.loginAsAdmin = isAdmin
    state.exchange.mockResolvedValue({ access_token: 'test-token', redirect })
    render(component)
    await flushPromises()
    expect(state.setToken).toHaveBeenCalledWith('test-token')
    expect(state.replace).toHaveBeenCalledWith(expected)
    expect(state.error).not.toHaveBeenCalled()
  })

  it('retains automatic landing through the legacy token fragment response', async () => {
    window.location.hash = '#access_token=legacy-token&redirect=%2Fauth%2Flanding'
    render(component)
    await flushPromises()
    expect(state.exchange).not.toHaveBeenCalled()
    expect(state.setToken).toHaveBeenCalledWith('legacy-token')
    expect(state.replace).toHaveBeenCalledWith('/admin/dashboard')
  })
})

describe('pending OAuth landing', () => {
  it.each(providers.slice(1))('keeps automatic intent through the %s two-step challenge', async (provider, component) => {
    state.exchange.mockResolvedValue({ requires_2fa: true, temp_token: 'pending-token', redirect: '/auth/landing' })
    const wrapper = render(component)
    await flushPromises()
    expect(state.replace).not.toHaveBeenCalled()
    expect(state.pending).toHaveBeenCalledWith(expect.objectContaining({ redirect: '/auth/landing' }))
    await wrapper.get(`[data-testid="${provider}-bind-login-totp"]`).setValue('123456')
    await wrapper.get(`[data-testid="${provider}-bind-login-totp-submit"]`).trigger('click')
    await flushPromises()
    expect(state.login2FA).toHaveBeenCalledWith({ temp_token: 'pending-token', totp_code: '123456' })
    expect(state.replace).toHaveBeenCalledWith('/admin/dashboard')
  })

  it('keeps email registration pending until completion and then uses the loaded role', async () => {
    state.exchange.mockResolvedValue({ error: 'registration_completion_required', provider: 'google', redirect: '/auth/landing', resolved_email: 'test@example.com' })
    const wrapper = render(OAuthCallbackView)
    await flushPromises()
    expect(state.replace).not.toHaveBeenCalled()
    const passwords = wrapper.findAll('input[type="password"]')
    await passwords[0].setValue('test-password')
    await passwords[1].setValue('test-password')
    await passwords[1].trigger('keyup', { key: 'Enter' })
    await flushPromises()
    expect(state.post).toHaveBeenCalledWith('/auth/oauth/google/complete-registration', { password: 'test-password' })
    expect(state.replace).toHaveBeenCalledWith('/admin/dashboard')
  })

  it('preserves intent across the DingTalk email-completion page', async () => {
    state.exchange.mockResolvedValue({ auth_result: 'pending_session', step: 'email_completion', redirect: '/auth/landing' })
    render(DingTalkCallbackView)
    await flushPromises()
    expect(state.replace).toHaveBeenCalledWith('/auth/dingtalk/email-completion?redirect=%2Fauth%2Flanding')
    wrappers.splice(0).forEach(wrapper => wrapper.unmount())
    state.route.query = { redirect: '/auth/landing' }
    const wrapper = render(DingTalkEmailCompletionView)
    wrapper.findComponent({ name: 'PendingOAuthCreateAccountForm' }).vm.$emit('submit', { email: 'test@example.com', password: 'test-password' })
    await flushPromises()
    expect(state.setToken).toHaveBeenCalledWith('completed-token')
    expect(state.replace).toHaveBeenLastCalledWith('/admin/dashboard')
  })

  it('preserves a binding completion target without treating it as a new login', async () => {
    state.exchange.mockResolvedValue({ auth_result: 'bind', redirect: '/profile/connections' })
    render(DingTalkCallbackView)
    await flushPromises()
    expect(state.setToken).not.toHaveBeenCalled()
    expect(state.replace).toHaveBeenCalledWith('/profile/connections')
  })
})
