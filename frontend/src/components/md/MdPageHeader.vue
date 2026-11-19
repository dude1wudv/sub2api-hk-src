<template>
  <header class="md-page-header">
    <div class="md-page-header-main">
      <p v-if="eyebrow" class="md-page-eyebrow">{{ eyebrow }}</p>
      <div class="md-page-title-row">
        <h1 class="md-page-title">{{ title }}</h1>
        <slot name="badge" />
      </div>
      <p v-if="description" class="md-page-description">{{ description }}</p>
    </div>
    <div v-if="$slots.actions" class="md-page-actions">
      <slot name="actions" />
    </div>
  </header>
</template>

<script setup lang="ts">
defineProps<{
  title: string
  description?: string
  eyebrow?: string
}>()
</script>

<style scoped>
.md-page-header {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-end;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 20px;
}

.md-page-header-main {
  min-width: 0;
  flex: 1 1 280px;
}

.md-page-eyebrow {
  margin: 0 0 4px;
  font-size: 0.75rem;
  font-weight: 600;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: var(--md-sys-color-primary);
}

.md-page-title-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}

.md-page-title {
  margin: 0;
  font-size: clamp(1.375rem, 2.2vw, 1.75rem);
  line-height: 1.25;
  font-weight: 650;
  letter-spacing: -0.02em;
  color: var(--md-sys-color-on-surface);
}

.md-page-description {
  margin: 6px 0 0;
  font-size: 0.875rem;
  line-height: 1.5;
  color: var(--md-sys-color-on-surface-variant);
  max-width: 72ch;
}

.md-page-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}
</style>
