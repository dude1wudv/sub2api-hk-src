import type { CustomMenuItem } from '@/types'

type SurfaceRoute = { path: string; params: Record<string, unknown>; meta: Record<string, unknown> }

/** Route ownership, never the account role alone, determines the visual shell. */
export function isAdminWorkspaceRoute(route: SurfaceRoute, isAdmin: boolean, menus: CustomMenuItem[] = []): boolean {
  if (route.path === '/admin' || route.path.startsWith('/admin/')) return true
  if (!isAdmin || !route.path.startsWith('/custom/')) return false
  const id = String(route.params.id ?? '')
  return menus.some(item => item.visibility === 'admin' && (item.id === id || item.page_slug === id))
}
