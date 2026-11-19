<template>
  <component
    :is="tagName"
    class="md-btn-host"
    :class="[`md-btn--${tone}`, `md-btn--${size}`, { 'md-btn--block': block, 'md-btn--loading': loading }]"
    :disabled="disabled || loading || undefined"
    :type="href ? undefined : type"
    :href="href || undefined"
    :target="href ? target : undefined"
    :rel="href && target === '_blank' ? (rel || 'noopener noreferrer') : rel"
    :aria-busy="loading || undefined"
    @click="onClick"
  >
    <md-circular-progress
      v-if="loading"
      slot="icon"
      indeterminate
      class="md-btn-spinner"
      aria-hidden="true"
    />
    <span v-else-if="$slots.icon" slot="icon" class="md-btn-icon">
      <slot name="icon" />
    </span>
    <slot />
  </component>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import '@material/web/button/filled-button.js'
import '@material/web/button/filled-tonal-button.js'
import '@material/web/button/outlined-button.js'
import '@material/web/button/text-button.js'
import '@material/web/button/elevated-button.js'
import '@material/web/progress/circular-progress.js'

export type MdButtonVariant = 'filled' | 'tonal' | 'outlined' | 'text' | 'elevated'
export type MdButtonTone = 'primary' | 'danger' | 'neutral'
export type MdButtonSize = 'sm' | 'md'

const props = withDefaults(
  defineProps<{
    variant?: MdButtonVariant
    tone?: MdButtonTone
    size?: MdButtonSize
    type?: 'button' | 'submit' | 'reset'
    disabled?: boolean
    loading?: boolean
    block?: boolean
    href?: string
    target?: string
    rel?: string
  }>(),
  {
    variant: 'filled',
    tone: 'primary',
    size: 'md',
    type: 'button',
    disabled: false,
    loading: false,
    block: false
  }
)

const emit = defineEmits<{
  (e: 'click', event: MouseEvent): void
}>()

const tagName = computed(() => {
  switch (props.variant) {
    case 'tonal':
      return 'md-filled-tonal-button'
    case 'outlined':
      return 'md-outlined-button'
    case 'text':
      return 'md-text-button'
    case 'elevated':
      return 'md-elevated-button'
    default:
      return 'md-filled-button'
  }
})

function onClick(event: MouseEvent) {
  if (props.disabled || props.loading) {
    event.preventDefault()
    event.stopPropagation()
    return
  }
  emit('click', event)
}
</script>

<style scoped>
.md-btn-host {
  display: inline-flex;
  vertical-align: middle;
  flex-shrink: 0;
  --md-filled-button-container-height: 40px;
  --md-filled-tonal-button-container-height: 40px;
  --md-outlined-button-container-height: 40px;
  --md-text-button-container-height: 40px;
  --md-elevated-button-container-height: 40px;
}

.md-btn--sm {
  --md-filled-button-container-height: 34px;
  --md-filled-tonal-button-container-height: 34px;
  --md-outlined-button-container-height: 34px;
  --md-text-button-container-height: 34px;
  --md-elevated-button-container-height: 34px;
  --md-filled-button-label-text-size: 0.8125rem;
  --md-filled-tonal-button-label-text-size: 0.8125rem;
  --md-outlined-button-label-text-size: 0.8125rem;
  --md-text-button-label-text-size: 0.8125rem;
  --md-elevated-button-label-text-size: 0.8125rem;
  --md-filled-button-leading-space: 14px;
  --md-filled-button-trailing-space: 14px;
  --md-filled-tonal-button-leading-space: 14px;
  --md-filled-tonal-button-trailing-space: 14px;
  --md-outlined-button-leading-space: 14px;
  --md-outlined-button-trailing-space: 14px;
  --md-text-button-leading-space: 10px;
  --md-text-button-trailing-space: 10px;
}

.md-btn--block {
  display: flex;
  width: 100%;
}

.md-btn--danger {
  --md-filled-button-container-color: var(--md-sys-color-error);
  --md-filled-button-label-text-color: var(--md-sys-color-on-error);
  --md-filled-button-hover-label-text-color: var(--md-sys-color-on-error);
  --md-filled-button-focus-label-text-color: var(--md-sys-color-on-error);
  --md-filled-button-pressed-label-text-color: var(--md-sys-color-on-error);
  --md-filled-tonal-button-container-color: var(--md-sys-color-error-container);
  --md-filled-tonal-button-label-text-color: var(--md-sys-color-on-error-container);
  --md-outlined-button-label-text-color: var(--md-sys-color-error);
  --md-outlined-button-outline-color: var(--md-sys-color-error);
  --md-text-button-label-text-color: var(--md-sys-color-error);
}

.md-btn--neutral {
  --md-filled-button-container-color: var(--md-sys-color-inverse-surface);
  --md-filled-button-label-text-color: var(--md-sys-color-inverse-on-surface);
  --md-filled-tonal-button-container-color: var(--md-sys-color-surface-container-high);
  --md-filled-tonal-button-label-text-color: var(--md-sys-color-on-surface);
  --md-outlined-button-label-text-color: var(--md-sys-color-on-surface);
  --md-text-button-label-text-color: var(--md-sys-color-on-surface-variant);
}

.md-btn-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
}

.md-btn-spinner {
  --md-circular-progress-size: 18px;
  --md-circular-progress-active-indicator-width: 12;
  --md-circular-progress-active-indicator-color: currentColor;
}
</style>
