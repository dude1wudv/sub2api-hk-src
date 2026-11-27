<template>
  <label
    class="inline-flex items-center gap-2 select-none"
    :class="disabled ? 'cursor-not-allowed opacity-50' : 'cursor-pointer'"
  >
    <md-checkbox
      :checked="modelValue"
      :indeterminate="indeterminate"
      :disabled="disabled"
      :aria-label="ariaLabel || label"
      @change="onChange"
    />
    <span v-if="label || $slots.default" class="text-sm text-[var(--atelier-ink)]">
      <slot>{{ label }}</slot>
    </span>
  </label>
</template>

<script setup lang="ts">
import '@material/web/checkbox/checkbox.js'

defineProps<{
  modelValue: boolean
  indeterminate?: boolean
  label?: string
  ariaLabel?: string
  disabled?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  change: [value: boolean]
}>()

function onChange(event: Event) {
  const target = event.target as HTMLElement & { checked?: boolean }
  const next = Boolean(target?.checked)
  emit('update:modelValue', next)
  emit('change', next)
}
</script>
