import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import PortalAccountSecurity from '../PortalAccountSecurity.vue'

const { getStatus, list } = vi.hoisted(() => ({ getStatus: vi.fn(), list: vi.fn() }))
vi.mock('@/api', () => ({ totpAPI: { getStatus }, passkeyAPI: { isSupported: () => false, list } }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ locale: { value: 'zh-CN' } }) }))

describe('account security failure states', () => {
  beforeEach(() => { vi.clearAllMocks() })
  afterEach(() => { vi.useRealTimers() })

  it('localizes a status rate limit, waits before retrying and then recovers', async () => {
    vi.useFakeTimers()
    getStatus.mockRejectedValueOnce({ status: 429, message: 'RATE_LIMITED', metadata: { retry_after_seconds: 2 } })
    getStatus.mockResolvedValue({ enabled: false, feature_enabled: true })
    const wrapper = mount(PortalAccountSecurity, { props: { passkeyEnabled: false } })
    await flushPromises()
    expect(getStatus).toHaveBeenCalledOnce()
    expect(wrapper.text()).toContain('操作过于频繁')
    expect(wrapper.text()).not.toContain('RATE_LIMITED')
    expect(wrapper.get('button').attributes('disabled')).toBeDefined()
    await vi.advanceTimersByTimeAsync(2100)
    await wrapper.get('button').trigger('click')
    await flushPromises()
    expect(getStatus).toHaveBeenCalledTimes(2)
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
    wrapper.unmount()
  })
})
