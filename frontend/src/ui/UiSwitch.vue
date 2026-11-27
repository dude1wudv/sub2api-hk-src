<template>
  <label
    class="inline-flex items-center gap-2.5 select-none"
    :class="disabled ? 'cursor-not-allowed opacity-50' : 'cursor-pointer'"
  >
    <md-switch
      :selected="modelValue"
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
import '@material/web/switch/switch.js'

defineProps<{
  modelValue: boolean
  label?: string
  ariaLabel?: string
  disabled?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  change: [value: boolean]
}>()

function onChange(event: Event) {
  const target = event.target as HTMLElement & { selected?: boolean }
  const next = Boolean(target?.selected)
  emit('update:modelValue', next)
  emit('change', next)
}
</script>
