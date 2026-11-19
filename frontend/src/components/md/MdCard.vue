<template>
  <component
    :is="as"
    class="md-card"
    :class="[
      `md-card--${variant}`,
      `md-card--pad-${padding}`,
      { 'md-card--interactive': interactive }
    ]"
  >
    <div v-if="title || subtitle || $slots.header || $slots.actions" class="md-card-header">
      <div class="md-card-heading">
        <slot name="header">
          <h2 v-if="title" class="md-card-title">{{ title }}</h2>
          <p v-if="subtitle" class="md-card-subtitle">{{ subtitle }}</p>
        </slot>
      </div>
      <div v-if="$slots.actions" class="md-card-actions">
        <slot name="actions" />
      </div>
    </div>
    <div class="md-card-body">
      <slot />
    </div>
    <div v-if="$slots.footer" class="md-card-footer">
      <slot name="footer" />
    </div>
  </component>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    as?: 'section' | 'div' | 'article'
    variant?: 'outlined' | 'filled' | 'elevated' | 'tonal'
    padding?: 'none' | 'sm' | 'md' | 'lg'
    title?: string
    subtitle?: string
    interactive?: boolean
  }>(),
  {
    as: 'section',
    variant: 'outlined',
    padding: 'md',
    interactive: false
  }
)
</script>

<style scoped>
.md-card {
  position: relative;
  border-radius: var(--md-sys-shape-corner-large);
  color: var(--md-sys-color-on-surface);
  transition: border-color var(--motion-fast, 150ms), box-shadow var(--motion-fast, 150ms), background-color var(--motion-fast, 150ms);
}

.md-card--outlined {
  background: var(--md-sys-color-surface-container-lowest);
  border: 1px solid var(--md-sys-color-outline-variant);
}

:global(:root.dark) .md-card--outlined {
  background: var(--md-sys-color-surface-container-low);
}

.md-card--filled {
  background: var(--md-sys-color-surface-container-low);
  border: 1px solid transparent;
}

.md-card--elevated {
  background: var(--md-sys-color-surface-container-lowest);
  border: 1px solid color-mix(in srgb, var(--md-sys-color-outline-variant) 55%, transparent);
  box-shadow: var(--md-elevation-1);
}

:global(:root.dark) .md-card--elevated {
  background: var(--md-sys-color-surface-container);
}

.md-card--tonal {
  background: var(--md-sys-color-primary-container);
  color: var(--md-sys-color-on-primary-container);
  border: 1px solid color-mix(in srgb, var(--md-sys-color-primary) 22%, transparent);
}

.md-card--interactive:hover {
  border-color: var(--md-sys-color-primary);
  box-shadow: var(--md-elevation-2);
}

.md-card--pad-none { padding: 0; }
.md-card--pad-sm { padding: 14px 16px; }
.md-card--pad-md { padding: 20px; }
.md-card--pad-lg { padding: 24px; }

.md-card-header {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 14px;
}

.md-card-heading {
  min-width: 0;
  flex: 1 1 auto;
}

.md-card-title {
  margin: 0;
  font-size: 1rem;
  line-height: 1.45;
  font-weight: 600;
  color: inherit;
}

.md-card-subtitle {
  margin: 4px 0 0;
  font-size: 0.8125rem;
  line-height: 1.45;
  color: var(--md-sys-color-on-surface-variant);
}

.md-card--tonal .md-card-subtitle {
  color: color-mix(in srgb, var(--md-sys-color-on-primary-container) 80%, transparent);
}

.md-card-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.md-card-footer {
  margin-top: 16px;
  padding-top: 12px;
  border-top: 1px solid var(--md-sys-color-outline-variant);
}
</style>
