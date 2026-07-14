import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, post } = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get,
    post,
  },
}))

import { paymentAPI } from '@/api/payment'

describe('payment api', () => {
  beforeEach(() => {
    get.mockReset()
    post.mockReset()
    get.mockResolvedValue({ data: {} })
    post.mockResolvedValue({ data: {} })
  })

  it('purchases a subscription plan with an idempotency key', async () => {
    await paymentAPI.purchaseSubscriptionWithBalance(7, 'balance-subscription-key')

    expect(post).toHaveBeenCalledWith('/payment/plans/7/purchase-with-balance', {}, {
      headers: { 'Idempotency-Key': 'balance-subscription-key' },
    })
  })

  it('creates an external subscription order with an idempotency key', async () => {
    const payload = { amount: 5, payment_type: 'alipay', order_type: 'subscription', plan_id: 8 }
    await paymentAPI.createOrder(payload, 'external-subscription-key')

    expect(post).toHaveBeenCalledWith('/payment/orders', payload, {
      headers: { 'Idempotency-Key': 'external-subscription-key' },
    })
  })

  it('keeps legacy public out_trade_no verification for upgrade compatibility', async () => {
    await paymentAPI.verifyOrderPublic('legacy-order-no')

    expect(post).toHaveBeenCalledWith('/payment/public/orders/verify', {
      out_trade_no: 'legacy-order-no',
    })
  })

  it('keeps signed public resume-token resolve endpoint', async () => {
    await paymentAPI.resolveOrderPublicByResumeToken('resume-token-123')

    expect(post).toHaveBeenCalledWith('/payment/public/orders/resolve', {
      resume_token: 'resume-token-123',
    })
  })
})
