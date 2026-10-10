<template>
  <div
    class="patrick-surface user-portal"
    data-workspace="user"
    :data-sidebar-collapsed="sidebarCollapsed"
    :data-nav-open="overlayOpen"
    :data-navigation-mode="desktop ? 'desktop' : railViewport ? 'rail' : 'mobile'"
  >
    <a class="user-skip-link" href="#user-workspace-content">{{ zh ? '跳至主要内容' : 'Skip to content' }}</a>

    <button
      v-if="overlayOpen"
      class="user-drawer-scrim"
      type="button"
      tabindex="-1"
      :aria-label="zh ? '关闭导航' : 'Close navigation'"
      @click="closeMobile()"
      @wheel.prevent
      @touchmove.prevent
    />

    <aside
      v-show="desktop || railViewport || mobileOpen"
      id="user-workspace-navigation"
      ref="sidebarRef"
      class="user-sidebar"
      :role="overlayOpen ? 'dialog' : undefined"
      :aria-modal="overlayOpen ? true : undefined"
      :aria-label="zh ? '用户导航' : 'User navigation'"
      tabindex="-1"
      @keydown="handleDrawerKeydown"
    >
      <div class="user-sidebar-brand">
        <RouterLink to="/home" class="user-brand-link" aria-label="patrickapi" @click="closeCurrentRoute('/home')">
          <PatrickBrand :compact="sidebarCollapsed" />
        </RouterLink>
        <button
          v-if="overlayOpen"
          ref="drawerCloseRef"
          class="user-icon-button user-drawer-close"
          type="button"
          :aria-label="zh ? '关闭导航' : 'Close navigation'"
          @click="closeMobile()"
        ><Icon name="x" size="md" aria-hidden="true" /></button>
      </div>

      <nav class="user-navigation" :aria-label="zh ? '主要导航' : 'Main navigation'">
        <div class="user-primary-navigation">
          <RouterLink
            v-for="item in primaryItems"
            :key="item.path"
            :to="{ path: item.path, query: item.query }"
            class="user-nav-item"
            :class="{ 'user-nav-active': isActive(item) }"
            :aria-current="isActive(item) ? 'page' : undefined"
            :aria-label="sidebarCollapsed ? item.label : undefined"
            :title="sidebarCollapsed ? item.label : undefined"
            :data-tour="item.path === '/keys' ? 'sidebar-my-keys' : undefined"
            :data-nav-path="item.path"
            @click="activateNavigation(item)"
            @pointerenter="prefetchItem(item, $event)"
            @focus="prefetchItem(item)"
          >
            <Icon :name="item.icon" size="md" aria-hidden="true" />
            <span v-if="!sidebarCollapsed" class="user-nav-label">{{ item.label }}</span>
          </RouterLink>
        </div>

        <div v-if="moreSections.length || docUrl" class="user-more-navigation">
          <button
            ref="moreToggleRef"
            type="button"
            class="user-nav-item user-more-toggle"
            :class="{ 'user-nav-active': moreActive && (!moreOpen || sidebarCollapsed) }"
            :aria-label="sidebarCollapsed ? (zh ? '更多' : 'More') : undefined"
            :title="sidebarCollapsed ? (zh ? '更多' : 'More') : undefined"
            :aria-expanded="moreOpen && !sidebarCollapsed"
            aria-controls="user-more-navigation"
            @click="toggleMore"
          >
            <Icon name="more" size="md" aria-hidden="true" />
            <span v-if="!sidebarCollapsed" class="user-nav-label">{{ zh ? '更多' : 'More' }}</span>
            <Icon v-if="!sidebarCollapsed" :name="moreOpen ? 'chevronUp' : 'chevronDown'" size="xs" class="user-nav-trailing" aria-hidden="true" />
          </button>

          <div v-if="moreOpen && !sidebarCollapsed" id="user-more-navigation" class="user-more-content">
            <section v-for="section in moreSections" :key="section.id" class="user-nav-section" :aria-label="section.label">
              <h2 class="user-nav-heading">{{ section.label }}</h2>
              <component
                :is="item.action ? 'button' : item.href ? 'a' : RouterLink"
                v-for="item in section.items"
                :key="item.path"
                v-bind="item.action ? { type: 'button' } : item.href ? { href: item.href } : { to: { path: item.path, query: item.query } }"
                class="user-nav-item"
                :class="{ 'user-nav-active': isActive(item) }"
                :aria-current="isActive(item) ? 'page' : undefined"
                :data-nav-path="item.path"
                @click="activateNavigation(item)"
                @pointerenter="prefetchItem(item, $event)"
                @focus="prefetchItem(item)"
              >
                <span v-if="item.iconSvg" class="user-custom-icon" aria-hidden="true" v-html="sanitizeSvg(item.iconSvg)" />
                <Icon v-else :name="item.icon" size="md" aria-hidden="true" />
                <span class="user-nav-label">{{ item.label }}</span>
                <Icon v-if="item.href || item.action" name="externalLink" size="xs" class="user-nav-trailing" aria-hidden="true" />
              </component>
            </section>
            <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer" class="user-nav-item user-doc-link" @click="closeMobile()">
              <Icon name="book" size="md" aria-hidden="true" />
              <span class="user-nav-label">{{ zh ? '文档' : 'Documentation' }}</span>
              <Icon name="externalLink" size="xs" class="user-nav-trailing" aria-hidden="true" />
            </a>
          </div>
        </div>

        <RouterLink
          v-if="!user"
          to="/home"
          class="user-nav-item"
          :aria-label="sidebarCollapsed ? (zh ? '返回首页' : 'Back to home') : undefined"
          :title="sidebarCollapsed ? (zh ? '返回首页' : 'Back to home') : undefined"
          @click="closeCurrentRoute('/home')"
        >
          <Icon name="home" size="md" aria-hidden="true" />
          <span v-if="!sidebarCollapsed">{{ zh ? '返回首页' : 'Back to home' }}</span>
        </RouterLink>
      </nav>

      <div class="user-sidebar-bottom">
        <RouterLink
          v-if="authStore.isAdmin"
          to="/admin/dashboard"
          class="user-nav-item user-admin-return"
          :title="sidebarCollapsed ? (zh ? '返回管理后台' : 'Back to administration') : undefined"
          :aria-label="sidebarCollapsed ? (zh ? '返回管理后台' : 'Back to administration') : undefined"
        >
          <Icon name="shield" size="md" aria-hidden="true" />
          <span v-if="!sidebarCollapsed" class="user-nav-label">{{ zh ? '管理后台' : 'Administration' }}</span>
        </RouterLink>

        <div class="user-sidebar-utilities">
          <div ref="localeRef" class="user-locale-control">
            <button
              ref="localeToggleRef"
              type="button"
              class="user-language-toggle"
              :aria-label="zh ? '切换语言' : 'Change language'"
              :aria-expanded="localeOpen"
              :disabled="localeSwitching"
              aria-controls="user-language-menu"
              aria-haspopup="menu"
              @click="toggleLocale"
            >
              <span>{{ zh ? '中' : 'EN' }}</span>
              <Icon name="chevronDown" size="xs" aria-hidden="true" />
            </button>
            <div v-if="localeOpen" id="user-language-menu" ref="localeMenuRef" class="user-language-menu" role="menu" :aria-label="zh ? '语言' : 'Language'" @keydown="handleLocaleKeydown">
              <button
                v-for="language in availableLocales"
                :key="language.code"
                type="button"
                role="menuitemradio"
                :aria-checked="language.code === currentLocaleCode"
                :disabled="localeSwitching"
                @click="changeLocale(language.code)"
              >
                <span>{{ language.name }}</span>
                <Icon v-if="language.code === currentLocaleCode" name="check" size="sm" aria-hidden="true" />
              </button>
            </div>
          </div>
          <button
            v-if="desktop"
            ref="desktopToggleRef"
            type="button"
            class="user-icon-button user-sidebar-toggle"
            :aria-label="sidebarCollapsed ? (zh ? '展开导航' : 'Expand navigation') : (zh ? '收起导航' : 'Collapse navigation')"
            :title="sidebarCollapsed ? (zh ? '展开导航' : 'Expand navigation') : (zh ? '收起导航' : 'Collapse navigation')"
            :aria-expanded="!sidebarCollapsed"
            aria-controls="user-workspace-navigation"
            @click="toggleNavigation"
          ><Icon :name="sidebarCollapsed ? 'chevronRight' : 'chevronLeft'" size="md" aria-hidden="true" /></button>
        </div>
      </div>
    </aside>

    <div class="user-workspace" :inert="overlayOpen ? true : undefined">
      <header class="user-topbar">
        <button
          v-if="!desktop"
          ref="navToggleRef"
          class="user-icon-button user-nav-toggle"
          type="button"
          aria-controls="user-workspace-navigation"
          :aria-expanded="overlayOpen"
          :aria-label="zh ? '打开导航' : 'Open navigation'"
          @click="toggleNavigation"
        ><Icon name="menu" size="md" aria-hidden="true" /></button>
        <h1 id="user-page-title" class="sr-only">{{ pageTitle }}</h1>

        <div class="user-topbar-actions">
          <RouterLink
            v-if="user && !authStore.isSimpleMode"
            to="/purchase"
            class="user-balance-pill"
            :aria-label="(zh ? '可用余额 ' : 'Available balance ') + formatMoney(availableBalance)"
          >
            <span class="user-balance-label">{{ zh ? '余额' : 'Balance' }}</span>
            <span>{{ formatMoney(availableBalance) }}</span>
          </RouterLink>
          <div v-if="user" class="user-announcement-control"><AnnouncementBell /></div>

          <div v-if="user" ref="accountRef" class="user-account-control">
            <button
              ref="accountToggleRef"
              type="button"
              class="user-account-trigger"
              :aria-label="displayName + ' — ' + (zh ? '账户菜单' : 'Account menu')"
              :aria-expanded="accountOpen"
              aria-controls="user-account-popover"
              @click="toggleAccount"
            >
              <span class="user-account-name">{{ displayName }}</span>
              <Icon name="chevronDown" size="xs" aria-hidden="true" />
            </button>
            <div v-if="accountOpen" id="user-account-popover" class="user-account-popover">
              <div class="user-account-summary"><strong>{{ displayName }}</strong><span>{{ user.email }}</span></div>
              <div v-if="frozenBalance > 0 && !authStore.isSimpleMode" class="user-account-balance">
                <span>{{ zh ? '冻结金额' : 'Frozen balance' }}</span><span>{{ formatMoney(frozenBalance) }}</span>
              </div>
              <RouterLink to="/profile" class="user-account-action" @click="accountOpen = false"><Icon name="user" size="md" aria-hidden="true" />{{ zh ? '个人资料' : 'Profile' }}</RouterLink>
              <button v-if="!authStore.isAdmin && !authStore.isSimpleMode" type="button" class="user-account-action" @click="replayUserGuide"><Icon name="questionCircle" size="md" aria-hidden="true" />{{ zh ? 'API 接入引导' : 'API quick start' }}</button>
              <p v-if="appStore.contactInfo" class="user-contact-info">{{ appStore.contactInfo }}</p>
              <button type="button" class="user-account-action user-signout" :disabled="loggingOut" @click="logout"><Icon name="arrowRight" size="md" aria-hidden="true" />{{ zh ? '退出登录' : 'Sign out' }}</button>
            </div>
          </div>
          <RouterLink v-else to="/login" class="user-signin-link">{{ t('home.login') }}</RouterLink>
        </div>
      </header>

      <main id="user-workspace-content" ref="contentRef" class="user-workspace-content" aria-labelledby="user-page-title" tabindex="-1">
        <slot />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, provide, ref, watch } from 'vue'
