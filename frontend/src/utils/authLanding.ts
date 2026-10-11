/** Keep an explicit local return path; choose the workspace only when none is valid. */
export function normalizeAuthRedirect(redirect: unknown): string | undefined {
  if (typeof redirect !== 'string' || !redirect.startsWith('/') || redirect.startsWith('//')) return undefined
  if (redirect.includes('\\') || redirect.includes('://') || [...redirect].some(char => char.charCodeAt(0) <= 32)) return undefined
  // These pages redirect authenticated visitors themselves; returning to them
  // would loop instead of honoring a usable destination.
  const pathname = redirect.split(/[?#]/, 1)[0].replace(/\/+$/, '')
  if (pathname === '/login' || pathname === '/register') return undefined
  return redirect
}

export function resolveAuthLanding(redirect: unknown, isAdmin: boolean): string {
  return normalizeAuthRedirect(redirect) ?? (isAdmin ? '/admin/dashboard' : '/dashboard')
}
