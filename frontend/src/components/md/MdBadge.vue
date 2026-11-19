<template>
  <span class="md-badge" :class="[`md-badge--${tone}`, { 'md-badge--dot': dot }]">
    <span v-if="dot" class="md-badge-dot" aria-hidden="true" />
    <slot />
  </span>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    tone?: 'neutral' | 'primary' | 'success' | 'warn' | 'error' | 'info'
    dot?: boolean
  }>(),
  {
    tone: 'neutral',
    dot: false
  }
)
</script>

<style scoped>
.md-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 3px 10px;
  border-radius: 9999px;
  font-size: 0.75rem;
  font-weight: 600;
  line-height: 1.3;
  white-space: nowrap;
}

.md-badge-dot {
  width: 6px;
  height: 6px;
  border-radius: 9999px;
  background: currentColor;
}

.md-badge--neutral {
  background: var(--md-sys-color-surface-container-high);
  color: var(--md-sys-color-on-surface-variant);
}

.md-badge--primary {
  background: var(--md-sys-color-primary-container);
  color: var(--md-sys-color-on-primary-container);
}

.md-badge--success {
  background: rgb(var(--status-ok-bg));
  color: rgb(var(--text-ok));
}

.md-badge--warn {
  background: rgb(var(--status-warn-bg));
  color: rgb(var(--text-warn));
}

.md-badge--error {
  background: var(--md-sys-color-error-container);
  color: var(--md-sys-color-on-error-container);
}

.md-badge--info {
  background: var(--md-sys-color-tertiary-container);
  color: var(--md-sys-color-on-tertiary-container);
}
</style>
