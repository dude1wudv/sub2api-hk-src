<template>
  <div class="relative flex min-h-screen flex-col bg-[var(--atelier-canvas)] text-[var(--atelier-ink)]">
    <header class="border-b border-[var(--atelier-border)] bg-[var(--atelier-canvas)]">
      <nav class="mx-auto flex h-14 max-w-6xl items-center justify-between px-5 sm:px-8" :aria-label="zh ? '主导航' : 'Main navigation'">
        <router-link to="/" aria-label="Home"><PatrickBrand /></router-link>
        <div class="flex items-center gap-2">
          <ColorModeToggle />
          <LocaleSwitcher />
        </div>
      </nav>
    </header>

    <main class="relative flex flex-1 flex-col items-center justify-center overflow-hidden px-5 py-16 text-center">
      <DotMatrixBackdrop mask="center" class="opacity-65" />
      <div class="relative z-10 max-w-md">
        <p class="font-mono text-xs font-medium uppercase tracking-[0.14em] text-[var(--atelier-ink-muted)]">404 · PAGE NOT FOUND</p>
        <h1 class="mt-3 text-3xl font-semibold tracking-tight text-[var(--atelier-ink)]">
          {{ t('errors.pageNotFound') }}
        </h1>
        <p class="mt-3 text-sm leading-relaxed text-[var(--atelier-ink-secondary)]">
          {{ zh ? '这个页面可能已被移动，或链接地址有误。你可以返回上一页，继续刚才的工作。' : 'This page may have moved, or the link may be incorrect. Go back to pick up where you left off.' }}
        </p>
        <div class="mt-7 flex flex-wrap items-center justify-center gap-3">
          <button type="button" class="atelier-btn atelier-btn-secondary h-9 px-4 text-xs" @click="goBack">
            <Icon name="arrowLeft" size="sm" />
            <span>{{ zh ? '返回上一页' : 'Go back' }}</span>
          </button>
          <router-link to="/dashboard" class="atelier-btn atelier-btn-primary h-9 px-4 text-xs">
            <Icon name="home" size="sm" />
            <span>{{ zh ? '前往控制台' : 'Go to console' }}</span>
          </router-link>
        </div>
        <router-link to="/" class="mt-5 inline-block text-xs text-[var(--atelier-ink-muted)] transition-colors hover:text-[var(--atelier-ink)]">
          {{ zh ? '或返回首页 ↗' : 'Or return home ↗' }}
        </router-link>
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import ColorModeToggle from '@/components/common/ColorModeToggle.vue'
import Icon from '@/components/icons/Icon.vue'
import PatrickBrand from '@/components/brand/PatrickBrand.vue'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import { DotMatrixBackdrop } from '@/ui'

const { t, locale } = useI18n()
const zh = computed(() => locale.value.startsWith('zh'))
const router = useRouter()

function goBack(): void {
  router.back()
}
</script>
