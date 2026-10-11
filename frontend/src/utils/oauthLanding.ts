import { normalizeAuthRedirect, resolveAuthLanding } from './authLanding'

// A real same-origin route preserves automatic vs explicit landing through the
// existing OAuth redirect field, including pending registration and reloads.
export const OAUTH_DEFAULT_LANDING = '/auth/landing'

export function resolveOAuthStartRedirect(redirect: unknown): string {
  const path = normalizeAuthRedirect(redirect)
  // Match the existing backend redirect limit (UTF-8 bytes) before the OAuth
  // round trip; otherwise its fallback would erase automatic landing intent.
  return path && new TextEncoder().encode(path).byteLength <= 2048 ? path : OAUTH_DEFAULT_LANDING
}

/** Call after setToken has loaded the authenticated user's role. */
export function resolveOAuthLanding(redirect: unknown, isAdmin: boolean): string {
  const path = normalizeAuthRedirect(redirect)
  return resolveAuthLanding(path === OAUTH_DEFAULT_LANDING ? undefined : path, isAdmin)
}
