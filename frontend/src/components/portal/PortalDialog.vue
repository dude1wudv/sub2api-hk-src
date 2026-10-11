<template>
  <Teleport to="body">
    <dialog ref="dialog" class="portal-dialog" :class="{ wide }" :aria-labelledby="titleId" tabindex="-1" @cancel.prevent="close" @close="onNativeClose" @keydown.esc.stop @click="onBackdrop">
      <header class="portal-dialog-header"><h2 :id="titleId">{{ title }}</h2><button type="button" class="portal-inline-button" :aria-label="zh ? '关闭' : 'Close'" @click="close">✕</button></header>
      <slot />
    </dialog>
  </Teleport>
</template>
<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, useId, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useScrollLock } from '@/composables/useScrollLock'
const props = defineProps<{ open: boolean; title: string; wide?: boolean }>()
const emit = defineEmits<{ close: [] }>()
const { locale } = useI18n()
const zh = computed(() => locale.value.startsWith('zh'))
const dialog = ref<HTMLDialogElement | null>(null)
const titleId = useId()
const modalOpen = ref(false)
useScrollLock(modalOpen)
let previousFocus: HTMLElement | null = null
function restoreFocus() {
  if (previousFocus?.isConnected && !previousFocus.closest('[inert]')) previousFocus.focus({ preventScroll: true })
  previousFocus = null
}
function onNativeClose() {
  if (dialog.value?.open) return
  modalOpen.value = false
  restoreFocus()
  if (props.open) emit('close')
}
function close() { emit('close') }
function onBackdrop(event: MouseEvent) {
  if (event.target !== dialog.value || !dialog.value) return
  const rect = dialog.value.getBoundingClientRect()
  if (event.clientX < rect.left || event.clientX > rect.right || event.clientY < rect.top || event.clientY > rect.bottom) close()
}
watch(() => props.open, async (open) => {
  if (!open) {
    dialog.value?.close()
    modalOpen.value = false
    restoreFocus()
    return
  }
  await nextTick()
  if (open !== props.open) return
  if (open && dialog.value && !dialog.value.open) {
    previousFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null
    dialog.value.showModal()
    modalOpen.value = true
  }
}, { immediate: true })
onBeforeUnmount(() => {
  dialog.value?.close()
  modalOpen.value = false
  restoreFocus()
})
</script>
<style scoped>
dialog.portal-dialog { margin: auto; color: rgb(var(--ink)); background: rgb(var(--surface)); border-color: rgb(var(--line)); font-family: var(--font-sans); overscroll-behavior: contain; opacity: 0; transform: translateY(12px) scale(.98); transition: opacity var(--motion-overlay, 260ms), transform var(--motion-overlay, 260ms) var(--motion-ease, cubic-bezier(.2,.8,.2,1)), display var(--motion-overlay, 260ms) allow-discrete, overlay var(--motion-overlay, 260ms) allow-discrete; }
dialog.portal-dialog[open] { opacity: 1; transform: none; }
dialog.portal-dialog::backdrop { background: rgb(0 0 0 / .4); opacity: 0; transition: opacity var(--motion-overlay, 260ms), display var(--motion-overlay, 260ms) allow-discrete, overlay var(--motion-overlay, 260ms) allow-discrete; }
dialog.portal-dialog[open]::backdrop { opacity: 1; }
.portal-dialog-header h2 { font-family: var(--font-sans); }
.portal-dialog-header > button { flex-shrink: 0; }
@starting-style {
  dialog.portal-dialog[open] { opacity: 0; transform: translateY(12px) scale(.98); }
  dialog.portal-dialog[open]::backdrop { opacity: 0; }
}
@media (max-width: 767px) {
  dialog.portal-dialog { max-width: calc(100vw - 24px); max-height: calc(100dvh - max(24px, env(safe-area-inset-top, 0px)) - max(24px, env(safe-area-inset-bottom, 0px))); padding: 20px; }
  .portal-dialog-header > button { min-width: 44px; min-height: 44px; display: inline-grid; place-items: center; margin-right: -8px; }
  dialog.portal-dialog :deep(input:not([type='checkbox']):not([type='radio'])), dialog.portal-dialog :deep(select), dialog.portal-dialog :deep(textarea) { font-size: 16px; }
  dialog.portal-dialog :deep(.portal-dialog-actions button) { min-height: 44px; }
}
@media (prefers-reduced-motion: reduce) {
  dialog.portal-dialog, dialog.portal-dialog::backdrop { transition: none; transform: none; }
}
</style>