import { useMediaQuery } from '@vueuse/core'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import PatrickBrand from '@/components/brand/PatrickBrand.vue'
import Icon from '@/components/icons/Icon.vue'
import AnnouncementBell from '@/components/common/AnnouncementBell.vue'
import { availableLocales, setLocale } from '@/i18n'
import { useAppStore, useAuthStore, useOnboardingStore } from '@/stores'
import { useUserNavigation, type UserNavigationItem } from '@/composables/useUserNavigation'
import { tableDensityKey, useGlacierPreferences } from '@/composables/useGlacierPreferences'
import { prefetchRoutePath } from '@/composables/useRoutePrefetch'
import { resolveRouteMetaKeys } from '@/router/title'
import { resolveSiteBillingMode } from '@/utils/siteBillingMode'
import { launchHeliosWorkbench } from '@/utils/heliosLaunch'
import { sanitizeSvg } from '@/utils/sanitize'
import { sanitizeUrl } from '@/utils/url'
import '@/styles/user-shell.css'

const { t, locale } = useI18n()
const zh = computed(() => locale.value.startsWith('zh'))
const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const authStore = useAuthStore()
const onboardingStore = useOnboardingStore()
const { primaryItems, moreSections, refreshBatchImageAccess } = useUserNavigation()
const { effectiveDensity } = useGlacierPreferences()
provide(tableDensityKey, effectiveDensity)

