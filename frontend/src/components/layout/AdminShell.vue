<template>
  <div
    class="admin-portal flex min-h-screen bg-[var(--atelier-canvas)] text-[var(--atelier-ink)]"
    :data-collapsed="collapsed"
    :data-overlay="overlayOpen"
    :data-mode="desktop ? 'desktop' : rail ? 'rail' : 'mobile'"
  >
    <a
      class="sr-only focus:not-sr-only focus:fixed focus:left-4 focus:top-4 focus:z-[120] focus:rounded-lg focus:bg-[var(--atelier-ink)] focus:px-3 focus:py-2 focus:text-xs focus:text-[var(--atelier-ink-inverse)]"
      href="#admin-content"
    >
      {{ zh ? '跳至主要内容' : 'Skip to content' }}
    </a>

    <button
      v-if="overlayOpen"
      type="button"
      tabindex="-1"
      class="fixed inset-0 z-40 bg-[var(--atelier-scrim)] backdrop-blur-[1px]"
      :aria-label="t('common.close')"
      @click="closeNavigation()"
      @wheel.prevent
      @touchmove.prevent
    />

    <!-- Unified 232px Warm-Ivory Sidebar (ZERO numeric count badges) -->
    <aside
      v-show="desktop || rail || navOpen"
      id="admin-navigation"
      ref="sidebar"
      class="admin-sidebar fixed inset-y-0 left-0 z-50 flex flex-col border-r border-[var(--atelier-border)] bg-[var(--atelier-canvas)] transition-[width,transform] duration-200"
      :class="[
        overlayOpen ? 'w-[248px] shadow-[var(--atelier-shadow-md)]' : collapsed ? 'w-[64px]' : 'w-[232px]'
      ]"
      :style="overlayOpen && viewport.width > 0 ? { top: `${viewport.top}px`, left: `${viewport.left}px`, bottom: 'auto', height: `${viewport.height}px`, maxHeight: `${viewport.height}px` } : undefined"
      :role="overlayOpen ? 'dialog' : undefined"
      :aria-modal="overlayOpen || undefined"
      :aria-label="zh ? '管理导航' : 'Administration navigation'"
      tabindex="-1"
      @keydown="onNavKeydown"
    >
      <!-- Brand Row (52px aligned with topbar) -->
      <div class="flex h-[52px] shrink-0 items-center justify-between gap-2 border-b border-[var(--atelier-border)] px-3.5">
        <RouterLink
          to="/admin/dashboard"
          class="flex min-w-0 items-center gap-2"
          aria-label="Admin dashboard"
          @click="closeNavigation()"
        >
          <PatrickBrand :compact="collapsed" />
        </RouterLink>
        <button
          v-if="overlayOpen"
          type="button"
          class="inline-flex h-8 w-8 items-center justify-center rounded-lg text-[var(--atelier-ink-secondary)] hover:bg-[var(--atelier-hover)] hover:text-[var(--atelier-ink)]"
          :aria-label="t('common.close')"
          @click="closeNavigation()"
        >
          <Icon name="x" size="sm" />
        </button>
      </div>

      <!-- Admin-Only Pure Segmented Switcher (User / Admin) -->
      <div v-if="!collapsed" class="px-3 pt-3">
        <div class="atelier-segmented grid w-full grid-cols-2" role="tablist" :aria-label="zh ? '工作区视角' : 'Workspace perspective'">
          <RouterLink
            to="/dashboard"
            class="atelier-segmented-item !px-2 !py-1 text-center text-xs"
            data-active="false"
          >
            {{ zh ? '用户视角' : 'User' }}
          </RouterLink>
          <span
            class="atelier-segmented-item !px-2 !py-1 text-center text-xs"
            data-active="true"
            aria-current="true"
          >
            {{ zh ? '管理后台' : 'Admin' }}
          </span>
        </div>
      </div>

      <!-- Quick Filter Search -->
      <div v-if="!collapsed" class="px-3 pt-2.5">
        <label class="relative block">
          <Icon name="search" size="xs" class="pointer-events-none absolute left-2.5 top-1/2 -translate-y-1/2 text-[var(--atelier-ink-muted)]" />
          <input
            v-model="query"
            class="h-8 w-full rounded-lg border border-[var(--atelier-border)] bg-[var(--atelier-surface)] pl-8 pr-2.5 text-xs text-[var(--atelier-ink)] placeholder:text-[var(--atelier-ink-muted)] focus:border-[var(--atelier-focus)] focus:outline-none"
            :placeholder="zh ? '筛选菜单…' : 'Filter menu…'"
            :aria-label="zh ? '搜索功能' : 'Search navigation'"
          />
        </label>
      </div>

      <!-- Navigation List (ZERO numeric badges, unified soft neutral active pill) -->
      <nav ref="navigation" class="flex-1 overflow-y-auto px-2.5 py-3" @scroll="rememberScroll">
        <section v-for="section in sections" :key="section.id" class="mb-4 last:mb-0">
          <h2
            v-if="!collapsed"
            class="mb-1 px-2.5 text-[11px] font-medium uppercase tracking-[0.06em] text-[var(--atelier-ink-muted)]"
          >
            {{ section.label }}
          </h2>
          <div class="space-y-0.5">
            <RouterLink
              v-for="item in section.items"
              :key="item.path"
              :to="item.path"
              class="admin-nav-item flex h-9 items-center gap-2.5 rounded-lg px-2.5 text-[13.5px] transition-colors"
              :class="[
                route.path === item.path
                  ? 'active bg-[var(--atelier-active)] font-semibold text-[var(--atelier-ink)]'
                  : 'font-normal text-[var(--atelier-ink-secondary)] hover:bg-[var(--atelier-hover)] hover:text-[var(--atelier-ink)]',
                collapsed ? 'justify-center px-0' : ''
              ]"
              :aria-label="item.label"
              :aria-current="route.path === item.path ? 'page' : undefined"
              :title="collapsed ? item.label : undefined"
              :data-nav-path="item.path"
              :id="item.path === '/admin/accounts' ? 'sidebar-channel-manage' : item.path === '/admin/groups' ? 'sidebar-group-manage' : undefined"
              @click="closeNavigation()"
              @pointerenter="prefetch(item.path)"
              @focus="prefetch(item.path)"
            >
              <span v-if="item.iconSvg" class="h-4 w-4 shrink-0" v-html="sanitizeSvg(item.iconSvg)" />
              <Icon v-else :name="item.icon" size="sm" class="shrink-0" />
              <span v-if="!collapsed" class="truncate">{{ item.label }}</span>
            </RouterLink>
          </div>
        </section>

        <p v-if="!sections.length" class="px-2.5 py-4 text-center text-xs text-[var(--atelier-ink-muted)]">
          {{ t('common.noData') }}
        </p>

        <section v-if="!auth.isSimpleMode && !query" class="mt-4 border-t border-[var(--atelier-border-subtle)] pt-3">
          <h2 v-if="!collapsed" class="mb-1 px-2.5 text-[11px] font-medium uppercase tracking-[0.06em] text-[var(--atelier-ink-muted)]">
            {{ zh ? '外部工具' : 'Tools' }}
          </h2>
          <div class="space-y-0.5">
            <a
              href="/image2/"
              class="flex h-9 items-center gap-2.5 rounded-lg px-2.5 text-[13.5px] text-[var(--atelier-ink-secondary)] transition-colors hover:bg-[var(--atelier-hover)] hover:text-[var(--atelier-ink)]"
              :class="collapsed ? 'justify-center px-0' : ''"
              :title="t('nav.image2')"
              :aria-label="t('nav.image2')"
            >
              <Icon name="sparkles" size="sm" class="shrink-0" />
              <span v-if="!collapsed" class="truncate">{{ t('nav.image2') }}</span>
            </a>
            <button
              type="button"
              class="flex h-9 w-full items-center gap-2.5 rounded-lg px-2.5 text-left text-[13.5px] text-[var(--atelier-ink-secondary)] transition-colors hover:bg-[var(--atelier-hover)] hover:text-[var(--atelier-ink)]"
              :class="collapsed ? 'justify-center px-0' : ''"
              :title="t('nav.infiniteCanvas')"
              :aria-label="t('nav.infiniteCanvas')"
              @click="launchHelios"
            >
              <Icon name="sparkles" size="sm" class="shrink-0" />
              <span v-if="!collapsed" class="truncate">{{ t('nav.infiniteCanvas') }}</span>
            </button>
            <RouterLink
              v-if="canUseBatchImage"
              to="/batch-image"
              class="flex h-9 items-center gap-2.5 rounded-lg px-2.5 text-[13.5px] text-[var(--atelier-ink-secondary)] transition-colors hover:bg-[var(--atelier-hover)] hover:text-[var(--atelier-ink)]"
              :class="collapsed ? 'justify-center px-0' : ''"
              :aria-label="t('nav.batchImage')"
              :title="t('nav.batchImage')"
            >
              <Icon name="sparkles" size="sm" class="shrink-0" />
              <span v-if="!collapsed" class="truncate">{{ t('nav.batchImage') }}</span>
            </RouterLink>
            <a
              v-if="docUrl"
              :href="docUrl"
              target="_blank"
              rel="noopener noreferrer"
              class="flex h-9 items-center gap-2.5 rounded-lg px-2.5 text-[13.5px] text-[var(--atelier-ink-secondary)] transition-colors hover:bg-[var(--atelier-hover)] hover:text-[var(--atelier-ink)]"
              :class="collapsed ? 'justify-center px-0' : ''"
              :title="zh ? '文档' : 'Documentation'"
            >
              <Icon name="book" size="sm" class="shrink-0" />
              <span v-if="!collapsed" class="truncate">{{ zh ? '文档' : 'Documentation' }}</span>
            </a>
          </div>
        </section>
      </nav>

      <!-- Bottom Fixed Account Strip -->
      <div class="shrink-0 border-t border-[var(--atelier-border)] p-2.5">
        <RouterLink
          v-if="collapsed"
          to="/dashboard"
          class="mb-1 flex h-9 items-center justify-center rounded-lg text-[var(--atelier-ink-secondary)] hover:bg-[var(--atelier-hover)] hover:text-[var(--atelier-ink)]"
          :aria-label="zh ? '用户工作区' : 'User workspace'"
          :title="zh ? '用户工作区' : 'User workspace'"
        >
          <Icon name="externalLink" size="sm" />
        </RouterLink>
        <RouterLink
          to="/profile"
          class="flex items-center gap-2.5 rounded-lg px-2 py-1.5 transition-colors hover:bg-[var(--atelier-hover)]"
          :class="collapsed ? 'justify-center px-0' : ''"
          :aria-label="t('nav.profile')"
        >
          <span class="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-full border border-[var(--atelier-border-strong)] bg-[var(--atelier-surface)] text-[11px] font-semibold text-[var(--atelier-ink)]">
            {{ initials }}
          </span>
          <div v-if="!collapsed" class="min-w-0 flex-1">
            <p class="truncate text-xs font-medium text-[var(--atelier-ink)]">{{ displayName }}</p>
            <p class="truncate text-[11px] text-[var(--atelier-ink-muted)]">{{ zh ? '管理员' : 'Administrator' }}</p>
          </div>
        </RouterLink>
      </div>
    </aside>

    <!-- Main Workspace Area -->
    <div
      class="admin-workspace flex min-w-0 flex-1 flex-col bg-[var(--atelier-surface)] transition-[padding-left] duration-200"
      :class="[
        desktop ? (collapsed ? 'pl-[64px]' : 'pl-[232px]') : rail ? 'pl-[64px]' : 'pl-0'
      ]"
      :inert="overlayOpen || undefined"
    >
      <!-- Unified Global 52px Topbar -->
      <header class="admin-topbar atelier-nav-surface sticky top-0 z-30 flex h-[52px] shrink-0 items-center gap-3 border-b border-[var(--atelier-border)] px-4 sm:px-6">
        <button
          ref="navToggle"
          type="button"
          class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-lg text-[var(--atelier-ink-secondary)] transition-colors hover:bg-[var(--atelier-hover)] hover:text-[var(--atelier-ink)]"
          :aria-label="zh ? '展开或收起导航' : 'Toggle navigation'"
          :aria-expanded="desktop ? !collapsed : overlayOpen"
          aria-controls="admin-navigation"
          @click="toggleNavigation"
        >
          <Icon name="menu" size="sm" />
        </button>
        <div class="mr-auto flex min-w-0 items-center gap-1.5 text-xs">
          <span class="hidden text-[var(--atelier-ink-muted)] sm:inline">{{ activeSection }}</span>
          <span class="hidden text-[var(--atelier-ink-muted)] sm:inline">/</span>
          <strong class="truncate font-semibold text-[var(--atelier-ink)]">{{ pageTitle }}</strong>
        </div>

        <div class="flex shrink-0 items-center gap-1.5">
          <WorkspaceCommandPalette :items="commands" />
          <button
            type="button"
            class="admin-mode-toggle inline-flex h-8 w-8 items-center justify-center rounded-lg text-[var(--atelier-ink-secondary)] transition-colors hover:bg-[var(--atelier-hover)] hover:text-[var(--atelier-ink)]"
            :aria-label="isDark ? (zh ? '切换至浅色' : 'Switch to light mode') : (zh ? '切换至深色' : 'Switch to dark mode')"
            :title="isDark ? (zh ? '切换至浅色' : 'Switch to light mode') : (zh ? '切换至深色' : 'Switch to dark mode')"
            :aria-pressed="isDark"
            @click="toggleMode"
          >
            <Icon :name="isDark ? 'sun' : 'moon'" size="sm" aria-hidden="true" />
          </button>
          <LocaleSwitcher />
          <AnnouncementBell />

          <div ref="accountMenu" class="relative" @keydown.esc.stop.prevent="closeAccount">
            <button
              ref="accountToggle"
              type="button"
              class="inline-flex h-8 w-8 items-center justify-center rounded-full border border-[var(--atelier-border-strong)] bg-[var(--atelier-canvas)] text-xs font-semibold text-[var(--atelier-ink)] transition-colors hover:bg-[var(--atelier-subtle)]"
              :aria-label="t('nav.profile')"
              :aria-expanded="accountOpen"
              aria-controls="admin-account-menu"
              @click="accountOpen = !accountOpen"
            >
              {{ initials }}
            </button>
            <div
              v-if="accountOpen"
              id="admin-account-menu"
              class="absolute right-0 top-full z-50 mt-2 w-56 rounded-xl border border-[var(--atelier-border-strong)] bg-[var(--atelier-surface)] p-1.5 shadow-[var(--atelier-shadow-md)]"
            >
              <div class="border-b border-[var(--atelier-border-subtle)] px-3 py-2">
                <p class="truncate text-xs font-semibold text-[var(--atelier-ink)]">{{ displayName }}</p>
                <p class="truncate text-[11px] text-[var(--atelier-ink-muted)]">{{ auth.user?.email }}</p>
              </div>
              <div class="px-1.5 py-1">
                <MotionPreferenceToggle show-label />
              </div>
              <RouterLink
                to="/profile"
                class="block rounded-lg px-3 py-2 text-xs text-[var(--atelier-ink-secondary)] hover:bg-[var(--atelier-hover)] hover:text-[var(--atelier-ink)]"
                @click="accountOpen = false"
              >
                {{ t('nav.profile') }}
              </RouterLink>
              <button
                v-if="!auth.isSimpleMode"
                type="button"
                class="block w-full rounded-lg px-3 py-2 text-left text-xs text-[var(--atelier-ink-secondary)] hover:bg-[var(--atelier-hover)] hover:text-[var(--atelier-ink)]"
                @click="replayTour(); accountOpen = false"
              >
                {{ zh ? '使用引导' : 'Product tour' }}
              </button>
              <button
                type="button"
                class="block w-full rounded-lg px-3 py-2 text-left text-xs text-[var(--atelier-err)] hover:bg-[var(--atelier-err-bg)]"
                @click="logout"
              >
                {{ t('nav.logout') }}
              </button>
            </div>
          </div>
        </div>
      </header>

      <main
        id="admin-content"
        class="mx-auto w-full max-w-[1440px] flex-1 px-4 py-6 sm:px-8"
        tabindex="-1"
      >
        <header v-if="route.path !== '/admin/ops'" class="mb-6">
          <h1 class="text-[22px] font-semibold tracking-[-0.02em] text-[var(--atelier-ink)]">
            {{ pageTitle }}
          </h1>
          <p v-if="pageDescription" class="mt-1 text-sm text-[var(--atelier-ink-secondary)]">
            {{ pageDescription }}
          </p>
        </header>
        <slot />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useMediaQuery, onClickOutside } from '@vueuse/core'
