<template>
  <component :is="'md-chip-set'" class="md-chip-set-host" :aria-label="ariaLabel || undefined">
    <slot />
  </component>
</template>

<script setup lang="ts">
import '@material/web/chips/chip-set.js'

defineProps<{
  ariaLabel?: string
}>()
</script>

<style scoped>
.md-chip-set-host {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
</style>
