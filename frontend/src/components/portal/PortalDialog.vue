<template>
  <Teleport to="body">
    <dialog ref="dialog" class="portal-dialog" :style="viewportStyle" :class="{ wide }" :aria-labelledby="titleId" tabindex="-1" @cancel.prevent="close" @close="onNativeClose" @keydown.esc.stop @click="onBackdrop">
      <header class="portal-dialog-header"><h2 :id="titleId">{{ title }}</h2><button type="button" class="portal-inline-button" :aria-label="zh ? '关闭' : 'Close'" @click="close">✕</button></header>
      <div class="portal-dialog-body"><slot /></div>
      <footer v-if="$slots.footer" class="portal-dialog-footer"><slot name="footer" /></footer>
    </dialog>
  </Teleport>
</template>
<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, provide, ref, useId, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useScrollLock } from '@/composables/useScrollLock'
import { useVisualViewport } from '@/composables/useVisualViewport'
import { overlayHostKey } from '@/composables/overlayHost'
const props = defineProps<{ open: boolean; title: string; wide?: boolean }>()
const emit = defineEmits<{ close: [] }>()
const { locale } = useI18n()
const zh = computed(() => locale.value.startsWith('zh'))
const dialog = ref<HTMLDialogElement | null>(null)
provide(overlayHostKey, dialog)
const titleId = useId()
const modalOpen = ref(false)
useScrollLock(modalOpen)
const viewport = useVisualViewport(() => props.open)
const viewportStyle = computed(() => {
  const rect = viewport.value
  return rect.width > 0 ? { top: `${rect.top}px`, left: `${rect.left}px`, right: `calc(100% - ${rect.right}px)`, bottom: `calc(100% - ${rect.bottom}px)`, '--visual-height': `${rect.height}px`, '--visual-width': `${rect.width}px` } : undefined
})
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
dialog.portal-dialog { margin: auto; height: fit-content; color: rgb(var(--ink)); background: rgb(var(--surface)); border-color: rgb(var(--line)); font-family: var(--font-sans); opacity: 0; transition: opacity var(--motion-overlay, 260ms), display var(--motion-overlay, 260ms) allow-discrete, overlay var(--motion-overlay, 260ms) allow-discrete; }
dialog.portal-dialog[open] { opacity: 1; display: flex; flex-direction: column; }
dialog.portal-dialog::backdrop { background: rgb(0 0 0 / .4); opacity: 0; transition: opacity var(--motion-overlay, 260ms), display var(--motion-overlay, 260ms) allow-discrete, overlay var(--motion-overlay, 260ms) allow-discrete; }
dialog.portal-dialog[open]::backdrop { opacity: 1; }
.portal-dialog-header h2 { min-width: 0; overflow-wrap: anywhere; font-family: var(--font-sans); }
.portal-dialog-header > button { flex-shrink: 0; }
.portal-dialog-header, .portal-dialog-footer { flex-shrink: 0; min-width: 0; }
.portal-dialog-body { min-width: 0; min-height: 0; overflow-y: auto; overscroll-behavior: contain; overflow-wrap: anywhere; }
@starting-style {
  dialog.portal-dialog[open] { opacity: 0; }
  dialog.portal-dialog[open]::backdrop { opacity: 0; }
}
@media (max-width: 767px), (max-width: 1023px) and (max-height: 500px) and (pointer: coarse) {
  dialog.portal-dialog { max-width: calc(var(--visual-width, 100vw) - 24px); max-height: calc(var(--visual-height, 100dvh) - 24px); padding: 20px; }
  dialog.portal-dialog.wide { width: var(--visual-width, 100vw); max-width: var(--visual-width, 100vw); height: var(--visual-height, 100dvh); max-height: var(--visual-height, 100dvh); border-radius: 0; padding: max(16px, env(safe-area-inset-top, 0px)) 16px max(16px, env(safe-area-inset-bottom, 0px)); }
  dialog.portal-dialog.wide .portal-dialog-body { flex: 1; }
  .portal-dialog-header > button { min-width: 44px; min-height: 44px; display: inline-grid; place-items: center; margin-right: -8px; }
  dialog.portal-dialog :deep(input:not([type='checkbox']):not([type='radio'])), dialog.portal-dialog :deep(select), dialog.portal-dialog :deep(textarea) { font-size: 16px; }
  dialog.portal-dialog :deep(.portal-dialog-actions button) { min-height: 44px; }
}
@media (prefers-reduced-motion: reduce) {
  dialog.portal-dialog, dialog.portal-dialog::backdrop { transition: none; }
}
</style>
