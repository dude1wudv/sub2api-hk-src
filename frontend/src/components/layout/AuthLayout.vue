<template>
  <div class="auth-shell">
    <section class="auth-story" :aria-label="siteName">
      <div class="hero-eyebrow">{{ siteName }} / API WORKSPACE</div>
      <h2>{{ zh ? '让灵感，\n连接无限可能。' : 'One connection.\nInfinite possibilities.' }}</h2>
      <p>{{ zh ? '从第一次 API 调用，到每一次规模化创新。一个清晰、可靠的工作区，连接你的模型、应用与创造力。' : 'From your first API call to your next breakthrough. A focused workspace for your models, applications and ideas.' }}</p>
      <div class="auth-story-features">
        <span><b>01 / CONNECT</b>{{ zh ? '统一接入 · 专注创造' : 'Unified access. Stay focused.' }}</span>
        <span><b>02 / OBSERVE</b>{{ zh ? '用量透明 · 成本可见' : 'Transparent usage and costs.' }}</span>
        <span><b>03 / CONTROL</b>{{ zh ? '精细权限 · 从容掌控' : 'Granular access. Clear control.' }}</span>
      </div>
    </section>
    <section class="auth-form-area">
      <div class="auth-appearance"><AppearanceSwitcher /></div>
      <div class="auth-form-inner">
        <div v-if="settingsLoaded" class="mb-7 flex items-center gap-3">
          <img :src="siteLogo || '/logo.svg'" alt="Logo" class="h-10 w-10 rounded-xl object-contain" />
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
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'
import AppearanceSwitcher from '@/components/common/AppearanceSwitcher.vue'
const appStore = useAppStore()
const { locale } = useI18n()
const zh = computed(() => locale.value.startsWith('zh'))
const siteName = computed(() => appStore.siteName || 'Sub2API')
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || 'AI API Workspace')
const settingsLoaded = computed(() => appStore.publicSettingsLoaded)
const currentYear = computed(() => new Date().getFullYear())
onMounted(() => { appStore.fetchPublicSettings() })
</script>
