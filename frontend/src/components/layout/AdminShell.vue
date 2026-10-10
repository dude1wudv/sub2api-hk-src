<template>
  <div class="admin-portal" :data-collapsed="collapsed" :data-overlay="overlayOpen" :data-mode="desktop ? 'desktop' : rail ? 'rail' : 'mobile'">
    <a class="admin-skip" href="#admin-content">{{ zh ? '跳至主要内容' : 'Skip to content' }}</a>
    <button v-if="overlayOpen" class="admin-nav-scrim" :aria-label="t('common.close')" @click="closeNavigation" @wheel.prevent @touchmove.prevent />
    <aside v-show="desktop || rail || navOpen" id="admin-navigation" ref="sidebar" class="admin-sidebar" :role="overlayOpen ? 'dialog' : undefined" :aria-modal="overlayOpen || undefined" :aria-label="zh ? '管理导航' : 'Administration navigation'" @keydown="onNavKeydown">
      <RouterLink to="/admin/dashboard" class="admin-brand" aria-label="patrickapi" @click="closeNavigation"><PatrickBrand :compact="collapsed" /><small v-if="!collapsed">{{ zh ? '管理后台' : 'Administration' }}</small></RouterLink>
      <button v-if="overlayOpen" class="admin-icon-button admin-nav-close" :aria-label="t('common.close')" @click="closeNavigation"><Icon name="x" size="md" /></button>
      <label v-if="!collapsed" class="admin-menu-search"><Icon name="search" size="sm" /><input v-model="query" :placeholder="zh ? '搜索功能…' : 'Search navigation…'" :aria-label="zh ? '搜索功能' : 'Search navigation'" /></label>
      <nav ref="navigation" class="admin-navigation" @scroll="rememberScroll">
        <section v-for="section in sections" :key="section.id" class="admin-nav-section">
          <h2 v-if="!collapsed">{{ section.label }}</h2>
          <RouterLink v-for="item in section.items" :key="item.path" :to="item.path" class="admin-nav-item" :class="{ active: route.path === item.path }" :aria-label="item.label" :aria-current="route.path === item.path ? 'page' : undefined" :title="collapsed ? item.label : undefined" :data-nav-path="item.path" :id="item.path === '/admin/accounts' ? 'sidebar-channel-manage' : item.path === '/admin/groups' ? 'sidebar-group-manage' : undefined" @click="closeNavigation" @pointerenter="prefetch(item.path)" @focus="prefetch(item.path)">
            <span v-if="item.iconSvg" class="admin-custom-icon" v-html="sanitizeSvg(item.iconSvg)" /><Icon v-else :name="item.icon" size="md" /><span v-if="!collapsed">{{ item.label }}</span>
          </RouterLink>
        </section>
        <p v-if="!sections.length" class="admin-nav-empty">{{ t('common.noData') }}</p>
        <section v-if="!auth.isSimpleMode && !query" class="admin-nav-section">
          <h2 v-if="!collapsed">{{ zh ? '外部工具' : 'Tools' }}</h2>
          <a href="/image2/" class="admin-nav-item" :title="t('nav.image2')" :aria-label="t('nav.image2')"><Icon name="sparkles" size="md" /><span v-if="!collapsed">{{ t('nav.image2') }}</span></a>
          <button class="admin-nav-item" :title="t('nav.infiniteCanvas')" :aria-label="t('nav.infiniteCanvas')" @click="launchHelios"><Icon name="sparkles" size="md" /><span v-if="!collapsed">{{ t('nav.infiniteCanvas') }}</span></button>
          <RouterLink v-if="canUseBatchImage" to="/batch-image" class="admin-nav-item" :aria-label="t('nav.batchImage')" :title="t('nav.batchImage')"><Icon name="sparkles" size="md" /><span v-if="!collapsed">{{ t('nav.batchImage') }}</span></RouterLink>
          <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer" class="admin-nav-item" :title="zh ? '文档' : 'Documentation'"><Icon name="book" size="md" /><span v-if="!collapsed">{{ zh ? '文档' : 'Documentation' }}</span></a>
        </section>
      </nav>
      <div class="admin-sidebar-bottom">
        <RouterLink to="/dashboard" class="admin-nav-item" :aria-label="zh ? '用户工作区' : 'User workspace'" :title="zh ? '用户工作区' : 'User workspace'"><Icon name="externalLink" size="md" /><span v-if="!collapsed">{{ zh ? '用户工作区' : 'User workspace' }}</span></RouterLink>
        <RouterLink to="/profile" class="admin-nav-item admin-sidebar-account" :aria-label="t('nav.profile')"><span class="admin-avatar">{{ initials }}</span><span v-if="!collapsed">{{ displayName }}<small>{{ zh ? '管理员' : 'Administrator' }}</small></span></RouterLink>
      </div>
    </aside>
    <div class="admin-workspace" :inert="overlayOpen || undefined">
      <header class="admin-topbar">
        <button ref="navToggle" class="admin-icon-button" :aria-label="zh ? '展开或收起导航' : 'Toggle navigation'" :aria-expanded="desktop ? !collapsed : overlayOpen" aria-controls="admin-navigation" @click="toggleNavigation"><Icon name="menu" size="md" /></button>
        <div class="admin-breadcrumb"><span>{{ activeSection }}</span><span>/</span><strong>{{ pageTitle }}</strong></div>
        <div class="admin-topbar-actions"><LocaleSwitcher /><AnnouncementBell /><div ref="accountMenu" class="admin-account-control"><button class="admin-avatar" :aria-label="t('nav.profile')" :aria-expanded="accountOpen" @click="accountOpen = !accountOpen">{{ initials }}</button><div v-if="accountOpen" class="admin-account-menu"><p>{{ displayName }}</p><RouterLink to="/profile" @click="accountOpen = false">{{ t('nav.profile') }}</RouterLink><button v-if="!auth.isSimpleMode" @click="replayTour(); accountOpen = false">{{ zh ? '使用引导' : 'Product tour' }}</button><button @click="logout">{{ t('nav.logout') }}</button></div></div></div>
      </header>
      <main id="admin-content" class="admin-content" tabindex="-1">
        <header v-if="route.path !== '/admin/ops'" class="admin-page-heading"><h1>{{ pageTitle }}</h1><p v-if="pageDescription">{{ pageDescription }}</p></header>
        <slot />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, provide, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useMediaQuery, onClickOutside } from '@vueuse/core'
