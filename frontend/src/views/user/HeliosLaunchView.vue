<template>
  <div class="flex min-h-[16rem] items-center justify-center text-sm text-gray-500 dark:text-gray-400">
    {{ t('helios.starting') }}
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { launchHeliosWorkbench } from '@/utils/heliosLaunch'
import type { HeliosLaunchFailure } from '@/utils/heliosLaunch'

const { t } = useI18n()
const appStore = useAppStore()

function showHeliosLaunchError(failure: HeliosLaunchFailure) {
  appStore.showError(failure === 'popup-blocked' ? t('helios.popupBlocked') : t('helios.launchFailed'))
}

onMounted(() => {
  void launchHeliosWorkbench({ mode: 'current', notify: showHeliosLaunchError })
})
</script>
