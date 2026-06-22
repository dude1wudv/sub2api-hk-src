<template>
  <div class="relative flex min-h-screen overflow-hidden bg-[#020403] text-white">
    <CyberBackground />

    <!-- Left side: Brand typography (hidden on mobile) -->
    <div
      class="relative z-10 hidden w-full items-center px-12 lg:flex lg:w-[56%] xl:px-20"
    >
      <div class="max-w-3xl">
        <router-link to="/home" class="mb-24 inline-flex items-center gap-3 text-white">
          <span class="brand-terminal">>_</span>
          <span class="font-mono text-2xl font-black uppercase tracking-normal">
            {{ siteName.replace(/\s+/g, '') }}
          </span>
        </router-link>
        <h1
          class="brand-heading mb-7 text-[72px] font-black uppercase leading-[0.88] text-white xl:text-[96px]"
        >
          <div>SUB2</div>
          <div class="text-[#2cff43]">API</div>
          <div>GATEWAY</div>
        </h1>
        <p class="max-w-xl text-2xl font-medium leading-snug text-zinc-400">
          {{ siteSubtitle || 'Enterprise-grade API infrastructure. Speed. Security. Scale.' }}
        </p>
        <div class="mt-8 flex max-w-lg items-center border border-[#2cff43]/70 bg-black/35 font-mono text-sm text-zinc-400 shadow-[0_0_24px_rgba(44,255,67,0.1)]">
          <span class="px-5 py-4 text-[#2cff43]">$</span>
          <span class="min-w-0 flex-1 truncate py-4">sub2api login --key ************</span>
          <span class="border-l border-[#2cff43]/70 px-4 py-4 text-[#2cff43]">
            <Icon name="copy" size="sm" />
          </span>
        </div>
        <div class="mt-24 flex items-center gap-4 font-mono text-xs uppercase text-zinc-500">
          <span class="h-2 w-2 rounded-full bg-[#2cff43] shadow-[0_0_12px_rgba(44,255,67,0.8)]"></span>
          <span>Status</span>
          <span class="text-[#2cff43]">All systems operational</span>
        </div>
      </div>
    </div>

    <div class="pointer-events-none absolute bottom-10 left-[40%] z-10 hidden font-mono text-xs uppercase text-zinc-600 lg:block">
      <div>Node&nbsp;&nbsp;12.05.25.11</div>
      <div class="mt-4">Region&nbsp;<span class="text-[#2cff43]">US-EAST-1</span></div>
    </div>

    <!-- Right side: Auth card -->
    <div
      class="relative z-10 flex w-full items-center justify-center px-4 py-10 sm:px-6 lg:w-[44%] lg:px-12"
    >
      <div class="w-full max-w-[480px]">
        <div class="mb-8 text-center lg:hidden">
          <router-link to="/home" class="inline-flex items-center gap-3 text-white">
            <span class="brand-terminal">>_</span>
            <span class="font-mono text-2xl font-black uppercase">{{ siteName.replace(/\s+/g, '') }}</span>
          </router-link>
          <p class="mt-3 text-sm text-zinc-500">{{ siteSubtitle }}</p>
        </div>

        <div class="auth-card px-6 py-8 sm:px-9 sm:py-10">
          <div class="mb-10 flex justify-center">
            <div class="flex h-16 w-16 items-center justify-center text-[#2cff43] shadow-[0_0_30px_rgba(44,255,67,0.35)]">
              <Icon name="terminal" size="xl" :stroke-width="2.2" />
            </div>
          </div>
          <slot />

          <div class="mt-7 text-center text-sm text-zinc-500">
            <slot name="footer" />
          </div>
        </div>

        <!-- Copyright -->
        <div class="mt-8 text-center text-xs text-zinc-700">
          &copy; {{ currentYear }} {{ siteName }}. All rights reserved.
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useAppStore } from '@/stores'
import CyberBackground from '@/components/auth/CyberBackground.vue'
import Icon from '@/components/icons/Icon.vue'

const appStore = useAppStore()

const siteName = computed(() => appStore.siteName || 'Sub2API')
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || 'Subscription to API Conversion Platform')

const currentYear = computed(() => new Date().getFullYear())

onMounted(() => {
  appStore.fetchPublicSettings()
})
</script>

<style scoped>
.brand-terminal {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 42px;
  height: 42px;
  border: 2px solid #2cff43;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 22px;
  font-weight: 900;
  color: #2cff43;
  box-shadow: 0 0 24px rgba(44, 255, 67, 0.25);
}

.brand-heading {
  font-family: Impact, Haettenschweiler, 'Arial Black', system-ui, sans-serif;
  letter-spacing: 0;
  text-shadow: 0 0 22px rgba(255, 255, 255, 0.12);
}

.auth-card {
  position: relative;
  overflow: hidden;
  border: 1px solid rgba(44, 255, 67, 0.58);
  border-radius: 18px;
  background:
    linear-gradient(90deg, rgba(44, 255, 67, 0.035) 1px, transparent 1px),
    linear-gradient(0deg, rgba(44, 255, 67, 0.025) 1px, transparent 1px),
    rgba(4, 8, 5, 0.78);
  background-size: 44px 44px, 44px 44px, auto;
  box-shadow:
    0 0 34px rgba(44, 255, 67, 0.14),
    inset 0 0 48px rgba(44, 255, 67, 0.045);
  backdrop-filter: blur(18px);
}

.auth-card::before,
.auth-card::after {
  position: absolute;
  width: 42px;
  height: 42px;
  content: '';
  pointer-events: none;
}

.auth-card::before {
  top: -1px;
  right: -1px;
  border-top: 2px solid #2cff43;
  border-right: 2px solid #2cff43;
}

.auth-card::after {
  bottom: -1px;
  left: -1px;
  border-bottom: 2px solid rgba(44, 255, 67, 0.6);
  border-left: 2px solid rgba(44, 255, 67, 0.6);
}

.auth-card :deep(h2) {
  color: #f4f4f5;
}

.auth-card :deep(p),
.auth-card :deep(.input-hint) {
  color: #71717a;
}

.auth-card :deep(.input-label) {
  color: #f4f4f5;
}

.auth-card :deep(.input) {
  border-color: rgba(113, 113, 122, 0.55);
  border-radius: 6px;
  background: rgba(0, 0, 0, 0.26);
  color: #f4f4f5;
}

.auth-card :deep(.input::placeholder) {
  color: #71717a;
}

.auth-card :deep(.input:focus) {
  border-color: rgba(44, 255, 67, 0.82);
  box-shadow: 0 0 0 3px rgba(44, 255, 67, 0.12);
}

.auth-card :deep(a) {
  color: #2cff43;
}
</style>
