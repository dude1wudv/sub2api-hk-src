<template>
  <div class="card">
    <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700 sm:px-6">
      <h2 class="text-base font-semibold text-gray-900 dark:text-white sm:text-lg">{{ t('dashboard.quickActions') }}</h2>
    </div>
    <div class="space-y-2.5 p-4 sm:p-5">
      <button @click="router.push('/keys')" class="group flex w-full items-center gap-3.5 rounded-xl border border-transparent bg-gray-50/80 p-3.5 text-left transition-colors hover:border-gray-200 hover:bg-gray-100/80 dark:border-transparent dark:bg-dark-800/40 dark:hover:border-dark-600 dark:hover:bg-dark-800">
        <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-primary-100/80 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300">
          <Icon name="key" size="md" />
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('dashboard.createApiKey') }}</p>
          <p class="truncate text-xs text-gray-500 dark:text-dark-300">{{ t('dashboard.generateNewKey') }}</p>
        </div>
        <Icon
          name="chevronRight"
          size="sm"
          class="text-gray-400 transition-colors group-hover:text-primary-600 dark:text-dark-400 dark:group-hover:text-primary-400"
        />
      </button>

      <button @click="router.push('/usage')" class="group flex w-full items-center gap-3.5 rounded-xl border border-transparent bg-gray-50/80 p-3.5 text-left transition-colors hover:border-gray-200 hover:bg-gray-100/80 dark:border-transparent dark:bg-dark-800/40 dark:hover:border-dark-600 dark:hover:bg-dark-800">
        <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-link-100/70 text-link-700 dark:bg-link-900/30 dark:text-link-300">
          <Icon name="chart" size="md" />
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('dashboard.viewUsage') }}</p>
          <p class="truncate text-xs text-gray-500 dark:text-dark-300">{{ t('dashboard.checkDetailedLogs') }}</p>
        </div>
        <Icon
          name="chevronRight"
          size="sm"
          class="text-gray-400 transition-colors group-hover:text-link-600 dark:text-dark-400 dark:group-hover:text-link-400"
        />
      </button>

      <button v-if="canUseBatchImage" @click="router.push('/batch-image')" class="group flex w-full items-center gap-3.5 rounded-xl border border-transparent bg-gray-50/80 p-3.5 text-left transition-colors hover:border-gray-200 hover:bg-gray-100/80 dark:border-transparent dark:bg-dark-800/40 dark:hover:border-dark-600 dark:hover:bg-dark-800">
        <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-sky-100/80 text-sky-700 dark:bg-sky-950/30 dark:text-sky-300">
          <Icon name="sparkles" size="md" />
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('dashboard.batchImageAgent') }}</p>
          <p class="truncate text-xs text-gray-500 dark:text-dark-300">{{ t('dashboard.batchImageAgentDesc') }}</p>
        </div>
        <Icon
          name="chevronRight"
          size="sm"
          class="text-gray-400 transition-colors group-hover:text-sky-600 dark:text-dark-400 dark:group-hover:text-sky-400"
        />
      </button>

      <button @click="router.push('/redeem')" class="group flex w-full items-center gap-3.5 rounded-xl border border-transparent bg-gray-50/80 p-3.5 text-left transition-colors hover:border-gray-200 hover:bg-gray-100/80 dark:border-transparent dark:bg-dark-800/40 dark:hover:border-dark-600 dark:hover:bg-dark-800">
        <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-amber-50 text-amber-700 dark:bg-amber-950/30 dark:text-amber-300">
          <Icon name="gift" size="md" />
        </div>
        <div class="min-w-0 flex-1">
          <p class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('dashboard.redeemCode') }}</p>
          <p class="truncate text-xs text-gray-500 dark:text-dark-300">{{ t('dashboard.addBalanceWithCode') }}</p>
        </div>
        <Icon
          name="chevronRight"
          size="sm"
          class="text-gray-400 transition-colors group-hover:text-amber-600 dark:text-dark-400 dark:group-hover:text-amber-400"
        />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { useBatchImageAccess } from '@/composables/useBatchImageAccess'
const router = useRouter()
const { t } = useI18n()
const { canUseBatchImage, refreshBatchImageAccess } = useBatchImageAccess()

onMounted(() => {
  void refreshBatchImageAccess()
})
</script>
