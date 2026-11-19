<template>
  <label class="md-checkbox-wrap" :class="{ 'md-checkbox-wrap--disabled': disabled }">
    <component
      :is="'md-checkbox'"
      :checked="modelValue || undefined"
      :indeterminate="indeterminate || undefined"
      :disabled="disabled || undefined"
      :aria-label="ariaLabel || label || undefined"
      @change="onChange"
    />
    <span v-if="label || $slots.default" class="md-checkbox-label">
      <slot>{{ label }}</slot>
    </span>
  </label>
</template>

<script setup lang="ts">
import '@material/web/checkbox/checkbox.js'

withDefaults(
  defineProps<{
    modelValue: boolean
    indeterminate?: boolean
    label?: string
    ariaLabel?: string
    disabled?: boolean
  }>(),
  {
    indeterminate: false,
    disabled: false
  }
)

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
  (e: 'change', value: boolean): void
}>()

function onChange(event: Event) {
  const target = event.target as HTMLElement & { checked?: boolean }
  const next = Boolean(target?.checked)
  emit('update:modelValue', next)
  emit('change', next)
}
</script>

<style scoped>
.md-checkbox-wrap {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  user-select: none;
}

.md-checkbox-wrap--disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.md-checkbox-label {
  font-size: 0.875rem;
  color: var(--md-sys-color-on-surface);
}
</style>
