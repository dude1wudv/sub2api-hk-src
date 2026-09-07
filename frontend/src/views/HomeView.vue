<template>
  <!-- Custom Home Content: Full Page Mode -->
  <div v-if="hasHomeContent" class="min-h-screen">
    <!-- iframe mode -->
    <iframe
      v-if="isHomeContentUrl"
      :src="homeContent.trim()"
      class="h-screen w-full border-0"
      allowfullscreen
    ></iframe>
    <!-- HTML mode - SECURITY: homeContent is admin-only setting, XSS risk is acceptable -->
    <div v-else v-html="homeContent"></div>
  </div>

  <!-- Compact Home Page -->
  <div
    v-else-if="compactHomeEnabled"
    data-testid="compact-home"
    class="home-surface compact-surface flex min-h-screen flex-col bg-[rgb(var(--canvas))] text-[rgb(var(--ink))]"
  >
    <header class="border-b border-gray-200/80 bg-white/80 px-4 py-3.5 backdrop-blur-md sm:px-6 dark:border-dark-800 dark:bg-dark-900/80">
      <nav class="mx-auto flex max-w-5xl flex-wrap items-center justify-between gap-3 sm:gap-4">
        <div class="flex min-w-0 flex-1 items-center gap-3">
          <img
            :src="siteLogo || '/logo.svg'"
            alt="Logo"
            class="h-9 w-9 shrink-0 rounded-lg object-contain ring-1 ring-gray-200/80 dark:ring-dark-700/80"
          />
          <span class="min-w-0 truncate text-base font-semibold text-gray-950 dark:text-white">{{ siteName }}</span>
        </div>
        <div class="flex max-w-full shrink-0 flex-wrap items-center justify-end gap-2">
          <LocaleSwitcher />
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-900 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40 dark:text-dark-300 dark:hover:bg-dark-800 dark:hover:text-white"
            :title="t('home.viewDocs')"
          >
            <Icon name="book" size="md" />
          </a>
          <router-link
            v-if="showModelPlazaEntry"
            to="/model-plaza"
            class="flex h-10 shrink-0 items-center gap-1.5 rounded-lg px-2.5 text-sm font-medium text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-900 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40 dark:text-dark-300 dark:hover:bg-dark-800 dark:hover:text-white"
            :title="t('nav.modelPlaza')"
          >
            <Icon name="grid" size="md" />
            <span class="hidden sm:inline">{{ t('nav.modelPlaza') }}</span>
          </router-link>
          <AppearanceSwitcher />
          <router-link
            :to="isAuthenticated ? dashboardPath : '/login'"
            class="inline-flex min-h-10 shrink-0 items-center justify-center rounded-lg bg-primary-600 px-4 py-2 text-sm font-medium text-white shadow-sm transition-colors hover:bg-primary-700 active:bg-primary-800 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40 dark:bg-primary-500 dark:hover:bg-primary-600"
          >
            {{ isAuthenticated ? t('home.dashboard') : t('home.login') }}
          </router-link>
        </div>
      </nav>
    </header>

    <main class="flex min-w-0 flex-1 items-center justify-center px-4 py-16 sm:px-6">
      <div class="compact-hero relative min-w-0 max-w-2xl text-center">
        <img
          :src="siteLogo || '/logo.svg'"
          alt="Logo"
          class="mx-auto mb-5 h-10 w-10 rounded-md object-contain ring-1 ring-gray-200 dark:ring-dark-700"
        />
        <h1 class="[overflow-wrap:anywhere] text-2xl font-semibold tracking-tight text-gray-950 dark:text-white md:text-[28px]">{{ siteName }}</h1>
        <p class="mt-3 whitespace-pre-wrap [overflow-wrap:anywhere] text-sm text-gray-600 dark:text-dark-300">{{ siteSubtitle }}</p>
        <router-link
          :to="isAuthenticated ? dashboardPath : '/login'"
          class="mt-8 inline-flex min-h-11 items-center justify-center rounded-lg bg-primary-600 px-6 py-2.5 text-sm font-semibold text-white shadow-sm transition-colors hover:bg-primary-700 active:bg-primary-800 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40 dark:bg-primary-500 dark:hover:bg-primary-600"
        >
          {{ isAuthenticated ? t('home.goToDashboard') : t('home.login') }}
        </router-link>
      </div>
    </main>

    <footer class="min-w-0 border-t border-gray-200/80 px-4 py-5 text-center text-sm text-gray-500 [overflow-wrap:anywhere] sm:px-6 dark:border-dark-800 dark:text-dark-400">
      &copy; {{ currentYear }} {{ siteName }}
    </footer>
  </div>

  <!-- Default Home Page -->
  <div
    v-else
    class="home-surface relative flex min-h-screen flex-col overflow-hidden bg-[rgb(var(--canvas))] text-[rgb(var(--ink))]"
  >
    <!-- Header -->
    <header class="relative z-20 border-b border-gray-200/60 bg-white/70 px-4 py-3.5 backdrop-blur-md sm:px-6 dark:border-dark-800/80 dark:bg-dark-900/70">
      <nav class="mx-auto flex max-w-6xl flex-wrap items-center justify-between gap-4">
        <!-- Logo -->
        <div class="flex min-w-0 items-center gap-3">
          <div class="flex h-9 w-9 items-center justify-center overflow-hidden rounded-lg bg-white ring-1 ring-gray-200/80 dark:bg-dark-800 dark:ring-dark-700/80">
            <img :src="siteLogo || '/logo.svg'" alt="Logo" class="h-full w-full object-contain" />
          </div>
          <span class="max-w-48 truncate text-sm font-bold tracking-tight text-gray-950 dark:text-white">{{ siteName }}</span>
        </div>

        <!-- Nav Actions -->
        <div class="flex flex-wrap items-center gap-2 sm:gap-3">
          <!-- Language Switcher -->
          <LocaleSwitcher />

          <!-- Doc Link -->
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-900 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40 dark:text-dark-300 dark:hover:bg-dark-800 dark:hover:text-white"
            :title="t('home.viewDocs')"
          >
            <Icon name="book" size="md" />
          </a>

          <!-- Model Plaza Link -->
          <router-link
            v-if="showModelPlazaEntry"
            to="/model-plaza"
            class="inline-flex h-10 shrink-0 items-center gap-1.5 rounded-lg px-2.5 text-sm font-medium text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-900 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40 dark:text-dark-300 dark:hover:bg-dark-800 dark:hover:text-white"
            :title="t('nav.modelPlaza')"
          >
            <Icon name="grid" size="md" />
            <span class="hidden sm:inline">{{ t('nav.modelPlaza') }}</span>
          </router-link>

          <!-- Theme Toggle -->
          <AppearanceSwitcher />

          <!-- Login / Dashboard Button -->
          <router-link
            v-if="isAuthenticated"
            :to="dashboardPath"
            class="inline-flex min-h-10 shrink-0 items-center gap-2 rounded-lg bg-gray-100 px-3 py-1.5 text-xs font-medium text-gray-900 transition-colors hover:bg-gray-200 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40 dark:bg-dark-800 dark:text-white dark:hover:bg-dark-700"
          >
            <span
              class="flex h-5 w-5 items-center justify-center rounded-full bg-primary-600 text-[10px] font-semibold text-white"
            >
              {{ userInitial }}
            </span>
            <span class="font-medium">{{ t('home.dashboard') }}</span>
            <svg
              class="h-3.5 w-3.5 text-gray-400 dark:text-dark-400"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              stroke-width="2"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M4.5 19.5l15-15m0 0H8.25m11.25 0v11.25"
              />
            </svg>
          </router-link>
          <router-link
            v-else
            to="/login"
            class="inline-flex min-h-10 shrink-0 items-center justify-center rounded-lg bg-primary-600 px-4 py-2 text-sm font-medium text-white shadow-sm transition-colors hover:bg-primary-700 active:bg-primary-800 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40 dark:bg-primary-500 dark:hover:bg-primary-600"
          >
            {{ t('home.login') }}
          </router-link>
        </div>
      </nav>
    </header>

    <main class="relative flex-1 px-4 py-10 sm:px-6">
      <div class="mx-auto max-w-5xl">
        <section class="gateway-heading">
          <div class="mb-3 flex items-center gap-2 text-xs text-gray-500 dark:text-dark-400">
            <Icon name="server" size="sm" />
            <span>{{ t('home.features.unifiedGateway') }}</span>
          </div>
          <h1 class="text-2xl font-semibold tracking-tight [overflow-wrap:anywhere] sm:text-[28px]">{{ siteName }}</h1>
          <p class="mt-2 max-w-2xl whitespace-pre-wrap text-sm text-gray-600 [overflow-wrap:anywhere] dark:text-dark-300">{{ siteSubtitle }}</p>
          <div class="mt-5 flex flex-wrap items-center gap-2">
            <router-link :to="isAuthenticated ? dashboardPath : '/login'" class="btn btn-primary">
              {{ isAuthenticated ? t('home.goToDashboard') : t('home.login') }}
              <Icon name="arrowRight" size="sm" />
            </router-link>
            <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer" class="btn btn-secondary">
              <Icon name="book" size="sm" />{{ t('home.viewDocs') }}
            </a>
            <router-link v-if="showModelPlazaEntry" to="/model-plaza" class="btn btn-secondary">
              <Icon name="grid" size="sm" />{{ t('nav.modelPlaza') }}
            </router-link>
          </div>
        </section>

        <section class="card mt-8 overflow-hidden" :aria-label="t('console.access.accessTitle')">
          <div class="card-header">
            <h2 class="text-sm font-semibold">{{ t('console.access.accessTitle') }}</h2>
            <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ t('console.access.accessDescription') }}</p>
          </div>
          <div class="grid divide-y divide-gray-200 dark:divide-dark-700 md:grid-cols-3 md:divide-x md:divide-y-0">
            <div class="p-5">
              <Icon name="key" size="md" class="mb-3 text-gray-500 dark:text-dark-400" />
              <h3 class="text-sm font-medium">{{ t('console.access.keysTitle') }}</h3>
              <p class="mt-2 text-[13px] leading-relaxed text-gray-500 dark:text-dark-400">{{ t('console.access.keysDescription') }}</p>
            </div>
            <div class="p-5">
              <Icon name="chart" size="md" class="mb-3 text-gray-500 dark:text-dark-400" />
              <h3 class="text-sm font-medium">{{ t('console.access.usageTitle') }}</h3>
              <p class="mt-2 text-[13px] leading-relaxed text-gray-500 dark:text-dark-400">{{ t('console.access.usageDescription') }}</p>
            </div>
            <div class="p-5">
              <Icon name="shield" size="md" class="mb-3 text-gray-500 dark:text-dark-400" />
              <h3 class="text-sm font-medium">{{ t('console.access.accountTitle') }}</h3>
              <p class="mt-2 text-[13px] leading-relaxed text-gray-500 dark:text-dark-400">{{ t('console.access.accountDescription') }}</p>
            </div>
          </div>
        </section>
      </div>
    </main>

    <!-- Footer -->
    <footer class="relative z-10 border-t border-gray-200/80 px-4 py-8 sm:px-6 dark:border-dark-800">
      <div
        class="mx-auto flex max-w-6xl flex-col items-center justify-between gap-4 text-center sm:flex-row sm:text-left"
      >
        <p class="text-sm text-gray-500 dark:text-dark-400">
          &copy; {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}
        </p>
        <div class="flex items-center gap-4">
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="rounded text-sm text-gray-500 transition-colors hover:text-gray-900 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40 dark:text-dark-400 dark:hover:text-white"
          >
            {{ t('home.docs') }}
          </a>
          <a
            :href="githubUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="rounded text-sm text-gray-500 transition-colors hover:text-gray-900 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40 dark:text-dark-400 dark:hover:text-white"
          >
            GitHub
          </a>
        </div>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'
