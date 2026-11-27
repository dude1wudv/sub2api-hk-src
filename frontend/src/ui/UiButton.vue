<template>
  <button
    :type="type"
    :disabled="disabled || loading"
    :class="[
      'atelier-btn',
      variantClass,
      sizeClass,
      block ? 'w-full' : ''
    ]"
    v-bind="$attrs"
  >
    <span
      v-if="loading"
      class="inline-block h-3.5 w-3.5 animate-spin rounded-full border-2 border-current border-t-transparent"
      aria-hidden="true"
    />
    <slot />
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue'

defineOptions({ inheritAttrs: false })

const props = withDefaults(
  defineProps<{
    variant?: 'primary' | 'secondary' | 'ghost' | 'danger'
    size?: 'sm' | 'md' | 'lg'
    type?: 'button' | 'submit' | 'reset'
    disabled?: boolean
    loading?: boolean
    block?: boolean
  }>(),
  {
    variant: 'primary',
    size: 'md',
    type: 'button',
    disabled: false,
    loading: false,
    block: false
  }
)

const variantClass = computed(() => `atelier-btn-${props.variant}`)
const sizeClass = computed(() => {
  switch (props.size) {
    case 'sm':
      return 'h-8 px-3 text-xs'
    case 'lg':
      return 'h-11 px-5 text-sm'
    default:
      return 'h-9 px-4 text-sm'
  }
})
</script>