import { useAdminSettingsStore, useAppStore, useAuthStore, useOnboardingStore } from '@/stores'
import Icon from '@/components/icons/Icon.vue'
import PatrickBrand from '@/components/brand/PatrickBrand.vue'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import AnnouncementBell from '@/components/common/AnnouncementBell.vue'
import WorkspaceCommandPalette from '@/components/common/WorkspaceCommandPalette.vue'
import MotionPreferenceToggle from '@/components/common/MotionPreferenceToggle.vue'
import { usePatrickApiBase } from '@/composables/usePatrickApiBase'
import { useClipboard } from '@/composables/useClipboard'
import type { WorkspaceCommand } from '@/utils/workspaceCommands'
import { adminNavigation, visibleAdminNavigation, type AdminIcon } from '@/utils/adminNavigation'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'
import { sanitizeSvg } from '@/utils/sanitize'
import { sanitizeUrl } from '@/utils/url'
import { resolveRouteMetaKeys } from '@/router/title'
import { resolveSiteBillingMode } from '@/utils/siteBillingMode'
import { prefetchRoutePath } from '@/composables/useRoutePrefetch'
import { useColorMode } from '@/composables/useColorMode'
import { useScrollLock } from '@/composables/useScrollLock'
import { useVisualViewport } from '@/composables/useVisualViewport'
import { useBatchImageAccess } from '@/composables/useBatchImageAccess'
import { launchHeliosWorkbench } from '@/utils/heliosLaunch'
import { useOnboardingTour } from '@/composables/useOnboardingTour'
import '@/styles/onboarding.css'

