<template>
  <BrandPlatformIcon :platform="provider" :size="`${size}px`" />
</template>

<script setup lang="ts">
import type { Provider } from '@/api/admin/channelMonitor'
import BrandPlatformIcon from '@/ui/icons/BrandPlatformIcon.vue'

withDefaults(
  defineProps<{
    provider: Provider
    size?: number
  }>(),
  {
    size: 16
  }
)
</script>
