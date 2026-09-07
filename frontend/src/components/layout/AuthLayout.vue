<template>
  <div class="auth-shell">
    <section class="auth-form-area">
      <div class="auth-appearance"><AppearanceSwitcher /></div>
      <div class="auth-form-inner">
        <div v-if="settingsLoaded" class="mb-7 flex items-center gap-3">
          <img :src="siteLogo || '/logo.svg'" alt="Logo" class="h-9 w-9 rounded-md object-contain ring-1 ring-gray-200 dark:ring-dark-700" />
          <div>
            <h1 class="text-lg font-semibold tracking-tight text-gray-900 dark:text-white">{{ siteName }}</h1>
            <p class="text-xs text-gray-500 dark:text-dark-300">{{ siteSubtitle }}</p>
          </div>
        </div>
        <div class="card"><slot /></div>
        <div class="mt-6 text-center text-sm"><slot name="footer" /></div>
        <div class="mt-8 text-center text-xs text-gray-500 dark:text-dark-300">&copy; {{ currentYear }} {{ siteName }}. All rights reserved.</div>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useAppStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'
import AppearanceSwitcher from '@/components/common/AppearanceSwitcher.vue'
const appStore = useAppStore()
const siteName = computed(() => appStore.siteName || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || 'AI API Workspace')
const settingsLoaded = computed(() => appStore.publicSettingsLoaded)
const currentYear = computed(() => new Date().getFullYear())
onMounted(() => { appStore.fetchPublicSettings() })
</script>

<style scoped>
.auth-form-inner :deep(.input) {
  min-height: 40px;
}

.auth-form-inner :deep(.btn) {
  min-height: 40px;
  font-weight: 500;
}

.auth-form-inner :deep(h2) {
  letter-spacing: -.025em;
  font-size: 24px;
  font-weight: 600;
}

.auth-form-inner :deep(.input-label) {
  font-size: 12px;
}

@media (max-width: 639px) {
  .auth-form-inner :deep(.input),
  .auth-form-inner :deep(.btn) { min-height: 44px; }
}
</style>