const { t, locale } = useI18n()
const zh = computed(() => locale.value.startsWith('zh'))
const route = useRoute()
const router = useRouter()
const app = useAppStore()
const auth = useAuthStore()
const settings = useAdminSettingsStore()
const onboarding = useOnboardingStore()
const { replayTour } = useOnboardingTour({ storageKey: 'admin_guide', autoStart: true })
const { isDark, toggleMode } = useColorMode()
const desktop = useMediaQuery('(min-width: 1200px)')
const rail = useMediaQuery('(min-width: 768px)')
const desktopCollapsed = ref(false)
const navOpen = ref(false)
const query = ref('')
const accountOpen = ref(false)
try {
  desktopCollapsed.value = localStorage.getItem('patrick-admin-sidebar-collapsed') === 'true'
} catch {
  /* optional storage */
}
const collapsed = computed(() => (desktop.value ? desktopCollapsed.value : rail.value && !navOpen.value))
const overlayOpen = computed(() => !desktop.value && navOpen.value)
useScrollLock(overlayOpen)
const viewport = useVisualViewport(overlayOpen)
const sidebar = ref<HTMLElement>()
const navigation = ref<HTMLElement>()
const navToggle = ref<HTMLButtonElement>()
const accountMenu = ref<HTMLElement>()
const accountToggle = ref<HTMLButtonElement>()
const { canUseBatchImage, refreshBatchImageAccess } = useBatchImageAccess()
const docUrl = computed(() => sanitizeUrl(app.docUrl))
const displayName = computed(() => auth.user?.username || auth.user?.email?.split('@')[0] || 'Admin')
const initials = computed(() => displayName.value.slice(0, 2).toUpperCase())
onClickOutside(accountMenu, () => {
  accountOpen.value = false
})

