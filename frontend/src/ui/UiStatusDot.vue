<template>
  <span
    class="inline-block h-1.5 w-1.5 shrink-0 rounded-full"
    :class="toneClass"
    aria-hidden="true"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    tone?: 'ok' | 'warn' | 'err' | 'info' | 'neutral'
  }>(),
  { tone: 'neutral' }
)

const toneClass = computed(() => {
  switch (props.tone) {
    case 'ok':
      return 'bg-[var(--atelier-ok)]'
    case 'warn':
      return 'bg-[var(--atelier-warn)]'
    case 'err':
      return 'bg-[var(--atelier-err)]'
    case 'info':
      return 'bg-[var(--atelier-info)]'
    default:
      return 'bg-[var(--atelier-ink-muted)]'
  }
})
</script>
