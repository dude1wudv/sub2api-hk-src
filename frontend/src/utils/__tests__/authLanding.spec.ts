import { describe, expect, it } from 'vitest'
import { resolveAuthLanding } from '../authLanding'

describe('authentication landing', () => {
  it('lands regular users on the overview and administrators on operations', () => {
    expect(resolveAuthLanding(undefined, false)).toBe('/dashboard')
    expect(resolveAuthLanding(undefined, true)).toBe('/admin/dashboard')
  })
  it('gives explicit local redirects priority for either role', () => {
    expect(resolveAuthLanding('/keys?create=1', false)).toBe('/keys?create=1')
    expect(resolveAuthLanding('/dashboard', true)).toBe('/dashboard')
    expect(resolveAuthLanding('/usage?model=claude%20sonnet', true)).toBe('/usage?model=claude%20sonnet')
  })
  it.each(['//example.com', 'https://example.com', '/\\example.com', '/\nexample', '/login', '/login/?redirect=/login', '/register#form', ['/', '//example.com'], null])('rejects an invalid redirect %s', value => {
    expect(resolveAuthLanding(value, true)).toBe('/admin/dashboard')
  })
})