const navigationSections = computed(() => {
  const groups = visibleAdminNavigation(auth.isSimpleMode, (flag) =>
    flag === 'ops'
      ? settings.opsMonitoringEnabled !== false
      : flag === 'adminPayment'
        ? settings.paymentEnabled !== false
        : isFeatureFlagEnabled(FeatureFlags[flag])
  ).map((group) => ({
    id: group.id,
    label: zh.value ? group.zh : group.en,
    items: group.items.map((item) => ({ path: item.path, label: t(item.key), icon: item.icon, iconSvg: '' }))
  }))
  const system = groups.find((group) => group.id === 'system')!
  if (auth.isSimpleMode) system.items.unshift({ path: '/keys', label: t('nav.apiKeys'), icon: 'key', iconSvg: '' })
  for (const item of [...settings.customMenuItems]
    .filter((item) => item.visibility === 'admin')
    .sort((a, b) => a.sort_order - b.sort_order)) {
    system.items.push({
      path: `/custom/${item.id}`,
      label: item.label,
      icon: 'document' as AdminIcon,
      iconSvg: item.icon_svg || ''
    })
  }
  return groups
})

const sections = computed(() => {
  const groups = navigationSections.value
  const q = query.value.trim().toLocaleLowerCase()
  return groups
    .map((group) => ({
      ...group,
      items: group.items.filter((item) => `${item.label} ${item.path}`.toLocaleLowerCase().includes(q))
    }))
    .filter((group) => group.items.length)
})