// These controls are local to the user shell; administrator preferences are not written.
const desktop = useMediaQuery('(min-width: 1100px)')
const railViewport = useMediaQuery('(min-width: 740px)')
const collapsed = ref(false)
const mobileOpen = ref(false)
const overlayOpen = computed(() => mobileOpen.value && !desktop.value)
const sidebarCollapsed = computed(() => desktop.value ? collapsed.value : railViewport.value && !mobileOpen.value)
const moreOpen = ref(false)
const accountOpen = ref(false)
const localeOpen = ref(false)
const localeSwitching = ref(false)
const loggingOut = ref(false)
const sidebarRef = ref<HTMLElement | null>(null)
const drawerCloseRef = ref<HTMLButtonElement | null>(null)
const navToggleRef = ref<HTMLButtonElement | null>(null)
const desktopToggleRef = ref<HTMLButtonElement | null>(null)
const moreToggleRef = ref<HTMLButtonElement | null>(null)
const contentRef = ref<HTMLElement | null>(null)
const accountRef = ref<HTMLElement | null>(null)
const accountToggleRef = ref<HTMLButtonElement | null>(null)
const localeRef = ref<HTMLElement | null>(null)
const localeToggleRef = ref<HTMLButtonElement | null>(null)
const localeMenuRef = ref<HTMLElement | null>(null)

