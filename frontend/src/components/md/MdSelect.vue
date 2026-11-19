<template>
  <md-outlined-select
    class="md-select-host"
    :class="{ 'md-select--block': block }"
    :value="stringValue"
    :label="label || undefined"
    :supporting-text="supportingText || undefined"
    :error="error || undefined"
    :error-text="errorText || undefined"
    :disabled="disabled || undefined"
    :required="required || undefined"
    :quick="quick || undefined"
    @change="onChange"
  >
    <md-select-option
      v-for="opt in options"
      :key="String(opt.value)"
      :value="String(opt.value)"
      :selected="String(opt.value) === stringValue || undefined"
      :disabled="opt.disabled || undefined"
    >
      <div slot="headline">{{ opt.headline || opt.label }}</div>
      <div v-if="opt.supportingText" slot="supporting-text">{{ opt.supportingText }}</div>
    </md-select-option>
  </md-outlined-select>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import '@material/web/select/outlined-select.js'
import '@material/web/select/select-option.js'

export interface MdSelectOption {
  value: string | number
  label: string
  headline?: string
  supportingText?: string
  disabled?: boolean
}

const props = withDefaults(
  defineProps<{
    modelValue?: string | number | null
    options: MdSelectOption[]
    label?: string
    supportingText?: string
    error?: boolean
    errorText?: string
    disabled?: boolean
    required?: boolean
    quick?: boolean
    block?: boolean
  }>(),
  {
    modelValue: '',
    error: false,
    disabled: false,
    required: false,
    quick: true,
    block: true
  }
)

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'change', value: string): void
}>()

const stringValue = computed(() =>
  props.modelValue === null || props.modelValue === undefined ? '' : String(props.modelValue)
)

function onChange(event: Event) {
  const target = event.target as HTMLElement & { value?: string }
  const val = target?.value ?? ''
  emit('update:modelValue', val)
  emit('change', val)
}
</script>

<style scoped>
.md-select-host {
  min-width: 140px;
  --md-outlined-select-text-field-container-shape: 12px;
  --md-outlined-select-text-field-top-space: 12px;
  --md-outlined-select-text-field-bottom-space: 12px;
}

.md-select--block {
  width: 100%;
}
</style>