const { openAIBase } = usePatrickApiBase()
const { copyToClipboard } = useClipboard()
const commands = computed<WorkspaceCommand[]>(() => {
  const entries: WorkspaceCommand[] = navigationSections.value.flatMap((section) =>
    section.items.map((item) => ({
      id: item.path,
      label: item.label,
      section: section.label,
      keywords: item.path,
      to: item.path
    }))
  )
  entries.unshift({
    id: 'copy-endpoint',
    label: zh.value ? '复制接口地址' : 'Copy API base URL',
    section: zh.value ? '快捷操作' : 'Quick actions',
    keywords: 'api base url',
    run: async () => {
      if (!(await copyToClipboard(openAIBase.value))) throw new Error('Copy failed')
    }
  })
  if (entries.some((item) => item.id === '/keys')) {
    entries.unshift({
      id: 'create-key',
      label: zh.value ? '创建 API 密钥' : 'Create API key',
      section: zh.value ? '快捷操作' : 'Quick actions',
      to: '/keys?create=1'
    })
  }
  entries.push(
    {
      id: 'user-workspace',
      label: zh.value ? '用户工作区' : 'User workspace',
      section: zh.value ? '账户' : 'Account',
      to: '/dashboard'
    },
    { id: 'profile', label: t('nav.profile'), section: zh.value ? '账户' : 'Account', to: '/profile' }
  )
  if (!auth.isSimpleMode) {
    entries.push(
      { id: 'image2', label: t('nav.image2'), section: zh.value ? '外部工具' : 'Tools', href: '/image2/' },
      { id: 'helios', label: t('nav.infiniteCanvas'), section: zh.value ? '外部工具' : 'Tools', run: launchHelios }
    )
    if (canUseBatchImage.value) {
      entries.push({ id: 'batch-image', label: t('nav.batchImage'), section: zh.value ? '工具' : 'Tools', to: '/batch-image' })
    }
    if (docUrl.value) {
      entries.push({
        id: 'docs',
        label: zh.value ? '开发者文档' : 'Documentation',
        section: zh.value ? '资源' : 'Resources',
        href: docUrl.value
      })
    }
  }
  return entries
})