const user = computed(() => authStore.user)
const displayName = computed(() => user.value?.username || user.value?.email?.split('@')[0] || (zh.value ? '我的账户' : 'My account'))
const docUrl = computed(() => sanitizeUrl(appStore.docUrl))
const availableBalance = computed(() => Number(user.value?.balance || 0))
const frozenBalance = computed(() => Number(user.value?.frozen_balance || 0))
const currentLocaleCode = computed(() => zh.value ? 'zh' : 'en')
const moreActive = computed(() => moreSections.value.some(section => section.items.some(isActive)))
const pageTitle = computed(() => {
  if (route.path === '/keys' || route.path === '/dashboard') return 'API'
  if (route.name === 'CustomPage') {
    const item = appStore.cachedPublicSettings?.custom_menu_items?.find(menu => menu.id === route.params.id)
    if (item?.label) return item.label
  }
  const { titleKey } = resolveRouteMetaKeys(route, { billingMode: resolveSiteBillingMode(appStore.cachedPublicSettings) })
  return titleKey ? t(titleKey) : String(route.meta.title || 'patrickapi')
})

function formatMoney(value: number) {
  return '$' + (Number.isFinite(value) ? value : 0).toFixed(2)
}

function isActive(item: UserNavigationItem) {
  return (item.activePaths || [item.path]).some(path => route.path === path || route.path.startsWith(path + '/'))
}

function focusNavigationToggle() {
  const toggle = desktop.value ? desktopToggleRef.value : navToggleRef.value
  toggle?.focus()
}

function closeMobile(restoreFocus = true) {
  if (!mobileOpen.value) return
  mobileOpen.value = false
  closeLocale(false)
  if (restoreFocus) void nextTick(focusNavigationToggle)
}

function closeCurrentRoute(path: string) {
  if (route.path === path) closeMobile()
}

function openNavigation() {
  accountOpen.value = false
  closeLocale(false)
  mobileOpen.value = true
  void nextTick(() => (drawerCloseRef.value || sidebarRef.value)?.focus())
}

function toggleNavigation() {
  if (desktop.value) {
    closeLocale(false)
    collapsed.value = !collapsed.value
  } else if (mobileOpen.value) {
    closeMobile()
  } else {
    openNavigation()
  }
}

function toggleMore() {
  if (sidebarCollapsed.value) {
    moreOpen.value = true
    if (desktop.value) {
      collapsed.value = false
      void nextTick(() => moreToggleRef.value?.focus())
    } else {
      openNavigation()
    }
  } else {
    moreOpen.value = !moreOpen.value
  }
}

function handleDrawerKeydown(event: KeyboardEvent) {
  if (!overlayOpen.value) return
  if (event.key === 'Escape') {
    event.preventDefault()
    event.stopPropagation()
    closeMobile()
    return
  }
  if (event.key !== 'Tab') return
  const controls = Array.from(sidebarRef.value?.querySelectorAll<HTMLElement>('a[href], button:not(:disabled), [tabindex="0"]') || [])
  const first = controls[0]
  const last = controls[controls.length - 1]
  if (event.shiftKey && document.activeElement === first) {
    event.preventDefault()
    last?.focus()
  } else if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault()
    first?.focus()
  }
}

function activateNavigation(item: UserNavigationItem) {
  if (item.action === 'helios') {
    // Preserve the click gesture for the existing authenticated popup handoff.
    void launchHeliosWorkbench({
      mode: 'popup',
      notify: failure => appStore.showError(t(failure === 'popup-blocked' ? 'helios.popupBlocked' : 'helios.launchFailed')),
    })
    closeMobile()
  } else if (item.href) {
    closeMobile(false)
  } else if (router.resolve({ path: item.path, query: item.query }).fullPath === route.fullPath) {
    closeMobile()
  }
  if (item.path === '/keys' && onboardingStore.isCurrentStep('[data-tour="sidebar-my-keys"]')) {
    void onboardingStore.nextStep(500)
  }
}

