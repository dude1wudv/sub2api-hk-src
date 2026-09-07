<template>
  <div
    class="relative flex min-h-screen items-center justify-center overflow-hidden bg-[rgb(var(--canvas))] px-4 py-12 text-[rgb(var(--ink))] sm:px-6"
  >

    <div class="relative z-10 w-full max-w-md text-center">
      <div class="mb-4 flex items-center justify-center gap-3 text-gray-500 dark:text-dark-400">
        <Icon name="search" size="lg" />
        <span class="font-mono text-sm">404</span>
      </div>

      <!-- Text Content -->
      <div class="mb-8">
        <h1 class="mb-2 text-2xl font-semibold tracking-tight text-gray-950 dark:text-white">
          {{ t('errors.pageNotFound') }}
        </h1>
        <p class="mx-auto max-w-sm text-sm leading-relaxed text-gray-600 dark:text-dark-300">
          {{ t('console.pageNotFoundDescription') }}
        </p>
      </div>

      <!-- Action Buttons -->
      <div class="flex flex-col justify-center gap-3 sm:flex-row">
        <button @click="goBack" class="btn btn-secondary min-h-[44px] px-5 text-sm font-medium">
          <Icon name="arrowLeft" size="md" class="mr-1.5" />
          {{ t('common.back') }}
        </button>
        <router-link to="/dashboard" class="btn btn-primary min-h-[44px] px-5 text-sm font-medium">
          <Icon name="home" size="md" class="mr-1.5" />
          {{ t('home.goToDashboard') }}
        </router-link>
      </div>

    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const router = useRouter()

function goBack(): void {
  router.back()
}
</script>
