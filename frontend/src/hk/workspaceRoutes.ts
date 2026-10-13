import type { RouteLocationNormalizedLoaded, RouteRecordRaw, Router } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { resetWorkspaceWarmup, scheduleWorkspaceWarmup } from '@/composables/useRoutePrefetch'

/** Routes that always render inside UserShell or AdminShell. */
const workspacePaths = new Set([
  '/dashboard',
  '/keys',
  '/batch-image',
  '/usage',
  '/redeem',
  '/affiliate',
  '/available-channels',
  '/profile',
  '/subscriptions',
  '/purchase',
  '/orders',
  '/payment/qrcode',
  '/payment/airwallex',
  '/custom/:id',
  '/admin/dashboard',
  '/admin/audit-logs',
  '/admin/users',
  '/admin/groups',
  '/admin/channels/pricing',
  '/admin/channels/monitor',
  '/monitor',
  '/admin/subscriptions',
  '/admin/accounts',
  '/admin/plugins',
  '/admin/announcements',
  '/admin/proxies',
  '/admin/redeem',
  '/admin/promo-codes',
  '/admin/settings',
  '/admin/risk-control',
  '/admin/prompt-audit',
  '/admin/usage',
  '/admin/affiliates/invites',
  '/admin/affiliates/rebates',
  '/admin/affiliates/transfers',
  '/admin/orders/dashboard',
  '/admin/orders',
  '/admin/orders/plans',
])

function embeddedPlaza(route: RouteLocationNormalizedLoaded) {
  return route.query.embedded === '1' && useAuthStore().isAuthenticated
}

function opsWorkspace(route: RouteLocationNormalizedLoaded) {
  return route.query.fullscreen !== '1' && route.query.fullscreen !== 'true'
}

function stripeWorkspace(route: RouteLocationNormalizedLoaded) {
  return !route.query.method
}

const conditionalLayout: Record<string, (route: RouteLocationNormalizedLoaded) => boolean> = {
  '/model-plaza': embeddedPlaza,
  '/admin/ops': opsWorkspace,
  '/payment/stripe': stripeWorkspace,
}

/** Card redemption stays available after self-serve gateway checkout is removed. */
const cardShopPaths = new Set(['/purchase', '/orders'])

const extraRoutes: RouteRecordRaw[] = [
  {
    path: '/auth/landing',
    name: 'AuthLanding',
    component: { render: () => null },
    beforeEnter: () => useAuthStore().isAdmin ? '/admin/dashboard' : '/dashboard',
    meta: { requiresAuth: true, title: 'Workspace' },
  },
  {
    path: '/image2',
    name: 'Image2',
    component: () => import('@/views/user/Image2RedirectView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Image Workbench',
      titleKey: 'nav.image2',
    },
  },
  {
    path: '/helios',
    name: 'HeliosLaunch',
    component: () => import('@/views/user/HeliosLaunchView.vue'),
    meta: {
      requiresAuth: true,
      requiresAdmin: false,
      title: 'Infinite Canvas',
      titleKey: 'nav.infiniteCanvas',
    },
  },
  {
    path: '/settings/profile',
    redirect: '/profile',
  },
]

export function applyHkWorkspaceRoutes(routes: RouteRecordRaw[]): RouteRecordRaw[] {
  for (const route of routes) {
    if (typeof route.path !== 'string') continue
    if (route.path === '/admin/groups') {
      route.component = () => import('@/views/admin/hk/GroupsView.vue')
    }
    if (workspacePaths.has(route.path)) {
      route.meta = { ...(route.meta ?? {}), appLayout: true }
    } else if (conditionalLayout[route.path]) {
      route.meta = { ...(route.meta ?? {}), appLayout: conditionalLayout[route.path] }
    }
    if (cardShopPaths.has(route.path) && route.meta && 'requiresPayment' in route.meta) {
      const meta = { ...route.meta }
      delete meta.requiresPayment
      route.meta = meta
    }
  }

  const catchAll = routes.findIndex(route => route.name === 'NotFound')
  if (catchAll >= 0) routes.splice(catchAll, 0, ...extraRoutes)
  else routes.push(...extraRoutes)
  return routes
}

export function installHkRouterHooks(router: Router) {
  router.afterEach((to) => {
    const authStore = useAuthStore()
    if (!authStore.isAuthenticated) {
      resetWorkspaceWarmup()
    } else if (to.meta.appLayout) {
      scheduleWorkspaceWarmup(router, authStore.isAdmin)
    }
  })
}