function prefetchItem(item: UserNavigationItem, event?: PointerEvent) {
  if (item.action || item.href || event?.pointerType === 'touch') return
  prefetchRoutePath(router, item.path)
}

function toggleAccount() {
  closeLocale(false)
  accountOpen.value = !accountOpen.value
}

function closeLocale(restoreFocus = true) {
  if (!localeOpen.value) return
  localeOpen.value = false
  if (restoreFocus) void nextTick(() => localeToggleRef.value?.focus())
}

function toggleLocale() {
  if (localeOpen.value) {
    closeLocale()
    return
  }
  accountOpen.value = false
  localeOpen.value = true
  void nextTick(() => localeMenuRef.value?.querySelector<HTMLButtonElement>('[aria-checked="true"]')?.focus())
}

async function changeLocale(code: string) {
  if (localeSwitching.value) return
  if (code === currentLocaleCode.value) {
    closeLocale()
    return
  }
  localeSwitching.value = true
  try {
    await setLocale(code)
  } catch {
    appStore.showError(zh.value ? '语言切换失败，请重试' : 'Could not change language. Please try again.')
  } finally {
    localeSwitching.value = false
    closeLocale()
  }
}

function handleLocaleKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    event.preventDefault()
    event.stopPropagation()
    closeLocale()
    return
  }
  if (!['ArrowDown', 'ArrowUp', 'Home', 'End'].includes(event.key)) return
  const buttons = Array.from(localeMenuRef.value?.querySelectorAll<HTMLButtonElement>('button:not(:disabled)') || [])
  if (!buttons.length) return
  event.preventDefault()
  const current = buttons.indexOf(document.activeElement as HTMLButtonElement)
  const index = event.key === 'Home' ? 0 : event.key === 'End' ? buttons.length - 1
    : (current + (event.key === 'ArrowDown' ? 1 : -1) + buttons.length) % buttons.length
  buttons[index]?.focus()
}

async function replayUserGuide() {
  accountOpen.value = false
  if (authStore.isAdmin || authStore.isSimpleMode) return
  closeMobile()
  await router.push({ path: '/keys', hash: '#api-quick-start' })
  await nextTick()
  document.getElementById('api-quick-start')?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

async function logout() {
  if (loggingOut.value) return
  loggingOut.value = true
  accountOpen.value = false
  try {
    await authStore.logout()
  } catch {
    // The auth store clears the local session even if server revocation fails.
  } finally {
    loggingOut.value = false
    await router.push('/login')
  }
}

function handlePointerDown(event: PointerEvent) {
  if (!(event.target instanceof Node)) return
  if (accountOpen.value && !accountRef.value?.contains(event.target)) accountOpen.value = false
  if (localeOpen.value && !localeRef.value?.contains(event.target)) closeLocale(false)
}

function handleWindowKeydown(event: KeyboardEvent) {
  if (event.key !== 'Escape') return
  if (localeOpen.value) {
    closeLocale()
  } else if (accountOpen.value) {
    accountOpen.value = false
    accountToggleRef.value?.focus()
  } else if (mobileOpen.value) {
    closeMobile()
  }
}

watch(() => route.fullPath, () => {
  const focusContent = overlayOpen.value
  closeMobile(false)
  closeLocale(false)
  accountOpen.value = false
  if (moreActive.value) moreOpen.value = true
  if (focusContent) void nextTick(() => contentRef.value?.focus())
}, { immediate: true })

watch(moreActive, active => {
  if (active) moreOpen.value = true
})

watch([desktop, railViewport], ([isDesktop, hasRail]) => {
  const focusedInSidebar = sidebarRef.value?.contains(document.activeElement)
  closeLocale(!!localeMenuRef.value?.contains(document.activeElement))
  if (isDesktop && mobileOpen.value) {
    closeMobile(false)
    if (focusedInSidebar) void nextTick(focusNavigationToggle)
  } else if (!isDesktop && !hasRail && !mobileOpen.value && focusedInSidebar) {
    void nextTick(focusNavigationToggle)
  }
})
watch(() => authStore.user?.id, () => { void refreshBatchImageAccess(true) })

onMounted(() => {
  void refreshBatchImageAccess()
  if (!authStore.isAdmin) onboardingStore.setReplayCallback(() => { void replayUserGuide() })
  document.addEventListener('pointerdown', handlePointerDown)
  window.addEventListener('keydown', handleWindowKeydown)
})
onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', handlePointerDown)
  window.removeEventListener('keydown', handleWindowKeydown)
})

defineExpose({ replayTour: replayUserGuide })
</script>
