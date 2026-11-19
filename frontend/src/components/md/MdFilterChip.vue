<template>
  <component
    :is="'md-filter-chip'"
    class="md-filter-chip-host"
    :label="label"
    :selected="modelValue || undefined"
    :elevated="elevated || undefined"
    :disabled="disabled || undefined"
    :removable="removable || undefined"
    @click="onToggle"
    @remove="onRemove"
  >
    <span v-if="$slots.icon" slot="icon" class="md-chip-icon">
      <slot name="icon" />
    </span>
  </component>
</template>

<script setup lang="ts">
import '@material/web/chips/filter-chip.js'

const props = withDefaults(
  defineProps<{
    modelValue?: boolean
    label: string
    elevated?: boolean
    disabled?: boolean
    removable?: boolean
  }>(),
  {
    modelValue: false,
    elevated: false,
    disabled: false,
    removable: false
  }
)

const emit = defineEmits<{
  (e: 'update:modelValue', selected: boolean): void
  (e: 'click', event: MouseEvent): void
  (e: 'remove'): void
}>()

function onToggle(event: MouseEvent) {
  if (props.disabled) return
  emit('update:modelValue', !props.modelValue)
  emit('click', event)
}

function onRemove(event: Event) {
  event.preventDefault()
  emit('remove')
}
</script>

<style scoped>
.md-filter-chip-host {
  --md-filter-chip-label-text-font: var(--font-sans);
  --md-filter-chip-container-shape: 10px;
}

.md-chip-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
}
</style>
