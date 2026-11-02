/** Public merchant destinations. Never append account information or credentials. */
export const PATRICK_CARD_SHOP_URL = 'https://catfk.com/shop/UQZYWSZC'
export const PATRICK_CARD_ORDER_URL = 'https://catfk.com/order'

/** History may identify a redemption by its suffix, but never reveals the usable code. */
export function maskPortalRedeemCode(code: string | null | undefined): string {
  const value = code?.trim() || ''
  if (!value) return '—'
  return value.length > 8 ? `•••• •••• ${value.slice(-4)}` : '•••• ••••'
}

export function formatPortalBalance(value: number, locale: string): string {
  if (!Number.isFinite(value)) return '—'
  const digits = value !== 0 && Math.abs(value) < 0.01 ? 6 : 2
  if (value !== 0 && Math.abs(value) < 10 ** -digits) return `${value < 0 ? '−' : ''}< USD ${(10 ** -digits).toFixed(digits)}`
  return `USD ${new Intl.NumberFormat(locale.startsWith('zh') ? 'zh-CN' : 'en-US', { minimumFractionDigits: digits, maximumFractionDigits: digits }).format(value)}`
}
