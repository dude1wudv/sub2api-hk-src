import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import LoginView from '@/views/auth/LoginView.vue'

const { getPublicSettingsMock, loginMock, pushMock } = vi.hoisted(() => ({
  getPublicSettingsMock: vi.fn(),
  loginMock: vi.fn(),
  pushMock: vi.fn()
}))

const publicSettings = {
  registration_enabled: true,
  turnstile_enabled: false,
  turnstile_site_key: '',
  tencent_captcha_enabled: false,
  tencent_captcha_app_id: '',
  aliyun_captcha_enabled: false,
  aliyun_captcha_scene_id: '',
  aliyun_captcha_prefix: '',
  linuxdo_oauth_enabled: false,
  dingtalk_oauth_enabled: false,
  wechat_oauth_enabled: false,
  backend_mode_enabled: false,
  oidc_oauth_enabled: false,
  oidc_oauth_provider_name: 'OIDC',
  github_oauth_enabled: false,
  google_oauth_enabled: false,
  password_reset_enabled: false,
  passkey_enabled: false,
  login_agreement_enabled: false,
  login_agreement_documents: []
}

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: pushMock,
    currentRoute: { value: { query: {} } }
  })
}))

vi.mock('vue-i18n', () => ({
  createI18n: () => ({
    global: {
      t: (key: string) => key
    }
  }),
  useI18n: () => ({
    t: (key: string) => key
  })
}))

vi.mock('@/stores', () => ({
  useAuthStore: () => ({
    login: loginMock,
    loginWithPasskey: vi.fn(),
    login2FA: vi.fn()
  }),
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
    showWarning: vi.fn()
  })
}))

vi.mock('@/api/auth', () => ({
  buildOAuthLoginStartURL: vi.fn(),
  getPublicSettings: (...args: unknown[]) => getPublicSettingsMock(...args),
  isTotp2FARequired: vi.fn(() => false),
  isWeChatWebOAuthEnabled: vi.fn(() => false),
  startOAuthLogin: vi.fn()
}))

function mountLogin() {
  return mount(LoginView, {
    global: {
      stubs: {
        AuthLayout: { template: '<div><slot /><slot name="footer" /></div>' },
        DingTalkOAuthSection: true,
        EmailOAuthButtons: true,
        Icon: true,
        LinuxDoOAuthSection: true,
        LoginAgreementPrompt: true,
        OidcOAuthSection: true,
        RouterLink: { template: '<a><slot /></a>' },
        TotpLoginModal: true,
        TurnstileWidget: {
          emits: ['verify', 'expire', 'error'],
          setup(_props: unknown, { emit, expose }: { emit: (event: string, ...args: unknown[]) => void; expose: (value: object) => void }) {
            expose({ reset: vi.fn(), verifyAction: vi.fn(async () => null) })
            return { emitVerify: () => emit('verify', 'turnstile-token', '') }
          },
          template: '<button type="button" data-test="verify-captcha" @click="emitVerify">verify</button>'
        },
        WechatOAuthSection: true,
        transition: false
      }
    }
  })
}

describe('LoginView registration entry', () => {
  beforeEach(() => {
    getPublicSettingsMock.mockReset()
    loginMock.mockReset()
    pushMock.mockReset()
    getPublicSettingsMock.mockResolvedValue(publicSettings)
  })

  it('shows the registration entry when registration is enabled', async () => {
    const wrapper = mountLogin()
    await flushPromises()

    expect(wrapper.text()).toContain('auth.signUp')
  })

  it('hides the registration entry when registration is disabled', async () => {
    getPublicSettingsMock.mockResolvedValueOnce({
      ...publicSettings,
      registration_enabled: false
    })

    const wrapper = mountLogin()
    await flushPromises()

    expect(wrapper.text()).not.toContain('auth.signUp')
  })

  it('does not submit by Enter while login settings are loading', async () => {
    let resolveSettings!: (value: typeof publicSettings) => void
    getPublicSettingsMock.mockReturnValueOnce(new Promise((resolve) => { resolveSettings = resolve }))
    const wrapper = mountLogin()
    await wrapper.get('#email').setValue('admin@example.com')
    await wrapper.get('#password').setValue('password123')
    await wrapper.get('form').trigger('submit')

    expect(loginMock).not.toHaveBeenCalled()
    resolveSettings(publicSettings)
    await flushPromises()
  })

  it('does not submit by Enter while settings are unavailable, and permits retrying settings', async () => {
    const consoleError = vi.spyOn(console, 'error').mockImplementation(() => {})
    getPublicSettingsMock.mockRejectedValueOnce(new Error('settings unavailable'))
    const wrapper = mountLogin()
    await flushPromises()
    await wrapper.get('#email').setValue('admin@example.com')
    await wrapper.get('#password').setValue('password123')
    await wrapper.get('form').trigger('submit')

    expect(loginMock).not.toHaveBeenCalled()
    expect(wrapper.get('[role="alert"]').text()).toContain('auth.loginSettingsFailed')

    await wrapper.get('[role="alert"] button').trigger('click')
    await flushPromises()
    await wrapper.get('#email').setValue('admin@example.com')
    await wrapper.get('#password').setValue('password123')
    await wrapper.get('form').trigger('submit')
    expect(loginMock).toHaveBeenCalledOnce()
    consoleError.mockRestore()
  })

  it('requires a Turnstile token and submits after verification', async () => {
    getPublicSettingsMock.mockResolvedValueOnce({
      ...publicSettings,
      turnstile_enabled: true,
      turnstile_site_key: 'site-key'
    })
    const wrapper = mountLogin()
    await flushPromises()
    await wrapper.get('#email').setValue('admin@example.com')
    await wrapper.get('#password').setValue('password123')
    await wrapper.get('form').trigger('submit')
    expect(loginMock).not.toHaveBeenCalled()

    await wrapper.get('[data-test="verify-captcha"]').trigger('click')
    await wrapper.get('form').trigger('submit')
    expect(loginMock).toHaveBeenCalledOnce()
    expect(loginMock).toHaveBeenCalledWith(expect.objectContaining({ turnstile_token: 'turnstile-token' }))
  })

  it('refreshes settings and shows Turnstile after the server requires verification', async () => {
    loginMock.mockRejectedValueOnce({ reason: 'TURNSTILE_VERIFICATION_FAILED' })
    getPublicSettingsMock
      .mockResolvedValueOnce(publicSettings)
      .mockResolvedValueOnce({
        ...publicSettings,
        turnstile_enabled: true,
        turnstile_site_key: 'site-key'
      })
    const wrapper = mountLogin()
    await flushPromises()
    await wrapper.get('#email').setValue('admin@example.com')
    await wrapper.get('#password').setValue('password123')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(getPublicSettingsMock).toHaveBeenCalledTimes(2)
    expect(wrapper.find('[data-test="verify-captcha"]').exists()).toBe(true)
    expect(loginMock).toHaveBeenCalledOnce()

    await wrapper.get('form').trigger('submit')
    expect(loginMock).toHaveBeenCalledOnce()
    await wrapper.get('[data-test="verify-captcha"]').trigger('click')
    await wrapper.get('form').trigger('submit')
    expect(loginMock).toHaveBeenCalledTimes(2)
    expect(loginMock.mock.calls[1]![0]).toMatchObject({ turnstile_token: 'turnstile-token' })
  })
})
