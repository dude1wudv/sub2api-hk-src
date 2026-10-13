import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, mount } from '@vue/test-utils'
import PaymentQRCodeView from '@/views/user/PaymentQRCodeView.vue'

vi.mock('vue-i18n', async (importOriginal) => ({
  ...await importOriginal<typeof import('vue-i18n')>(),
  useI18n: () => ({ t: (key: string) => key }),
}))
vi.mock('@/stores/payment', () => ({
  usePaymentStore: () => ({ pollOrderStatus: vi.fn().mockResolvedValue(null) }),
}))
vi.mock('@/stores', () => ({ useAppStore: () => ({ showError: vi.fn() }) }))
vi.mock('@/api/payment', () => ({ paymentAPI: {} }))
vi.mock('qrcode', () => ({ default: { toCanvas: vi.fn() } }))
vi.mock('vue-router', () => ({
  useRoute: () => ({ query: { order_id: '42', expires_at: new Date(Date.now() + 120_000).toISOString() } }),
  useRouter: () => ({ push: vi.fn() }),
}))

enableAutoUnmount(afterEach)
beforeEach(() => {
  vi.useFakeTimers()
  vi.setSystemTime(new Date('2026-09-30T12:00:00Z'))
})
afterEach(() => vi.useRealTimers())

describe('payment countdown: page', () => {
  it('catches up after timers were suspended', async () => {
    const wrapper = mount(PaymentQRCodeView)
    await vi.advanceTimersByTimeAsync(0)
    expect(wrapper.text()).toContain('02:00')
    vi.setSystemTime(Date.now() + 65_000)
    await vi.advanceTimersByTimeAsync(1000)
    expect(wrapper.text()).toContain('00:54')
  })

  it('expires and stops polling on the first overdue tick', async () => {
    const wrapper = mount(PaymentQRCodeView)
    vi.setSystemTime(Date.now() + 121_000)
    await vi.advanceTimersByTimeAsync(1000)
    expect(wrapper.text()).toContain('payment.qr.expired')
    expect(vi.getTimerCount()).toBe(0)
  })

  it('still counts down normally', async () => {
    const wrapper = mount(PaymentQRCodeView)
    await vi.advanceTimersByTimeAsync(1000)
    expect(wrapper.text()).toContain('01:59')
  })
})