import { sanitizeUrl } from '@/utils/url'
import { FeatureFlags, isFeatureFlagEnabled } from '@/utils/featureFlags'
import AppearanceSwitcher from '@/components/common/AppearanceSwitcher.vue'

const { t } = useI18n()

const authStore = useAuthStore()
const appStore = useAppStore()

// Site settings - directly from appStore (already initialized from injected config)
const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || 'AI API Gateway Platform')
const docUrl = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.doc_url || appStore.docUrl || ''))
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')
const hasHomeContent = computed(() => homeContent.value.trim().length > 0)
const compactHomeEnabled = computed(() => appStore.cachedPublicSettings?.compact_home_enabled === true)
const modelPlazaEnabled = computed(() => isFeatureFlagEnabled(FeatureFlags.modelPlaza))

// Check if homeContent is a URL (for iframe display)
const isHomeContentUrl = computed(() => {
  const content = homeContent.value.trim()
  return content.startsWith('http://') || content.startsWith('https://')
})


// GitHub URL
const githubUrl = 'https://github.com/Wei-Shaw/sub2api'

// Auth state
const isAuthenticated = computed(() => authStore.isAuthenticated)
const modelPlazaRequiresAuth = computed(
  () => appStore.cachedPublicSettings?.model_plaza_require_auth === true,
)
const showModelPlazaEntry = computed(
  () => modelPlazaEnabled.value && (isAuthenticated.value || !modelPlazaRequiresAuth.value),
)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => isAdmin.value ? '/admin/dashboard' : '/dashboard')
const userInitial = computed(() => {
  const user = authStore.user
  if (!user || !user.email) return ''
  return user.email.charAt(0).toUpperCase()
})

// Current year for footer
const currentYear = computed(() => new Date().getFullYear())


onMounted(() => {

  // Check auth state
  authStore.checkAuth()

  // Ensure public settings are loaded (will use cache if already loaded from injected config)
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
})
</script>

<style scoped>
.gateway-heading { padding-block: 12px 8px; }
.compact-hero { width: 100%; max-width: 440px; padding: 28px; border: 1px solid rgb(var(--line)); border-radius: var(--radius-card); background: rgb(var(--surface)); }
.home-surface > header { background: rgb(var(--surface-header)); border-color: rgb(var(--line)); backdrop-filter: none; }
.home-surface > footer { border-color: rgb(var(--line)); }
@media (max-width: 639px) {
  .compact-hero { padding: 24px 20px; }
  .gateway-heading .btn { flex: 1 1 auto; }
}
</style>
