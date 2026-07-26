<template>
  <div class="flex min-h-[16rem] items-center justify-center text-sm text-gray-500 dark:text-gray-400">
    {{ t('nav.image2') }}
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

onMounted(() => {
  window.location.replace('/image2/')
})
</script>