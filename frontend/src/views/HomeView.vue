<template>
  <!-- Custom Home Content: Full Page Mode -->
  <div v-if="homeContent" class="min-h-screen">
    <iframe
      v-if="isHomeContentUrl"
      :src="homeContent.trim()"
      class="h-screen w-full border-0"
      allowfullscreen
    ></iframe>
    <!-- HTML mode - SECURITY: homeContent is admin-only setting, XSS risk is acceptable -->
    <div v-else v-html="homeContent"></div>
  </div>

  <!-- Default Home Page -->
  <div v-else class="home-cyber relative min-h-screen overflow-hidden bg-[#020403] text-white">
    <CyberBackground />

    <header class="relative z-20 px-5 py-5 sm:px-8">
      <nav class="mx-auto flex max-w-7xl items-center justify-between gap-5">
        <router-link to="/home" class="flex min-w-0 items-center gap-3">
          <span class="brand-terminal">>_</span>
          <span class="truncate font-mono text-xl font-black uppercase tracking-normal text-white">
            {{ siteName }}
          </span>
        </router-link>

        <div class="flex shrink-0 items-center gap-2 sm:gap-3">
          <LocaleSwitcher />

          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="nav-icon"
            :title="t('home.viewDocs')"
          >
            <Icon name="book" size="md" />
          </a>

          <button
            class="nav-icon"
            :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
            @click="toggleTheme"
          >
            <Icon v-if="isDark" name="sun" size="md" />
            <Icon v-else name="moon" size="md" />
          </button>

          <router-link
            :to="isAuthenticated ? dashboardPath : '/login'"
            class="home-cta"
          >
            <span v-if="isAuthenticated" class="user-dot">{{ userInitial }}</span>
            <span>{{ isAuthenticated ? t('home.dashboard') : t('home.login') }}</span>
            <Icon name="arrowRight" size="sm" :stroke-width="2" />
          </router-link>
        </div>
      </nav>
    </header>

    <main class="relative z-10 px-5 pb-12 sm:px-8">
      <section class="mx-auto grid min-h-[calc(100vh-84px)] max-w-7xl items-center gap-10 py-10 lg:grid-cols-[1.05fr_0.95fr] lg:gap-14">
        <div class="hero-copy">
          <h1 class="hero-title">
            <span>SUB2</span>
            <span class="text-[#2cff43]">API</span>
            <span>GATEWAY</span>
          </h1>

          <p class="mt-7 max-w-2xl text-xl font-medium leading-relaxed text-zinc-400 sm:text-2xl">
            {{ siteSubtitle || 'Enterprise-grade API infrastructure. Speed. Security. Scale.' }}
          </p>

          <div class="mt-9 flex flex-col gap-3 sm:flex-row">
            <router-link :to="isAuthenticated ? dashboardPath : '/login'" class="primary-action">
              {{ isAuthenticated ? t('home.goToDashboard') : t('home.getStarted') }}
              <Icon name="arrowRight" size="md" :stroke-width="2" />
            </router-link>
            <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer" class="secondary-action">
              {{ t('home.docs') }}
              <Icon name="externalLink" size="sm" />
            </a>
          </div>

          <div class="command-line mt-10">
            <span class="text-[#2cff43]">$</span>
            <span class="min-w-0 flex-1 truncate">curl https://sub.sunmmyapi.xyz/v1/messages</span>
            <Icon name="copy" size="sm" class="text-[#2cff43]" />
          </div>

          <div class="mt-10 grid max-w-2xl gap-3 sm:grid-cols-3">
            <div v-for="item in trustItems" :key="item.label" class="status-tile">
              <Icon :name="item.icon" size="sm" />
              <span>{{ item.label }}</span>
            </div>
          </div>
        </div>

        <div class="hero-panel">
          <div class="panel-header">
            <span class="h-2 w-2 rounded-full bg-[#2cff43] shadow-[0_0_14px_rgba(44,255,67,0.8)]"></span>
            <span>API CONTROL PLANE</span>
            <span class="ml-auto text-[#2cff43]">200 OK</span>
          </div>

          <div class="terminal-preview">
            <div class="terminal-line">
              <span class="terminal-key">POST</span>
              <span>/v1/chat/completions</span>
            </div>
            <div class="terminal-line text-zinc-500">
              <span>Authorization:</span>
              <span>Bearer sk-************</span>
            </div>
            <div class="terminal-line text-zinc-500">
              <span>Group:</span>
              <span>production-default</span>
            </div>
            <div class="response-box">
              <div class="flex items-center justify-between text-xs uppercase text-zinc-500">
                <span>stream response</span>
                <span class="text-[#2cff43]">live</span>
              </div>
              <pre>{
  "model": "gpt-5.5",
  "route": "sticky",
  "latency_ms": 287
}</pre>
            </div>
          </div>

          <div class="metric-grid">
            <div>
              <span>RPM</span>
              <strong>12.8k</strong>
            </div>
            <div>
              <span>Uptime</span>
              <strong>99.99%</strong>
            </div>
            <div>
              <span>Keys</span>
              <strong>active</strong>
            </div>
          </div>
        </div>
      </section>

      <section class="mx-auto max-w-7xl border-t border-[#2cff43]/18 py-12">
        <div class="grid gap-8 lg:grid-cols-[0.9fr_1.1fr]">
          <div>
            <h2 class="section-title">{{ t('home.features.unifiedGateway') }}</h2>
            <p class="mt-4 max-w-xl text-sm leading-7 text-zinc-500">
              {{ t('home.features.unifiedGatewayDesc') }}
            </p>
          </div>
          <div class="feature-list">
            <div v-for="feature in features" :key="feature.title" class="feature-row">
              <div class="feature-icon">
                <Icon :name="feature.icon" size="md" />
              </div>
              <div>
                <h3>{{ feature.title }}</h3>
                <p>{{ feature.description }}</p>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section class="mx-auto max-w-7xl pb-12">
        <div class="provider-strip">
          <span class="provider-label">{{ t('home.providers.title') }}</span>
          <span v-for="provider in providers" :key="provider">{{ provider }}</span>
          <span class="text-zinc-600">{{ t('home.providers.more') }}</span>
        </div>
      </section>
    </main>

    <footer class="relative z-10 border-t border-[#2cff43]/12 px-5 py-7 sm:px-8">
      <div class="mx-auto flex max-w-7xl flex-col gap-4 text-sm text-zinc-600 sm:flex-row sm:items-center sm:justify-between">
        <p>&copy; {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}</p>
        <div class="flex items-center gap-5">
          <a v-if="docUrl" :href="docUrl" target="_blank" rel="noopener noreferrer" class="footer-link">
            {{ t('home.docs') }}
          </a>
          <a :href="githubUrl" target="_blank" rel="noopener noreferrer" class="footer-link">
            GitHub
          </a>
        </div>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'
