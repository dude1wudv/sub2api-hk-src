<template>
  <md-linear-progress
    v-if="variant === 'linear'"
    class="md-linear-progress-host"
    :class="[`md-progress--${tone}`]"
    :value="indeterminate ? undefined : clampedValue"
    :indeterminate="indeterminate || undefined"
    :four-color="fourColor || undefined"
    :aria-label="ariaLabel || undefined"
  />
  <md-circular-progress
    v-else
    class="md-circular-progress-host"
    :class="[`md-progress--${tone}`]"
    :style="size ? { '--md-circular-progress-size': `${size}px` } : undefined"
    :value="indeterminate ? undefined : clampedValue"
    :indeterminate="indeterminate || undefined"
    :four-color="fourColor || undefined"
    :aria-label="ariaLabel || undefined"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import '@material/web/progress/linear-progress.js'
import '@material/web/progress/circular-progress.js'

const props = withDefaults(
  defineProps<{
    variant?: 'linear' | 'circular'
    value?: number
    max?: number
    indeterminate?: boolean
    fourColor?: boolean
    tone?: 'primary' | 'warn' | 'error'
    size?: number
    ariaLabel?: string
  }>(),
  {
    variant: 'linear',
    value: 0,
    max: 1,
    indeterminate: false,
    fourColor: false,
    tone: 'primary'
  }
)

const clampedValue = computed(() => {
  const ratio = props.max > 0 ? props.value / props.max : 0
  return Math.max(0, Math.min(1, ratio))
})
</script>

<style scoped>
.md-linear-progress-host {
  width: 100%;
  --md-linear-progress-track-height: 6px;
  --md-linear-progress-active-indicator-height: 6px;
  --md-linear-progress-track-shape: 9999px;
}

.md-progress--warn {
  --md-linear-progress-active-indicator-color: rgb(var(--color-warn-500));
  --md-circular-progress-active-indicator-color: rgb(var(--color-warn-500));
}

.md-progress--error {
  --md-linear-progress-active-indicator-color: var(--md-sys-color-error);
  --md-circular-progress-active-indicator-color: var(--md-sys-color-error);
}
</style>
