<template>
  <Teleport to="body">
    <dialog ref="element" class="portal-account-modal" :aria-labelledby="titleId" tabindex="-1" @cancel.prevent="close" @keydown.esc.stop @close="onNativeClose" @click="onBackdropClick">
      <header class="portal-account-modal__header">
        <h2 :id="titleId">{{ title }}</h2>
        <button type="button" class="portal-account-close" :aria-label="closeLabel" :disabled="busy" @click="close">×</button>
      </header>
      <slot />
    </dialog>
  </Teleport>
</template>

<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref, useId } from 'vue'
import { useScrollLock } from '@/composables/useScrollLock'

const props = withDefaults(defineProps<{ title: string; busy?: boolean; closeLabel?: string }>(), {
  busy: false,
  closeLabel: '关闭'
})
const emit = defineEmits<{ close: [] }>()
const element = ref<HTMLDialogElement | null>(null)
const titleId = useId()
const modalOpen = ref(false)
useScrollLock(modalOpen)
let previousFocus: HTMLElement | null = null
let unmounting = false

function restoreFocus() {
  if (previousFocus?.isConnected && !previousFocus.closest('[inert]')) previousFocus.focus({ preventScroll: true })
  previousFocus = null
}

function onNativeClose() {
  if (unmounting || element.value?.open) return
  modalOpen.value = false
  restoreFocus()
  emit('close')
}

function close() {
  if (!props.busy) emit('close')
}

function onBackdropClick(event: MouseEvent) {
  const dialog = element.value
  if (!dialog || event.target !== dialog) return
  const bounds = dialog.getBoundingClientRect()
  if (event.clientX < bounds.left || event.clientX > bounds.right || event.clientY < bounds.top || event.clientY > bounds.bottom) close()
}

onMounted(async () => {
  previousFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null
  await nextTick()
  if (unmounting) return
  element.value?.showModal()
  modalOpen.value = true
  element.value?.querySelector<HTMLElement>('[autofocus]:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled])')?.focus({ preventScroll: true })
})

onBeforeUnmount(() => {
  unmounting = true
  element.value?.close()
  modalOpen.value = false
  restoreFocus()
})
</script>

<style scoped>
.portal-account-modal { color: rgb(var(--ink)); background: rgb(var(--surface)); border-color: rgb(var(--line)); font-family: var(--font-sans); overscroll-behavior: contain; animation: account-dialog-enter var(--motion-overlay, 260ms) var(--motion-ease, cubic-bezier(.2,.8,.2,1)); }
.portal-account-modal::backdrop { animation: account-backdrop-enter var(--motion-overlay, 260ms) ease-out; }
@keyframes account-dialog-enter { from { opacity: 0; transform: translateY(12px) scale(.98); } to { opacity: 1; transform: none; } }
@keyframes account-backdrop-enter { from { opacity: 0; } to { opacity: 1; } }
@media (max-width: 767px) {
  .portal-account-modal { max-height: calc(100dvh - max(24px, env(safe-area-inset-top, 0px)) - max(24px, env(safe-area-inset-bottom, 0px))); }
  .portal-account-close { min-width: 44px; min-height: 44px; }
  .portal-account-modal :deep(input:not([type='checkbox']):not([type='radio'])), .portal-account-modal :deep(select), .portal-account-modal :deep(textarea) { font-size: 16px; }
}
@media (prefers-reduced-motion: reduce) { .portal-account-modal, .portal-account-modal::backdrop { animation: none; } }
</style>