import { useAdminSettingsStore, useAppStore, useAuthStore, useOnboardingStore } from '@/stores'
import Icon from '@/components/icons/Icon.vue'
import PatrickBrand from '@/components/brand/PatrickBrand.vue'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import AnnouncementBell from '@/components/common/AnnouncementBell.vue'
import { adminNavigation, visibleAdminNavigation, type AdminIcon } from '@/utils/adminNavigation'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'
import { sanitizeSvg } from '@/utils/sanitize'
import { sanitizeUrl } from '@/utils/url'
import { resolveRouteMetaKeys } from '@/router/title'
import { resolveSiteBillingMode } from '@/utils/siteBillingMode'
import { prefetchRoutePath } from '@/composables/useRoutePrefetch'
import { tableDensityKey } from '@/composables/useGlacierPreferences'
import { useBatchImageAccess } from '@/composables/useBatchImageAccess'
import { launchHeliosWorkbench } from '@/utils/heliosLaunch'
import { useOnboardingTour } from '@/composables/useOnboardingTour'
import '@/styles/admin-portal.css'
import '@/styles/onboarding.css'

const { t, locale } = useI18n()
const zh = computed(() => locale.value.startsWith('zh'))
const route = useRoute(), router = useRouter()
const app = useAppStore(), auth = useAuthStore(), settings = useAdminSettingsStore()
const onboarding = useOnboardingStore()
const { replayTour } = useOnboardingTour({ storageKey: 'admin_guide', autoStart: true })
provide(tableDensityKey, computed(() => 'compact' as const))
const desktop = useMediaQuery('(min-width: 1200px)'), rail = useMediaQuery('(min-width: 768px)')
const desktopCollapsed = ref(false), navOpen = ref(false), query = ref(''), accountOpen = ref(false)
try { desktopCollapsed.value = localStorage.getItem('patrick-admin-sidebar-collapsed') === 'true' } catch { /* session preference */ }
const collapsed = computed(() => desktop.value ? desktopCollapsed.value : rail.value && !navOpen.value)
const overlayOpen = computed(() => !desktop.value && navOpen.value)
const sidebar = ref<HTMLElement>(), navigation = ref<HTMLElement>(), navToggle = ref<HTMLButtonElement>(), accountMenu = ref<HTMLElement>()
const { canUseBatchImage, refreshBatchImageAccess } = useBatchImageAccess()
const docUrl = computed(() => sanitizeUrl(app.docUrl))
const displayName = computed(() => auth.user?.username || auth.user?.email?.split('@')[0] || 'Admin')
const initials = computed(() => displayName.value.slice(0, 2).toUpperCase())
onClickOutside(accountMenu, () => { accountOpen.value = false })
const sections = computed(() => {
  const groups = visibleAdminNavigation(auth.isSimpleMode, flag => flag === 'ops' ? settings.opsMonitoringEnabled !== false : flag === 'adminPayment' ? settings.paymentEnabled !== false : isFeatureFlagEnabled(FeatureFlags[flag]))
    .map(group => ({ id: group.id, label: zh.value ? group.zh : group.en, items: group.items.map(item => ({ path: item.path, label: t(item.key), icon: item.icon, iconSvg: '' })) }))
  const system = groups.find(group => group.id === 'system')!
  if (auth.isSimpleMode) system.items.unshift({ path: '/keys', label: t('nav.apiKeys'), icon: 'key', iconSvg: '' })
  for (const item of [...settings.customMenuItems].filter(item => item.visibility === 'admin').sort((a,b) => a.sort_order - b.sort_order)) {
    system.items.push({ path: `/custom/${item.id}`, label: item.label, icon: 'document' as AdminIcon, iconSvg: item.icon_svg || '' })
  }
  const q = query.value.trim().toLocaleLowerCase()
  return groups.map(group => ({ ...group, items: group.items.filter(item => `${item.label} ${item.path}`.toLocaleLowerCase().includes(q)) })).filter(group => group.items.length)
})
const activeSection = computed(() => {
  const group = adminNavigation.find(g => g.items.some(i => i.path === route.path))
  return group ? zh.value ? group.zh : group.en : zh.value ? '系统' : 'System'
})
const meta = computed(() => resolveRouteMetaKeys(route, { billingMode: resolveSiteBillingMode(app.cachedPublicSettings) }))
const pageTitle = computed(() => {
  if (route.path === '/admin/dashboard') return zh.value ? '运营概览' : 'Operations overview'
  if (route.name === 'CustomPage') return settings.customMenuItems.find(item => item.id === String(route.params.id) || item.page_slug === String(route.params.id))?.label || t('nav.settings')
  return meta.value.titleKey ? t(meta.value.titleKey) : String(route.meta.title || '')
})
const pageDescription = computed(() => meta.value.descriptionKey ? t(meta.value.descriptionKey) : String(route.meta.description || ''))
function closeNavigation() { const restore = overlayOpen.value; navOpen.value = false; if (restore) void nextTick(() => navToggle.value?.focus()) }
function toggleNavigation() {
  if (desktop.value) { desktopCollapsed.value = !desktopCollapsed.value; try { localStorage.setItem('patrick-admin-sidebar-collapsed', String(desktopCollapsed.value)) } catch { /* optional storage */ } }
  else { navOpen.value = !navOpen.value; if (navOpen.value) void nextTick(() => sidebar.value?.querySelector<HTMLElement>('input,a,button')?.focus()) }
}
function onNavKeydown(event: KeyboardEvent) {
  if (!overlayOpen.value) return
  if (event.key === 'Escape') { event.preventDefault(); closeNavigation() }
  if (event.key === 'Tab') {
    const items = sidebar.value?.querySelectorAll<HTMLElement>('a[href],button,input')
    if (!items?.length) return
    const first = items[0], last = items[items.length - 1]
    if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last.focus() }
    else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first.focus() }
  }
}
function rememberScroll() { if (navigation.value) app.sidebarScrollTop = navigation.value.scrollTop }
function prefetch(path: string) { prefetchRoutePath(router, path) }
function launchHelios() { void launchHeliosWorkbench({ mode: 'popup', notify: failure => app.showError(t(failure === 'popup-blocked' ? 'helios.popupBlocked' : 'helios.launchFailed')) }) }
async function logout() { accountOpen.value = false; try { await auth.logout() } finally { await router.push('/login') } }
watch(() => route.fullPath, () => { closeNavigation(); accountOpen.value = false })
watch(desktop, () => { navOpen.value = false })
watch(overlayOpen, open => document.body.classList.toggle('admin-navigation-open', open))
onMounted(() => { void settings.fetch(); void refreshBatchImageAccess(); onboarding.setReplayCallback(replayTour); if (navigation.value) navigation.value.scrollTop = app.sidebarScrollTop })
onBeforeUnmount(() => { rememberScroll(); document.body.classList.remove('admin-navigation-open') })
</script>
