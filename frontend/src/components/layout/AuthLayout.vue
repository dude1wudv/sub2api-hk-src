<template>
  <div class="relative flex min-h-screen">
    <!-- Cyber-punk background (dark mode only) -->
    <CyberBackground v-if="isDarkMode" />

    <!-- Light mode fallback background -->
    <div
      v-else
      class="absolute inset-0 bg-gradient-to-br from-gray-50 via-primary-50/30 to-gray-100"
    ></div>

    <!-- Left side: Brand typography (hidden on mobile) -->
    <div
      class="relative z-10 hidden w-full items-center justify-center px-12 lg:flex lg:w-1/2"
    >
      <div class="max-w-lg">
        <h1
          class="mb-4 text-6xl font-black leading-tight tracking-tight"
          :class="isDarkMode ? 'text-white' : 'text-gray-900'"
        >
          <div class="mb-2">SECURE</div>
          <div class="mb-2">
            <span :class="isDarkMode ? 'text-cyber-green-500' : 'text-primary-600'">API</span>
          </div>
          <div>GATEWAY</div>
        </h1>
        <p
          class="text-lg leading-relaxed"
          :class="isDarkMode ? 'text-gray-400' : 'text-gray-600'"
        >
          {{ siteSubtitle || 'Enterprise-grade API access control and subscription management platform' }}
        </p>
      </div>
    </div>

    <!-- Right side: Login card -->
    <div
      class="relative z-10 flex w-full items-center justify-center px-4 py-12 lg:w-1/2 lg:px-12"
    >
      <div class="w-full max-w-md">
        <!-- Logo (mobile only) -->
        <div class="mb-8 text-center lg:hidden">
          <template v-if="settingsLoaded">
            <div
              class="mb-4 inline-flex h-12 w-12 items-center justify-center overflow-hidden rounded-xl"
              :class="isDarkMode ? 'shadow-neon-glow-sm' : 'shadow-lg shadow-primary-500/30'"
            >
              <img :src="siteLogo || '/logo.png'" alt="Logo" class="h-full w-full object-contain" />
            </div>
            <h1
              class="text-2xl font-bold"
              :class="isDarkMode ? 'text-cyber-green-500' : 'text-primary-600'"
            >
              {{ siteName }}
            </h1>
          </template>
        </div>

        <!-- Card Container -->
        <div
          class="rounded-xl p-8"
          :class="
            isDarkMode
              ? 'border border-cyber-border bg-cyber-card-bg/90 shadow-neon-glow backdrop-blur-sm'
              : 'bg-white shadow-glass'
          "
        >
          <slot />
        </div>

        <!-- Footer Links -->
        <div class="mt-6 text-center text-sm">
          <slot name="footer" />
        </div>

        <!-- Copyright -->
        <div
          class="mt-8 text-center text-xs"
          :class="isDarkMode ? 'text-gray-600' : 'text-gray-400'"
        >
          &copy; {{ currentYear }} {{ siteName }}. All rights reserved.
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useAppStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'
import CyberBackground from '@/components/auth/CyberBackground.vue'

const appStore = useAppStore()

const siteName = computed(() => appStore.siteName || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || 'Subscription to API Conversion Platform')
const settingsLoaded = computed(() => appStore.publicSettingsLoaded)
const isDarkMode = computed(() => document.documentElement.classList.contains('dark'))

const currentYear = computed(() => new Date().getFullYear())

onMounted(() => {
  appStore.fetchPublicSettings()
})
</script>
