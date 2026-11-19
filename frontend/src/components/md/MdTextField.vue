<template>
  <md-outlined-text-field
    class="md-tf-host"
    :class="{ 'md-tf--block': block, 'md-tf--mono': mono }"
    :value="stringValue"
    :label="label || undefined"
    :placeholder="placeholder || undefined"
    :type="type"
    :rows="type === 'textarea' ? rows : undefined"
    :supporting-text="supportingText || undefined"
    :error="error || undefined"
    :error-text="errorText || undefined"
    :prefix-text="prefixText || undefined"
    :suffix-text="suffixText || undefined"
    :disabled="disabled || undefined"
    :readonly="readonly || undefined"
    :required="required || undefined"
    :maxlength="maxlength"
    :min="min"
    :max="max"
    :step="step"
    :autocomplete="autocomplete || undefined"
    :name="name || undefined"
    @input="onInput"
    @change="onChange"
  >
    <span v-if="$slots['leading-icon']" slot="leading-icon" class="md-tf-slot-icon">
      <slot name="leading-icon" />
    </span>
    <span v-if="$slots['trailing-icon']" slot="trailing-icon" class="md-tf-slot-icon">
      <slot name="trailing-icon" />
    </span>
  </md-outlined-text-field>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import '@material/web/textfield/outlined-text-field.js'

const props = withDefaults(
  defineProps<{
    modelValue?: string | number | null
    label?: string
    placeholder?: string
    type?: 'text' | 'password' | 'email' | 'number' | 'search' | 'url' | 'tel' | 'textarea'
    rows?: number
    supportingText?: string
    error?: boolean
    errorText?: string
    prefixText?: string
    suffixText?: string
    disabled?: boolean
    readonly?: boolean
    required?: boolean
    maxlength?: number
    min?: number | string
    max?: number | string
    step?: number | string
    autocomplete?: string
    name?: string
    block?: boolean
    mono?: boolean
  }>(),
  {
    modelValue: '',
    type: 'text',
    rows: 3,
    error: false,
    disabled: false,
    readonly: false,
    required: false,
    block: true,
    mono: false
  }
)

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'change', value: string): void
}>()

const stringValue = computed(() =>
  props.modelValue === null || props.modelValue === undefined ? '' : String(props.modelValue)
)

function readValue(event: Event): string {
  const target = event.target as HTMLElement & { value?: string }
  return target?.value ?? ''
}

function onInput(event: Event) {
  emit('update:modelValue', readValue(event))
}

function onChange(event: Event) {
  const val = readValue(event)
  emit('update:modelValue', val)
  emit('change', val)
}
</script>

<style scoped>
.md-tf-host {
  --md-outlined-text-field-container-shape: 12px;
  --md-outlined-text-field-top-space: 12px;
  --md-outlined-text-field-bottom-space: 12px;
}

.md-tf--block {
  width: 100%;
}

.md-tf--mono {
  --md-outlined-text-field-input-text-font: var(--font-mono);
}

.md-tf-slot-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
}
</style>
