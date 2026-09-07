<template>
  <div
    class="relative flex min-h-screen items-center justify-center overflow-hidden bg-[rgb(var(--canvas))] px-4 py-12 text-[rgb(var(--ink))] sm:px-6"
  >
    <!-- Background Decoration: Subtle Warm Ambient Glow -->
    <div class="pointer-events-none absolute inset-0 overflow-hidden" aria-hidden="true">
      <div
        class="absolute -top-32 left-1/2 h-80 w-full max-w-4xl -translate-x-1/2 bg-gradient-to-b from-primary-500/[0.06] to-transparent dark:from-primary-500/[0.04]"
      ></div>
    </div>

    <div class="relative z-10 w-full max-w-md text-center">
      <!-- 404 Display -->
      <div class="mb-6">
        <div class="relative inline-block select-none">
          <span class="text-8xl font-black tracking-tight text-gray-200/90 dark:text-dark-800 sm:text-9xl md:text-[10rem]">404</span>
          <div class="absolute inset-0 flex items-center justify-center">
            <div
              class="flex h-16 w-16 items-center justify-center rounded-2xl border border-primary-400/30 bg-primary-600 text-white shadow-lg shadow-primary-600/25 ring-4 ring-[rgb(var(--canvas))] dark:bg-primary-500 dark:shadow-primary-500/20 sm:h-20 sm:w-20"
            >
              <svg
                class="h-8 w-8 sm:h-10 sm:w-10"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                stroke-width="1.75"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z"
                />
              </svg>
            </div>
          </div>
        </div>
      </div>

      <!-- Text Content -->
      <div class="mb-8">
        <h1 class="mb-2.5 text-2xl font-bold tracking-tight text-gray-950 dark:text-white sm:text-3xl">
          {{ t('errors.pageNotFound') }}
        </h1>
        <p class="mx-auto max-w-sm text-sm leading-relaxed text-gray-600 dark:text-dark-300 sm:text-base">
          The page you are looking for doesn't exist or has been moved.
        </p>
      </div>

      <!-- Action Buttons -->
      <div class="flex flex-col justify-center gap-3 sm:flex-row">
        <button @click="goBack" class="btn btn-secondary min-h-[44px] px-5 text-sm font-medium">
          <Icon name="arrowLeft" size="md" class="mr-1.5" />
          Go Back
        </button>
        <router-link to="/dashboard" class="btn btn-primary min-h-[44px] px-5 text-sm font-medium shadow-sm">
          <Icon name="home" size="md" class="mr-1.5" />
          Go to Dashboard
        </router-link>
      </div>

      <!-- Help Link -->
      <p class="mt-8 text-xs text-gray-500 dark:text-dark-400 sm:text-sm">
        Need help?
        <a
          href="#"
          class="font-medium text-primary-600 underline underline-offset-4 decoration-primary-500/30 transition-colors hover:text-primary-700 hover:decoration-primary-500 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/40 rounded dark:text-primary-400 dark:hover:text-primary-300"
        >
          Contact support
        </a>
      </p>
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