import CyberBackground from '@/components/auth/CyberBackground.vue'
import { sanitizeUrl } from '@/utils/url'

const { t } = useI18n()

const authStore = useAuthStore()
const appStore = useAppStore()

const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API')
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || 'AI API Gateway Platform')
const docUrl = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.doc_url || appStore.docUrl || ''))
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')

const isHomeContentUrl = computed(() => {
  const content = homeContent.value.trim()
  return content.startsWith('http://') || content.startsWith('https://')
})

const isDark = ref(document.documentElement.classList.contains('dark'))
const githubUrl = 'https://github.com/Wei-Shaw/sub2api'

const isAuthenticated = computed(() => authStore.isAuthenticated)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => isAdmin.value ? '/admin/dashboard' : '/dashboard')
const userInitial = computed(() => authStore.user?.email?.charAt(0).toUpperCase() || '')
const currentYear = computed(() => new Date().getFullYear())

const trustItems = computed(() => [
  { icon: 'shield' as const, label: t('home.tags.stickySession') },
  { icon: 'chart' as const, label: t('home.tags.realtimeBilling') },
  { icon: 'swap' as const, label: t('home.tags.subscriptionToApi') },
])

const features = computed(() => [
  {
    icon: 'server' as const,
    title: t('home.features.unifiedGateway'),
    description: t('home.features.unifiedGatewayDesc')
  },
  {
    icon: 'key' as const,
    title: t('home.features.multiAccount'),
    description: t('home.features.multiAccountDesc')
  },
  {
    icon: 'database' as const,
    title: t('home.features.balanceQuota'),
    description: t('home.features.balanceQuotaDesc')
  },
])

