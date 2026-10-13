import { describe, expect, it } from 'vitest'
import router from '@/router'

/** Paths the backend or an email/OAuth client may open. Each must render something other than NotFound. */
const backendLinkedPaths = [
  '/dashboard',
  '/keys',
  '/usage',
  '/profile',
  '/settings/profile',
  '/purchase',
  '/redeem',
  '/subscriptions',
  '/orders',
  '/payment/result',
  '/reset-password',
  '/auth/callback',
  '/auth/landing',
  '/model-plaza',
]

describe('backend linked frontend paths', () => {
  it.each(backendLinkedPaths)('%s resolves to a real route', (path) => {
    const resolved = router.resolve(path)
    expect(resolved.name).not.toBe('NotFound')
    expect(resolved.matched.some(record => record.name === 'NotFound')).toBe(false)
  })

  it('sends the legacy identity settings path to the portal account page', () => {
    const resolved = router.resolve('/settings/profile')
    const redirect = resolved.matched.find(record => record.path === '/settings/profile')?.redirect
    expect(redirect).toBe('/profile')
  })
})
