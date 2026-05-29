<template>
  <div class="relative">
    <span
      v-if="displayVersion"
      class="flex items-center gap-1.5 rounded-lg bg-gray-100 px-2 py-1 text-xs text-gray-600 dark:bg-dark-800 dark:text-dark-400"
      :title="`v${displayVersion}`"
    >
      <span class="font-medium">v{{ displayVersion }}</span>
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useAppStore } from '@/stores'

const props = defineProps<{
  version?: string
}>()

const appStore = useAppStore()
const displayVersion = computed(() => appStore.currentVersion || props.version || '')
</script>