const providers = computed(() => [
  t('home.providers.claude'),
  'GPT',
  t('home.providers.gemini'),
  t('home.providers.antigravity'),
])

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

function initTheme() {
  const savedTheme = localStorage.getItem('theme')
  if (
    savedTheme === 'dark' ||
    (!savedTheme && window.matchMedia('(prefers-color-scheme: dark)').matches)
  ) {
    isDark.value = true
    document.documentElement.classList.add('dark')
  }
}

onMounted(() => {
  initTheme()
  authStore.checkAuth()
  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
})
</script>

<style scoped>
.home-cyber {
  isolation: isolate;
}

.brand-terminal {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 38px;
  height: 38px;
  border: 2px solid #2cff43;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 19px;
  font-weight: 900;
  color: #2cff43;
  box-shadow: 0 0 20px rgba(44, 255, 67, 0.22);
}

.nav-icon {
  display: inline-flex;
  width: 40px;
  height: 40px;
  align-items: center;
  justify-content: center;
  border: 1px solid rgba(44, 255, 67, 0.2);
  border-radius: 6px;
  color: #a1a1aa;
  background: rgba(0, 0, 0, 0.28);
  transition: border-color 0.18s ease, color 0.18s ease, background-color 0.18s ease;
}

.nav-icon:hover {
  border-color: rgba(44, 255, 67, 0.65);
  color: #2cff43;
  background: rgba(44, 255, 67, 0.07);
}

.home-cta,
.primary-action,
.secondary-action {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  border-radius: 6px;
  font-weight: 800;
  transition: transform 0.18s ease, box-shadow 0.18s ease, background-color 0.18s ease, border-color 0.18s ease;
}

.home-cta {
  min-height: 40px;
  padding: 0 14px;
  background: #2cff43;
  color: #020403;
  font-size: 0.85rem;
}

.user-dot {
  display: inline-flex;
  width: 20px;
  height: 20px;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  background: #020403;
  color: #2cff43;
  font-size: 0.7rem;
}

.hero-title {
  display: flex;
  flex-direction: column;
  font-family: Impact, Haettenschweiler, 'Arial Black', system-ui, sans-serif;
  font-size: clamp(4.8rem, 13vw, 9.8rem);
  font-weight: 900;
  line-height: 0.86;
  letter-spacing: 0;
  text-transform: uppercase;
  text-shadow: 0 0 22px rgba(255, 255, 255, 0.12);
}

.primary-action {
  min-height: 54px;
  padding: 0 22px;
  background: #2cff43;
  color: #020403;
  box-shadow: 0 0 24px rgba(44, 255, 67, 0.35);
}

.primary-action:hover,
.home-cta:hover {
  transform: translateY(-1px);
  box-shadow: 0 0 32px rgba(44, 255, 67, 0.42);
}

.secondary-action {
  min-height: 54px;
  padding: 0 20px;
  border: 1px solid rgba(44, 255, 67, 0.5);
  color: #2cff43;
}

.secondary-action:hover {
  border-color: #2cff43;
  background: rgba(44, 255, 67, 0.08);
}

.command-line {
  display: flex;
  max-width: 560px;
  align-items: center;
  gap: 14px;
  border: 1px solid rgba(44, 255, 67, 0.55);
  border-radius: 6px;
  background: rgba(0, 0, 0, 0.32);
  padding: 15px 18px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.9rem;
  color: #a1a1aa;
  box-shadow: 0 0 24px rgba(44, 255, 67, 0.1);
}

.status-tile {
  display: flex;
  align-items: center;
  gap: 10px;
  border: 1px solid rgba(44, 255, 67, 0.2);
  border-radius: 6px;
  background: rgba(0, 0, 0, 0.24);
  padding: 12px 13px;
  color: #a1a1aa;
  font-size: 0.8rem;
}

