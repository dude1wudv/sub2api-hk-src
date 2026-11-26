<template>
  <div
    class="atelier-dot-matrix pointer-events-none absolute inset-0"
    :class="maskClass"
    aria-hidden="true"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    mask?: 'edges' | 'right' | 'center'
  }>(),
  { mask: 'edges' }
)

const maskClass = computed(() => `atelier-dot-mask-${props.mask}`)
</script>