const activeSection = computed(() => {
  const group = adminNavigation.find((g) => g.items.some((i) => i.path === route.path))
  return group ? (zh.value ? group.zh : group.en) : zh.value ? '系统' : 'System'
})

const meta = computed(() => resolveRouteMetaKeys(route, { billingMode: resolveSiteBillingMode(app.cachedPublicSettings) }))
const pageTitle = computed(() => {
  if (route.path === '/admin/dashboard') return zh.value ? '运营概览' : 'Operations overview'
  if (route.name === 'CustomPage') {
    return (
      settings.customMenuItems.find((item) => item.id === String(route.params.id) || item.page_slug === String(route.params.id))
        ?.label || t('nav.settings')
    )
  }
  return meta.value.titleKey ? t(meta.value.titleKey) : String(route.meta.title || '')
})
const pageDescription = computed(() => (meta.value.descriptionKey ? t(meta.value.descriptionKey) : String(route.meta.description || '')))

function closeNavigation(restoreFocus = true) {
  const restore = overlayOpen.value
  navOpen.value = false
  if (restore && restoreFocus) void nextTick(() => navToggle.value?.focus())
}
function closeAccount() {
  accountOpen.value = false
  accountToggle.value?.focus()
}
function toggleNavigation() {
  if (desktop.value) {
    desktopCollapsed.value = !desktopCollapsed.value
    try {
      localStorage.setItem('patrick-admin-sidebar-collapsed', String(desktopCollapsed.value))
    } catch {
      /* optional storage */
    }
  } else {
    navOpen.value = !navOpen.value
    if (navOpen.value) void nextTick(() => sidebar.value?.querySelector<HTMLElement>('input,a,button')?.focus())
  }
}
function onNavKeydown(event: KeyboardEvent) {
  if (!overlayOpen.value) return
  if (event.key === 'Escape') {
    event.preventDefault()
    event.stopPropagation()
    closeNavigation()
  }
  if (event.key === 'Tab') {
    const items = [...(sidebar.value?.querySelectorAll<HTMLElement>('a[href],button:not(:disabled),input:not(:disabled)') ?? [])].filter(
      (item) => !item.closest('[hidden],[inert]') && getComputedStyle(item).display !== 'none' && getComputedStyle(item).visibility !== 'hidden'
    )
    if (!items.length) {
      event.preventDefault()
      sidebar.value?.focus()
      return
    }
    const first = items[0]
    const last = items[items.length - 1]
    if (event.shiftKey && (document.activeElement === first || !sidebar.value?.contains(document.activeElement))) {
      event.preventDefault()
      last.focus()
    } else if (!event.shiftKey && (document.activeElement === last || !sidebar.value?.contains(document.activeElement))) {
      event.preventDefault()
      first.focus()
    }
  }
}
function rememberScroll() {
  if (navigation.value) app.sidebarScrollTop = navigation.value.scrollTop
}
function prefetch(path: string) {
  prefetchRoutePath(router, path)
}
function launchHelios() {
  void launchHeliosWorkbench({
    mode: 'popup',
    notify: (failure) => app.showError(t(failure === 'popup-blocked' ? 'helios.popupBlocked' : 'helios.launchFailed'))
  })
}
async function logout() {
  accountOpen.value = false
  try {
    await auth.logout()
  } finally {
    await router.push('/login')
  }
}

watch(
  () => route.fullPath,
  () => {
    const focusContent = overlayOpen.value
    closeNavigation(false)
    accountOpen.value = false
    if (focusContent) void nextTick(() => document.getElementById('admin-content')?.focus())
  }
)
watch([desktop, rail], ([isDesktop, hasRail]) => {
  const focusedInSidebar = sidebar.value?.contains(document.activeElement)
  if (isDesktop) closeNavigation(false)
  if (focusedInSidebar && (isDesktop || (!hasRail && !navOpen.value))) void nextTick(() => navToggle.value?.focus())
})
onMounted(() => {
  void settings.fetch()
  void refreshBatchImageAccess()
  onboarding.setReplayCallback(replayTour)
  if (navigation.value) navigation.value.scrollTop = app.sidebarScrollTop
})
onBeforeUnmount(rememberScroll)
</script>
