<template>
  <label class="md-switch-wrap" :class="{ 'md-switch-wrap--disabled': disabled }">
    <component
      :is="'md-switch'"
      :selected="modelValue || undefined"
      :disabled="disabled || undefined"
      :icons="icons || undefined"
      :show-only-selected-icon="showOnlySelectedIcon || undefined"
      :aria-label="ariaLabel || label || undefined"
      @change="onChange"
    />
    <span v-if="label || description || $slots.default" class="md-switch-text">
      <span class="md-switch-label">
        <slot>{{ label }}</slot>
      </span>
      <span v-if="description" class="md-switch-desc">{{ description }}</span>
    </span>
  </label>
</template>

<script setup lang="ts">
import '@material/web/switch/switch.js'

withDefaults(
  defineProps<{
    modelValue: boolean
    label?: string
    description?: string
    ariaLabel?: string
    disabled?: boolean
    icons?: boolean
    showOnlySelectedIcon?: boolean
  }>(),
  {
    disabled: false,
    icons: true,
    showOnlySelectedIcon: true
  }
)

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
  (e: 'change', value: boolean): void
}>()

function onChange(event: Event) {
  const target = event.target as HTMLElement & { selected?: boolean }
  const next = Boolean(target?.selected)
  emit('update:modelValue', next)
  emit('change', next)
}
</script>

<style scoped>
.md-switch-wrap {
  display: inline-flex;
  align-items: center;
  gap: 12px;
  cursor: pointer;
  user-select: none;
}

.md-switch-wrap--disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.md-switch-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.md-switch-label {
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--md-sys-color-on-surface);
}

.md-switch-desc {
  font-size: 0.75rem;
  color: var(--md-sys-color-on-surface-variant);
}
</style>
