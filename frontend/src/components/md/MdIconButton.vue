<template>
  <component
    :is="tagName"
    class="md-icon-btn-host"
    :class="[`md-icon-btn--${tone}`]"
    :disabled="disabled || undefined"
    :aria-label="ariaLabel"
    :title="title || ariaLabel"
    :toggle="toggle || undefined"
    :selected="toggle ? selected : undefined"
    :type="type"
    @click="onClick"
    @change="onChange"
  >
    <slot />
  </component>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import '@material/web/iconbutton/icon-button.js'
import '@material/web/iconbutton/filled-tonal-icon-button.js'
import '@material/web/iconbutton/outlined-icon-button.js'
import '@material/web/iconbutton/filled-icon-button.js'

const props = withDefaults(
  defineProps<{
    variant?: 'standard' | 'tonal' | 'outlined' | 'filled'
    tone?: 'default' | 'primary' | 'danger'
    ariaLabel?: string
    title?: string
    disabled?: boolean
    toggle?: boolean
    selected?: boolean
    type?: 'button' | 'submit' | 'reset'
  }>(),
  {
    variant: 'standard',
    tone: 'default',
    disabled: false,
    toggle: false,
    selected: false,
    type: 'button'
  }
)

const emit = defineEmits<{
  (e: 'click', event: MouseEvent): void
  (e: 'update:selected', value: boolean): void
}>()

const tagName = computed(() => {
  switch (props.variant) {
    case 'tonal':
      return 'md-filled-tonal-icon-button'
    case 'outlined':
      return 'md-outlined-icon-button'
    case 'filled':
      return 'md-filled-icon-button'
    default:
      return 'md-icon-button'
  }
})

function onClick(event: MouseEvent) {
  if (props.disabled) {
    event.preventDefault()
    event.stopPropagation()
    return
  }
  emit('click', event)
}

function onChange(event: Event) {
  if (!props.toggle) return
  const target = event.target as HTMLElement & { selected?: boolean }
  emit('update:selected', Boolean(target?.selected))
}
</script>

<style scoped>
.md-icon-btn-host {
  display: inline-flex;
  vertical-align: middle;
  flex-shrink: 0;
}

.md-icon-btn--danger {
  --md-icon-button-icon-color: var(--md-sys-color-error);
  --md-icon-button-hover-icon-color: var(--md-sys-color-error);
  --md-filled-tonal-icon-button-container-color: var(--md-sys-color-error-container);
  --md-filled-tonal-icon-button-icon-color: var(--md-sys-color-on-error-container);
}
</style>
