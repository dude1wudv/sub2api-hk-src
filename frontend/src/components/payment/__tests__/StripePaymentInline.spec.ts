import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import { useColorMode } from '@/composables/useColorMode'
import StripePaymentInline from '../StripePaymentInline.vue'

enableAutoUnmount(afterEach)
const { elements, stripe, paymentElement, loadStripe } = vi.hoisted(() => {
  const paymentElement = { mount: vi.fn(), on: vi.fn() }
  const elements = { create: vi.fn(() => paymentElement), update: vi.fn() }
  const stripe = { elements: vi.fn(() => elements) }
  return { elements, stripe, paymentElement, loadStripe: vi.fn(async () => stripe) }
})
vi.mock('@stripe/stripe-js/pure', () => ({ loadStripe }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('vue-router', () => ({ useRouter: () => ({ resolve: vi.fn() }) }))
vi.mock('@/stores', () => ({ useAppStore: () => ({ showError: vi.fn() }) }))
vi.mock('@/api/payment', () => ({ paymentAPI: {} }))

beforeEach(() => {
  vi.clearAllMocks()
  useColorMode().setMode('light')
  paymentElement.on.mockImplementation((event: string, callback: () => void) => { if (event === 'ready') callback() })
})

it('keeps the inline payment mounted while applying color-mode changes to Elements', async () => {
  const wrapper = mount(StripePaymentInline, {
    props: { orderId: 42, amount: 10, payAmount: 10, clientSecret: 'preview-secret', publishableKey: 'preview-key' },
    global: { stubs: { Icon: true } },
  })
  await flushPromises()
  expect(stripe.elements).toHaveBeenCalledWith(expect.objectContaining({ appearance: expect.objectContaining({ theme: 'stripe' }) }))
  useColorMode().setMode('dark')
  await flushPromises()
  expect(elements.update).toHaveBeenCalledWith({ appearance: expect.objectContaining({ theme: 'night' }) })
  expect(paymentElement.mount).toHaveBeenCalledTimes(1)
  wrapper.unmount()
  useColorMode().setMode('light')
  await flushPromises()
  expect(elements.update).toHaveBeenCalledTimes(1)
})
