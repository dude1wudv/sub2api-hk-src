import { describe, expect, it } from 'vitest'
import { OAUTH_DEFAULT_LANDING, resolveOAuthLanding, resolveOAuthStartRedirect } from '../oauthLanding'

describe('OAuth landing contract', () => {
  it('carries automatic landing through an existing same-origin redirect field', () => {
    expect(resolveOAuthStartRedirect(undefined)).toBe(OAUTH_DEFAULT_LANDING)
    expect(resolveOAuthLanding(OAUTH_DEFAULT_LANDING, false)).toBe('/dashboard')
    expect(resolveOAuthLanding(OAUTH_DEFAULT_LANDING, true)).toBe('/admin/dashboard')
  })

  it.each(['/dashboard', '/profile', '/profile/connections?tab=oauth#providers', '/usage?model=claude%20sonnet'])('preserves the explicit target %s for either role', target => {
    expect(resolveOAuthStartRedirect(target)).toBe(target)
    expect(resolveOAuthLanding(target, true)).toBe(target)
    expect(resolveOAuthLanding(target, false)).toBe(target)
  })

  it.each([undefined, null, '', ['/', '/profile'], '//example.com', 'https://example.com', '/\\example.com', '/\nexample', '/ bad'])('falls back safely for invalid input %s', target => {
    expect(resolveOAuthStartRedirect(target)).toBe(OAUTH_DEFAULT_LANDING)
    expect(resolveOAuthLanding(target, true)).toBe('/admin/dashboard')
    expect(resolveOAuthLanding(target, false)).toBe('/dashboard')
  })

  it('preserves automatic intent for paths rejected by the backend byte limit', () => {
    expect(resolveOAuthStartRedirect('/' + 'a'.repeat(2047))).toHaveLength(2048)
    expect(resolveOAuthStartRedirect('/' + 'a'.repeat(2048))).toBe(OAUTH_DEFAULT_LANDING)
    expect(resolveOAuthStartRedirect('/' + '中'.repeat(683))).toBe(OAUTH_DEFAULT_LANDING)
  })
})
