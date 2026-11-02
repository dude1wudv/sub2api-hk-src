import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ResetPasswordView from '../ResetPasswordView.vue'

const { getPublicSettings, resetPassword, showError, route } = vi.hoisted(() => ({
  getPublicSettings: vi.fn(), resetPassword: vi.fn(), showError: vi.fn(),
  route: { query: { email: 'user@example.com', token: 'test-reset-token' } },
}))
vi.mock('@/api/auth', () => ({ getPublicSettings, resetPassword }))
vi.mock('@/stores', () => ({ useAppStore: () => ({ showError, showSuccess: vi.fn() }) }))
vi.mock('vue-router', () => ({ useRoute: () => route }))
vi.mock('vue-i18n', async importOriginal => ({ ...(await importOriginal<typeof import('vue-i18n')>()), useI18n: () => ({ t: (key: string) => key, locale: { value: 'zh-CN' } }) }))

function mountReset() {
  return mount(ResetPasswordView, { global: { stubs: {
    AuthLayout: { template: '<main><slot /></main>' }, Icon: true, RouterLink: true,
  } } })
}

describe('reset password capability and policy', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    delete window.__APP_CONFIG__
    getPublicSettings.mockResolvedValue({ password_reset_enabled: true, backend_mode_enabled: false })
    resetPassword.mockResolvedValue(undefined)
  })

  it('does not expose a submit form when password reset is disabled', async () => {
    getPublicSettings.mockResolvedValueOnce({ password_reset_enabled: false })
    const wrapper = mountReset()
    await flushPromises()
    expect(wrapper.find('form').exists()).toBe(false)
    expect(wrapper.text()).toContain('本站暂未开放密码重置')
    expect(resetPassword).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('rejects an invalid byte length and submits a valid Unicode password unchanged', async () => {
    const wrapper = mountReset()
    await flushPromises()
    await wrapper.get('#password').setValue('密'.repeat(25))
    await wrapper.get('#confirmPassword').setValue('密'.repeat(25))
    await wrapper.get('form').trigger('submit')
    expect(resetPassword).not.toHaveBeenCalled()
    expect(showError).toHaveBeenCalledWith(expect.stringContaining('72'))
    const password = '😀'.repeat(12)
    await wrapper.get('#password').setValue(password)
    await wrapper.get('#confirmPassword').setValue(password)
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(resetPassword).toHaveBeenCalledWith({ email: 'user@example.com', token: 'test-reset-token', new_password: password })
    expect(wrapper.text()).toContain('auth.passwordResetSuccess')
    wrapper.unmount()
  })
})
