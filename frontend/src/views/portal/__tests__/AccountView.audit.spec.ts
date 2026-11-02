import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AccountView from '../AccountView.vue'

const { auth, app, changePassword, bindEmailIdentity } = vi.hoisted(() => ({
  auth: { user: {} as Record<string, unknown>, refreshUser: vi.fn() },
  app: { cachedPublicSettings: {} as Record<string, unknown>, fetchPublicSettings: vi.fn() },
  changePassword: vi.fn(),
  bindEmailIdentity: vi.fn(),
}))

vi.mock('@/stores/app', () => ({ useAppStore: () => app }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => auth }))
vi.mock('@/api', () => ({ userAPI: { changePassword, bindEmailIdentity } }))
vi.mock('vue-router', () => ({ useRoute: () => ({ fullPath: '/profile' }) }))
vi.mock('vue-i18n', async importOriginal => ({ ...(await importOriginal<typeof import('vue-i18n')>()), useI18n: () => ({ locale: { value: 'zh-CN' } }) }))

function mountAccount() {
  return mount(AccountView, {
    global: { stubs: {
      PortalPageHeader: true,
      PortalAccountSecurity: true,
      PortalAccountNotifications: true,
      MotionPreferenceToggle: true,
      PortalAccountDialog: { template: '<div role="dialog"><slot /></div>' },
    } },
  })
}

describe('account audit fixes', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    auth.user = { username: '', email: 'hidden.invalid', email_bound: false, auth_bindings: { linuxdo: { bound: true, display_name: 'groktest' } } }
    app.cachedPublicSettings = { password_policy: { min_length: 12, max_bytes: 72 }, passkey_enabled: false }
    app.fetchPublicSettings.mockResolvedValue(app.cachedPublicSettings)
    auth.refreshUser.mockResolvedValue(undefined)
    changePassword.mockResolvedValue(undefined)
    bindEmailIdentity.mockResolvedValue(auth.user)
  })

  it('keeps a real linked identity while showing an unset nickname without a brand fallback', async () => {
    const wrapper = mountAccount()
    await flushPromises()
    expect(wrapper.text()).toContain('未设置昵称')
    expect(wrapper.text()).toContain('groktest')
    expect(wrapper.text()).not.toContain('patrickapi')
    wrapper.unmount()
  })

  it('retries only failed settings and retains successfully loaded profile information', async () => {
    app.fetchPublicSettings.mockResolvedValueOnce(null)
    const wrapper = mountAccount()
    await flushPromises()
    expect(wrapper.text()).toContain('账户功能设置读取失败')
    expect(wrapper.text()).toContain('groktest')
    await wrapper.findAll('button').find(button => button.text() === '重试设置')!.trigger('click')
    await flushPromises()
    expect(app.fetchPublicSettings).toHaveBeenLastCalledWith(true)
    expect(auth.refreshUser).toHaveBeenCalledOnce()
    expect(wrapper.text()).not.toContain('账户功能设置读取失败')
    wrapper.unmount()
  })

  it('validates the new password but allows a short current password', async () => {
    const wrapper = mountAccount()
    await flushPromises()
    await wrapper.get('#account-security button').trigger('click')
    const inputs = wrapper.get('[role="dialog"]').findAll('input[type="password"]')
    await inputs[0].setValue('old')
    await inputs[1].setValue('😀'.repeat(6))
    await inputs[2].setValue('😀'.repeat(6))
    await wrapper.get('form').trigger('submit')
    expect(changePassword).not.toHaveBeenCalled()
    expect(wrapper.get('[role="alert"]').text()).toContain('12')
    await inputs[1].setValue('new-password-12')
    await inputs[2].setValue('new-password-12')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(changePassword).toHaveBeenCalledWith('old', 'new-password-12')
    wrapper.unmount()
  })
})
