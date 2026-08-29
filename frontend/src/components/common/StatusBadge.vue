<template>
  <div class="inline-flex items-center gap-1.5 font-medium tabular-nums">
    <span
      :class="[
        'inline-block h-2 w-2 rounded-full ring-2 ring-transparent',
        variantDotClass
      ]"
    ></span>
    <span :class="['text-xs', variantTextClass]">
      {{ label }}
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  status: string
  label: string
}>()

const variantDotClass = computed(() => {
  switch (props.status) {
    case 'active':
    case 'success':
      return 'bg-ok-500 ring-ok-500/20'
    case 'disabled':
    case 'inactive':
    case 'warning':
      return 'bg-warn-500 ring-warn-500/20'
    case 'error':
    case 'danger':
      return 'bg-err-500 ring-err-500/20'
    default:
      return 'bg-gray-400 ring-gray-400/20 dark:bg-dark-300'
  }
})

const variantTextClass = computed(() => {
  switch (props.status) {
    case 'active':
    case 'success':
      return 'text-ok-700 dark:text-ok-400'
    case 'disabled':
    case 'inactive':
    case 'warning':
      return 'text-warn-700 dark:text-warn-400'
    case 'error':
    case 'danger':
      return 'text-err-700 dark:text-err-400'
    default:
      return 'text-gray-600 dark:text-dark-300'
  }
})
</script>
