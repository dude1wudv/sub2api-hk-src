<template>
  <div class="app-shell min-h-screen" :data-workspace="isAdminRoute ? 'operations' : 'developer'">
    <!-- Background Decoration -->
    <div class="app-shell-bg pointer-events-none fixed inset-0"></div>
    <a href="#workspace-content" class="skip-link">{{ zh ? '跳至主要内容' : 'Skip to content' }}</a>

    <!-- Sidebar -->
    <AppSidebar />

    <!-- Main Content Area -->
    <div
      class="app-main-content relative min-h-screen"
      :class="[sidebarCollapsed ? 'lg:ml-[72px]' : 'lg:ml-64']"
    >
      <!-- Header -->
      <AppHeader />

      <!-- Main Content -->
      <main id="workspace-content" tabindex="-1" class="workspace-content p-4 md:p-6 lg:p-8">
        <slot />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import '@/styles/onboarding.css'
import { computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { useAuthStore } from '@/stores/auth'
import { useOnboardingTour } from '@/composables/useOnboardingTour'
import { useOnboardingStore } from '@/stores/onboarding'
import AppSidebar from './AppSidebar.vue'
import AppHeader from './AppHeader.vue'

const appStore = useAppStore()
const authStore = useAuthStore()
const route = useRoute()
const { locale } = useI18n()
const zh = computed(() => locale.value.startsWith('zh'))
const isAdminRoute = computed(() => route.path.startsWith('/admin'))
const sidebarCollapsed = computed(() => appStore.sidebarCollapsed)
const isAdmin = computed(() => authStore.user?.role === 'admin')

const { replayTour } = useOnboardingTour({
  storageKey: isAdmin.value ? 'admin_guide' : 'user_guide',
  autoStart: true
})

const onboardingStore = useOnboardingStore()

onMounted(() => {
  onboardingStore.setReplayCallback(replayTour)
})

defineExpose({ replayTour })
</script>

<style scoped>
.app-main-content {
  transition-property: margin-left;
  transition-duration: 200ms;
  transition-timing-function: cubic-bezier(0.4, 0, 0.2, 1);
}

@media (prefers-reduced-motion: reduce) {
  .app-main-content {
    transition-duration: 0.01ms !important;
  }
}
</style>
