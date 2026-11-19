<template>
  <div class="md-slider-wrap">
    <div v-if="label || showValue" class="md-slider-head">
      <span v-if="label" class="md-slider-label">{{ label }}</span>
      <span v-if="showValue" class="md-slider-value">{{ formattedValue }}</span>
    </div>
    <component
      :is="'md-slider'"
      class="md-slider-host"
      :value="modelValue"
      :min="min"
      :max="max"
      :step="step"
      :labeled="labeled || undefined"
      :ticks="ticks || undefined"
      :disabled="disabled || undefined"
      :aria-label="ariaLabel || label || undefined"
      @input="onInput"
      @change="onChange"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import '@material/web/slider/slider.js'

const props = withDefaults(
  defineProps<{
    modelValue: number
    min?: number
    max?: number
    step?: number
    label?: string
    ariaLabel?: string
    unit?: string
    showValue?: boolean
    labeled?: boolean
    ticks?: boolean
    disabled?: boolean
  }>(),
  {
    min: 0,
    max: 100,
    step: 1,
    showValue: true,
    labeled: true,
    ticks: false,
    disabled: false
  }
)

const emit = defineEmits<{
  (e: 'update:modelValue', value: number): void
  (e: 'change', value: number): void
}>()

const formattedValue = computed(() =>
  props.unit ? `${props.modelValue}${props.unit}` : String(props.modelValue)
)

function readNumber(event: Event): number {
  const target = event.target as HTMLElement & { value?: number }
  const val = Number(target?.value)
  return Number.isFinite(val) ? val : props.modelValue
}

function onInput(event: Event) {
  emit('update:modelValue', readNumber(event))
}

function onChange(event: Event) {
  const next = readNumber(event)
  emit('update:modelValue', next)
  emit('change', next)
}
</script>

<style scoped>
.md-slider-wrap {
  display: flex;
  flex-direction: column;
  gap: 2px;
  width: 100%;
}

.md-slider-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 0.8125rem;
}

.md-slider-label {
  color: var(--md-sys-color-on-surface-variant);
  font-weight: 500;
}

.md-slider-value {
  font-family: var(--font-mono);
  color: var(--md-sys-color-on-surface);
  font-weight: 600;
}

.md-slider-host {
  width: 100%;
}
</style>
