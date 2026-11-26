<template>
  <div class="atelier-progress-track" role="progressbar" :aria-valuenow="clamped" aria-valuemin="0" aria-valuemax="100">
    <div
      class="atelier-progress-fill"
      :class="toneClass"
      :style="{ width: `${clamped}%` }"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(
  defineProps<{
    value: number
    tone?: 'ink' | 'ok' | 'warn' | 'err'
  }>(),
  {
    tone: 'ink'
  }
)

const clamped = computed(() => Math.max(0, Math.min(100, Number.isFinite(props.value) ? props.value : 0)))

const toneClass = computed(() => {
  switch (props.tone) {
    case 'ok':
      return '!bg-[var(--atelier-ok)]'
    case 'warn':
      return '!bg-[var(--atelier-warn)]'
    case 'err':
      return '!bg-[var(--atelier-err)]'
    default:
      return ''
  }
})
</script>
