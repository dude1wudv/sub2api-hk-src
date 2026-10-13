import { computed, onBeforeUnmount, watch, type ComputedRef, type InjectionKey } from 'vue'
import { useRoute, type RouteLocationNormalizedLoaded } from 'vue-router'
import { useAdminSettingsStore, useAppStore, useAuthStore } from '@/stores'
import { adminSurfaceActive } from '@/composables/adminSurface'
import { useDelegatedPointerSurfaces } from '@/composables/useDelegatedPointerSurfaces'
import { isAdminWorkspaceRoute } from '@/utils/workspaceSurface'

export interface WorkspaceChrome {
  adminCustomPending: ComputedRef<boolean>
  userSurface: ComputedRef<boolean>
  brandName: ComputedRef<string>
  usesAppShell: (target: RouteLocationNormalizedLoaded) => boolean
  usesAdminShell: (target: RouteLocationNormalizedLoaded) => boolean
}

export const workspaceChromeKey: InjectionKey<WorkspaceChrome> = Symbol('workspaceChrome')

export function useWorkspaceChrome(): WorkspaceChrome {
  const route = useRoute()
  const appStore = useAppStore()
  const authStore = useAuthStore()
  const adminSettingsStore = useAdminSettingsStore()
  useDelegatedPointerSurfaces(() => route.fullPath)

  function usesAppShell(target: RouteLocationNormalizedLoaded): boolean {
    const appLayout = target.meta.appLayout
    return typeof appLayout === 'function' ? appLayout(target) : appLayout === true
  }

  function usesAdminShell(target: RouteLocationNormalizedLoaded): boolean {
    return isAdminWorkspaceRoute(target, authStore.isAdmin, adminSettingsStore.customMenuItems)
  }

  // A direct administrator custom-page visit must not depend on the shell
  // mounting before its route ownership can be resolved.
  watch([() => authStore.isAdmin, () => route.path], ([isAdmin, path]) => {
    if (isAdmin && path.startsWith('/custom/')) void adminSettingsStore.fetch()
  }, { immediate: true })

  const adminCustomPending = computed(() => authStore.isAdmin
    && route.path.startsWith('/custom/')
    && !adminSettingsStore.loaded
    && adminSettingsStore.customMenuItems.length === 0)
  const userSurface = computed(() => route.path !== '/setup' && !adminCustomPending.value && !usesAdminShell(route))
  const brandName = computed(() => userSurface.value ? 'patrickapi' : appStore.siteName)

  watch(() => usesAdminShell(route), (active) => {
    adminSurfaceActive.value = active
    if (active) document.body.dataset.adminSurface = 'true'
    else delete document.body.dataset.adminSurface
  }, { immediate: true })
  watch(userSurface, (active) => {
    if (active) document.body.dataset.patrickSurface = 'true'
    else delete document.body.dataset.patrickSurface
  }, { immediate: true })
  onBeforeUnmount(() => {
    adminSurfaceActive.value = false
    delete document.body.dataset.adminSurface
    delete document.body.dataset.patrickSurface
  })

  return { adminCustomPending, userSurface, brandName, usesAppShell, usesAdminShell }
}
