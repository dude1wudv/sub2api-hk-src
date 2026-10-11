<template>
  <button ref="trigger" type="button" class="action-trigger" :aria-label="label" :title="label" aria-haspopup="menu" :aria-expanded="open" @click.stop="toggle" @keydown.down.prevent="show">
    <Icon name="more" size="sm" />
  </button>
  <Teleport to="body">
    <div v-if="open" ref="panel" role="menu" :aria-label="label" class="action-menu" :style="position" @keydown="onKeydown">
      <slot :close="close" />
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, nextTick, onUnmounted, watch } from 'vue'
import Icon from '@/components/icons/Icon.vue'
import { useColorMode } from '@/composables/useColorMode'
defineProps<{ label: string }>()
const { mode } = useColorMode()
const open = ref(false)
const trigger = ref<HTMLButtonElement>()
const panel = ref<HTMLDivElement>()
const position = ref({ left: '0px', top: '0px' })
const buttons = () => Array.from(panel.value?.querySelectorAll<HTMLButtonElement>('button:not(:disabled)') ?? [])

function close(restoreFocus = true) {
  open.value = false
  document.removeEventListener('pointerdown', outside)
  window.removeEventListener('resize', dismiss)
  window.removeEventListener('scroll', scroll, true)
  if (restoreFocus) trigger.value?.focus()
}
function dismiss() { close(false) }
function scroll(event: Event) {
  if (event.target instanceof Node && panel.value?.contains(event.target)) return
  dismiss()
}
function outside(event: PointerEvent) {
  if (event.target instanceof Node && !panel.value?.contains(event.target) && !trigger.value?.contains(event.target)) dismiss()
}
async function show() {
  if (open.value || !trigger.value) return
  open.value = true
  await nextTick()
  if (!open.value || !trigger.value || !panel.value) return
  const rect = trigger.value.getBoundingClientRect()
  const height = panel.value.offsetHeight
  const width = panel.value.offsetWidth
  position.value = {
    left: `${Math.max(12, Math.min(rect.right - width, window.innerWidth - width - 12))}px`,
    top: `${Math.max(12, rect.bottom + 8 + height <= window.innerHeight - 12 ? rect.bottom + 8 : rect.top - height - 8)}px`
  }
  buttons().forEach(button => { button.setAttribute('role', 'menuitem'); button.tabIndex = -1 })
  buttons()[0]?.focus()
  document.addEventListener('pointerdown', outside)
  window.addEventListener('resize', dismiss)
  window.addEventListener('scroll', scroll, true)
}
function toggle() { if (open.value) close(); else void show() }
function onKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') { event.preventDefault(); event.stopPropagation(); close(); return }
  if (event.key === 'Tab') { close(); return }
  if (!['ArrowDown', 'ArrowUp', 'Home', 'End'].includes(event.key)) return
  event.preventDefault()
  const items = buttons()
  if (!items.length) return
  const index = items.indexOf(document.activeElement as HTMLButtonElement)
  const next = event.key === 'Home' ? 0 : event.key === 'End' ? items.length - 1 : (index + (event.key === 'ArrowDown' ? 1 : -1) + items.length) % items.length
  items[next]?.focus()
}
watch(mode, dismiss)
onUnmounted(dismiss)
</script>

<style scoped>
.action-trigger { display: inline-flex; min-width: 44px; min-height: 44px; align-items: center; justify-content: center; border: 1px solid rgb(var(--line)); border-radius: 8px; color: rgb(var(--ink-2)); background: rgb(var(--surface)); transition: color var(--motion-fast) var(--motion-ease), background-color var(--motion-fast) var(--motion-ease), border-color var(--motion-fast) var(--motion-ease); }
.action-trigger:hover, .action-trigger[aria-expanded='true'] { color: rgb(var(--ink)); background: rgb(var(--surface-2)); border-color: rgb(var(--line-strong)); }
.action-trigger:focus-visible, .action-menu :deep(button:focus-visible) { outline: 2px solid rgb(var(--focus-ring)); outline-offset: 2px; }
.action-menu { position: fixed; z-index: 250; width: max-content; min-width: 180px; max-width: calc(100vw - 24px); max-height: calc(100dvh - 24px); overflow: auto; padding: 6px; border: 1px solid rgb(var(--line-strong)); border-radius: 12px; color: rgb(var(--ink)); background: rgb(var(--surface-elevated)); box-shadow: 0 12px 36px rgb(0 0 0 / .16); animation: menu-enter var(--motion-menu) var(--motion-ease); }
.action-menu :deep(button) { display: flex; width: 100%; min-height: 44px; align-items: center; gap: 9px; border: 0; border-radius: 6px; padding: 8px 12px; background: transparent; color: inherit; font: inherit; font-size: 14px; text-align: left; white-space: normal; cursor: pointer; transition: background-color var(--motion-fast) var(--motion-ease); }
.action-menu :deep(button:hover) { background: rgb(var(--surface-2)); }
.action-menu :deep(button[data-danger='true']) { color: rgb(var(--text-err)); }
.action-menu :deep(button:disabled) { opacity: .5; cursor: not-allowed; }
@keyframes menu-enter { from { opacity: 0; transform: translateY(-3px); } to { opacity: 1; transform: translateY(0); } }
@media (prefers-reduced-motion: reduce) { .action-trigger, .action-menu, .action-menu :deep(button) { animation: none; transition: none; } }
</style>
