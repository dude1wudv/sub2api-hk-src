import { describe, expect, it } from 'vitest'
import {
  formatPortalBalance,
  maskPortalRedeemCode,
  PATRICK_CARD_ORDER_URL,
  PATRICK_CARD_SHOP_URL,
} from '../portalPurchase'

describe('portal purchase helpers', () => {
  it('uses the public patrickapi card-shop routes without adding account data', () => {
    expect(PATRICK_CARD_SHOP_URL).toBe('https://catfk.com/shop/UQZYWSZC')
    expect(PATRICK_CARD_ORDER_URL).toBe('https://catfk.com/order')
    for (const url of [PATRICK_CARD_SHOP_URL, PATRICK_CARD_ORDER_URL]) {
      expect(url).not.toMatch(/email|token|code|user_id/i)
    }
  })

  it('masks short and long redemption codes and handles empty values', () => {
    expect(maskPortalRedeemCode('  abc  ')).toBe('•••• ••••')
    expect(maskPortalRedeemCode('FULL-REDEEM-SECRET')).toBe('•••• •••• CRET')
    expect(maskPortalRedeemCode('   ')).toBe('—')
    expect(maskPortalRedeemCode(undefined)).toBe('—')
  })

  it('formats balances by locale while rejecting invalid numeric values', () => {
    expect(formatPortalBalance(12.5, 'en-US')).toBe('$12.50')
    expect(formatPortalBalance(12.5, 'zh-CN')).toContain('12.50')
    expect(formatPortalBalance(Number.NaN, 'en-US')).toBe('—')
  })
})
