import { describe, expect, it } from 'vitest'

import { isAdminWorkspaceRoute } from '../workspaceSurface'

describe('workspace surface ownership', () => {
  const adminMenu = [{ id: 'ops-home', page_slug: 'operations', visibility: 'admin' }] as never

  it('assigns every admin route to the admin surface regardless of account role', () => {
    expect(isAdminWorkspaceRoute({ path: '/admin/dashboard', params: {}, meta: {} }, false)).toBe(true)
    expect(isAdminWorkspaceRoute({ path: '/admin/users', params: {}, meta: {} }, true)).toBe(true)
  })

  it('assigns only administrator-owned custom pages to admins', () => {
    expect(isAdminWorkspaceRoute({ path: '/custom/ops-home', params: { id: 'ops-home' }, meta: {} }, true, adminMenu)).toBe(true)
    expect(isAdminWorkspaceRoute({ path: '/custom/operations', params: { id: 'operations' }, meta: {} }, true, adminMenu)).toBe(true)
    expect(isAdminWorkspaceRoute({ path: '/custom/ops-home', params: { id: 'ops-home' }, meta: {} }, false, adminMenu)).toBe(false)
    expect(isAdminWorkspaceRoute({ path: '/custom/user-page', params: { id: 'user-page' }, meta: {} }, true, adminMenu)).toBe(false)
  })

  it('leaves personal and public paths on the Patrick surface', () => {
    for (const path of ['/home', '/login', '/dashboard', '/keys', '/custom/user-page']) {
      expect(isAdminWorkspaceRoute({ path, params: { id: 'user-page' }, meta: {} }, true, adminMenu)).toBe(false)
    }
  })
})
