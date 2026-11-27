<template>
  <div class="w-full">
    <div v-if="label || $slots.value" class="mb-1.5 flex items-center justify-between gap-2 text-xs">
      <span class="font-medium text-[var(--atelier-ink-secondary)]">{{ label }}</span>
      <span class="font-mono tabular-nums font-medium text-[var(--atelier-ink)]">
        <slot name="value">{{ modelValue }}</slot>
      </span>
    </div>
    <md-slider
      class="w-full"
      :value="modelValue"
      :min="min"
      :max="max"
      :step="step"
      :disabled="disabled"
      :labeled="labeled"
      :aria-label="ariaLabel || label"
      @input="onInput"
      @change="onChange"
    />
  </div>
</template>

<script setup lang="ts">
import '@material/web/slider/slider.js'

withDefaults(
  defineProps<{
    modelValue: number
    min?: number
    max?: number
    step?: number
    label?: string
    ariaLabel?: string
    disabled?: boolean
    labeled?: boolean
  }>(),
  {
    min: 0,
    max: 100,
    step: 1,
    label: '',
    ariaLabel: undefined,
    disabled: false,
    labeled: false
  }
)

const emit = defineEmits<{
  'update:modelValue': [value: number]
  change: [value: number]
}>()

function readNumeric(event: Event): number {
  const target = event.target as HTMLElement & { value?: number | string }
  const num = Number(target?.value ?? 0)
  return Number.isFinite(num) ? num : 0
}

function onInput(event: Event) {
  emit('update:modelValue', readNumeric(event))
}

function onChange(event: Event) {
  const num = readNumeric(event)
  emit('update:modelValue', num)
  emit('change', num)
}
</script>
