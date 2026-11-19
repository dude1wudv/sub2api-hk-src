<template>
  <component
    :is="'md-dialog'"
    ref="dialogRef"
    class="md-dialog-host"
    :style="{ '--md-dialog-max-width': maxWidth }"
    :open="open || undefined"
    @cancel="onCancel"
    @close="onClose"
  >
    <div v-if="headline || $slots.headline" slot="headline" class="md-dialog-headline">
      <slot name="headline">{{ headline }}</slot>
    </div>
    <div slot="content" class="md-dialog-content">
      <slot />
    </div>
    <div v-if="$slots.actions" slot="actions" class="md-dialog-actions">
      <slot name="actions" />
    </div>
  </component>
</template>

<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import '@material/web/dialog/dialog.js'

const props = withDefaults(
  defineProps<{
    open: boolean
    headline?: string
    maxWidth?: string
    closeOnEsc?: boolean
  }>(),
  {
    maxWidth: '560px',
    closeOnEsc: true
  }
)

const emit = defineEmits<{
  (e: 'update:open', value: boolean): void
  (e: 'close'): void
}>()

type MdDialogElement = HTMLElement & {
  open?: boolean
  show?: () => Promise<void> | void
  close?: (returnValue?: string) => Promise<void> | void
}

const dialogRef = ref<MdDialogElement | null>(null)

function syncOpen(next: boolean) {
  const el = dialogRef.value
  if (!el) return
  if (next) {
    if (typeof el.show === 'function' && !el.open) {
      void el.show()
    } else {
      el.open = true
    }
  } else if (typeof el.close === 'function' && el.open) {
    void el.close()
  } else {
    el.open = false
  }
}

onMounted(() => {
  if (props.open) {
    syncOpen(true)
  }
})

watch(
  () => props.open,
  (next) => {
    syncOpen(next)
  }
)

function onCancel(event: Event) {
  if (!props.closeOnEsc) {
    event.preventDefault()
  }
}

function onClose() {
  if (props.open) {
    emit('update:open', false)
  }
  emit('close')
}
</script>

<style scoped>
.md-dialog-host {
  max-width: min(var(--md-dialog-max-width, 560px), calc(100vw - 24px));
  width: 100%;
  --md-dialog-container-shape: 24px;
  --md-dialog-headline-font: var(--font-sans);
  --md-dialog-supporting-text-font: var(--font-sans);
}

.md-dialog-headline {
  font-size: 1.125rem;
  font-weight: 600;
  color: var(--md-sys-color-on-surface);
}

.md-dialog-content {
  color: var(--md-sys-color-on-surface);
}

.md-dialog-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 8px;
}
</style>
