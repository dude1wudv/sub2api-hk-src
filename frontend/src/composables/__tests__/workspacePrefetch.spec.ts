import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createMemoryHistory, createRouter, type Router } from 'vue-router'

import { prefetchRoutePath, resetWorkspaceWarmup, scheduleWorkspaceWarmup } from '../useRoutePrefetch'

const flush = async () => {
  for (let i = 0; i < 10; i++) {
    await new Promise((resolve) => setTimeout(resolve, 0))
  }
}

function createTestRouter() {
  const importers = {
    adminGroups: vi.fn().mockResolvedValue({ default: {} }),
    adminUsers: vi.fn().mockResolvedValue({ default: {} }),
    userKeys: vi.fn().mockResolvedValue({ default: {} }),
    login: vi.fn().mockResolvedValue({ default: {} }),
  }
  const router: Router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/admin/groups', component: importers.adminGroups, meta: { appLayout: true, requiresAdmin: true } },
      { path: '/admin/users', component: importers.adminUsers, meta: { appLayout: true, requiresAdmin: true } },
      { path: '/keys', component: importers.userKeys, meta: { appLayout: true } },
      { path: '/login', component: importers.login, meta: { requiresAuth: false } },
    ],
  })
  return { router, importers }
}

describe('workspace prefetch', () => {
  beforeEach(() => {
    resetWorkspaceWarmup()
    vi.stubGlobal('requestIdleCallback', (cb: IdleRequestCallback) =>
      setTimeout(() => cb({ didTimeout: false, timeRemaining: () => 50 }), 0)
    )
    vi.stubGlobal('cancelIdleCallback', (id: number) => clearTimeout(id))
  })

  afterEach(() => {
    resetWorkspaceWarmup()
    vi.unstubAllGlobals()
  })

  it('prefetchRoutePath 立即加载目标页面且只加载一次', async () => {
    const { router, importers } = createTestRouter()

    prefetchRoutePath(router, '/admin/groups')
    prefetchRoutePath(router, '/admin/groups')
    await flush()

    expect(importers.adminGroups).toHaveBeenCalledTimes(1)
    expect(importers.adminUsers).not.toHaveBeenCalled()
  })

  it('prefetchRoutePath 忽略外部链接和未知路径', () => {
    const { router, importers } = createTestRouter()

    expect(() => prefetchRoutePath(router, 'https://example.com/admin/groups')).not.toThrow()
    expect(() => prefetchRoutePath(router, '/not-exist')).not.toThrow()
    expect(Object.values(importers).every((fn) => fn.mock.calls.length === 0)).toBe(true)
  })

  it('管理员预热全部工作区页面，但不加载非工作区页面', async () => {
    const { router, importers } = createTestRouter()

    scheduleWorkspaceWarmup(router, true)
    await flush()

    expect(importers.adminGroups).toHaveBeenCalledTimes(1)
    expect(importers.adminUsers).toHaveBeenCalledTimes(1)
    expect(importers.userKeys).toHaveBeenCalledTimes(1)
    expect(importers.login).not.toHaveBeenCalled()
  })

  it('普通用户不预热管理员页面', async () => {
    const { router, importers } = createTestRouter()

    scheduleWorkspaceWarmup(router, false)
    await flush()

    expect(importers.userKeys).toHaveBeenCalledTimes(1)
    expect(importers.adminGroups).not.toHaveBeenCalled()
    expect(importers.adminUsers).not.toHaveBeenCalled()
  })

  it('省流量模式下跳过空闲预热', async () => {
    vi.stubGlobal('navigator', { ...navigator, connection: { saveData: true } })
    const { router, importers } = createTestRouter()

    scheduleWorkspaceWarmup(router, true)
    await flush()

    expect(Object.values(importers).every((fn) => fn.mock.calls.length === 0)).toBe(true)
  })
})
