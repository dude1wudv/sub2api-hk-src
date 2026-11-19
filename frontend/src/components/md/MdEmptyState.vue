<template>
  <div class="md-empty-state" :class="{ 'md-empty-state--compact': compact }">
    <div v-if="$slots.icon" class="md-empty-icon">
      <slot name="icon" />
    </div>
    <h3 class="md-empty-title">{{ title }}</h3>
    <p v-if="description" class="md-empty-desc">{{ description }}</p>
    <div v-if="$slots.actions" class="md-empty-actions">
      <slot name="actions" />
    </div>
  </div>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    title: string
    description?: string
    compact?: boolean
  }>(),
  {
    compact: false
  }
)
</script>

<style scoped>
.md-empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  padding: 40px 20px;
  border-radius: var(--md-sys-shape-corner-large);
  background: var(--md-sys-color-surface-container-low);
  border: 1px dashed var(--md-sys-color-outline-variant);
}

.md-empty-state--compact {
  padding: 24px 16px;
}

.md-empty-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  margin-bottom: 12px;
  border-radius: 16px;
  background: var(--md-sys-color-secondary-container);
  color: var(--md-sys-color-on-secondary-container);
}

.md-empty-title {
  margin: 0;
  font-size: 1rem;
  font-weight: 600;
  color: var(--md-sys-color-on-surface);
}

.md-empty-desc {
  margin: 6px 0 0;
  font-size: 0.875rem;
  color: var(--md-sys-color-on-surface-variant);
  max-width: 46ch;
}

.md-empty-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 8px;
  margin-top: 16px;
}
</style>