.status-tile svg {
  color: #2cff43;
}

.hero-panel {
  position: relative;
  overflow: hidden;
  border: 1px solid rgba(44, 255, 67, 0.55);
  border-radius: 18px;
  background:
    linear-gradient(90deg, rgba(44, 255, 67, 0.035) 1px, transparent 1px),
    linear-gradient(0deg, rgba(44, 255, 67, 0.025) 1px, transparent 1px),
    rgba(4, 8, 5, 0.78);
  background-size: 44px 44px, 44px 44px, auto;
  padding: 24px;
  box-shadow: 0 0 36px rgba(44, 255, 67, 0.13), inset 0 0 44px rgba(44, 255, 67, 0.04);
  backdrop-filter: blur(18px);
}

.panel-header {
  display: flex;
  align-items: center;
  gap: 10px;
  border-bottom: 1px solid rgba(44, 255, 67, 0.18);
  padding-bottom: 16px;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.75rem;
  color: #71717a;
}

.terminal-preview {
  padding: 26px 0;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}

.terminal-line {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  padding: 8px 0;
  color: #d4d4d8;
}

.terminal-key {
  color: #2cff43;
}

.response-box {
  margin-top: 22px;
  border: 1px solid rgba(44, 255, 67, 0.22);
  border-radius: 6px;
  background: rgba(0, 0, 0, 0.35);
  padding: 18px;
}

.response-box pre {
  margin-top: 14px;
  overflow: auto;
  color: #d4d4d8;
  font-size: 0.82rem;
  line-height: 1.7;
}

.metric-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 1px;
  overflow: hidden;
  border: 1px solid rgba(44, 255, 67, 0.18);
  border-radius: 6px;
  background: rgba(44, 255, 67, 0.18);
}

.metric-grid div {
  background: rgba(0, 0, 0, 0.42);
  padding: 15px;
}

.metric-grid span {
  display: block;
  color: #71717a;
  font-size: 0.72rem;
  text-transform: uppercase;
}

.metric-grid strong {
  margin-top: 7px;
  display: block;
  color: #2cff43;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 1rem;
}

.section-title {
  font-size: clamp(2rem, 5vw, 4.5rem);
  font-weight: 900;
  line-height: 0.95;
  letter-spacing: 0;
  text-transform: uppercase;
}

.feature-list {
  border-top: 1px solid rgba(44, 255, 67, 0.2);
}

.feature-row {
  display: grid;
  grid-template-columns: auto 1fr;
  gap: 16px;
  border-bottom: 1px solid rgba(44, 255, 67, 0.16);
  padding: 20px 0;
}

.feature-icon {
  display: flex;
  width: 42px;
  height: 42px;
  align-items: center;
  justify-content: center;
  border: 1px solid rgba(44, 255, 67, 0.35);
  border-radius: 6px;
  color: #2cff43;
  background: rgba(44, 255, 67, 0.06);
}

.feature-row h3 {
  font-weight: 800;
  color: #f4f4f5;
}

.feature-row p {
  margin-top: 6px;
  color: #71717a;
  font-size: 0.9rem;
  line-height: 1.65;
}

.provider-strip {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
  border: 1px solid rgba(44, 255, 67, 0.16);
  border-radius: 6px;
  background: rgba(0, 0, 0, 0.22);
  padding: 14px;
  color: #a1a1aa;
}

.provider-strip span {
  border: 1px solid rgba(44, 255, 67, 0.14);
  border-radius: 6px;
  padding: 8px 12px;
}

.provider-strip .provider-label {
  border-color: transparent;
  color: #2cff43;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  text-transform: uppercase;
}

.footer-link {
  color: #71717a;
  transition: color 0.18s ease;
}

.footer-link:hover {
  color: #2cff43;
}

@media (max-width: 640px) {
  .hero-panel {
    padding: 18px;
  }

  .metric-grid {
    grid-template-columns: 1fr;
  }
}
</style>
